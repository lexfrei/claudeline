package promotion

import "time"

const (
	peakStartMarch2026 = 8  // 8 AM ET (Eastern Time, as defined by Anthropic).
	peakEndMarch2026   = 14 // 2 PM ET (Eastern Time, as defined by Anthropic).
)

// knownPromotions lists all Anthropic usage promotions.
// Add new entries here and rebuild to support future promotions.
var knownPromotions = []Promotion{
	{
		Name:  "March 2026",
		Start: time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 28, 7, 0, 0, 0, time.UTC), // March 28 00:00 PT (PDT, UTC-7)
		Peak: PeakSchedule{
			StartHour: peakStartMarch2026,
			EndHour:   peakEndMarch2026,
			// Weekdays: Anthropic defines peak as "8 AM-2 PM ET" on business days.
			// Weekends are implicitly off-peak since peak hours only apply to weekdays.
			Weekdays: true,
			Location:  mustLoadLocation("America/New_York"),
		},
	},
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic("promotion: failed to load timezone " + name + ": " + err.Error())
	}

	return loc
}
