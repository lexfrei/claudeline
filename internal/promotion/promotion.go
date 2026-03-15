// Package promotion provides client-side detection of Anthropic usage promotions.
package promotion

import (
	"time"

	_ "time/tzdata" // Embed timezone database for portability.
)

// NowFn is the time source, replaceable in tests.
var NowFn = time.Now

// PeakSchedule defines when peak hours apply (limits NOT doubled).
type PeakSchedule struct {
	StartHour int            // inclusive, 0-23
	EndHour   int            // exclusive, 0-23
	Weekdays  bool           // true = Mon-Fri only (weekends always off-peak)
	Location  *time.Location // timezone for hour evaluation
}

// Promotion defines a time-limited usage promotion.
type Promotion struct {
	Name  string
	Start time.Time // inclusive (UTC)
	End   time.Time // exclusive (UTC)
	Peak  PeakSchedule
}

// Status holds the off-peak indicators to append to quota labels.
type Status struct {
	Active   bool
	FiveHour string // " 🌈" or ""
	SevenDay string // " ⏸" or ""
}

// Current checks all known promotions and returns the current off-peak status.
// First matching promotion wins; overlapping promotions are not supported.
func Current() Status {
	now := NowFn()

	for idx := range knownPromotions {
		if isOffPeak(now, knownPromotions[idx]) {
			return Status{
				Active:   true,
				FiveHour: " 🌈",
				SevenDay: " ⏸",
			}
		}
	}

	return Status{}
}

func isOffPeak(now time.Time, promo Promotion) bool {
	if now.Before(promo.Start) || !now.Before(promo.End) {
		return false
	}

	local := now.In(promo.Peak.Location)

	return !isPeakHour(local.Hour(), local.Weekday(), promo.Peak)
}

func isPeakHour(hour int, weekday time.Weekday, sched PeakSchedule) bool {
	if sched.Weekdays && (weekday == time.Saturday || weekday == time.Sunday) {
		return false
	}

	return hour >= sched.StartHour && hour < sched.EndHour
}
