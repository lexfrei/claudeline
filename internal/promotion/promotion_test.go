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
		End:   time.Date(2026, 3, 28, 8, 0, 0, 0, time.UTC),
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
			time.Date(2026, 3, 28, 8, 0, 0, 0, time.UTC), // exactly at end
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

func TestCurrent(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			origNow := NowFn

			t.Cleanup(func() { NowFn = origNow })

			NowFn = func() time.Time { return tt.now }

			status := Current()
			if status.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", status.Active, tt.wantActive)
			}

			if status.FiveHour != tt.wantFiveH {
				t.Errorf("FiveHour = %q, want %q", status.FiveHour, tt.wantFiveH)
			}

			if status.SevenDay != tt.wantSevenD {
				t.Errorf("SevenDay = %q, want %q", status.SevenDay, tt.wantSevenD)
			}
		})
	}
}

func TestCurrentMarch2026Promo(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata not available: %v", err)
	}

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

	_ = loc // DST handled by knownPromotions using real location.

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			origNow := NowFn

			t.Cleanup(func() { NowFn = origNow })

			NowFn = func() time.Time { return tt.now }

			status := Current()
			if status.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", status.Active, tt.wantActive)
			}

			if status.FiveHour != tt.wantFiveH {
				t.Errorf("FiveHour = %q, want %q", status.FiveHour, tt.wantFiveH)
			}

			if status.SevenDay != tt.wantSevenD {
				t.Errorf("SevenDay = %q, want %q", status.SevenDay, tt.wantSevenD)
			}
		})
	}
}
