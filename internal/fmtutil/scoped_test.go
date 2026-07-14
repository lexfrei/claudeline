package fmtutil_test

import (
	"testing"

	"github.com/lexfrei/claudeline/internal/fmtutil"
)

const (
	fableID          = "claude-fable-5"
	fableDisplay     = "Fable 5"
	fableBucket      = "Fable"
	opusID           = "claude-opus-4-8"
	opusDisplay      = "Opus 4.8"
	sonnetID         = "claude-sonnet-5"
	sonnetDisplay    = "Sonnet 5"
	sonnetBucketName = "sonnet"
)

func TestMatchesModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modelID     string
		displayName string
		bucket      string
		want        bool
	}{
		{"fable id matches server bucket", fableID, fableDisplay, fableBucket, true},
		{"opus id matches opus bucket", opusID, opusDisplay, "opus", true},
		{"sonnet id matches sonnet bucket", sonnetID, sonnetDisplay, sonnetBucketName, true},
		{"opus model does not match fable bucket", opusID, opusDisplay, fableBucket, false},
		{"falls back to display name when id is empty", "", sonnetDisplay, sonnetBucketName, true},
		{"empty bucket matches nothing", sonnetID, sonnetDisplay, "", false},
		{"non-model bucket does not match", opusID, opusDisplay, "oauth", false},
		{"multi-word bucket still matches the id", fableID, fableDisplay, fableDisplay, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := fmtutil.MatchesModel(tt.modelID, tt.displayName, tt.bucket)
			if got != tt.want {
				t.Errorf("MatchesModel(%q, %q, %q) = %v, want %v",
					tt.modelID, tt.displayName, tt.bucket, got, tt.want)
			}
		})
	}
}

func TestScopedLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		bucket string
		want   string
	}{
		{"single word", fableBucket, "7d-fable"},
		{"spaces become dashes", fableDisplay, "7d-fable-5"},
		{"already lowercase", "opus", "7d-opus"},
		{"surrounding space trimmed", " Fable ", "7d-fable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := fmtutil.ScopedLabel(tt.bucket); got != tt.want {
				t.Errorf("ScopedLabel(%q) = %q, want %q", tt.bucket, got, tt.want)
			}
		})
	}
}
