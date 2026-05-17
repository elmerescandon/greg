# greg

Multi-agent Claude Code manager with a terminal UI.

## What it is

**greg CLI** — spawn, message, and schedule Claude Code sessions from the terminal.  
**greg UI** — a 3-pane terminal workspace: session history | claude panel | shell.

```
┌─────────────────┬──────────────────────────┬──────────────┐
│   sessions      │      claude-code          │   terminal   │
│                 │                           │              │
│  ACTIVAS        │  ● claude  abc12345       │              │
│  ● my-task      │  main │ task-1            │              │
│                 │                           │              │
│  HISTORIAL      │  > your message here      │              │
│  ○ old-task     │                           │              │
└─────────────────┴──────────────────────────┴──────────────┘
```

## Requirements

- [Claude Code](https://docs.anthropic.com/claude-code) (`claude` in PATH)
- tmux
- Node.js 18+
- jq

## Setup

### 1. Set your working directory

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
export GREG_VAULT="/path/to/your/project"
```

This is the directory Claude Code will work in by default.

### 2. Install the CLI

```bash
# Clone the repo
git clone https://github.com/your-username/greg
cd greg

# Add CLI to PATH
ln -s "$(pwd)/cli/greg" /usr/local/bin/greg

# Install UI dependencies
cd ui && npm install
```

### 3. Add the UI alias

```bash
echo 'alias greg-ui="bash /path/to/greg/ui/greg-ui.sh"' >> ~/.zshrc
source ~/.zshrc
```

### 4. Configure Ghostty navigation (optional)

Add to `~/.config/ghostty/config`:

```
macos-option-as-alt = true
keybind = ctrl+super+left=text:\x02\x1b[D
keybind = ctrl+super+right=text:\x02\x1b[C
keybind = ctrl+super+up=text:\x02\x1b[A
keybind = ctrl+super+down=text:\x02\x1b[B
```

## Usage

### CLI

```bash
greg spawn                          # new Claude Code session in $GREG_VAULT
greg spawn --name "my-task" --prompt "refactor the auth module"
greg list                           # list active sessions and scheduled tasks
greg attach greg-xxxxxxxx           # attach to a session
greg send --to greg-xxxxxxxx "add error handling"
greg kill greg-xxxxxxxx             # stop and archive session
greg schedule --prompt "run tests" --at "2026-01-15 09:00"
```

### UI

```bash
greg-ui
```

| Key | Action |
|-----|--------|
| `Ctrl+Cmd+←/→` | Navigate panes |
| `n` | New greg session (in sessions pane) |
| `x` | Close selected session |
| `Enter` | Open session in claude panel |
| `Ctrl+T` | New claude tab |
| `Ctrl+W` | Close current tab |
| `Ctrl+←/→` | Switch tabs |
| `Ctrl+Q` | Quit panel |

## How it works

- Sessions are stored in `~/.greg/sessions.json`
- Finished sessions move to `~/.greg/history.json` with their `claude_session_id`
- The UI reuses conversation context via `claude -p --resume <session_id>`
- Each session gets a mailbox at `~/.greg/mailbox/<id>/inbox.md`
