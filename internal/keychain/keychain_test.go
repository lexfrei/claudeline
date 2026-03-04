package keychain

import "testing"

func TestGetDefaultNoKeychain(t *testing.T) {
	t.Parallel()

	_, err := getDefault()
	if err == nil {
		return
	}
	// Any error is acceptable — we just verify it doesn't panic.
}
