# claudeline

[![CI](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml/badge.svg)](https://github.com/lexfrei/claudeline/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod-go-version/lexfrei/claudeline)](https://go.dev/)
[![License](https://img.shields.io/github/license/lexfrei/claudeline)](LICENSE)

Real-time statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) showing live quota usage directly from stdin data.

## Example output

```text
🤖 Claude | 💰 $1.37 | 🟢 7d: 45% (4d 2h) | 🟢 5h: 12% (4h 23m)
```

```text
🤖 Claude | 💰 $0.00 | ⚠️ degraded | 🧠 67% | 🔄 2 | 🟠 7d: 74% (1d 17h) | 🔴 5h: 91% (27m)
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
| 🟢/🟡/🟠/🔴 5h | 5-hour rolling quota utilization with time until reset (⬆ during off-peak promotions) |

Quota indicators compare your usage rate against elapsed time to warn about hitting limits:

- 🟢 usage pace is sustainable
- 🟡 usage is slightly ahead of schedule
- 🟠 usage is significantly ahead of schedule
- 🔴 on track to hit the limit before reset

### Off-peak promotions

During [Anthropic usage promotions](https://support.claude.com/en/articles/14063676-claude-march-2026-usage-promotion), off-peak hours provide boosted 5-hour limits. claudeline shows ⬆ next to the 5h quota segment when a promotion is active and you are in an off-peak window. The 7-day limit is unaffected — only bonus usage above the normal 5h cap is excluded from weekly counting.

Disable with `--no-offpeak` or `offpeak = false` in config.

## Requirements

- Claude Code v2.1.80+ (provides `rate_limits` in statusline stdin)

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
offpeak = true

[cache]
status_ttl = "15s"
```

Set any segment to `false` to hide it.

### CLI flags

Flags override config file settings:

```bash
claudeline --no-cost --no-status
claudeline --config /path/to/config.toml
```

Available flags: `--no-model`, `--no-cost`, `--no-status`, `--no-context`, `--no-compactions`, `--no-quota`, `--no-offpeak`.

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
