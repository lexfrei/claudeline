// Package fmtutil provides formatting helpers for the Claude Code statusline.
package fmtutil

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiReset  = "\033[0m"

	minutesPerDay  = 1440
	minutesPerHour = 60
	percentBase    = 100
	halfRound      = 0.5

	rateSlightlyAhead = 5
	rateFarAhead      = 15

	contextWarnPct = 50
	contextCritPct = 80
)

// ErrCannotParseTimestamp is returned when a timestamp string cannot be parsed.
var ErrCannotParseTimestamp = errors.New("cannot parse timestamp")

// DurationNow is the rendering used for non-positive durations.
const DurationNow = "now"

// Duration returns a human-readable duration from minutes.
func Duration(minutes int) string {
	if minutes <= 0 {
		return DurationNow
	}

	days := minutes / minutesPerDay
	hours := (minutes % minutesPerDay) / minutesPerHour
	mins := minutes % minutesPerHour

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

// RateIndicator compares usage% with elapsed time% to produce a traffic-light emoji.
func RateIndicator(usagePct, remainingMins, totalMins int) string {
	elapsed := max(totalMins-remainingMins, 0)

	timePct := 0
	if totalMins > 0 {
		timePct = elapsed * percentBase / totalMins
	}

	diff := usagePct - timePct

	switch {
	case diff <= 0:
		return "🟢"
	case diff <= rateSlightlyAhead:
		return "🟡"
	case diff <= rateFarAhead:
		return "🟠"
	default:
		return "🔴"
	}
}

// ContextSegment returns an ANSI-colored context percentage segment.
func ContextSegment(pct float64) string {
	rounded := int(pct + halfRound)

	var color string

	switch {
	case rounded >= contextCritPct:
		color = ansiRed
	case rounded >= contextWarnPct:
		color = ansiYellow
	default:
		color = ansiGreen
	}

	return color + Part(fmt.Sprintf("%d%%", rounded), "🧠") + ansiReset
}

// ParseISOUTC parses an ISO-8601 timestamp to UTC time.
func ParseISOUTC(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+00:00",
	}

	for _, layout := range formats {
		parsed, parseErr := time.Parse(layout, raw)
		if parseErr == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: %s", ErrCannotParseTimestamp, raw)
}

// JoinPipe joins string segments with " | " separator.
func JoinPipe(parts []string) string {
	return strings.Join(parts, " | ")
}

// Part renders a statusline part from its text and zero or more icons. The
// first icon (when present and non-empty) is the leading icon, placed to the
// left of the text; any further icons are qualifiers of the entity (e.g. the
// model's effort/thinking/fast-mode markers) and are glued to the right without
// separators: "icon text subicons". Zero icons is valid and yields just the
// text — the case a no-emoji theme relies on.
func Part(text string, icons ...string) string {
	var lead, sub string
	if len(icons) > 0 {
		lead = icons[0]
		sub = strings.Join(icons[1:], "")
	}

	out := text
	if lead != "" {
		out = lead + " " + text
	}

	if sub != "" {
		out += " " + sub
	}

	return out
}

// Quota window constants.
const (
	FiveHourWindowMinutes = 300
	SevenDayWindowMinutes = 10_080
	exhaustedThresholdPct = 99
)

// QuotaWindow represents a single usage quota window (5-hour or 7-day).
type QuotaWindow struct {
	Utilization      float64
	ResetsAt         time.Time
	TotalMinutes     int
	RemainingMinutes int
}

// ExtraUsage represents monthly extra/overuse budget.
type ExtraUsage struct {
	MonthlyLimit float64
	UsedCredits  float64
}

// ExhaustedWindow contains information about which rate limit window was exhausted.
type ExhaustedWindow struct {
	Name    string // "5h", "7d", "7d-opus", "7d-sonnet", "7d-cowork", or "7d-oauth"
	Minutes int    // minutes until reset
}

// Data is the parsed quota usage data.
type Data struct {
	FiveHour          *QuotaWindow
	SevenDay          *QuotaWindow
	SevenDayOpus      *QuotaWindow
	SevenDaySonnet    *QuotaWindow
	SevenDayCowork    *QuotaWindow
	SevenDayOAuthApps *QuotaWindow
	Extra             *ExtraUsage
	ErrorType         string
}

// FormatStaleQuotaWindow formats a window with ?% but real time and indicator.
func FormatStaleQuotaWindow(win *QuotaWindow, label string) string {
	pct := int(win.Utilization + halfRound)
	indicator := RateIndicator(pct, win.RemainingMinutes, win.TotalMinutes)
	dur := Duration(win.RemainingMinutes)

	return Part(fmt.Sprintf("%s: ?%% (%s)", label, dur), indicator)
}

// FormatQuotaWindow formats a single quota window for display.
func FormatQuotaWindow(win *QuotaWindow, label string) string {
	pct := int(win.Utilization + halfRound)
	indicator := RateIndicator(pct, win.RemainingMinutes, win.TotalMinutes)
	dur := Duration(win.RemainingMinutes)

	return Part(fmt.Sprintf("%s: %d%% (%s)", label, pct, dur), indicator)
}

// FormatRateLimitSegment formats the explicit exhausted-limit segment.
func FormatRateLimitSegment(exhausted *ExhaustedWindow) string {
	if exhausted == nil {
		return Part("limit hit", "⛔")
	}

	if exhausted.Minutes <= 0 {
		return Part(exhausted.Name+" limit hit", "⛔")
	}

	return Part(fmt.Sprintf("%s limit hit (%s)", exhausted.Name, Duration(exhausted.Minutes)), "⛔")
}

// FindExhaustedWindow returns the most saturated active window that is exhausted.
// When perModel is true, per-model windows (opus, sonnet, cowork, oauth) are included.
func FindExhaustedWindow(data *Data, perModel bool) *ExhaustedWindow {
	if data == nil {
		return nil
	}

	type windowEntry struct {
		win  *QuotaWindow
		name string
	}

	windows := []windowEntry{
		{data.FiveHour, "5h"},
		{data.SevenDay, "7d"},
	}

	if perModel {
		windows = append(windows,
			windowEntry{data.SevenDayOpus, "7d-opus"},
			windowEntry{data.SevenDaySonnet, "7d-sonnet"},
			windowEntry{data.SevenDayCowork, "7d-cowork"},
			windowEntry{data.SevenDayOAuthApps, "7d-oauth"},
		)
	}

	var best *ExhaustedWindow

	bestPct := -1

	for _, entry := range windows {
		if entry.win == nil || entry.win.RemainingMinutes <= 0 {
			continue
		}

		pct := int(entry.win.Utilization + halfRound)
		if pct < exhaustedThresholdPct {
			continue
		}

		if pct > bestPct || (pct == bestPct && (best == nil || entry.win.RemainingMinutes < best.Minutes)) {
			bestPct = pct
			best = &ExhaustedWindow{
				Name:    entry.name,
				Minutes: entry.win.RemainingMinutes,
			}
		}
	}

	return best
}
