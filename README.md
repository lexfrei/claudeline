# claudeline

[![CI](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml/badge.svg)](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod/go-version/lexfrei/claudeline)](https://go.dev/)
[![License](https://img.shields.io/github/license/lexfrei/claudeline)](LICENSE)

Real-time statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) showing live quota usage directly from stdin data.

## Example output

```text
🤖 Opus 4.7 ⏫💭 | 🧠 67% | 🔄 2 | 🟡 7d: 42% (4d 2h) | 🔴 5h: 91% (27m) | 🐙 lexfrei/claudeline 📝 #19 🌳 feat-api 🌿 feat/api
```

## Segments

| Segment | Description |
| --- | --- |
| 🤖 Model | Active model name, with effort / thinking / fast-mode indicators (Claude Code v2.1.119+) |
| 🐙 Repo | Repository host icon, `owner/name`, optional `#PR <state>`, optional `🌳 worktree` (linked worktrees only) and `🌿 branch` (Claude Code v2.1.145+) |
| 🌳 Worktree / 🌿 Branch | Linked-worktree directory name and current branch; branch alone falls back here when no repository info is available |
| 💰 Cost | Cumulative session cost in USD (hidden by default for subscribers, see [Cost mode](#cost-mode)) |
| ⚠️/🔶/🔴 Status | Anthropic platform status: ⚠️ degraded, 🔶 major outage, 🔴 critical (hidden when all clear) |
| 🧠 Context | Context window usage percentage (color-coded) |
| 🔄 Compactions | Number of context compactions in current session |
| 🟢/🟡/🟠/🔴 7d | 7-day rolling quota utilization with time until reset |
| 🟢/🟡/🟠/🔴 5h | 5-hour rolling quota utilization with time until reset |

### Model indicators

  - effort `low` → `⬇️`, `medium` → no indicator, `high` → `⬆️`, `xhigh` → `⏫`, `max` → `🚀`
  - thinking enabled → `💭`
  - fast mode → `⚡`

### Themes

The icon style is selectable with `theme` in config or `--theme` on the CLI:

  - `emoji` (default) — the rendering shown above.
  - `text` — drops every emoji icon. Where an emoji encoded status by color (the `🟢/🟡/🟠/🔴` rate circles, the context meter, a changes-requested PR, a critical platform status), that color is carried onto the segment's text instead. Identifying emoji (`🤖`, `🐙`, `📝`) are removed, since the text already names the segment.

Two kinds of state have no text form and are unavailable in this theme: the model's effort / thinking / fast-mode markers (`⏫`/`💭`/`⚡`) disappear entirely, and every PR review state except changes-requested (which survives as red) collapses to a plain `#N`.

The same state as the example above, under `--theme text` (status shown here in **bold** to stand in for color):

```text
Opus 4.7 | **67%** | 2 | **7d: 42% (4d 2h)** | **5h: 91% (27m)** | lexfrei/claudeline #19 feat-api feat/api
```

### Auto-wrap on narrow terminals

When Claude Code exports `$COLUMNS` (v2.1.153+) and the joined statusline exceeds the available width, segments overflow onto additional lines instead of running past the host's line budget. Segments are never split mid-content — a single oversized segment lands on its own line.

A small safety margin (2 cells) is subtracted from `$COLUMNS` as a buffer: the terminal may have resized between Claude Code reading `$COLUMNS` and the script running, and empirically rows exactly equal to `$COLUMNS` were observed to drop their rightmost character. Tuning this further is out of scope until someone reports a concrete miss.

When `$COLUMNS` is unset (older Claude Code, non-terminal hosts), output stays on a single line — current behavior is preserved.

### Repo segment

Renders when Claude Code reports `workspace.repo` (a git remote pointing at a known host):

  - `🐙` github.com, `🦊` gitlab.com, `🪣` bitbucket.org, `📦` other hosts (with `host/` prefix)
  - review state followed by `#N`: `📝` draft, `👀` pending, `💬` commented, `🔴` changes requested, `✅` approved
  - `🌳 worktree` — directory name of the linked worktree, shown only when `cwd` is a linked worktree (in the main clone it would just duplicate the repo name, so it is omitted)
  - `🌿 branch` — current branch read directly from `cwd/.git/HEAD`; when HEAD is detached or unreadable it falls back to the worktree name from stdin, but only if the `🌳` marker is not already shown (so the same name is never printed twice)

When no `workspace.repo` is present (non-git directory), the segment falls back to the bare `🌳 worktree 🌿 branch` form showing the same sources.

Quota indicators compare your usage rate against elapsed time to warn about hitting limits:

- 🟢 usage pace is sustainable
- 🟡 usage is slightly ahead of schedule
- 🟠 usage is significantly ahead of schedule
- 🔴 on track to hit the limit before reset

### Cost mode

The cost segment has three modes:

- `auto` (default) — hide for Claude.ai subscribers (who have rate limits), show for API users
- `true` — always show
- `false` — never show

Set via `--cost auto|true|false` or `cost = "auto"` in config.

## Requirements

- Claude Code v2.1.82+ (provides `rate_limits` in statusline stdin)
- Claude Code v2.1.97+ recommended (adds `workspace.git_worktree` and `refreshInterval`)
- Claude Code v2.1.119+ enables effort, thinking, and fast-mode indicators
- Claude Code v2.1.145+ enables the combined repo / PR segment
- Claude Code v2.1.153+ enables terminal-width-aware line wrapping (via the `COLUMNS` env var)
- Current branch in `🌿 branch` is read directly from `cwd/.git/HEAD`, no `git` binary required

## Installation

### Homebrew

```bash
brew install lexfrei/tap/claudeline
```

### From source

```bash
go install github.com/lexfrei/claudeline/cmd/claudeline@latest
```

## Usage

Add the `statusLine` block to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "claudeline",
    "padding": 0
  }
}
```

Claude Code pipes session data as JSON to stdin. claudeline reads it and outputs a formatted statusline string.

Restart Claude Code after changing settings.

### Keep quota timers ticking

By default Claude Code re-runs the statusline command on each new assistant message, on permission mode changes, and on vim mode toggles. While the session is idle (for example waiting on background subagents), the command is not re-run and quota reset timers freeze on screen.

Set `refreshInterval` in `~/.claude/settings.json` to also re-run the command on a fixed timer. This **adds** to event-driven updates, it does not replace them. Requires Claude Code v2.1.97+.

```json
{
  "statusLine": {
    "type": "command",
    "command": "claudeline",
    "padding": 0,
    "refreshInterval": 5
  }
}
```

## Configuration

Optional config file at `~/.claudelinerc.toml`:

```toml
theme = "emoji" # or "text"

[segments]
model = true
effort = true
thinking = true
fast_mode = true
repo = true
worktree = true
cost = "auto"
status = true
context = true
compactions = true
quota = true

[cache]
status_ttl = "15s"
```

Set any segment to `false` to hide it (`cost` accepts `"auto"`, `"true"`, `"false"`; `theme` accepts `"emoji"`, `"text"`).

Run `claudeline validate --config ~/.claudelinerc.toml` to check your config for typos and invalid values.

### CLI flags

Flags override config file settings:

```bash
claudeline --cost false --no-status
claudeline --config /path/to/config.toml
```

Available flags: `--theme`, `--no-model`, `--no-effort`, `--no-thinking`, `--no-fast-mode`, `--no-repo`, `--no-worktree`, `--cost`, `--no-status`, `--no-context`, `--no-compactions`, `--no-quota`, `--mac-insecure`, `--per-model-quota`, `--no-credits`. The last two only take effect with `--mac-insecure`.

## Advanced: `--mac-insecure` mode

For additional data not available in stdin, claudeline can access the Anthropic usage API directly via macOS Keychain. This gives you:

- Per-model 7-day quotas (Opus, Sonnet, Cowork, OAuth apps)
- Extra credit usage (💳 segment)

**Security note:** this mode reads your OAuth token from macOS Keychain. Only enable it if you understand the implications.

```bash
claudeline --mac-insecure --per-model-quota
```

```toml
mac_insecure = true

[segments]
per_model_quota = true
credits = true

[cache]
usage_ttl = "10m"
```

Additional flags with `--mac-insecure`: `--per-model-quota`, `--no-credits`.

**Known limitations of `--mac-insecure`:** the Anthropic usage API has a very low rate limit (~5 requests per window). Responses are cached for 10 minutes by default (`usage_ttl`). macOS only.

## License

[BSD-3-Clause](LICENSE)
