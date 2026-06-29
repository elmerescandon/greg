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
- **Fault tolerant** — if an agent's session crashes before writing `done`, the coordinator moves it to `review` after 120 seconds so the director can verify before closing
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
greg kill <session-id>                              # stop and archive
greg resume <session-id>                            # resume a finished session
greg history [--limit N]                            # show last N finished sessions
greg schedule --prompt "..." --at "2026-07-01 09:00"  # one-shot scheduled task
greg cancel <task-id>                               # cancel a scheduled task
```

### Multi-agent tasks

```bash
greg task run --goal "..." --agent "id:role" [--agent ...] [--criteria-file "id:/path/to/criteria.md" ...] [--preset coding|research] [--model alias|id]
greg task status <task-id>                # per-agent status, coordinator health, tmux state
greg task list                            # all tasks with statuses
greg peek <task-id> [-n 30]               # tail all agents at once
greg task message <task-id> "redirect X"  # send a message to the director
greg task done <task-id> <agent-id>       # force-mark an agent as done
greg task close <task-id>                 # close task (requires all agents done)
greg task resume <task-id> <agent-id>     # resume a finished agent with full context

# Agent messaging (used inside agent sessions, via skills)
greg send-msg --from <id> --to <id> --workspace <path> "message"   # send with timestamp
greg wait-msg --agent <id> --workspace <path> [--timeout <secs>]   # block until message arrives
greg check-msgs --agent <id> --workspace <path>                     # drain unread messages
```

### Model aliases

`--model` accepts short aliases or full IDs:

| Alias | Model ID |
|-------|----------|
| `opus` | `claude-opus-4-8` |
| `sonnet` | `claude-sonnet-4-6` |
| `haiku` | `claude-haiku-4-5-20251001` |

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
    americas.status                # working | waiting | needs-help | review | done
    ...
```

1. `greg task run` spawns all agents in parallel, each in its own tmux session
2. A **director** agent is auto-injected to coordinate, cross-pollinate, and synthesize
3. Agents write to `workspace/<agent-id>.md` progressively — teammates can read in real time
4. Agents communicate via `greg send-msg` — each message is appended with a timestamp; `greg wait-msg` blocks deterministically on `fswatch` until a reply arrives; `greg check-msgs` drains unread messages
5. A background **coordinator** process polls `status/` files every 15 seconds
6. When a specialist finishes, it moves to `review`; the director verifies against `<agent-id>.criteria.md` and writes a verdict to `<agent-id>.review.md`, then moves the agent to `done` (or back to `working` with gaps)
7. When all agents write `done`, the coordinator closes the task

## Setup

### Requirements

