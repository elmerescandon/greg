# greg

Multi-agent Claude Code manager with a terminal UI.

## What it is

**greg CLI** — spawn, message, and schedule Claude Code sessions from the terminal.  
**greg UI** — a 3-pane terminal workspace: session history | claude panel | shell.

```
┌──────────────┬──────────────────────────────────────────────┬──────────┐
│   sessions   │  main  │  task-1 ●                           │ terminal │
│              │  ⠿ claude  a1b2c3  $0.042  ctx:68% · 136k/200k│          │
│  ACTIVAS     ├──────────────────────────────────────────────┤          │
│  ● a1b2c3    │                                              │          │
│  ● task-1    │  output scrollable...                        │          │
│              │                                              │          │
│  MÉTRICAS    │                                              │          │
│  12k tokens  │                                              │          │
│  $0.04/mes   ├──────────────────────────────────────────────┤          │
│              │  > your message here_                        │          │
│  HISTORIAL   ├──────────────────────────────────────────────┤          │
│  ○ finished  │  Enter  Alt+Enter  Ctrl+K  Ctrl+W  Ctrl+Q   │          │
└──────────────┴──────────────────────────────────────────────┴──────────┘
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

### Multi-agent tasks

Launch a team of parallel agents that collaborate via a shared workspace:

```bash
greg task run \
  --goal "Research report on LLMs in 2026" \
  --agent "models:Analyze benchmarks and model capabilities" \
  --agent "research:Analyze academic research trends" \
  --agent "industry:Analyze industry adoption and use cases"

greg task list                         # show all tasks with agent statuses grouped
greg task status mtask-xxxxxxxx        # detailed status: agents, coordinator, sessions
greg task recover mtask-xxxxxxxx       # unblock a task if an agent crashed mid-work
greg task revise mtask-xxxxxxxx \
  --agent greg-xxxxxxxx \
  --message "Go deeper on section 3"  # resume a finished agent with feedback
```

**How it works:**

1. A **director** agent is auto-added to coordinate the team
2. All agents run in parallel, each in its own tmux session
3. Agents write to `~/.greg/multi-tasks/<task-id>/workspace/<agent>.md` progressively
4. Agents can message each other via `messages/<from>→<to>.md`
5. A background **coordinator** polls status files every 15s
6. When all agents (including director) mark `done`, the coordinator marks the task as `completed`
7. Director leaves synthesis notes in `workspace/director-synthesis-notes.md`

**Messaging:** Send messages to the director mid-task:

```bash
greg task message mtask-xxxxxxxx "How is the analysis going?"
```

**Monitoring:**

```bash
greg peek mtask-xxxxxxxx           # tail last 30 lines of all agents
greg peek greg-xxxxxxxx            # tail a specific agent session
greg peek mtask-xxxxxxxx -n 50     # custom line count
```

**Resuming agents:**

```bash
greg task resume mtask-xxxxxxxx interaction   # resume a finished agent
```

### Skills

| Skill | Purpose |
|-------|---------|
| `greg-mailbox.md` | Workspace/messaging protocol — injected into every agent |
| `greg-director.md` | Director agent: coordinate team, cross-pollinate, trigger synthesis |
| `greg-teammate.md` | Specialist agent: progressive writing, proactive reading, status protocol |
| `greg-task` | `/greg-task` — design and launch a multi-agent task interactively |
| `greg-revise` | `/greg-revise` — close a revision session and archive it cleanly |
| `greg-learn` | `/greg-learn` — consolidate learnings into persistent memory |

### UI (Go — ui-v2)

```bash
cd ui-v2 && go build -o greg-ui . && ./greg-ui
```

Two views: **Chat** (Ctrl+1) and **Agente** (Ctrl+2).

#### Agente tab — Office View

When viewing a task detail, agents appear as animated ASCII tamagotchi desks:

```
┌──────────────────────┐  ┌──────────────────────┐
│       ]=[            │  │      (o_o)            │
│      (^o^)           │  │       ⌨▒░             │
│  director   working  │  │  datos      working   │
│  role: coordinator   │  │  role: data gathering  │
└──────────────────────┘  └──────────────────────┘
```

Below the desks: navigable message channel tabs and a chat panel for reading/sending messages.

| Key | Action |
|-----|--------|
| `←/→` | Switch message channel |
| `↑/↓` | Scroll chat |
| `f` or `i` | Focus chat input |
| `Enter` | Send message (in input) / View agent output (in nav) |
| `Esc` | Cancel input / Go back |

#### Claude panel

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Alt+Enter` | New line in input |
| `↑ / ↓` | Navigate input history |
| `← / →` | Move cursor in input |
| `Home / End` | Jump to start/end of input |
| `PgUp / PgDn` | Scroll output (big jump) |
| `Ctrl+↑ / Ctrl+↓` | Scroll output (line by line) |
| `Ctrl+K` | Pre-fill `/compact ` for guided context compaction |
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
