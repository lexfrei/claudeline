package fmtutil

import (
	"slices"
	"strings"
	"unicode"
)

// ScopedWindow is a per-model quota window reported by the server with its own
// bucket name (for example "Fable"). The server names these buckets itself, so
// new models appear without a client change.
type ScopedWindow struct {
	// Name is the server-supplied bucket label, e.g. "Fable".
	Name string
	// ID is the model id the server scoped the bucket to. It is null in every
	// response seen so far, but when present it identifies the model exactly and
	// beats matching on the name.
	ID     string
	Window *QuotaWindow
}

// MatchesScopedModel reports whether a quota bucket belongs to the selected
// model, preferring the server's own model id when it sends one: an id names the
// model outright, while a display name has to be matched by token.
//
// The two ids are still compared by token rather than for equality, because one
// model is written both dated and undated ("claude-sonnet-4-5" and
// "claude-sonnet-4-5-20250929"). Equality would read those as different models
// and drop the segment; the token match accepts the trailing build date while
// still refusing a different version ("claude-sonnet-4"). It is tried both ways
// round, since either side may be the dated one.
func MatchesScopedModel(modelID, modelDisplayName string, bucket ScopedWindow) bool {
	if bucket.ID != "" && modelID != "" {
		return MatchesModel(modelID, "", bucket.ID) || MatchesModel(bucket.ID, "", modelID)
	}

	return MatchesModel(modelID, modelDisplayName, bucket.Name)
}

// modelTokenSeparators are the characters a model name may be split on. The
// server writes bucket names for humans ("Claude Opus 4.5") while model ids are
// slugs ("claude-opus-4-5"); folding both onto dashes makes them comparable.
var modelTokenSeparators = strings.NewReplacer(" ", "-", ".", "-", "_", "-")

