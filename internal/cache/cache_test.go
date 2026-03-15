package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadHit(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "cache.json")
	data := []byte(`{"test": true}`)

	if err := os.WriteFile(tmp, data, fileMode); err != nil {
		t.Fatal(err)
	}

	got, ok := Read(tmp, 10*time.Second)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if string(got) != string(data) {
		t.Errorf("Read = %q, want %q", got, data)
	}
}

func TestReadMiss(t *testing.T) {
	t.Parallel()

	_, ok := Read("/nonexistent/path/cache.json", 10*time.Second)
	if ok {
		t.Error("expected cache miss for nonexistent file")
	}
}

func TestReadExpired(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "cache.json")

	if err := os.WriteFile(tmp, []byte("old"), fileMode); err != nil {
		t.Fatal(err)
	}

	past := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(tmp, past, past); err != nil {
		t.Fatal(err)
	}

	_, ok := Read(tmp, 1*time.Second)
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "cache.json")
	data := []byte(`hello`)

	writeErr := Write(tmp, data)
	if writeErr != nil {
		t.Fatalf("Write failed: %v", writeErr)
	}

	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	if string(got) != string(data) {
		t.Errorf("Write wrote %q, want %q", got, data)
	}
}

func TestWriteAtomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tmp := filepath.Join(dir, "cache.json")

	if err := Write(tmp, []byte("first")); err != nil {
		t.Fatalf("first Write failed: %v", err)
	}

	if err := Write(tmp, []byte("second")); err != nil {
		t.Fatalf("second Write failed: %v", err)
	}

	got, _ := os.ReadFile(tmp)
	if string(got) != "second" {
		t.Errorf("expected second write to win, got %q", got)
	}

	_, err := os.Stat(tmp + ".tmp")
	if err == nil {
		t.Error("temp file should not exist after rename")
	}
}
