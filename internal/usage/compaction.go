package usage

import (
	"bufio"
	"os"
	"strings"

	"github.com/tidwall/gjson"
)

const scanBufSize = 256 * 1024

// CountCompactions counts compact_boundary entries in a session transcript JSONL.
func CountCompactions(transcriptPath string) int {
	if transcriptPath == "" {
		return 0
	}

	file, err := os.Open(transcriptPath)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, scanBufSize), scanBufSize*4) //nolint:mnd // 1MB max line

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "compact_boundary") {
			continue
		}

		if gjson.Get(line, "subtype").Str == "compact_boundary" {
			count++
		}
	}

	// Return partial count even on scanner error: lines read before the error
	// were valid, and a partial count is more useful than zero.
	return count
}
