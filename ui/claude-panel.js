#!/usr/bin/env node
'use strict';

const blessed   = require('blessed');
const { spawn } = require('child_process');
const fs        = require('fs');
const path      = require('path');
const os        = require('os');

const VAULT        = process.env.GREG_VAULT || os.homedir();
const GREG_HOME    = path.join(os.homedir(), '.greg');
const SESSION_FILE = path.join(GREG_HOME, 'claude-ui-session');
const CMD_FILE     = path.join(GREG_HOME, 'ui-cmd.json');
const SESSIONS_FILE = path.join(GREG_HOME, 'sessions.json');
const MAILBOX_DIR  = path.join(GREG_HOME, 'mailbox');

// ── Greg session helpers ──────────────────────────────────────────────────────
const crypto = require('crypto');

function gregSpawn(name) {
  const id      = `greg-${crypto.randomBytes(4).toString('hex')}`;
  const label   = name || id;
  const started = new Date().toISOString().replace('T', ' ').slice(0, 19);
  try {
    fs.mkdirSync(path.join(MAILBOX_DIR, id), { recursive: true });
    fs.writeFileSync(path.join(MAILBOX_DIR, id, 'inbox.md'), '');
    const sessions = JSON.parse(fs.readFileSync(SESSIONS_FILE, 'utf8') || '[]');
    sessions.push({ id, name: label, dir: VAULT, started, status: 'active' });
    fs.writeFileSync(SESSIONS_FILE, JSON.stringify(sessions, null, 2));
  } catch {}
  return { id, name: label };
}

// ── CLI args ──────────────────────────────────────────────────────────────────
const cliArgs         = process.argv.slice(2);
const argClaudeSession = cliArgs[cliArgs.indexOf('--claude-session') + 1] || null;

// ── Screen ────────────────────────────────────────────────────────────────────
const screen = blessed.screen({ smartCSR: true, fullUnicode: true, title: 'claude' });

const statusBar = blessed.box({
  parent: screen, top: 0, left: 0,
  width: '100%', height: 1,
  tags: true, style: { bg: '#111' },
});

const tabBar = blessed.box({
  parent: screen, top: 1, left: 0,
  width: '100%', height: 1,
  tags: true, style: { bg: '#0d0d0d' },
});

const output = blessed.log({
  parent: screen, top: 2, left: 0,
  width: '100%', height: '100%-5',
  tags: true, scrollable: true, alwaysScroll: true,
  padding: { left: 1, right: 1 },
});

const inputLine = blessed.box({
  parent: screen, bottom: 2, left: 0,
  width: '100%', height: 1,
  tags: true, style: { bg: '#1a1a1a' },
});

blessed.line({
  parent: screen, bottom: 1, left: 0,
  width: '100%', orientation: 'horizontal',
  style: { fg: '#333' },
});

blessed.box({
  parent: screen, bottom: 0, left: 0,
  width: '100%', height: 1,
  tags: true,
  content: '  {gray-fg}Enter enviar  Ctrl+←/→ tabs  Ctrl+T nueva  Ctrl+W cerrar  Ctrl+C cancelar  Ctrl+Q salir{/}',
  style: { bg: '#111' },
});

// ── Tab management ────────────────────────────────────────────────────────────
function createTab(name, claudeSession) {
  return {
    name,
    claudeSession: claudeSession || null,
    content: '',
    inputBuf: '',   // input buffer por tab
    running: false,
    proc: null,
    cost: 0,
    contextPct: null,
  };
}

const tabs = [createTab('main', argClaudeSession)];
let tabIdx = 0;

function tab() { return tabs[tabIdx]; }

function renderTabBar() {
  const parts = tabs.map((t, i) => {
    const active  = i === tabIdx;
    const spinner = t.running ? `{yellow-fg}${FRAMES[spinIdx]}{/} ` : '';
    if (active) return `{bold}{cyan-fg} ${t.name} {/}{/}${spinner}`;
    return `{gray-fg} ${t.name} {/}`;
  });
  tabBar.setContent('  ' + parts.join('{gray-fg} │ {/}'));
  screen.render();
}

function switchTab(newIdx) {
  if (newIdx < 0 || newIdx >= tabs.length) return;
  // Guardar estado del tab actual
  if (tabs[tabIdx]) {
    tabs[tabIdx].content  = output.getContent();
    tabs[tabIdx].inputBuf = inputBuf;
  }
  tabIdx   = newIdx;
  inputBuf = tab().inputBuf;
  // Restaurar contenido
  output.setContent(tab().content);
  output.setScrollPerc(100);
  renderTabBar();
  renderStatus();
  renderInput();
  screen.render();
}

function closeTab() {
  if (tabs.length === 1) return;
  const t = tab();
  if (t.proc) t.proc.kill('SIGINT');
  const newIdx = tabIdx > 0 ? tabIdx - 1 : 0;
  tabs.splice(tabIdx, 1);
  tabIdx   = newIdx;
  inputBuf = tab().inputBuf;
  output.setContent(tab().content);
  output.setScrollPerc(100);
  renderTabBar();
  renderStatus();
  renderInput();
  screen.render();
}

