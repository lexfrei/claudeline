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

// Duration returns a human-readable duration from minutes.
func Duration(minutes int) string {
	if minutes <= 0 {
		return "now"
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

	return fmt.Sprintf("%s🧠 %d%%%s", color, rounded, ansiReset)
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