- [Claude Code](https://docs.anthropic.com/claude-code) (`claude` in PATH)
- tmux
- Go 1.22+ (for the TUI)
- jq
- fswatch (optional — enables event-driven `wait-msg`; falls back to 2s polling)

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

### TUI

```bash
cd ui
go build -o greg-ui .
ln -s "$(pwd)/greg-ui" ~/.local/bin/greg-ui
```

The TUI is a Go/bubbletea terminal app with four views:

| Tab | Key | Description |
|-----|-----|-------------|
| **Chat** | `Ctrl+1` | Session sidebar + Claude output panel |
| **Agente** | `Ctrl+2` / `Ctrl+Space` | Multi-agent task browser with Command Center |
| **Métricas** | `Ctrl+3` | Usage charts |
| **Config** | `Ctrl+4` | Persistent preferences |

---

#### Chat view (`Ctrl+1`)

The main interface. Left sidebar lists standalone sessions; right panel shows the active Claude output.

**Sidebar navigation**

| Key | Action |
|-----|--------|
| `Shift+↑/↓` | Navigate sessions |
| `Shift+Enter` | Open selected session |

**Chat panel**

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Alt+Enter` | Insert newline |
| `↑/↓` | Scroll output |
| `PgUp/PgDn` | Scroll by page |
| `Alt+↑/↓` | Navigate input history |
| `Ctrl+K` | Pre-fill `/compact` |
| `Ctrl+V` | Paste from clipboard |

---

#### Agente view (`Ctrl+2`)

Task list with status, ID, creation date, and goal. Coding tasks are marked with `⌨`.

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate tasks |
| `PgUp/PgDn` | Scroll task list |
| `Enter` | Open Command Center for selected task |
| `x` | Close task (if all agents are done) |

**Command Center** — opened with `Enter` on a task:

Shows a table of all agents (status, messages in/out), workspace files, and message channels.

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate agents / files / messages |
| `Enter` | Open selected file (glamour-rendered) or peek agent's tmux session |
| `Esc` | Go back |

**File / tmux peek sub-views**

| Key | Action |
|-----|--------|
| `↑/↓`, `PgUp/PgDn` | Scroll |
| `Esc` | Go back |

---

#### Config view (`Ctrl+4`)

Preferences are persisted to `~/.config/greg/config.json`.

| Setting | Options | Default |
|---------|---------|---------|
| Tema | Dark mode / Light mode | Dark |
| Modelo por defecto | opus / sonnet / haiku | sonnet-4-6 |
| Esfuerzo por defecto | low / medium / high / xhigh / max | high |
| Umbral compactación | 70–95% | 90% |
| Auto-compact | activado / desactivado | desactivado |
| Timeout idle | nunca / 4h / 8h / 24h | nunca |

Navigate with `↑/↓`, change values with `←/→`.

---

#### Global keys

| Key | Action |
|-----|--------|
| `Ctrl+T` | New session tab |
| `Ctrl+W` | Close current tab |
| `Ctrl+Shift+←/→` | Switch between tabs |
| `Ctrl+Q` | Quit |

---

## Presets

### `--preset coding`

Creates an isolated git worktree at `/tmp/greg-worktree-<task_id>` and gives each agent role a specific skill:

- **Specialists** receive `coding/workflow` — atomic commits with conventional format, build verification per stack, no push
- **Director** receives `coding/director` — waits for all specialists to finish, runs integration build, resolves conflicts, does the single `git push` and creates the consolidated PR with `gh pr create`

The director is the only agent that pushes and opens a PR. Specialists only commit.

### `--preset research`

Detects agent role by keyword (`collector` vs `analyzer`) and injects role-specific methodology:

- **Collector** agents gather raw evidence only — structured output with source, date, quality flag, content, and contradictions
- **Analyzer** agents work exclusively from workspace files to prevent anchoring bias; produce explicit confidence levels and bias detection checklist

---

## Skills

greg injects prompt templates (skills) into each agent to define their behavior.

**Agent templates** — injected by the CLI into every task:

| File | Purpose |
|------|---------|
| `agents/mailbox.md` | Workspace and messaging protocol — injected into every agent; exposes `send-msg`, `wait-msg`, `check-msgs` |
| `agents/director.md` | Director: coordinate team, cross-pollinate findings, write synthesis |
| `agents/teammate.md` | Specialist: progressive writing, proactive reading, status protocol |

**Invocable skills** — used by humans via `/skill-name`, also injected by presets:

| Skill path | Invocation | Purpose |
|------------|-----------|---------|
| `coding/workflow/SKILL.md` | `--preset coding` (specialists) | Git workflow, build/test checklist, atomic commits |
| `coding/director/SKILL.md` | `--preset coding` (director) | Integration build, push, and consolidated PR |
| `coding/issue/SKILL.md` | `/greg-issue` | Build a well-defined issue before handing work to a coding agent |
| `research/collector/SKILL.md` | `--preset research` (collectors) | Raw evidence gathering protocol |
| `research/analyzer/SKILL.md` | `--preset research` (analyzers) | Workspace-only analysis, bias detection, confidence levels |
| `human/greg-task/SKILL.md` | `/greg-task` | Design and launch a multi-agent task |
| `human/greg-learn/SKILL.md` | `/greg-learn` | Consolidate learnings from a conversation into persistent memory |
| `human/greg-revise/SKILL.md` | `/greg-revise` | Revise and improve written content |

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
