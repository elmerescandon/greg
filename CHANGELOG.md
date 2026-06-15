# Changelog

All notable changes to this project will be documented in this file.

## [0.4.3] - 2026-06-15

### Added

**CLI**
- `--preset <coding|research>` flag in `greg task run` — behavioral modes that shape how agents operate without fixing team composition:
  - `coding`: auto-creates an isolated git worktree at `/tmp/greg-worktree-<task_id>` and spawns all agents there; injects structured coding standards (git workflow, build/test verification, quality principles) via `skills/greg-preset-coding.md`
  - `research`: keyword-detects agent role (collector vs analyzer) and injects role-specific methodology skill files (`skills/greg-preset-research-collector.md`, `skills/greg-preset-research-analyzer.md`); collectors gather raw evidence only, analyzers work exclusively from workspace to prevent anchoring bias
- `--model <alias|id>` flag in `greg task run`, `greg spawn`, and `greg start` — passes `--model <id>` directly to the Claude invocation; `cmd_start` inherits for free via delegation to `cmd_spawn`
- `_resolve_model` helper — maps short aliases to full model IDs: `opus→claude-opus-4-8`, `sonnet→claude-sonnet-4-6`, `haiku→claude-haiku-4-5-20251001`; full IDs pass through unchanged

**Skills**
- `greg-preset-coding.md` — standalone skill injected into coding-preset agents; covers git workflow, build/test verification checklist, quality standards, and cross-agent collaboration protocol
- `greg-preset-research-collector.md` — standalone skill for collector-role agents; enforces raw evidence gathering with structured output format (source, date, quality flag, content, contradictions)
- `greg-preset-research-analyzer.md` — standalone skill for analyzer-role agents; enforces workspace-only analysis, explicit confidence levels, bias detection checklist, and structured conclusions format

## [0.4.2] - 2026-06-15

### Removed
- `ui/` (Node.js/blessed) — deleted from the repo
- `ui-v2/` renamed to `ui/` — the Go/bubbletea UI is now simply `ui/`

## [0.4.1] - 2026-06-15

### Fixed

**TUI (ui-v2)**
- Model persistence per session — selected model is saved to `sessions.json` and restored when resuming (previously always reset to Opus 4.6 on restart)
- Sidebar sessions opened from the session list now also restore their saved model
- Config dialog (`Ctrl+T`) pre-selects the current tab's model instead of always highlighting Opus 4.6
- Default model changed from `claude-opus-4-6` to `claude-sonnet-4-6` for all new sessions
- Completed multi-agent task sessions are now automatically moved from `sessions.json` to `history.json` when the sidebar refreshes — no more orphaned "active" sessions from finished tasks

## [0.4.0] - 2026-06-15

### Added

**CLI**
- `greg peek <session-id|task-id> [-n lines]` — show last N lines (default 30) from a tmux pane, with ANSI escape sequences stripped for clean piped output; pass a task ID to peek all agents at once
- `capture_id` mechanism for reliable `claude_session_id` lookup — a unique marker is prepended to each agent's prompt and later grepped from `.jsonl` files, replacing the fragile mtime-based approach
- `{{SESSION_ID}}` template variable in skill resolution

**TUI**
- `ui-v2/` — new terminal UI written in Go with bubbletea, replacing the Node.js/blessed UI; includes Chat and Agente tabs
- Agente tab — Office View: animated ASCII tamagotchi sprites per agent status (`(o_o) ⌨▒░` working, `(-_-) zzZ` waiting, `(o_O)!` needs-help, `(^_^) ✔✔` done, `]=[ (^o^)` director)
- Message channel tabs navigable with `←/→` arrows, displaying all `messages/*.md` from the task workspace
- Chat panel with scrollable message history and markdown header highlighting
- Chat input: `f`/`i` to focus, `Enter` to send via `greg task message`, `Esc` to cancel and return to navigation

### Changed
- Task system v2 (`schema_version: 2`): the synthesizer agent is removed; the director now produces consolidated output directly in `workspace/director-synthesis-notes.md`
- Coordinator closes the task when all agents (including director) write `done` — no extra synthesizer step
- `task recover` and `task revise` replaced by `task done`, `task close`, `task message`, and `task resume`
- Updated `greg-task` skill docs and `greg-director.md` to reflect the new flow

### Deprecated
- `ui/` (Node.js/blessed) — the active UI is now `ui-v2/` (Go/bubbletea)

## [0.3.3] - 2026-05-31

### Added
- UI — model and effort selector on new tabs

## [0.3.2] - 2026-05-20

### Fixed
- UI — `claude-panel.js`: write `\n` to stdin immediately after spawn to satisfy claude's 3-second stdin check and eliminate the "Warning: no stdin data received in 3s" message that appeared on every message send; prompt is delivered via `-p`, not stdin, so `AskUserQuestion` stdin writes are unaffected

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
