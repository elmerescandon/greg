#!/usr/bin/env node
'use strict';

const blessed      = require('blessed');
const { execSync } = require('child_process');
const crypto       = require('crypto');
const fs           = require('fs');
const path         = require('path');
const os           = require('os');

const GREG_HOME     = path.join(os.homedir(), '.greg');
const SESSIONS_FILE = path.join(GREG_HOME, 'sessions.json');
const HISTORY_FILE  = path.join(GREG_HOME, 'history.json');
const MAILBOX_DIR   = path.join(GREG_HOME, 'mailbox');
const CMD_FILE      = path.join(GREG_HOME, 'ui-cmd.json');
const VAULT         = process.env.GREG_VAULT || os.homedir();
const TMUX_SESSION  = 'greg-ui';

const screen = blessed.screen({ smartCSR: true, fullUnicode: true });

const header = blessed.box({
  parent: screen,
  top: 0, left: 0, width: '100%', height: 1,
  tags: true, style: { bg: '#111' },
});

const list = blessed.list({
  parent: screen,
  top: 1, left: 0, width: '100%', height: '100%-2',
  keys: true, vi: true, tags: true,
  scrollable: true, alwaysScroll: true,
  style: {
    selected: { bg: 'cyan', fg: 'black', bold: true },
    item: { fg: 'white' },
  },
});

const metricsBar = { setContent: () => {} }; // eliminado, métricas en la lista

blessed.box({
  parent: screen,
  bottom: 0, left: 0, width: '100%', height: 1,
  tags: true,
  content: '  {cyan-fg}Enter{/} abrir  {cyan-fg}n{/} nueva  {cyan-fg}x{/} cerrar  {gray-fg}j/k navegar{/}',
  style: { bg: '#111' },
});

// ── Greg session management ───────────────────────────────────────────────────

function read(file) {
  try { return JSON.parse(fs.readFileSync(file, 'utf8')); }
  catch { return []; }
}

function write(file, data) {
  fs.writeFileSync(file, JSON.stringify(data, null, 2));
}

// Crea una sesión greg sin tmux (para el UI panel)
function gregSpawnUI() {
  const id      = `greg-${crypto.randomBytes(4).toString('hex')}`;
  const started = new Date().toISOString().replace('T', ' ').slice(0, 19);

  fs.mkdirSync(path.join(MAILBOX_DIR, id), { recursive: true });
  fs.writeFileSync(path.join(MAILBOX_DIR, id, 'inbox.md'), '');

  const sessions = read(SESSIONS_FILE);
  sessions.push({ id, dir: VAULT, started, status: 'active' });
  write(SESSIONS_FILE, sessions);

  return { id };
}

// Guarda claude_session_id en la sesión greg activa
function saveClaudeSession(gregId, claudeSessionId) {
  try {
    const sessions = read(SESSIONS_FILE);
    const s = sessions.find(s => s.id === gregId);
    if (s) {
      s.claude_session_id = claudeSessionId;
      write(SESSIONS_FILE, sessions);
    }
  } catch { /* ignore */ }
}

// ── UI helpers ────────────────────────────────────────────────────────────────

function tmuxAlive(id) {
  try { execSync(`tmux has-session -t "${id}"`, { stdio: 'pipe' }); return true; }
  catch { return false; }
}

function openInCenter(name, claudeSession, gregSessionId) {
  const cmd = {
    action: 'new-tab',
    name,
    claudeSession: claudeSession || null,
    gregSessionId: gregSessionId || null,
  };
  try { fs.writeFileSync(CMD_FILE, JSON.stringify(cmd)); } catch {}
  try { execSync(`tmux select-pane -t "${TMUX_SESSION}:0.1"`, { stdio: 'pipe' }); } catch {}
}

// ── Render ────────────────────────────────────────────────────────────────────

let itemData = [];

