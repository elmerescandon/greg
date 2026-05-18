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

function gregSpawn() {
  const id      = `greg-${crypto.randomBytes(4).toString('hex')}`;
  const started = new Date().toISOString().replace('T', ' ').slice(0, 19);
  try {
    fs.mkdirSync(path.join(MAILBOX_DIR, id), { recursive: true });
    fs.writeFileSync(path.join(MAILBOX_DIR, id, 'inbox.md'), '');
    const sessions = JSON.parse(fs.readFileSync(SESSIONS_FILE, 'utf8') || '[]');
    sessions.push({ id, dir: VAULT, started, status: 'active' });
    fs.writeFileSync(SESSIONS_FILE, JSON.stringify(sessions, null, 2));
  } catch {}
  return { id };
}

// ── CLI args ──────────────────────────────────────────────────────────────────
const cliArgs         = process.argv.slice(2);
const argClaudeSession = cliArgs[cliArgs.indexOf('--claude-session') + 1] || null;

// ── Screen ────────────────────────────────────────────────────────────────────
const screen = blessed.screen({ smartCSR: true, fullUnicode: true, title: 'claude', mouse: true });

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

const MAX_INPUT_LINES = 6;

const output = blessed.log({
  parent: screen, top: 2, left: 0,
  width: '100%', bottom: 5,
  tags: true, scrollable: true, alwaysScroll: false,
  padding: { left: 1, right: 1 },
  scrollbar: { ch: '▐', style: { fg: '#333' } },
});

screen.on('mouse', data => {
  if (data.action === 'wheelup') {
    tab().scrollLock = true;
    output.scroll(-2);
    screen.render();
  } else if (data.action === 'wheeldown') {
    output.scroll(2);
    if (output.getScrollPerc() >= 99) tab().scrollLock = false;
    screen.render();
  }
});

const inputLine = blessed.box({
  parent: screen, bottom: 2, left: 0,
  width: '100%', height: 3,
  tags: true, wrap: true,
  padding: { left: 1, right: 1 },
  style: { bg: '#1a1a1a' },
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
  content: '  {gray-fg}Enter enviar  Alt+Enter nueva línea  Ctrl+↑/↓ scroll  PgUp/PgDn saltar  Ctrl+Shift+←/→ tabs  Ctrl+T nueva  Ctrl+W cerrar  Ctrl+K compactar  Ctrl+Q salir{/}',
  style: { bg: '#111' },
});

// ── Tab management ────────────────────────────────────────────────────────────
function createTab(name, claudeSession) {
  return {
    name,
    claudeSession: claudeSession || null,
    content: '',
    lines: [],
    scrollLock: false,
    hasNew: false,
    inputBuf: '',
    cursorPos: 0,
    running: false,
    proc: null,
    cost: 0,
    contextPct: null,
    compactWarned: false,
    compactPending: false,
  };
}

// Cargar sesión activa previa de Greg, o crear una nueva registrada
function buildInitialTab() {
  if (argClaudeSession) {
    const t = createTab('manual', argClaudeSession);
    return t;
  }
  try {
    const sessions = JSON.parse(fs.readFileSync(SESSIONS_FILE, 'utf8'));
    if (sessions.length > 0) {
      const active = sessions.filter(s => s.status === 'active').pop() || sessions[sessions.length - 1];
      const t = createTab(active.id.replace(/^greg-/, ''), active.claude_session_id || null);
      t.gregSessionId = active.id;
      return t;
    }
  } catch {}
  // Solo crear nueva si no existe ninguna sesión
  const { id } = gregSpawn();
  const t = createTab(id.replace(/^greg-/, ''), null);
  t.gregSessionId = id;
  return t;
}

const tabs = [buildInitialTab()];
let tabIdx = 0;

function tab() { return tabs[tabIdx]; }

