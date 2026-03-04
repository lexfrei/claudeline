// Package cache provides file-based caching with TTL for statusline data.
package cache

import (
	"os"
	"time"
)

const fileMode = 0o600

// Read returns cached content if the file exists and is younger than ttl.
func Read(path string, ttl time.Duration) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}

	if time.Since(info.ModTime()) >= ttl {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	return data, true
}

// Write stores data to path via temp file + rename for crash safety.
func Write(path string, data []byte) {
	tmp := path + ".tmp"

	err := os.WriteFile(tmp, data, fileMode)
	if err != nil {
		return
	}

	_ = os.Rename(tmp, path)
}