function refresh() {
  const sessions = read(SESSIONS_FILE);
  const history  = read(HISTORY_FILE).slice().reverse();

  const items = [];
  itemData = [];

  // ── Activas ───────────────────────────────────────────────────────────────
  items.push(' {bold}ACTIVAS{/}');
  itemData.push(null);

  if (sessions.length === 0) {
    items.push('  {gray-fg}(ninguna){/}');
    itemData.push(null);
  } else {
    for (const s of sessions) {
      const alive   = tmuxAlive(s.id);
      const dot     = alive ? '{green-fg}●{/}' : '{yellow-fg}●{/}';
      const shortId = s.id.replace(/^greg-/, '');
      const time    = (s.started || '').slice(5, 16);
      items.push(`  ${dot} ${shortId} {gray-fg}${time}{/}`);
      itemData.push({ type: 'active', s, alive });
    }
  }

  // ── Métricas ──────────────────────────────────────────────────────────────
  const allSessions = [...sessions, ...history];
  const thisMonth = new Date().toISOString().slice(0, 7);
  const monthSessions = allSessions.filter(s => (s.started || '').startsWith(thisMonth));
  const totalSessions = allSessions.length;
  const monthOutputTokens = monthSessions.reduce((sum, s) => sum + (s.output_tokens || 0), 0);
  const monthCost = monthSessions.reduce((sum, s) => sum + (s.cost_usd || 0), 0);
  const fmtTokens = monthOutputTokens >= 1000
    ? `${(monthOutputTokens / 1000).toFixed(1)}k`
    : String(monthOutputTokens);

  items.push('');
  itemData.push(null);
  items.push(' {bold}MÉTRICAS{/}');
  itemData.push(null);
  items.push(`  {gray-fg}sesiones totales{/}   {white-fg}${totalSessions}{/}`);
  itemData.push(null);
  items.push(`  {gray-fg}output tokens/mes{/}  {white-fg}${fmtTokens}{/}`);
  itemData.push(null);
  items.push(`  {gray-fg}costo/mes{/}          {white-fg}$${monthCost.toFixed(3)}{/}`);
  itemData.push(null);

  // ── Historial ─────────────────────────────────────────────────────────────
  items.push('');
  itemData.push(null);
  items.push(' {bold}HISTORIAL{/}');
  itemData.push(null);

  if (history.length === 0) {
    items.push('  {gray-fg}(vacío){/}');
    itemData.push(null);
  } else {
    for (const h of history.slice(0, 30)) {
      const shortId = h.id.replace(/^greg-/, '');
      const started = (h.started || '').slice(5, 16);
      const ended   = (h.ended   || '').slice(11, 16);
      const resume  = h.claude_session_id ? '{gray-fg}↩{/}' : ' ';
      items.push(`  {gray-fg}○{/} ${resume} ${shortId} {gray-fg}${started}→${ended}{/}`);
      itemData.push({ type: 'history', h });
    }
  }

  // ── Métricas ──────────────────────────────────────────────────────────────
  metricsBar.setContent('');

  header.setContent(
    `  {bold}{cyan-fg}greg{/}{/} {gray-fg}${sessions.length} activa${sessions.length !== 1 ? 's' : ''}{/}`
  );
  list.setItems(items);
  screen.render();
}

// ── Eventos ───────────────────────────────────────────────────────────────────

function killSession(s) {
  // Matar tmux si existe
  try { execSync(`tmux kill-session -t "${s.id}"`, { stdio: 'pipe' }); } catch {}

  // Quitar de sessions.json
  const sessions = read(SESSIONS_FILE).filter(x => x.id !== s.id);
  write(SESSIONS_FILE, sessions);

  // Mover a history.json con ended
  const history = read(HISTORY_FILE);
  history.push({
    ...s,
    status: 'finished',
    ended: new Date().toISOString().replace('T', ' ').slice(0, 19),
  });
  write(HISTORY_FILE, history);

  refresh();
}

list.on('select', (_item, idx) => {
  const d = itemData[idx];
  if (!d) return;

  if (d.type === 'active') {
    openInCenter(d.s.id.replace(/^greg-/, ''), d.s.claude_session_id || null, d.s.id);

  } else if (d.type === 'history') {
    openInCenter(d.h.id.replace(/^greg-/, ''), d.h.claude_session_id || null, d.h.id);
  }
});

// x → cerrar sesión activa seleccionada (no si es la última)
screen.key(['x'], () => {
  const d = itemData[list.selected];
  if (!d || d.type !== 'active') return;
  const activeSessions = read(SESSIONS_FILE);
  if (activeSessions.length <= 1) return;
  killSession(d.s);
});

// n → nueva sesión greg + abre tab en el centro
screen.key(['n'], () => {
  const { id } = gregSpawnUI();
  openInCenter(id.replace(/^greg-/, ''), null, id);
});

screen.key(['C-q'], () => process.exit(0));

// ── Auto-refresh: fs.watch para cambios instantáneos ─────────────────────────

refresh();

// Reacciona inmediatamente cuando greg modifica sus archivos
try {
  fs.watch(GREG_HOME, { persistent: false }, (event, filename) => {
    if (filename === 'sessions.json' || filename === 'history.json') {
      refresh();
    }
  });
} catch {}

// Fallback polling
setInterval(refresh, 3000);

list.focus();
screen.render();
