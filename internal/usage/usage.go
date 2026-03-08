package usage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lexfrei/claudeline/internal/cache"
	"github.com/lexfrei/claudeline/internal/fmtutil"
	"github.com/lexfrei/claudeline/internal/httpclient"
	"github.com/lexfrei/claudeline/internal/keychain"
)

const (
	apiURL     = "https://api.anthropic.com/api/oauth/usage"
	apiTimeout = 3 * time.Second

	fiveHourWindowMinutes = 300
	sevenDayWindowMinutes = 10_080

	halfRound = 0.5
)

// CacheTTL is the cache duration for usage data. Configurable at startup.
var CacheTTL = 10 * time.Minute

// CachePath is the path to the usage cache file. Replaceable for testing.
var CachePath = "/tmp/claude-usage-cache.json"

// HTTPGetFn is the function used for HTTP requests. Replaceable for testing.
var HTTPGetFn httpclient.GetFn = httpclient.Get

// Fetch retrieves quota usage from Anthropic API (with caching).
func Fetch() (*Data, error) {
	if cached, ok := cache.Read(CachePath, CacheTTL); ok {
		return ParseBody(cached)
	}

	token, err := keychain.GetFn()
	if err != nil {
		return nil, fmt.Errorf("getting oauth token: %w", err)
	}

	body, err := HTTPGetFn(apiURL, map[string]string{
		"Authorization":  "Bearer " + token,
		"anthropic-beta": "oauth-2025-04-20",
	}, apiTimeout)
	if err != nil {
		return nil, fmt.Errorf("fetching usage: %w", err)
	}

	cache.Write(CachePath, body)

	return ParseBody(body)
}

// ParseBody parses the usage API response body.
func ParseBody(body []byte) (*Data, error) {
	var resp apiResponse

	unmarshalErr := json.Unmarshal(body, &resp)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("parsing usage response: %w", unmarshalErr)
	}

	if resp.Error != nil {
		return &Data{ErrorType: resp.Error.Type}, nil
	}

	result := &Data{}

	if resp.FiveHour != nil {
		result.FiveHour = parseWindow(resp.FiveHour, fiveHourWindowMinutes)
	}

	if resp.SevenDay != nil {
		result.SevenDay = parseWindow(resp.SevenDay, sevenDayWindowMinutes)
	}

	if resp.ExtraUsage != nil && resp.ExtraUsage.IsEnabled && resp.ExtraUsage.UsedCredits > 0 {
		result.Extra = &ExtraUsage{
			MonthlyLimit: resp.ExtraUsage.MonthlyLimit,
			UsedCredits:  resp.ExtraUsage.UsedCredits,
		}
	}

	return result, nil
}

func parseWindow(win *apiWindow, totalMinutes int) *QuotaWindow {
	resetsAt, err := fmtutil.ParseISOUTC(win.ResetsAt)
	if err != nil {
		return nil
	}

	remaining := max(int(time.Until(resetsAt).Minutes()), 0)

	return &QuotaWindow{
		Utilization:      win.Utilization,
		ResetsAt:         resetsAt,
		TotalMinutes:     totalMinutes,
		RemainingMinutes: remaining,
	}
}

// FormatQuotaWindow formats a single quota window for display.
func FormatQuotaWindow(win *QuotaWindow, label string) string {
	pct := int(win.Utilization + halfRound)
	indicator := fmtutil.RateIndicator(pct, win.RemainingMinutes, win.TotalMinutes)
	dur := fmtutil.Duration(win.RemainingMinutes)

	return fmt.Sprintf("%s %s: %d%% (%s)", indicator, label, pct, dur)
}
