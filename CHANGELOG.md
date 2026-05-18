# Changelog

All notable changes to this project will be documented in this file.

## [0.3.1] - 2026-05-17

### Added

**UI — claude-panel.js**
- `AskUserQuestion` interactive widget — when Greg generates selection prompts, a bordered overlay appears above the input showing the question and options; navigate with `↑/↓`, confirm with `Enter`, cancel with `Esc`
- Multi-question sequences supported: questions are presented one at a time, answers collected and written to the process stdin as JSON on completion
- Overlay position adjusts dynamically when input box height changes (multiline input)
- Overlay auto-hides if the process closes while a question is pending

### Changed
- UI branding renamed from "claude" to "Greg" across status bar, title, welcome and new-session messages
- Process `stdio` changed from `['ignore', ...]` to `['pipe', ...]` to support mid-run stdin writes for tool interactions

## [0.3.0] - 2026-05-17

### Added

**Multi-agent task system (`greg task`)**
- `greg task run --goal "..." --agent "id:role" [--agent ...]` — launch a team of parallel agents; director is auto-added
- `greg task status <task-id>` — per-agent status, tmux state, coordinator health
- `greg task list` — flat list of all multi-agent tasks
- `greg task recover <task-id>` — force-complete crashed agents and restart coordinator
- `greg task revise <task-id> --agent <session-id> --message "<feedback>"` — resume a finished agent with feedback
- Background coordinator polls status files every 15s; when all agents write `done`, triggers a final synthesizer
- Crash recovery: if an agent's tmux session dies before writing `done`, coordinator auto-completes it after 120s
- Synthesizer produces `final-output.md` from all agent outputs and director synthesis notes

**Skills system**
- `skills/greg-mailbox.md` — shared workspace/messaging protocol injected into every agent prompt
- `skills/greg-director.md` — director agent prompt: coordinate team, cross-pollinate, unblock agents, trigger synthesis
- `skills/greg-teammate.md` — specialist agent prompt: progressive writing, proactive reading, status protocol
- `_resolve_skill()` — resolves `{{> greg-mailbox}}` partials and `{{TASK_ID}}`, `{{AGENT_ID}}`, `{{AGENT_ROLE}}`, `{{TASK_GOAL}}`, `{{WORKSPACE}}` variables

**UI — claude-panel.js**
- Visible cursor in input with `←/→` navigation and `Home`/`End` keys
- Input history: `↑/↓` to cycle through sent messages
- `Ctrl+K` — pre-fill `/compact ` for guided context compaction
- Context color: gray <75%, yellow ≥75%, red ≥90%
- Compaction warnings: alert at 90%, auto-prompt at 95%; pre-fills `/compact ` when turn ends with context at limit
- Tab badge — green dot on inactive tabs with unread output
- `compactPending` preserved across tab switches; prompt appears when switching to the tab
- `Ctrl+↑/↓` — scroll output line by line

### Changed
- `greg list` now groups multi-agent tasks separately (with per-agent status) above standalone sessions
- Mouse scroll handling moved to `screen.on('mouse', ...)` for more reliable trackpad support
- `closeTab()` now calls `greg kill` instead of sending SIGINT directly to the process
- `cmdMtime` initialized to current file mtime on startup to ignore stale IPC commands from previous sessions
- Footer help bar updated with all new keybindings

## [0.2.0] - 2026-05-18

### Added
- **Tabs aislados** — output de cada sesión buffereado independientemente; múltiples sesiones concurrentes ya no se mezclan en el mismo panel
- **Scroll con touchpad** — soporte de mouse habilitado en el panel de Claude, scroll natural con trackpad
- **Scroll con teclado** — `PgUp` / `PgDn` para navegar el output; `scrollLock` inteligente que pausa el auto-scroll al subir y lo reactiva al llegar al fondo
- **Input dinámico** — la caja de input crece de 1 a 6 líneas según el contenido; `Alt+Enter` inserta saltos de línea
- **Métricas en el panel de sesiones** — sección MÉTRICAS con sesiones totales, output tokens del mes y costo mensual en USD, embebida en la lista de sesiones
- **Captura de tokens** — `claude-panel.js` guarda `output_tokens` y `cost_usd` por sesión en `sessions.json` al recibir el evento `result`

### Changed
- **Navegación entre tabs** — cambiado de `Ctrl+←/→` a `Ctrl+Shift+←/→` para evitar conflicto con Mission Control de macOS
- **Sesión inicial** — al abrir el UI carga la última sesión activa de Greg en vez de crear una pestaña "main" genérica
- **Layout** — proporciones ajustadas a 15% / 70% / 15% (sessions | claude | terminal) usando `resize-pane` con columnas exactas para mayor precisión
- **Historial** — métricas movidas de la barra inferior a la lista principal, eliminando la barra negra vacía

### Fixed
- Sesiones duplicadas al reiniciar el UI cuando ya existía una sesión activa

## [0.1.0] - 2026-05-17

### Added
- **greg CLI** — spawn, list, attach, send, kill y schedule de sesiones Claude Code
- **greg UI** — workspace de 3 paneles en tmux: historial de sesiones | panel Claude | terminal
- Panel de Claude con soporte de múltiples tabs, spinner de actividad, barra de estado con costo y contexto
- Historial de sesiones con navegación vi (`j/k`), apertura con `Enter`, cierre con `x`, nueva sesión con `n`
- IPC entre `historial.js` y `claude-panel.js` vía `~/.greg/ui-cmd.json`
- Persistencia de sesiones en `~/.greg/sessions.json` y `~/.greg/history.json`
- Reanudación de contexto Claude via `--resume <session_id>`
- Configuración de directorio de trabajo vía `GREG_VAULT`
