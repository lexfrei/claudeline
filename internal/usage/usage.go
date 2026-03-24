// Package usage provides access to the Anthropic quota usage API.
package usage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lexfrei/claudeline/internal/cache"
	"github.com/lexfrei/claudeline/internal/fmtutil"
	"github.com/lexfrei/claudeline/internal/httpclient"
	"github.com/lexfrei/claudeline/internal/keychain"
)

// ErrUnexpectedStatus is returned when the usage API returns a non-200/401/429 status code.
var ErrUnexpectedStatus = errors.New("unexpected status from usage API")

const (
	apiURL     = "https://api.anthropic.com/api/oauth/usage"
	apiTimeout = 3 * time.Second

	retryAfterBuffer         = 5 * time.Second
	defaultRetryAfterSeconds = 30
	authFailTTL              = 1 * time.Hour
)

// CacheTTL is the cache duration for usage data. Configurable at startup.
var CacheTTL = 10 * time.Minute

// CachePath is the path to the usage cache file. Replaceable for testing.
var CachePath = "/tmp/claude-usage-cache.json"

// LastGoodCachePath stores the last successful API response. Replaceable for testing.
var LastGoodCachePath = "/tmp/claude-usage-last-good.json"

// RetryAfterPath stores the retry-after deadline. Replaceable for testing.
var RetryAfterPath = "/tmp/claude-usage-retry-after"

// AuthFailPath stores the token hash of the last authentication failure. Replaceable for testing.
var AuthFailPath = "/tmp/claude-usage-auth-failed"

// HTTPGetFn is the function used for HTTP requests. Replaceable for testing.
var HTTPGetFn httpclient.GetFn = httpclient.Get

// apiResponse mirrors the JSON structure from the Anthropic usage API.
type apiResponse struct {
	FiveHour          *apiWindow `json:"five_hour"`            //nolint:tagliatelle // External API format
	SevenDay          *apiWindow `json:"seven_day"`            //nolint:tagliatelle // External API format
	SevenDayOpus      *apiWindow `json:"seven_day_opus"`       //nolint:tagliatelle // External API format
	SevenDaySonnet    *apiWindow `json:"seven_day_sonnet"`     //nolint:tagliatelle // External API format
	SevenDayCowork    *apiWindow `json:"seven_day_cowork"`     //nolint:tagliatelle // External API format
	SevenDayOAuthApps *apiWindow `json:"seven_day_oauth_apps"` //nolint:tagliatelle // External API format
	ExtraUsage        *struct {
		IsEnabled    bool    `json:"is_enabled"`    //nolint:tagliatelle // External API format
		MonthlyLimit float64 `json:"monthly_limit"` //nolint:tagliatelle // External API format
		UsedCredits  float64 `json:"used_credits"`  //nolint:tagliatelle // External API format
	} `json:"extra_usage"` //nolint:tagliatelle // External API format
	Error *struct {
		Type string `json:"type"`
	} `json:"error"`
}

// apiWindow represents a single window in the API response.
type apiWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"` //nolint:tagliatelle // External API format
}

