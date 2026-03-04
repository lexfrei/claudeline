package usage

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
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

		var entry struct {
			Subtype string `json:"subtype"`
		}

		if json.Unmarshal([]byte(line), &entry) == nil && entry.Subtype == "compact_boundary" {
			count++
		}
	}

	return count
}
