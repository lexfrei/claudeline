package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"time"
)

// The statusline stdin payload is not documented anywhere; the installed Claude
// Code binary is the only source of truth. These tests extract the payload
// object literal from that binary and compare its field names against the
// pinned set, so a harness update that adds, renames, or drops a field turns
// into a test failure instead of a silently ignored field.
//
// The extraction anchors on field-name strings (stable across builds), not on
// minified variable names (not stable).

// harnessVersionsDir is where the native Claude Code installer keeps versioned
// binaries. Absent on CI, so the drift test skips there and runs on the
// machines that actually have a harness to drift against.
const harnessVersionsDir = ".local/share/claude/versions"

// payloadMarker sits inside the rate_limits assignment directly preceding the
// statusline payload literal. Both windows of interest surround it.
const payloadMarker = "five_hour:{used_percentage"

var errPayloadNotFound = errors.New("statusline payload literal not found in harness binary")

// knownHarnessPayloadFields pins the snake_case field names of the statusline
// stdin payload as of Claude Code 2.1.257. On mismatch, re-verify the schema
// against cmd/claudeline/main.go's stdinData (new fields may deserve a
// segment; renames need a parser change), then update this list.
func knownHarnessPayloadFields() []string {
	return []string{
		"added_dirs",
		"agent",
		"agent_id",
		"agent_type",
		"branch",
		"context_window",
		"cost",
		"current_dir",
		"cwd",
		"display_name",
		"effort",
		"enabled",
		"exceeds_200k_tokens",
		"fast_mode",
		"five_hour",
		"git_worktree",
		"id",
		"kind",
		"level",
		"mode",
		"model",
		"name",
		"number",
		"original_branch",
		"original_cwd",
		"output_style",
		"path",
		"permission_mode",
		"pr",
		"project_dir",
		"prompt_id",
		"rate_limits",
		"remote",
		"repo",
		"resets_at",
		"review_state",
		"scratchpad_dir",
		"session_id",
		"session_name",
		"seven_day",
		"spend_limit",
		"thinking",
		"total_api_duration_ms",
		"total_cost_usd",
		"total_duration_ms",
		"total_lines_added",
		"total_lines_removed",
		"transcript_path",
		"url",
		"used_percentage",
		"version",
		"vim",
		"workspace",
		"worktree",
	}
}

var (
	quotedStringRE = regexp.MustCompile(`"[^"]*"`)
	// A payload field is a snake_case key in a minified object literal:
	// preceded by an object/argument delimiter, followed by a colon. camelCase
	// identifiers (internal JS, not payload) do not match.
	payloadFieldRE = regexp.MustCompile(`[{,(&]([a-z_][a-z0-9_]*):`)
)

// harnessPayloadFields extracts the sorted, deduplicated field names of the
// statusline stdin payload from a Claude Code binary. Two windows are read: the
// payload object literal (anchored on payloadMarker inside the rate_limits
// assignment preceding it) and the base-payload literal it spreads in (anchored
// on basePayloadAnchor), which carries transcript_path — a field the parser
// actively consumes.
//
// Known blind spots, by name: the sub-fields of repo (host, owner, name) and
// context_window (used_percentage) arrive by reference and stay invisible to
// this extraction — the pinned occurrences of used_percentage and name come
// from other inline literals, not from these objects. A field the harness
// appends past the fixed window's end (~150 bytes of slack after the payload
// literal) would also be missed silently; everything else degrades loudly.
func harnessPayloadFields(bin []byte) ([]string, error) {
	markerAt := bytes.Index(bin, []byte(payloadMarker))
	if markerAt < 0 {
		return nil, errPayloadNotFound
	}

	tail := bin[markerAt:min(markerAt+800, len(bin))]

	returnAt := bytes.Index(tail, []byte("return{"))
	if returnAt < 0 {
		return nil, errPayloadNotFound
	}

	base, err := basePayloadWindow(bin)
	if err != nil {
		return nil, err
	}

	// A small pre-window keeps the delimiter right before the marker in view
	// without reaching back into the enclosing function's signature, whose
	// destructured parameters would read as payload fields.
	window := bin[max(markerAt-10, 0):min(markerAt+returnAt+1400, len(bin))]

	fields := fieldsInWindow(window)

	for _, field := range fieldsInWindow(base) {
		if !slices.Contains(fields, field) {
			fields = append(fields, field)
		}
	}

	slices.Sort(fields)

	return fields, nil
}

// basePayloadAnchor is a field of the base payload every hook and the
// statusline share; the literal returning it starts with "return{" right
// before. The proximity requirement is what rejects the same substring inside
// agent_transcript_path elsewhere in the binary.
const basePayloadAnchor = "transcript_path:"

// basePayloadWindow locates the base-payload literal — the object the
// statusline payload spreads in first, invisible to the main window.
func basePayloadWindow(bin []byte) ([]byte, error) {
	for from := 0; ; {
		rel := bytes.Index(bin[from:], []byte(basePayloadAnchor))
		if rel < 0 {
			return nil, errPayloadNotFound
		}

		anchorAt := from + rel

		returnAt := bytes.LastIndex(bin[max(anchorAt-40, 0):anchorAt], []byte("return{"))
		if returnAt < 0 {
			from = anchorAt + 1

			continue
		}

		start := max(anchorAt-40, 0) + returnAt
		window := bin[start:min(start+250, len(bin))]

		// The base literal is flat, so its first closing brace ends it; cutting
		// there keeps whatever follows the literal from reading as fields.
		if end := bytes.IndexByte(window, '}'); end >= 0 {
			window = window[:end]
		}

		return window, nil
	}
}

