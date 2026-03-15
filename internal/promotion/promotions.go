package promotion

import "time"

// knownPromotions lists all Anthropic usage promotions.
// Add new entries here and rebuild to support future promotions.
var knownPromotions = []Promotion{
	{
		Name:  "March 2026",
		Start: time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 28, 8, 0, 0, 0, time.UTC), // March 27 23:59 PT
		Peak: PeakSchedule{
			StartHour: 8,
			EndHour:   14,
			Weekdays:  true,
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
