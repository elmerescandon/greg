# Changelog

All notable changes to this project will be documented in this file.

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