// fieldsInWindow returns the deduplicated payload field names in a window.
func fieldsInWindow(window []byte) []string {
	// Quoted values (URLs, version strings) contain colons that would read as
	// keys; blank them out before matching.
	window = quotedStringRE.ReplaceAll(window, nil)

	var fields []string

	for _, match := range payloadFieldRE.FindAllSubmatch(window, -1) {
		field := string(match[1])
		if !slices.Contains(fields, field) {
			fields = append(fields, field)
		}
	}

	return fields
}

// installedHarnessBinary returns the newest Claude Code binary on this machine,
// or "" when none is installed (CI).
func installedHarnessBinary(t *testing.T) string {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return newestBinaryIn(filepath.Join(home, harnessVersionsDir))
}

// newestBinaryIn picks the newest non-empty file in a directory, or "" when
// there is none. Empty entries are download stubs the auto-updater leaves for
// the version it is fetching; reading one would misreport a failed download as
// a restructured payload.
func newestBinaryIn(dir string) string {
	entries, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return ""
	}

	newest := ""

	var newestMod int64

	for _, path := range entries {
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() || info.Size() == 0 {
			continue
		}

		if mod := info.ModTime().UnixNano(); newest == "" || mod > newestMod {
			newest, newestMod = path, mod
		}
	}

	return newest
}

// The auto-updater leaves a zero-byte stub for the version it is downloading;
// by mtime that stub is the newest entry, and reading it would misreport a
// failed download as a restructured payload.
func TestNewestBinaryInSkipsEmptyStubs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	older := filepath.Join(dir, "2.1.226")
	if err := os.WriteFile(older, []byte("binary"), 0o600); err != nil {
		t.Fatalf("writing binary fixture: %v", err)
	}

	stub := filepath.Join(dir, "2.1.227")
	if err := os.WriteFile(stub, nil, 0o600); err != nil {
		t.Fatalf("writing stub fixture: %v", err)
	}

	// Make the stub unambiguously the newest entry, as it is during a download.
	if err := os.Chtimes(stub, time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("bumping stub mtime: %v", err)
	}

	if got := newestBinaryIn(dir); got != older {
		t.Errorf("newestBinaryIn() = %q, want the non-empty %q", got, older)
	}
}

func TestNewestBinaryInEmptyDir(t *testing.T) {
	t.Parallel()

	if got := newestBinaryIn(t.TempDir()); got != "" {
		t.Errorf("expected no binary in an empty dir, got %q", got)
	}
}

func TestHarnessStdinSchemaDrift(t *testing.T) {
	t.Parallel()

	binPath := installedHarnessBinary(t)
	if binPath == "" {
		t.Skip("no installed Claude Code binary to check against")
	}

	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("reading harness binary: %v", err)
	}

	fields, err := harnessPayloadFields(bin)
	if err != nil {
		t.Fatalf("%v in %s — the payload was restructured; re-verify the schema against stdinData and update this test", err, binPath)
	}

	known := knownHarnessPayloadFields()

	if !slices.Equal(fields, known) {
		var news, gone []string

		for _, f := range fields {
			if !slices.Contains(known, f) {
				news = append(news, f)
			}
		}

		for _, f := range known {
			if !slices.Contains(fields, f) {
				gone = append(gone, f)
			}
		}

		t.Errorf("statusline payload of %s drifted from the pinned schema:\n  new fields: %v\n  missing fields: %v\n"+
			"Check whether stdinData should parse the new fields, then update knownHarnessPayloadFields.",
			binPath, news, gone)
	}
}

// The extractor itself is pinned against a synthetic minified payload, so a
// silent extraction regression cannot masquerade as "no drift". The base
// payload arrives via spread (...F()), so its fields must come from the second
// window — the main literal alone cannot see them.
func TestHarnessPayloadFieldsExtraction(t *testing.T) {
	t.Parallel()

	blob := []byte(`function F(){return{session_id:n,transcript_path:uL(n),effort:a}}` +
		`function G(o){agent_transcript_path:Pk(o)};` +
		`X={...C.five_hour&&{five_hour:{used_percentage:C.five_hour.utilization*100,resets_at:C.five_hour.resets_at}}};` +
		`return{...F(),cwd:d,model:{id:g,display_name:W(g)},version:{URL:"https://example.com/x"}.V,...(X.five_hour)&&{rate_limits:X},camelCaseKey:1}`)

	fields, err := harnessPayloadFields(blob)
	if err != nil {
		t.Fatalf("harnessPayloadFields() error = %v", err)
	}

	want := []string{
		"cwd", "display_name", "effort", "five_hour", "id", "model",
		"rate_limits", "resets_at", "session_id", "transcript_path", "used_percentage", "version",
	}
	if !slices.Equal(fields, want) {
		t.Errorf("fields = %v, want %v", fields, want)
	}
}

func TestHarnessPayloadFieldsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		blob string
	}{
		{"no marker at all", "no payload here"},
		{"marker without a following return", `A={five_hour:{used_percentage:1}}; nothing else`},
		{"payload without a base-payload literal", `A={five_hour:{used_percentage:1}};return{cwd:d}`},
		// agent_transcript_path contains the anchor as a substring, but hook
		// payloads assemble it far from any return{ — proximity rejects it.
		{"anchor only inside agent_transcript_path", `A={five_hour:{used_percentage:1}};return{cwd:d}` +
			`function G(o){m={hook_event_name:v,stop_hook_active:n,agent_transcript_path:Pk(o)}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := harnessPayloadFields([]byte(tt.blob)); !errors.Is(err, errPayloadNotFound) {
				t.Errorf("expected errPayloadNotFound, got %v", err)
			}
		})
	}
}
