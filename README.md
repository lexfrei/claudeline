# claudeline

[![CI](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml/badge.svg)](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod/go-version/lexfrei/claudeline)](https://go.dev/)
[![License](https://img.shields.io/github/license/lexfrei/claudeline)](LICENSE)

Real-time statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) showing live quota usage from the Anthropic API.

## Example output

```text
🤖 Claude | 💰 $1.37 | 🟢 7d: 45% (4d 2h) | 🟢 5h: 12% (4h 23m)
```

```text
🤖 Claude | 💰 $0.00 | ⚠️ degraded | 📊 67% | 🔄 2 | 🟡 7d: 74% (1d 17h) | 🔴 5h: 91% (27m) | 💳 $12/$100
```

## Segments

| Segment | Description |
| --- | --- |
| 🤖 Model | Active model name |
| 💰 Cost | Cumulative session cost in USD |
| ⚠️ Status | Anthropic platform status (only shown when degraded/down) |
| 📊 Context | Context window usage percentage (color-coded) |
| 🔄 Compactions | Number of context compactions in current session |
| 🟢/🟡/🔴 7d | 7-day rolling quota utilization with time until reset |
| 🟢/🟡/🔴 5h | 5-hour rolling quota utilization with time until reset |
| 💳 Credits | Monthly extra credit usage (only shown when active) |

Quota indicators compare your usage rate against elapsed time to warn about hitting limits:

- 🟢 usage pace is sustainable
- 🟡 usage is ahead of schedule
- 🔴 on track to hit the limit before reset

## Requirements

- **macOS only** (uses macOS Keychain for OAuth token storage)
- Claude Code with OAuth login (`claude login`)

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

Add to your Claude Code statusline configuration in `~/.claude/settings.json`:

```json
{
  "env": {
    "CLAUDE_CODE_ENABLE_STATUSLINE": "1"
  },
  "statusline": {
    "command": "claudeline",
    "interval": 30
  }
}
```

Claude Code pipes session data as JSON to stdin. claudeline reads it and outputs a formatted statusline string.

## Configuration

Optional config file at `~/.claudelinerc.toml`:

```toml
[segments]
model = true
cost = true
status = true
context = true
compactions = true
quota = true
credits = true

[cache]
usage_ttl = "60s"
status_ttl = "15s"
```

Set any segment to `false` to hide it. Cache TTLs control how often API data is refreshed.

### CLI flags

Flags override config file settings:

```bash
claudeline --no-cost --no-status
claudeline --config /path/to/config.toml
```

Available flags: `--no-model`, `--no-cost`, `--no-status`, `--no-context`, `--no-compactions`, `--no-quota`, `--no-credits`.

## License

[BSD-3-Clause](LICENSE)