function renderTabBar() {
  const parts = tabs.map((t, i) => {
    const active  = i === tabIdx;
    const spinner = t.running ? `{yellow-fg}${FRAMES[spinIdx]}{/} ` : '';
    const badge   = !active && t.hasNew ? ' {green-fg}●{/}' : '';
    if (active) return `{bold}{cyan-fg} ${t.name} {/}{/}${spinner}`;
    return `{gray-fg} ${t.name}${badge} {/}`;
  });
  tabBar.setContent('  ' + parts.join('{gray-fg} │ {/}'));
  screen.render();
}

function switchTab(newIdx) {
  if (newIdx < 0 || newIdx >= tabs.length) return;
  // Guardar estado del tab actual
  if (tabs[tabIdx]) {
    tabs[tabIdx].content   = output.getContent();
    tabs[tabIdx].inputBuf  = inputBuf;
    tabs[tabIdx].cursorPos = cursorPos;
  }
  tabIdx    = newIdx;
  inputBuf  = tab().inputBuf;
  cursorPos = tab().cursorPos;
  tab().hasNew = false;
  output.setContent(tab().content);
  for (const line of tab().lines) output.log(line);
  tab().lines = [];
  output.setScrollPerc(100);
  renderTabBar();
  renderStatus();

  if (tab().compactPending && !tab().running) {
    tab().compactPending = false;
    tab().compactWarned  = false;
    tabLog('{red-fg}⚡ contexto al límite — escribe qué quieres preservar y presiona Enter{/}');
    inputBuf  = '/compact ';
    cursorPos = inputBuf.length;
    syncInput();
  }

  renderInput();
  screen.render();
}

