package fmtutil

import (
	"strings"
	"testing"
)

func TestDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		minutes int
		want    string
	}{
		{0, "now"},
		{-5, "now"},
		{7, "7m"},
		{60, "1h 0m"},
		{125, "2h 5m"},
		{1440, "1d 0h"},
		{1500, "1d 1h"},
		{10080, "7d 0h"},
		{4337, "3d 0h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := Duration(tt.minutes); got != tt.want {
				t.Errorf("Duration(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestRateIndicator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                       string
		usagePct, remaining, total int
		want                       string
	}{
		{"on_track", 45, 150, 300, "🟢"},
		{"slightly_ahead", 53, 150, 300, "🟡"},
		{"ahead", 60, 150, 300, "🟠"},
		{"way_ahead", 70, 150, 300, "🔴"},
		{"start_of_window", 0, 300, 300, "🟢"},
		{"end_of_window", 100, 0, 300, "🟢"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := RateIndicator(tt.usagePct, tt.remaining, tt.total); got != tt.want {
				t.Errorf("RateIndicator(%d, %d, %d) = %q, want %q",
					tt.usagePct, tt.remaining, tt.total, got, tt.want)
			}
		})
	}
}

func TestContextSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pct       float64
		wantColor string
	}{
		{"green_low", 30, ansiGreen},
		{"green_border", 49.4, ansiGreen},
		{"yellow_border", 50, ansiYellow},
		{"yellow_high", 79, ansiYellow},
		{"red_border", 80, ansiRed},
		{"red_high", 95, ansiRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ContextSegment(tt.pct)
			if !strings.HasPrefix(got, tt.wantColor) {
				t.Errorf("ContextSegment(%.1f) starts with wrong color, got %q", tt.pct, got)
			}
		})
	}
}

func TestParseISOUTC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{"basic_z", "2025-02-26T12:34:56Z"},
		{"nano_z", "2025-02-26T12:34:56.123456Z"},
		{"offset", "2025-02-26T12:34:56+00:00"},
		{"nano_offset", "2025-02-26T12:34:56.123456+00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := ParseISOUTC(tt.raw)
			if err != nil {
				t.Errorf("ParseISOUTC(%q) failed: %v", tt.raw, err)

				return
			}

			if parsed.Year() != 2025 || parsed.Month() != 2 || parsed.Day() != 26 {
				t.Errorf("ParseISOUTC(%q) = %v, wrong date", tt.raw, parsed)
			}
		})
	}
}

func TestParseISOUTCInvalid(t *testing.T) {
	t.Parallel()

	if _, err := ParseISOUTC("not-a-date"); err == nil {
		t.Error("ParseISOUTC(invalid) should return error")
	}
}

func TestJoinPipe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{"nil", nil, ""},
		{"single", []string{"a"}, "a"},
		{"multiple", []string{"a", "b", "c"}, "a | b | c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := JoinPipe(tt.parts); got != tt.want {
				t.Errorf("JoinPipe(%v) = %q, want %q", tt.parts, got, tt.want)
			}
		})
	}
}