// normalizeModelToken lowercases a model or bucket name and folds its separators
// to dashes, so "Claude Opus 4.5" and "claude-opus-4-5" become comparable.
func normalizeModelToken(raw string) string {
	return modelTokenSeparators.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

// versionTokenMaxLen is the greatest length at which an all-digit token still
// reads as a version component ("4", "5", "10") rather than a build date
// ("20250929").
const versionTokenMaxLen = 2

// MatchesModel reports whether a quota bucket belongs to the currently selected
// model. The bucket name comes from the server ("Fable", "Claude Sonnet 4.5"),
// the model from the statusline stdin ("claude-sonnet-4-5-20250929" / "Sonnet
// 4.5"): a bucket matches when its tokens appear, in order and whole, in either.
//
// Whole tokens, not raw substrings, because several versions of a family are in
// service at once and the server reports a bucket per version: "Claude Sonnet 4"
// is a plain prefix of claude-sonnet-4-5-20250929, and letting it match would
// show the quota of a model the session is not running. A bucket that ends on a
// version therefore must not be followed by a further version component — while
// a bucket naming no version at all ("sonnet") stays the family bucket and
// matches every version of it.
//
// Buckets that name a surface rather than a model (cowork, oauth) never match:
// no model is called that. An empty bucket name matches nothing.
func MatchesModel(modelID, modelDisplayName, bucket string) bool {
	key := modelTokens(bucket)
	if len(key) == 0 {
		return false
	}

	for _, haystack := range []string{modelID, modelDisplayName} {
		if tokensMatch(modelTokens(haystack), key) {
			return true
		}
	}

	return false
}

// modelTokens splits a model or bucket name into its normalized tokens.
func modelTokens(raw string) []string {
	normalized := normalizeModelToken(raw)
	if normalized == "" {
		return nil
	}

	return strings.Split(normalized, "-")
}

// tokensMatch reports whether key occurs as a whole-token run inside haystack,
// not continued by a further version component.
func tokensMatch(haystack, key []string) bool {
	for start := 0; start+len(key) <= len(haystack); start++ {
		if !slices.Equal(haystack[start:start+len(key)], key) {
			continue
		}

		next := start + len(key)
		if isVersionToken(key[len(key)-1]) && next < len(haystack) && isVersionToken(haystack[next]) {
			continue // the bucket names an older version of this model
		}

		return true
	}

	return false
}

// isVersionToken reports whether a token is a version component: all digits and
// short enough not to be a build date.
func isVersionToken(token string) bool {
	if token == "" || len(token) > versionTokenMaxLen {
		return false
	}

	return strings.IndexFunc(token, func(r rune) bool { return !unicode.IsDigit(r) }) < 0
}

// PerModelCandidates returns every per-model quota window the server reported —
// the fixed top-level buckets first, then the server-named ones from limits[] —
// with no collapsing. Two buckets may share a label while naming different models
// (the family bucket "opus" and a bucket scoped to one Opus version both label
// 7d-opus), so a caller selecting by model must filter this list, not the
// collapsed one, or a bucket for another model could shadow the applicable one.
func (d *Data) PerModelCandidates() []ScopedWindow {
	// A nil Data is a legal value here — FetchLastGood returns one on a cache
	// miss, and FindExhaustedWindow accepts it — so the pair of them being called
	// together must not panic.
	if d == nil {
		return nil
	}

	topLevel := []ScopedWindow{
		{Name: "opus", Window: d.SevenDayOpus},
		{Name: "sonnet", Window: d.SevenDaySonnet},
		{Name: "cowork", Window: d.SevenDayCowork},
		{Name: "oauth", Window: d.SevenDayOAuthApps},
	}

	candidates := make([]ScopedWindow, 0, len(topLevel)+len(d.Scoped))

	for _, candidate := range append(topLevel, d.Scoped...) {
		if candidate.Window != nil {
			candidates = append(candidates, candidate)
		}
	}

	return candidates
}

// PerModelWindows returns the reported windows with one window per label, since a
// label can only render once. On a collision it keeps the window closest to its
// limit, the same rule MostBinding follows. Note this collapses labels, not
// models: two buckets naming one model under different labels (7d-sonnet and
// 7d-sonnet-4-5) both survive here.
func (d *Data) PerModelWindows() []ScopedWindow {
	candidates := d.PerModelCandidates()

	windows := make([]ScopedWindow, 0, len(candidates))
	indexByLabel := make(map[string]int, len(candidates))

	for _, candidate := range candidates {
		label := ScopedLabel(candidate.Name)

		if i, collides := indexByLabel[label]; collides {
			if candidate.Window.Utilization > windows[i].Window.Utilization {
				windows[i] = candidate
			}

			continue
		}

		indexByLabel[label] = len(windows)

		windows = append(windows, candidate)
	}

	return windows
}

// MostBinding reduces windows that all describe the same model to the one
// closest to its limit. The server may report a model twice — a coarse family
// bucket (seven_day_sonnet) and a finer named one ("Sonnet 4.5") — and only one
// of them can be shown without naming the same model twice. Picking the highest
// utilization keeps the window that will bite first: the narrower bucket usually
// is that window, but when the family quota is the one about to run out, hiding
// it would be exactly the wrong half to drop. Ties keep the first window, which
// is the caller's order — top-level buckets ahead of the server-named ones.
// Windows without data are skipped: the function is exported and takes a slice
// the caller built, so it must not depend on their filtering.
func MostBinding(windows []ScopedWindow) []ScopedWindow {
	var best *ScopedWindow

	for i, win := range windows {
		if win.Window == nil {
			continue
		}

		if best == nil || win.Window.Utilization > best.Window.Utilization {
			best = &windows[i]
		}
	}

	if best == nil {
		return nil
	}

	return []ScopedWindow{*best}
}

// ScopedLabel renders the statusline label for a server-named quota bucket:
// "Fable" becomes "7d-fable", matching the existing 7d-opus / 7d-sonnet labels.
func ScopedLabel(bucket string) string {
	return "7d-" + normalizeModelToken(bucket)
}