function newTab(name, claudeSession, gregSessionId) {
  const t = createTab(name, claudeSession || null);
  t.gregSessionId = gregSessionId || null;
  tabs.push(t);
  switchTab(tabs.length - 1);
  if (claudeSession) {
    tabLog(`{gray-fg}sesión: ${claudeSession.slice(0, 8)}… (contexto preservado){/}`);
    tabLog('');
  } else {
    tabLog('{gray-fg}nueva sesión de claude{/}');
    tabLog('');
  }
  renderInput();
  screen.render();
}

// ── Session ───────────────────────────────────────────────────────────────────
let globalCost = 0;

function renderStatus() {
  const t   = tab();
  const sid = t.claudeSession ? t.claudeSession.slice(0, 8) : '—';
  const cost = globalCost > 0 ? ` {gray-fg}$${globalCost.toFixed(3)}{/}` : '';
  const ctx  = t.contextPct !== null ? ` {gray-fg}ctx:${t.contextPct}%{/}` : '';

  if (t.running) {
    statusBar.setContent(`  {yellow-fg}${FRAMES[spinIdx]}{/} {bold}claude{/} {gray-fg}${sid}{/}${currentAction ? ` {gray-fg}${escTags(currentAction)}{/}` : ''}${cost}${ctx}`);
  } else {
    statusBar.setContent(`  {green-fg}●{/} {bold}claude{/} {gray-fg}${sid}{/}${cost}${ctx}`);
  }
  screen.render();
}

// ── Spinner ───────────────────────────────────────────────────────────────────
const FRAMES = ['⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏'];
let spinIdx  = 0;
let spinTimer = null;

function startSpinner() {
  if (spinTimer) return;
  spinTimer = setInterval(() => {
    spinIdx = (spinIdx + 1) % FRAMES.length;
    renderStatus();
    renderTabBar();
  }, 80);
}

function stopSpinner() {
  if (spinTimer) { clearInterval(spinTimer); spinTimer = null; }
}

// ── Output helpers ────────────────────────────────────────────────────────────
let currentAction = '';

function tabLog(line) {
  output.log(line);
}

// ── Input ─────────────────────────────────────────────────────────────────────
let inputBuf = '';  // sincronizado con tab().inputBuf

function renderInput() {
  inputLine.setContent(`{gray-fg}>{/} ${escTags(inputBuf)}`);
  screen.render();
}

function syncInput() {
  tab().inputBuf = inputBuf;
}

// ── Send ──────────────────────────────────────────────────────────────────────
function send(text) {
  const t = tab();
  if (!text.trim() || t.running) return;

  t.running = true;
  inputBuf  = '';
  currentAction = 'pensando…';
  renderInput();

  tabLog('');
  tabLog(`{cyan-fg}▶ ${escTags(text)}{/}`);
  tabLog('');
  startSpinner();

  const args = [
    '-p', text,
    '--output-format', 'stream-json',
    '--verbose',
    '--dangerously-skip-permissions',
  ];
  if (t.claudeSession) args.push('--resume', t.claudeSession);

  t.proc = spawn('claude', args, { cwd: VAULT, stdio: ['ignore', 'pipe', 'pipe'] });
  let buf = '';

  t.proc.stdout.on('data', chunk => {
    buf += chunk.toString();
    const lines = buf.split('\n');
    buf = lines.pop();
    lines.forEach(line => { if (line.trim()) handleEvent(line, t); });
  });

  t.proc.stderr.on('data', chunk => {
    const s = chunk.toString().trim();
    if (s) { tabLog(`{red-fg}${escTags(s)}{/}`); screen.render(); }
  });

  t.proc.on('close', () => {
    t.running = false;
    t.proc    = null;
    currentAction = '';
    const anyRunning = tabs.some(tb => tb.running);
    if (!anyRunning) stopSpinner();
    tabLog('');
    renderStatus();
    renderTabBar();
  });
}