function closeTab() {
  if (tabs.length === 1) return;
  const t = tab();
  if (t.gregSessionId) spawn('greg', ['kill', t.gregSessionId], { stdio: 'ignore' }).unref();
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
    tabLog(`{gray-fg}retomando ${name} (contexto preservado){/}`);
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

function ctxColor(pct) {
  if (pct >= 90) return `{red-fg}ctx:${pct}%{/}`;
  if (pct >= 75) return `{yellow-fg}ctx:${pct}%{/}`;
  return `{gray-fg}ctx:${pct}%{/}`;
}

function renderStatus() {
  const t   = tab();
  const sid = t.gregSessionId ? t.gregSessionId.replace(/^greg-/, '') : '—';
  const cost = globalCost > 0 ? ` {gray-fg}$${globalCost.toFixed(3)}{/}` : '';
  const ctx  = t.contextPct !== null ? ` ${ctxColor(t.contextPct)}` : '';

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

function tabLog(line, t) {
  if (!t || t === tab()) {
    output.log(line);
    if (!tab().scrollLock) output.setScrollPerc(100);
  } else {
    t.lines.push(line);
    t.hasNew = true;
    renderTabBar();
  }
}

// ── Input ─────────────────────────────────────────────────────────────────────
let inputBuf = '';
let cursorPos = 0;
const inputHistory = [];
let historyIdx = -1;
let savedInput = '';

function renderInput() {
  const rawLines = inputBuf.split('\n');
  const visibleLines = Math.min(MAX_INPUT_LINES, rawLines.length);
  const inputHeight = visibleLines + 2;
  inputLine.height = inputHeight;
  inputLine.bottom = 2;
  output.bottom = inputHeight + 2;

  // Renderizar con cursor
  const before     = inputBuf.slice(0, cursorPos);
  const cursorChar = inputBuf[cursorPos];
  const after      = inputBuf.slice(cursorPos + (cursorChar !== undefined ? 1 : 0));

  let full;
  if (cursorChar === '\n') {
    full = escTags(before) + '{inverse} {/}\n' + escTags(after);
  } else if (cursorChar !== undefined) {
    full = escTags(before) + `{inverse}${escTags(cursorChar)}{/}` + escTags(after);
  } else {
    full = escTags(before) + '{inverse} {/}';
  }

  const lines = full.split('\n');
  const content = lines
    .map((l, i) => (i === 0 ? `{gray-fg}>{/} ${l}` : `  ${l}`))
    .join('\n');
  inputLine.setContent(content);
  screen.render();
}

function syncInput() {
  tab().inputBuf  = inputBuf;
  tab().cursorPos = cursorPos;
}

// ── Send ──────────────────────────────────────────────────────────────────────
function send(text) {
  const t = tab();
  if (!text.trim() || t.running) return;

  if (text.trim()) { inputHistory.push(text); historyIdx = -1; savedInput = ''; }
  t.running = true;
  inputBuf  = '';
  cursorPos = 0;
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
    if (s) { tabLog(`{red-fg}${escTags(s)}{/}`, t); screen.render(); }
  });

  t.proc.on('close', () => {
    t.running = false;
    t.proc    = null;
    currentAction = '';
    const anyRunning = tabs.some(tb => tb.running);
    if (!anyRunning) stopSpinner();
    tabLog('', t);
    renderStatus();
    renderTabBar();

    if (t.compactPending) {
      tabLog('{red-fg}⚡ contexto al límite — escribe qué quieres preservar y presiona Enter{/}', t);
      screen.render();
      if (t === tab()) {
        t.compactPending = false;
        t.compactWarned  = false;
        inputBuf  = '/compact ';
        cursorPos = inputBuf.length;
        syncInput();
        renderInput();
      }
      // Si es background tab, compactPending queda activo hasta que se haga switch
    }
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
          b.text.split('\n').forEach(l => tabLog(escTags(l), t));
        }
        if (b.type === 'tool_use') {
          const label = formatToolLabel(b.name, b.input);
          currentAction = `${b.name}…`;
          tabLog(`{yellow-fg}⚙ ${b.name}{/} {gray-fg}${escTags(label)}{/}`, t);
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
            lines.forEach(l => tabLog(`  {gray-fg}${escTags(l)}{/}`, t));
            if (txt.split('\n').length > 6) tabLog(`  {gray-fg}…(${txt.split('\n').length} líneas){/}`, t);
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

          if (t.contextPct < 75) {
            t.compactWarned = false;
            t.compactPending = false;
          } else if (t.contextPct >= 95) {
            t.compactPending = true;
          } else if (t.contextPct >= 90 && !t.compactWarned) {
            t.compactWarned = true;
            tabLog(`{yellow-fg}⚠ contexto al ${t.contextPct}% — Ctrl+K para compactar{/}`, t);
            screen.render();
          }
        }
        // Acumular output tokens y costo en la sesión de Greg
        if (t.gregSessionId) {
          try {
            const outputTokens = Object.values(ev.modelUsage)
              .reduce((sum, m) => sum + (m.outputTokens || 0), 0);
            const sdata = JSON.parse(fs.readFileSync(SESSIONS_FILE, 'utf8'));
            const s = sdata.find(s => s.id === t.gregSessionId);
            if (s) {
              s.output_tokens = (s.output_tokens || 0) + outputTokens;
              s.cost_usd = (s.cost_usd || 0) + (ev.total_cost_usd || 0);
              fs.writeFileSync(SESSIONS_FILE, JSON.stringify(sdata, null, 2));
            }
          } catch {}
        }
      }
      if (ev.subtype === 'error') {
        tabLog(`{red-fg}Error: ${escTags(String(ev.error || ''))}{/}`, t);
        screen.render();
      }
      break;
    }
  }
}

