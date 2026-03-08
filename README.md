# claudeline

[![CI](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml/badge.svg)](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod/go-version/lexfrei/claudeline)](https://go.dev/)
[![License](https://img.shields.io/github/license/lexfrei/claudeline)](LICENSE)

Real-time statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) showing live quota usage from the Anthropic API.

> **⚠️ claudeline uses an undocumented Anthropic API.** If you encounter an unexpected error or status message, please [open an issue](https://github.com/lexfrei/claudeline/issues/new) with the error text — it helps us handle more edge cases!

## Known limitations

The Anthropic usage API (`/api/oauth/usage`) has a very low rate limit — roughly 5 requests per access token before it starts returning HTTP 429 indefinitely. claudeline caches responses for **5 minutes** by default to stay within this budget. This means quota data may be up to 5 minutes stale. You can tune `usage_ttl` in the config, but lower values will burn through the rate limit faster.

## Example output

```text
🤖 Claude | 💰 $1.37 | 🟢 7d: 45% (4d 2h) | 🟢 5h: 12% (4h 23m)
```

```text
🤖 Claude | 💰 $0.00 | ⚠️ degraded | 🧠 67% | 🔄 2 | 🟠 7d: 74% (1d 17h) | 🔴 5h: 91% (27m) | 💳 $12/$100
```

## Segments

| Segment | Description |
| --- | --- |
| 🤖 Model | Active model name |
| 💰 Cost | Cumulative session cost in USD |
| ⚠️/🔶/🔴 Status | Anthropic platform status: ⚠️ degraded, 🔶 major outage, 🔴 critical (hidden when all clear) |
| 🧠 Context | Context window usage percentage (color-coded) |
| 🔄 Compactions | Number of context compactions in current session |
| 🟢/🟡/🟠/🔴 7d | 7-day rolling quota utilization with time until reset |
| 🟢/🟡/🟠/🔴 5h | 5-hour rolling quota utilization with time until reset |
| 💳 Credits | Monthly extra credit usage (only shown when active) |

Quota indicators compare your usage rate against elapsed time to warn about hitting limits:

- 🟢 usage pace is sustainable
- 🟡 usage is slightly ahead of schedule
- 🟠 usage is significantly ahead of schedule
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
usage_ttl = "5m"
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
