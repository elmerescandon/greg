#!/usr/bin/env bash
set -e

SESSION="greg-ui"
VAULT="${GREG_VAULT:-$HOME}"
UI_DIR="$(cd "$(dirname "$0")" && pwd)"

# Si ya existe la sesión, reattach
if tmux has-session -t "$SESSION" 2>/dev/null; then
  tmux attach -t "$SESSION"
  exit 0
fi

# Crear sesión nueva
tmux new-session -d -s "$SESSION"

# Layout 3 columnas: historial(15%) | claude(70%) | terminal(15%)
COLS=$(tput cols)
HIST_COLS=$((COLS * 15 / 100))
CLAUDE_COLS=$((COLS * 70 / 100))

tmux split-window -h -t "$SESSION:0.0"
tmux split-window -h -t "$SESSION:0.1"
tmux resize-pane -t "$SESSION:0.0" -x "$HIST_COLS"
tmux resize-pane -t "$SESSION:0.1" -x "$CLAUDE_COLS"

# Títulos y estilo
tmux select-pane -t "$SESSION:0.0" -T "sessions"
tmux select-pane -t "$SESSION:0.1" -T "claude-code"
tmux select-pane -t "$SESSION:0.2" -T "terminal"
tmux set-option -t "$SESSION" pane-border-status top
tmux set-option -t "$SESSION" pane-border-format " #{?pane_active,#[fg=white bold],#[fg=colour240]}#{pane_title}#[default] "
tmux set-option -t "$SESSION" pane-active-border-style "fg=white,bold"
tmux set-option -t "$SESSION" pane-border-style fg=colour240
tmux set-option -t "$SESSION" mouse on
tmux set-option -t "$SESSION" status off

# Arrancar procesos
tmux send-keys -t "$SESSION:0.0" "node '$UI_DIR/historial.js'" Enter
tmux send-keys -t "$SESSION:0.1" "node '$UI_DIR/claude-panel.js'" Enter

# Foco en claude y attach
# Al cerrar la pestaña/ventana, matar la sesión tmux automáticamente
tmux select-pane -t "$SESSION:0.1"
tmux attach -t "$SESSION"