// ── IPC desde historial ───────────────────────────────────────────────────────
// Inicializar con mtime actual para ignorar comandos previos al inicio
let cmdMtime = 0;
try { cmdMtime = fs.statSync(CMD_FILE).mtimeMs; } catch {}
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

  if (key.full === 'C-k') {
    const t = tab();
    if (!t.running) {
      t.compactWarned  = false;
      t.compactPending = false;
      inputBuf  = '/compact ';
      cursorPos = inputBuf.length;
      syncInput();
      renderInput();
    }
    return;
  }

  // Scroll del output
  if (key.name === 'pageup') {
    tab().scrollLock = true;
    output.scroll(-Math.floor(output.height / 2));
    screen.render();
    return;
  }
  if (key.name === 'pagedown') {
    output.scroll(Math.floor(output.height / 2));
    if (output.getScrollPerc() >= 99) tab().scrollLock = false;
    screen.render();
    return;
  }
  if (key.name === 'up' && key.ctrl) {
    tab().scrollLock = true;
    output.scroll(-3);
    screen.render();
    return;
  }
  if (key.name === 'down' && key.ctrl) {
    output.scroll(3);
    if (output.getScrollPerc() >= 99) tab().scrollLock = false;
    screen.render();
    return;
  }

  // Navegar tabs — Ctrl+Shift+Flechas para no colisionar con Mission Control de macOS
  const isCtrlShiftRight = key.full === 'C-S-right' || (key.ctrl && key.shift && key.name === 'right') || key.sequence === '\x1b[1;6C';
  const isCtrlShiftLeft  = key.full === 'C-S-left'  || (key.ctrl && key.shift && key.name === 'left')  || key.sequence === '\x1b[1;6D';
  if (isCtrlShiftRight) { switchTab(tabIdx + 1); return; }
  if (isCtrlShiftLeft)  { switchTab(tabIdx - 1); return; }

  // Nueva tab (registra en greg)
  if (key.full === 'C-t') {
    const { id } = gregSpawn();
    newTab(id.replace(/^greg-/, ''), null, id);
    return;
  }

  // Cerrar tab actual
  if (key.full === 'C-w') { closeTab(); return; }

  if (tab().running) return;

  // Historial de inputs con ↑/↓
  if (key.name === 'up' && !key.ctrl && !key.shift && !key.meta) {
    if (inputHistory.length === 0) return;
    if (historyIdx === -1) savedInput = inputBuf;
    historyIdx = Math.min(historyIdx + 1, inputHistory.length - 1);
    inputBuf = inputHistory[inputHistory.length - 1 - historyIdx];
    cursorPos = inputBuf.length;
    syncInput(); renderInput(); return;
  }
  if (key.name === 'down' && !key.ctrl && !key.shift && !key.meta) {
    if (historyIdx === -1) return;
    historyIdx--;
    inputBuf  = historyIdx === -1 ? savedInput : inputHistory[inputHistory.length - 1 - historyIdx];
    cursorPos = inputBuf.length;
    syncInput(); renderInput(); return;
  }

  // Cursor ←/→
  if (key.name === 'left'  && !key.ctrl && !key.shift) { cursorPos = Math.max(0, cursorPos - 1); syncInput(); renderInput(); return; }
  if (key.name === 'right' && !key.ctrl && !key.shift) { cursorPos = Math.min(inputBuf.length, cursorPos + 1); syncInput(); renderInput(); return; }
  if (key.name === 'home') { cursorPos = 0; syncInput(); renderInput(); return; }
  if (key.name === 'end')  { cursorPos = inputBuf.length; syncInput(); renderInput(); return; }

  // Alt+Enter = nueva línea en el input
  if ((key.full === 'M-return' || key.full === 'M-enter' || (key.meta && (key.name === 'return' || key.name === 'enter')))) {
    inputBuf = inputBuf.slice(0, cursorPos) + '\n' + inputBuf.slice(cursorPos);
    cursorPos++;
    syncInput(); renderInput(); return;
  }

  if (key.name === 'enter' || key.name === 'return') {
    send(inputBuf);
    return;
  }

  if (key.name === 'backspace') {
    if (cursorPos > 0) {
      inputBuf  = inputBuf.slice(0, cursorPos - 1) + inputBuf.slice(cursorPos);
      cursorPos--;
    }
  } else if (ch && !key.ctrl && !key.meta && ch.length === 1) {
    inputBuf  = inputBuf.slice(0, cursorPos) + ch + inputBuf.slice(cursorPos);
    cursorPos++;
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
const t = tab();

renderStatus();
renderTabBar();
tabLog('{gray-fg}claude panel — escribe tu mensaje y presiona Enter{/}');
if (t.claudeSession) tabLog(`{gray-fg}sesión previa cargada: ${t.claudeSession.slice(0, 8)}…{/}`);
tabLog('');
renderInput();
screen.render();
