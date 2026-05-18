# greg

Multi-agent Claude Code manager with a terminal UI.

## What it is

**greg CLI** — spawn, message, and schedule Claude Code sessions from the terminal.  
**greg UI** — a 3-pane terminal workspace: session history | claude panel | shell.

```
┌──────────────┬──────────────────────────────────────┬──────────────┐
│   sessions   │           claude-code                │   terminal   │
│              │                                      │              │
│  ACTIVAS     │  ● claude  abc12345                  │              │
│  ● abc12345  │  main │ task-1                       │              │
│              │                                      │              │
│  MÉTRICAS    │  > your message here                 │              │
│  tokens 12k  │                                      │              │
│  costo $0.04 │                                      │              │
│              │                                      │              │
│  HISTORIAL   │                                      │              │
│  ○ old-task  │                                      │              │
└──────────────┴──────────────────────────────────────┴──────────────┘
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
git clone https://github.com/elmerescandon/greg
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

#### Claude panel

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Alt+Enter` | New line in input |
| `PgUp / PgDn` | Scroll output |
| `Ctrl+Shift+←/→` | Switch tabs |
| `Ctrl+T` | New tab (new Greg session) |
| `Ctrl+W` | Close current tab |
| `Ctrl+C` | Cancel running request |
| `Ctrl+Q` | Quit panel |

#### Sessions panel

| Key | Action |
|-----|--------|
| `Enter` | Open session in claude panel |
| `n` | New Greg session |
| `x` | Close selected session |
| `j / k` | Navigate list |

## How it works

- Sessions are stored in `~/.greg/sessions.json`
- Finished sessions move to `~/.greg/history.json` with their `claude_session_id`
- The UI reuses conversation context via `claude -p --resume <session_id>`
- Each session gets a mailbox at `~/.greg/mailbox/<id>/inbox.md`
- Token usage and cost are tracked per session and aggregated monthly in the metrics section

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
