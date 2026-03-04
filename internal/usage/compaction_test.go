package usage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountCompactionsEmpty(t *testing.T) {
	t.Parallel()

	if got := CountCompactions(""); got != 0 {
		t.Errorf("expected 0 for empty path, got %d", got)
	}
}

func TestCountCompactionsNoFile(t *testing.T) {
	t.Parallel()

	if got := CountCompactions("/nonexistent/transcript.jsonl"); got != 0 {
		t.Errorf("expected 0 for missing file, got %d", got)
	}
}

func TestCountCompactionsWithData(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "transcript.jsonl")

	lines := `{"type":"message","subtype":"user"}
{"type":"boundary","subtype":"compact_boundary"}
{"type":"message","subtype":"assistant"}
{"type":"boundary","subtype":"compact_boundary"}
{"type":"message","subtype":"user"}
`

	if err := os.WriteFile(tmp, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CountCompactions(tmp); got != 2 {
		t.Errorf("expected 2 compactions, got %d", got)
	}
}

func TestCountCompactionsNoMatches(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "transcript.jsonl")

	lines := `{"type":"message","subtype":"user"}
{"type":"message","subtype":"assistant"}
`

	if err := os.WriteFile(tmp, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CountCompactions(tmp); got != 0 {
		t.Errorf("expected 0 compactions, got %d", got)
	}
}

func TestCountCompactionsInvalidJSON(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "transcript.jsonl")

	if err := os.WriteFile(tmp, []byte("compact_boundary not valid json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := CountCompactions(tmp); got != 0 {
		t.Errorf("expected 0 for invalid JSON, got %d", got)
	}
}
