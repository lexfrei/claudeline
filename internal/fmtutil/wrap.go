package fmtutil

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

// ansiEscape strips SGR escape sequences (color codes) before measuring
// visual width. Without this `\033[32m🧠 50%\033[0m` would over-count.
// Covers SGR only. If future segments embed OSC 8 hyperlinks
// (`\x1b]8;;url\x07text\x1b]8;;\x07`), cursor moves, or other CSI escapes,
// extend the pattern accordingly.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

const (
	segmentSeparator = " | "
	// minWrapWidth is the smallest terminal width at which wrapping kicks in.
	// Below this, the per-line budget cannot meaningfully fit even a single
	// emoji + separator + label, so we degrade to JoinPipe to avoid emitting
	// a near-useless one-segment-per-line output. Callers driving from
	// COLUMNS should additionally subtract a safety margin upstream; this
	// constant only guards the JoinPipeWrap API surface.
	minWrapWidth = 10
)

// VisualWidth returns the visible cell width of s with ANSI SGR codes stripped
// and emoji counted as two cells.
func VisualWidth(s string) int {
	return runewidth.StringWidth(ansiEscape.ReplaceAllString(s, ""))
}

// JoinPipeWrap packs segments separated by " | " into lines no wider than
// maxWidth visual cells. A segment that is itself wider than maxWidth lands
// on its own line (no mid-segment splitting). When maxWidth is too small
// to be meaningful, returns a single-line join (same as JoinPipe).
func JoinPipeWrap(segments []string, maxWidth int) string {
	if maxWidth < minWrapWidth || len(segments) == 0 {
		return JoinPipe(segments)
	}

	sepWidth := VisualWidth(segmentSeparator)

	var (
		lines        []string
		current      strings.Builder
		currentWidth int
	)

	appendInline := func(seg string, segWidth int) {
		if current.Len() > 0 {
			current.WriteString(segmentSeparator)

			currentWidth += sepWidth
		}

		current.WriteString(seg)

		currentWidth += segWidth
	}

	flush := func() {
		if current.Len() == 0 {
			return
		}

		lines = append(lines, current.String())
		currentWidth = 0

		current.Reset()
	}

	for _, seg := range segments {
		segWidth := VisualWidth(seg)

		fits := currentWidth == 0 || currentWidth+sepWidth+segWidth <= maxWidth
		if !fits {
			flush()
		}

		appendInline(seg, segWidth)
	}

	flush()

	return strings.Join(lines, "\n")
}
