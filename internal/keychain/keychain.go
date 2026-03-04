// Package keychain provides macOS Keychain access for OAuth tokens.
package keychain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"time"
)

const (
	service        = "Claude Code-credentials"
	commandTimeout = 5 * time.Second
)

// ErrNoToken is returned when no valid OAuth token is found.
var ErrNoToken = errors.New("no oauth token found")

// GetFn is the function used to retrieve OAuth tokens.
// Replaceable for testing.
var GetFn = getDefault

func getDefault() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("getting current user: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, //nolint:gosec // Arguments are constants
		"security", "find-generic-password",
		"-s", service,
		"-a", usr.Username,
		"-w",
	)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("reading keychain: %w", err)
	}

	var creds struct {
		ClaudeAiOauth *struct {
			AccessToken string `json:"accessToken"` //nolint:gosec // Not a credential, just a field name
		} `json:"claudeAiOauth"`
	}

	unmarshalErr := json.Unmarshal(out, &creds)
	if unmarshalErr != nil {
		return "", fmt.Errorf("parsing keychain data: %w", unmarshalErr)
	}

	if creds.ClaudeAiOauth == nil || creds.ClaudeAiOauth.AccessToken == "" {
		return "", ErrNoToken
	}

	return creds.ClaudeAiOauth.AccessToken, nil
}