// ── Eventos ───────────────────────────────────────────────────────────────────
function handleEvent(raw, t) {
  let ev;
  try { ev = JSON.parse(raw); } catch { return; }

  if (ev.session_id && ev.session_id !== t.claudeSession) {
    t.claudeSession = ev.session_id;
    // Guardar en archivo si es el tab principal
    if (tabIdx === 0 && !argClaudeSession) {
      try { fs.writeFileSync(SESSION_FILE, ev.session_id); } catch {}
    }
    // Sincronizar de vuelta a sessions.json de greg
    if (t.gregSessionId) {
      try {
        const SESSIONS = path.join(os.homedir(), '.greg', 'sessions.json');
        const sessions = JSON.parse(fs.readFileSync(SESSIONS, 'utf8'));
        const s = sessions.find(s => s.id === t.gregSessionId);
        if (s) {
          s.claude_session_id = ev.session_id;
          fs.writeFileSync(SESSIONS, JSON.stringify(sessions, null, 2));
        }
      } catch {}
    }
  }

  switch (ev.type) {
    case 'assistant': {
      const blocks = ev.message?.content || [];
      for (const b of blocks) {
        if (b.type === 'text' && b.text) {
          currentAction = '';
          b.text.split('\n').forEach(l => tabLog(escTags(l)));
        }
        if (b.type === 'tool_use') {
          const label = formatToolLabel(b.name, b.input);
          currentAction = `${b.name}…`;
          tabLog(`{yellow-fg}⚙ ${b.name}{/} {gray-fg}${escTags(label)}{/}`);
        }
      }
      screen.render();
      break;
    }
    case 'user': {
      const blocks = ev.message?.content || [];
      for (const b of blocks) {
        if (b.type === 'tool_result') {
          currentAction = 'pensando…';
          const txt = extractResult(b);
          if (txt) {
            const lines = txt.split('\n').slice(0, 6);
            lines.forEach(l => tabLog(`  {gray-fg}${escTags(l)}{/}`));
            if (txt.split('\n').length > 6) tabLog(`  {gray-fg}…(${txt.split('\n').length} líneas){/}`);
          }
        }
      }
      screen.render();
      break;
    }
    case 'result': {
      if (ev.total_cost_usd) globalCost += ev.total_cost_usd;
      if (ev.modelUsage) {
        const m = Object.values(ev.modelUsage).find(v => v.contextWindow);
        if (m) {
          const used = (m.inputTokens || 0) + (m.cacheReadInputTokens || 0) + (m.cacheCreationInputTokens || 0);
          t.contextPct = Math.round(used / m.contextWindow * 100);
        }
      }
      if (ev.subtype === 'error') {
        tabLog(`{red-fg}Error: ${escTags(String(ev.error || ''))}{/}`);
        screen.render();
      }
      break;
    }
  }
}

// ── IPC desde historial ───────────────────────────────────────────────────────
let cmdMtime = 0;
setInterval(() => {
  try {
    const stat = fs.statSync(CMD_FILE);
    if (stat.mtimeMs <= cmdMtime) return;
    cmdMtime = stat.mtimeMs;
    const cmd = JSON.parse(fs.readFileSync(CMD_FILE, 'utf8'));
    if (cmd.action === 'new-tab') newTab(cmd.name || 'session', cmd.claudeSession || null, cmd.gregSessionId || null);
  } catch { /* ignore */ }
}, 300);

// ── Teclado ───────────────────────────────────────────────────────────────────
screen.on('keypress', (ch, key) => {
  if (!key) return;

  if (key.full === 'C-q') { process.exit(0); return; }

  if (key.full === 'C-c') {
    const t = tab();
    if (t.proc) { t.proc.kill('SIGINT'); tabLog('{gray-fg}cancelado{/}'); screen.render(); }
    return;
  }

  // Navegar tabs
  if (key.full === 'C-right') { switchTab(tabIdx + 1); return; }
  if (key.full === 'C-left')  { switchTab(tabIdx - 1); return; }

  // Nueva tab (registra en greg)
  if (key.full === 'C-t') {
    const count = tabs.length + 1;
    const { id, name } = gregSpawn(`sesión-${count}`);
    newTab(name, null, id);
    return;
  }

  // Cerrar tab actual
  if (key.full === 'C-w') { closeTab(); return; }

  if (tab().running) return;

  if (key.name === 'enter' || key.name === 'return') {
    send(inputBuf);
    return;
  }

  if (key.name === 'backspace') {
    inputBuf = inputBuf.slice(0, -1);
  } else if (ch && !key.ctrl && !key.meta && ch.length === 1) {
    inputBuf += ch;
  }

  syncInput();
  renderInput();
});

// ── Helpers ───────────────────────────────────────────────────────────────────
function escTags(s) { return String(s).replace(/\{/g, '\\{'); }

function formatToolLabel(name, input) {
  if (!input) return '';
  if (input.command)     return input.command.slice(0, 100);
  if (input.path)        return input.path;
  if (input.file_path)   return input.file_path;
  if (input.description) return input.description.slice(0, 80);
  return JSON.stringify(input).slice(0, 80);
}

function extractResult(b) {
  if (Array.isArray(b.content)) return b.content.map(c => c.text || '').join('').trim();
  if (typeof b.content === 'string') return b.content.trim();
  return '';
}

// ── Init ──────────────────────────────────────────────────────────────────────
// Cargar sesión guardada para el tab main
const t = tab();
if (!t.claudeSession) {
  try { t.claudeSession = fs.readFileSync(SESSION_FILE, 'utf8').trim(); } catch {}
}

renderStatus();
renderTabBar();
tabLog('{gray-fg}claude panel — escribe tu mensaje y presiona Enter{/}');
if (t.claudeSession) tabLog(`{gray-fg}sesión: ${t.claudeSession.slice(0, 8)}… (contexto preservado){/}`);
tabLog('');
renderInput();
screen.render();
