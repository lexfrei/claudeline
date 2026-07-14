package fmtutil

import "strings"

// ScopedWindow is a per-model quota window reported by the server with its own
// bucket name (for example "Fable"). The server names these buckets itself, so
// new models appear without a client change.
type ScopedWindow struct {
	// Name is the server-supplied bucket label, e.g. "Fable".
	Name   string
	Window *QuotaWindow
}

// normalizeModelToken lowercases a model or bucket name and collapses spaces to
// dashes, so "Fable 5" and "claude-fable-5" become comparable.
func normalizeModelToken(raw string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), " ", "-")
}

// MatchesModel reports whether a quota bucket belongs to the currently selected
// model. The bucket name comes from the server ("Fable", "opus"), the model from
// the statusline stdin ("claude-fable-5" / "Fable 5"). An empty bucket matches
// nothing, so non-model buckets (cowork, oauth) never match a model.
func MatchesModel(modelID, modelDisplayName, bucket string) bool {
	key := normalizeModelToken(bucket)
	if key == "" {
		return false
	}

	for _, haystack := range []string{modelID, modelDisplayName} {
		if strings.Contains(normalizeModelToken(haystack), key) {
			return true
		}
	}

	return false
}

// ScopedLabel renders the statusline label for a server-named quota bucket:
// "Fable" becomes "7d-fable", matching the existing 7d-opus / 7d-sonnet labels.
func ScopedLabel(bucket string) string {
	return "7d-" + normalizeModelToken(bucket)
}
