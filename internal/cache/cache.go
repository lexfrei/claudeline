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

// ReadAny returns cached content regardless of age.
func ReadAny(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	return data, true
}

// Write stores data to path via temp file + rename for crash safety.
// Errors are intentionally not returned: cache is best-effort in a statusline tool.
// A failed write means slightly stale data on the next read, which is acceptable.
func Write(path string, data []byte) {
	tmp := path + ".tmp"

	err := os.WriteFile(tmp, data, fileMode)
	if err != nil {
		return
	}

	renameErr := os.Rename(tmp, path)
	if renameErr != nil {
		_ = os.Remove(tmp)
	}
}