// Fetch retrieves quota usage from Anthropic API (with caching).
func Fetch() (*fmtutil.Data, error) {
	if cached, ok := cache.Read(CachePath, CacheTTL); ok {
		return ParseBody(cached)
	}

	if retryAfterActive() {
		return &fmtutil.Data{ErrorType: "rate_limit_error"}, nil
	}

	token, err := keychain.GetFn()
	if err != nil {
		return nil, fmt.Errorf("getting oauth token: %w", err)
	}

	if token == "" {
		return nil, fmt.Errorf("getting oauth token: %w", keychain.ErrNoToken)
	}

	if authFailedForToken(token) {
		return &fmtutil.Data{ErrorType: "authentication_error"}, nil
	}

	resp, err := HTTPGetFn(apiURL, map[string]string{
		"Authorization":  "Bearer " + token,
		"anthropic-beta": "oauth-2025-04-20",
	}, apiTimeout)
	if err != nil {
		return nil, fmt.Errorf("fetching usage: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		writeRetryAfter(resp.Header)

		return &fmtutil.Data{ErrorType: "rate_limit_error"}, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		writeAuthFailed(token)

		return &fmtutil.Data{ErrorType: "authentication_error"}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	cache.Write(CachePath, resp.Body)

	return ParseBody(resp.Body)
}

// retryAfterActive returns true if a retry-after deadline is set and has not passed.
func retryAfterActive() bool {
	data, ok := cache.ReadAny(RetryAfterPath)
	if !ok {
		return false
	}

	deadline, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return false
	}

	return time.Now().UTC().Before(deadline)
}

// writeRetryAfter stores a retry-after deadline computed from the Retry-After header.
// If the header is missing or unparseable, defaultRetryAfterSeconds is used.
// A retryAfterBuffer is always added on top of the parsed value.
func writeRetryAfter(header http.Header) {
	seconds := parseRetryAfterSeconds(header)
	deadline := time.Now().UTC().Add(time.Duration(seconds)*time.Second + retryAfterBuffer)
	cache.Write(RetryAfterPath, []byte(deadline.Format(time.RFC3339)))
}

// parseRetryAfterSeconds extracts the number of seconds from a Retry-After header.
// Returns defaultRetryAfterSeconds if the header is missing or not a valid integer.
// Note: HTTP-date format (RFC 7231) is intentionally not supported; the Anthropic API
// uses integer seconds in practice.
func parseRetryAfterSeconds(header http.Header) int {
	val := header.Get("Retry-After")
	if val == "" {
		return defaultRetryAfterSeconds
	}

	seconds, err := strconv.Atoi(val)
	if err != nil {
		return defaultRetryAfterSeconds
	}

	return max(seconds, 0)
}

// authFailedForToken returns true if the given token received a 401 within authFailTTL.
func authFailedForToken(token string) bool {
	data, ok := cache.Read(AuthFailPath, authFailTTL)
	if !ok {
		return false
	}

	return string(data) == hashToken(token)
}

// writeAuthFailed stores the hash of the token that got a 401 response.
func writeAuthFailed(token string) {
	cache.Write(AuthFailPath, []byte(hashToken(token)))
}

// hashToken returns a hex-encoded SHA-256 hash of the token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))

	return hex.EncodeToString(h[:])
}

// ParseBody parses the usage API response body.
func ParseBody(body []byte) (*fmtutil.Data, error) {
	var resp apiResponse

	unmarshalErr := json.Unmarshal(body, &resp)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("parsing usage response: %w", unmarshalErr)
	}

	if resp.Error != nil {
		return &fmtutil.Data{ErrorType: resp.Error.Type}, nil
	}

	cache.Write(LastGoodCachePath, body)

	result := &fmtutil.Data{}

	if resp.FiveHour != nil {
		result.FiveHour = parseWindow(resp.FiveHour, fmtutil.FiveHourWindowMinutes)
	}

	if resp.SevenDay != nil {
		result.SevenDay = parseWindow(resp.SevenDay, fmtutil.SevenDayWindowMinutes)
	}

	if resp.SevenDayOpus != nil {
		result.SevenDayOpus = parseWindow(resp.SevenDayOpus, fmtutil.SevenDayWindowMinutes)
	}

	if resp.SevenDaySonnet != nil {
		result.SevenDaySonnet = parseWindow(resp.SevenDaySonnet, fmtutil.SevenDayWindowMinutes)
	}

	if resp.SevenDayCowork != nil {
		result.SevenDayCowork = parseWindow(resp.SevenDayCowork, fmtutil.SevenDayWindowMinutes)
	}

	if resp.SevenDayOAuthApps != nil {
		result.SevenDayOAuthApps = parseWindow(resp.SevenDayOAuthApps, fmtutil.SevenDayWindowMinutes)
	}

	if resp.ExtraUsage != nil && resp.ExtraUsage.IsEnabled && resp.ExtraUsage.UsedCredits > 0 {
		result.Extra = &fmtutil.ExtraUsage{
			MonthlyLimit: resp.ExtraUsage.MonthlyLimit,
			UsedCredits:  resp.ExtraUsage.UsedCredits,
		}
	}

	return result, nil
}

func parseWindow(win *apiWindow, totalMinutes int) *fmtutil.QuotaWindow {
	resetsAt, err := fmtutil.ParseISOUTC(win.ResetsAt)
	if err != nil {
		return nil
	}

	remaining := max(int(time.Until(resetsAt).Minutes()), 0)

	return &fmtutil.QuotaWindow{
		Utilization:      win.Utilization,
		ResetsAt:         resetsAt,
		TotalMinutes:     totalMinutes,
		RemainingMinutes: remaining,
	}
}

// FetchLastGood returns the last successful usage data (no TTL).
func FetchLastGood() *fmtutil.Data {
	body, ok := cache.ReadAny(LastGoodCachePath)
	if !ok {
		return nil
	}

	data, err := ParseBody(body)
	if err != nil || data.ErrorType != "" {
		return nil
	}

	return data
}
