package promotion

import (
	"testing"
	"time"
)

func TestIsPeakHour(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("ET", -5*3600)

	sched := PeakSchedule{
		StartHour: 8,
		EndHour:   14,
		Weekdays:  true,
		Location:  loc,
	}

	tests := []struct {
		name    string
		hour    int
		weekday time.Weekday
		want    bool
	}{
		{"before peak", 7, time.Monday, false},
		{"peak start boundary", 8, time.Monday, true},
		{"mid peak", 10, time.Wednesday, true},
		{"last peak hour", 13, time.Friday, true},
		{"peak end boundary", 14, time.Monday, false},
		{"after peak", 20, time.Tuesday, false},
		{"midnight", 0, time.Thursday, false},
		{"saturday peak hour", 10, time.Saturday, false},
		{"sunday peak hour", 10, time.Sunday, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isPeakHour(tt.hour, tt.weekday, sched)
			if got != tt.want {
				t.Errorf("isPeakHour(%d, %s) = %v, want %v", tt.hour, tt.weekday, got, tt.want)
			}
		})
	}
}

func TestIsPeakHourAllDays(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("ET", -5*3600)

	sched := PeakSchedule{
		StartHour: 8,
		EndHour:   14,
		Weekdays:  false,
		Location:  loc,
	}

	tests := []struct {
		name    string
		hour    int
		weekday time.Weekday
		want    bool
	}{
		{"saturday peak hour all days", 10, time.Saturday, true},
		{"sunday peak hour all days", 10, time.Sunday, true},
		{"weekday peak hour all days", 10, time.Monday, true},
		{"off peak all days", 20, time.Saturday, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isPeakHour(tt.hour, tt.weekday, sched)
			if got != tt.want {
				t.Errorf("isPeakHour(%d, %s) = %v, want %v", tt.hour, tt.weekday, got, tt.want)
			}
		})
	}
}

