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
	exhaustedThresholdPct = 99

	halfRound = 0.5
)

// CacheTTL is the cache duration for usage data. Configurable at startup.
var CacheTTL = 10 * time.Minute

// CachePath is the path to the usage cache file. Replaceable for testing.
var CachePath = "/tmp/claude-usage-cache.json"

// LastGoodCachePath stores the last successful API response. Replaceable for testing.
var LastGoodCachePath = "/tmp/claude-usage-last-good.json"

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

	cache.Write(LastGoodCachePath, body)

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

// FetchLastGood returns the last successful usage data (no TTL).
func FetchLastGood() *Data {
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

// FormatStaleQuotaWindow formats a window with ?% but real time and indicator.
func FormatStaleQuotaWindow(win *QuotaWindow, label string) string {
	pct := int(win.Utilization + halfRound)
	indicator := fmtutil.RateIndicator(pct, win.RemainingMinutes, win.TotalMinutes)
	dur := fmtutil.Duration(win.RemainingMinutes)

	return fmt.Sprintf("%s %s: ?%% (%s)", indicator, label, dur)
}

// FormatQuotaWindow formats a single quota window for display.
func FormatQuotaWindow(win *QuotaWindow, label string) string {
	pct := int(win.Utilization + halfRound)
	indicator := fmtutil.RateIndicator(pct, win.RemainingMinutes, win.TotalMinutes)
	dur := fmtutil.Duration(win.RemainingMinutes)

	return fmt.Sprintf("%s %s: %d%% (%s)", indicator, label, pct, dur)
}

// FormatRateLimitSegment formats the explicit exhausted-limit segment.
func FormatRateLimitSegment(exhausted *ExhaustedWindow) string {
	if exhausted == nil {
		return "⛔ limit hit"
	}

	if exhausted.Minutes <= 0 {
		return fmt.Sprintf("⛔ %s limit hit", exhausted.Name)
	}

	return fmt.Sprintf("⛔ %s limit hit (%s)", exhausted.Name, fmtutil.Duration(exhausted.Minutes))
}

// ExhaustedWindow returns the most saturated active window that is exhausted.
func FindExhaustedWindow(data *Data) *ExhaustedWindow {
	if data == nil {
		return nil
	}

	type windowEntry struct {
		win  *QuotaWindow
		name string
	}

	windows := []windowEntry{
		{data.FiveHour, "5h"},
		{data.SevenDay, "7d"},
	}

	var best *ExhaustedWindow

	bestPct := -1

	for _, entry := range windows {
		if entry.win == nil || entry.win.RemainingMinutes <= 0 {
			continue
		}

		pct := int(entry.win.Utilization + halfRound)
		if pct < exhaustedThresholdPct {
			continue
		}

		if pct > bestPct || (pct == bestPct && (best == nil || entry.win.RemainingMinutes < best.Minutes)) {
			bestPct = pct
			best = &ExhaustedWindow{
				Name:    entry.name,
				Minutes: entry.win.RemainingMinutes,
			}
		}
	}

	return best
}
