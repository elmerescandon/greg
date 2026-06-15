# greg

Orchestrate teams of Claude Code agents that collaborate in parallel.

## The problem

A single Claude Code session hits its limits fast on big tasks — research reports, codebase audits, multi-file migrations. You run out of context, wait too long, or end up copy-pasting outputs between sessions manually. That doesn't scale.

## How greg solves it

greg spawns multiple Claude Code agents that work **in parallel** on a shared workspace. Each agent owns a perspective, writes progressively, and can read what the others are producing in real time.

- **Parallel agents** — split a task across specialists that run simultaneously, each in its own tmux session
- **Shared workspace** — agents read and reference each other's outputs as they write; no copy-paste, no handoffs
- **Automatic director** — a coordinator agent is injected into every task to monitor progress, unblock teammates, and produce synthesis notes
- **Monitor without interrupting** — `greg peek` shows you what any agent (or all agents in a task) is doing right now, without attaching to their session
- **Fault tolerant** — if an agent's session crashes before writing `done`, the coordinator detects it and auto-completes after 120 seconds
- **Terminal-native** — bash CLI + Go TUI, no external services, no API keys beyond Claude Code itself

## Quick start

```bash
# Launch a 3-agent research task
greg task run \
  --goal "Research report on the state of AI regulation worldwide" \
  --agent "americas:Cover US, Canada, Brazil, and LATAM regulations. Skip Europe and Asia." \
  --agent "europe:Cover EU AI Act, UK, and European national frameworks. Skip other regions." \
  --agent "asia:Cover China, Japan, South Korea, India, and Singapore. Skip other regions."

# A director agent is added automatically to coordinate the team.
# All 4 agents start working in parallel immediately.

# Check what they're doing
greg peek mtask-xxxxxxxx

# ━━━ director (greg-a1b2c3d4) ━━━
# ⏺ Reading americas output... cross-referencing with europe findings on
#   bilateral AI safety agreements.
# ✻ Brewed for 3m 12s
#
# ━━━ americas (greg-e5f6a7b8) ━━━
# ⏺ Writing section on Brazil's AI regulatory framework...
# ✻ Crunched for 4m 5s
# ...

# When all agents finish, the director leaves synthesis notes:
cat ~/.greg/multi-tasks/mtask-xxxxxxxx/workspace/director-synthesis-notes.md
```

## CLI reference

### Sessions

```bash
greg spawn                                          # new session in $GREG_VAULT
greg spawn --name "refactor" --prompt "refactor auth module"
greg list                                           # sessions, tasks, and recent history
greg peek <session-id> [-n 50]                      # last N lines from a session (default 30)
greg attach <session-id>                            # attach to tmux session
greg send --to <session-id> "add error handling"    # send a message
greg kill <session-id>                              # stop and archive
greg resume <session-id>                            # resume a finished session
greg history [--limit N]                            # show last N finished sessions
greg schedule --prompt "..." --at "2026-07-01 09:00"  # one-shot scheduled task
greg cancel <task-id>                               # cancel a scheduled task
```

### Multi-agent tasks

```bash
greg task run --goal "..." --agent "id:role" [--agent ...]
greg task status <task-id>                # per-agent status, coordinator health, tmux state
greg task list                            # all tasks with statuses
greg peek <task-id> [-n 30]               # tail all agents at once
greg task message <task-id> "redirect X"  # send a message to the director
greg task done <task-id> <agent-id>       # force-mark an agent as done
greg task close <task-id>                 # close task (requires all agents done)
greg task resume <task-id> <agent-id>     # resume a finished agent with full context
```

## How multi-agent tasks work

```
~/.greg/multi-tasks/mtask-xxxxxxxx/
  manifest.json                    # task metadata, agent roles, session IDs
  workspace/
    director.md                    # director's coordination log
    director-synthesis-notes.md    # final consolidated output
    americas.md                    # each agent writes here progressively
    europe.md
    asia.md
  messages/
    americas→director.md           # agents message each other
    director→americas.md
  status/
    director.status                # working | waiting | needs-help | done
    americas.status
    ...
```

1. `greg task run` spawns all agents in parallel, each in its own tmux session
2. A **director** agent is auto-injected to coordinate, cross-pollinate, and synthesize
3. Agents write to `workspace/<agent-id>.md` progressively — teammates can read in real time
4. Agents communicate via `messages/<from>→<to>.md` when they need to coordinate
5. A background **coordinator** process polls `status/` files every 15 seconds
6. When all agents (including the director) write `done`, the coordinator closes the task

## Setup

### Requirements

- [Claude Code](https://docs.anthropic.com/claude-code) (`claude` in PATH)
- tmux
- Go 1.26+ (for the TUI)
- jq

### Install

```bash
git clone https://github.com/elmerescandon/greg
cd greg

# Add CLI to PATH
ln -s "$(pwd)/cli/greg" ~/.local/bin/greg

# Set your default working directory
echo 'export GREG_VAULT="/path/to/your/project"' >> ~/.zshrc
source ~/.zshrc
```

### TUI (optional)

```bash
cd ui-v2
go build -o greg-ui .
ln -s "$(pwd)/greg-ui" ~/.local/bin/greg-ui
```

The TUI has two views: **Chat** (`Ctrl+1`) and **Agente** (`Ctrl+2`).

The Agente tab shows task details with animated ASCII tamagotchi sprites per agent:

```
┌──────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐
│       ]=[            │  │      (o_o)            │  │      (-_-)           │
│      (^o^)           │  │       ⌨▒░             │  │       zzZ            │
│  director   working  │  │  americas   working   │  │  europe    waiting   │
└──────────────────────┘  └──────────────────────┘  └──────────────────────┘
```

Below the desks: navigable message channel tabs and a chat panel for reading/sending messages.

| Key | Action |
|-----|--------|
| `←/→` | Switch message channel |
| `↑/↓` | Scroll chat |
| `f` or `i` | Focus chat input |
| `Enter` | Send message (in input) / View agent output (in nav) |
| `Esc` | Cancel input / Go back |

### Skills

greg injects prompt templates (skills) into each agent to define their behavior:

| Skill | Purpose |
|-------|---------|
| `greg-mailbox.md` | Workspace and messaging protocol — injected into every agent |
| `greg-director.md` | Director: coordinate team, cross-pollinate findings, write synthesis |
| `greg-teammate.md` | Specialist: progressive writing, proactive reading, status protocol |
| `greg-task` | `/greg-task` — interactive skill to design and launch a multi-agent task |
| `greg-learn` | `/greg-learn` — consolidate learnings from a conversation into persistent memory |

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