func TestIsOffPeak(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("ET", -5*3600)

	promo := Promotion{
		Name:  "Test Promo",
		Start: time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 28, 7, 0, 0, 0, time.UTC),
		Peak: PeakSchedule{
			StartHour: 8,
			EndHour:   14,
			Weekdays:  true,
			Location:  loc,
		},
	}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			"weekday off-peak evening",
			time.Date(2026, 3, 16, 1, 0, 0, 0, time.UTC), // 20:00 ET Monday
			true,
		},
		{
			"weekday peak morning",
			time.Date(2026, 3, 16, 15, 0, 0, 0, time.UTC), // 10:00 ET Monday
			false,
		},
		{
			"weekend always off-peak",
			time.Date(2026, 3, 14, 15, 0, 0, 0, time.UTC), // 10:00 ET Saturday
			true,
		},
		{
			"before promo start",
			time.Date(2026, 3, 12, 1, 0, 0, 0, time.UTC),
			false,
		},
		{
			"after promo end",
			time.Date(2026, 3, 29, 1, 0, 0, 0, time.UTC),
			false,
		},
		{
			"peak start boundary",
			time.Date(2026, 3, 17, 13, 0, 0, 0, time.UTC), // 08:00 ET Tuesday
			false,
		},
		{
			"peak end boundary",
			time.Date(2026, 3, 17, 19, 0, 0, 0, time.UTC), // 14:00 ET Tuesday
			true,
		},
		{
			"promo start boundary inclusive",
			time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC), // exactly at start, midnight UTC Friday
			true,
		},
		{
			"promo end boundary exclusive",
			time.Date(2026, 3, 28, 7, 0, 0, 0, time.UTC), // exactly at end
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isOffPeak(tt.now, promo)
			if got != tt.want {
				t.Errorf("isOffPeak(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestIsOffPeakDST(t *testing.T) {
	t.Parallel()

	// Use real America/New_York for DST testing.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata not available: %v", err)
	}

	// DST spring forward: March 8, 2026 at 2:00 AM ET.
	promo := Promotion{
		Name:  "DST Test",
		Start: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Peak: PeakSchedule{
			StartHour: 8,
			EndHour:   14,
			Weekdays:  true,
			Location:  loc,
		},
	}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			"before DST 10am EST Monday",
			// March 2 2026 is Monday. 10:00 EST = 15:00 UTC.
			time.Date(2026, 3, 2, 15, 0, 0, 0, time.UTC),
			false, // peak
		},
		{
			"after DST 10am EDT Monday",
			// March 9 2026 is Monday. 10:00 EDT = 14:00 UTC.
			time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC),
			false, // peak
		},
		{
			"after DST 20:00 EDT Monday",
			// March 9 2026. 20:00 EDT = 00:00 UTC next day.
			time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			true, // off-peak
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isOffPeak(tt.now, promo)
			if got != tt.want {
				t.Errorf("isOffPeak(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestIsOffPeakEndDateBoundary(t *testing.T) {
	t.Parallel()

	// Verify the UTC end boundary derived from March 28 00:00 PDT (= March 28 07:00 UTC).
	// Peak schedule location (ET) is irrelevant to end date check — End is stored in UTC.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata not available: %v", err)
	}

	promo := Promotion{
		Name:  "End Date Test",
		Start: time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 28, 7, 0, 0, 0, time.UTC),
		Peak: PeakSchedule{
			StartHour: 8,
			EndHour:   14,
			Weekdays:  true,
			Location:  loc,
		},
	}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			"exactly at end March 28 00:00 PDT",
			// March 28 00:00 PDT = March 28 07:00 UTC. End is exclusive, so this is outside.
			time.Date(2026, 3, 28, 7, 0, 0, 0, time.UTC),
			false,
		},
		{
			"last minute before end",
			// March 28 06:59 UTC is before 07:00 UTC end.
			time.Date(2026, 3, 28, 6, 59, 0, 0, time.UTC),
			true, // Saturday off-peak
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isOffPeak(tt.now, promo)
			if got != tt.want {
				t.Errorf("isOffPeak(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

// TestCurrent and TestCurrentMarch2026Promo do NOT use t.Parallel on the parent
// because they mutate the package-level NowFn. Subtests also run sequentially.
func TestCurrent(t *testing.T) {
	tests := []struct {
		name       string
		now        time.Time
		wantActive bool
		wantFiveH  string
		wantSevenD string
	}{
		{
			"no active promo",
			time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
			false, "", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origNow := NowFn

			t.Cleanup(func() { NowFn = origNow })

			NowFn = func() time.Time { return tt.now }

			got := Current()
			if got.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", got.Active, tt.wantActive)
			}

			if got.FiveHour != tt.wantFiveH {
				t.Errorf("FiveHour = %q, want %q", got.FiveHour, tt.wantFiveH)
			}

			if got.SevenDay != tt.wantSevenD {
				t.Errorf("SevenDay = %q, want %q", got.SevenDay, tt.wantSevenD)
			}
		})
	}
}

func TestCurrentMarch2026Promo(t *testing.T) {
	tests := []struct {
		name       string
		now        time.Time
		wantActive bool
		wantFiveH  string
		wantSevenD string
	}{
		{
			"off-peak evening",
			// March 16 2026 Monday 20:00 EDT = March 17 00:00 UTC.
			time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
			true, " 🌈", " ⏸",
		},
		{
			"peak morning",
			// March 16 2026 Monday 10:00 EDT = March 16 14:00 UTC.
			time.Date(2026, 3, 16, 14, 0, 0, 0, time.UTC),
			false, "", "",
		},
		{
			"weekend",
			// March 14 2026 Saturday 10:00 EDT.
			time.Date(2026, 3, 14, 14, 0, 0, 0, time.UTC),
			true, " 🌈", " ⏸",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origNow := NowFn

			t.Cleanup(func() { NowFn = origNow })

			NowFn = func() time.Time { return tt.now }

			got := Current()
			if got.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", got.Active, tt.wantActive)
			}

			if got.FiveHour != tt.wantFiveH {
				t.Errorf("FiveHour = %q, want %q", got.FiveHour, tt.wantFiveH)
			}

			if got.SevenDay != tt.wantSevenD {
				t.Errorf("SevenDay = %q, want %q", got.SevenDay, tt.wantSevenD)
			}
		})
	}
}
