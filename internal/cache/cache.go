// Package cache provides file-based caching with TTL for statusline data.
package cache

import (
	"fmt"
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
func Write(path string, data []byte) error {
	tmp := path + ".tmp"

	err := os.WriteFile(tmp, data, fileMode)
	if err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	renameErr := os.Rename(tmp, path)
	if renameErr != nil {
		return fmt.Errorf("rename temp file: %w", renameErr)
	}

	return nil
}
