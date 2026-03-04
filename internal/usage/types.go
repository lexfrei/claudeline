// Package usage provides access to the Anthropic quota usage API.
package usage

import "time"

// QuotaWindow represents a single usage quota window (5-hour or 7-day).
type QuotaWindow struct {
	Utilization      float64
	ResetsAt         time.Time
	TotalMinutes     int
	RemainingMinutes int
}

// ExtraUsage represents monthly extra/overuse budget.
type ExtraUsage struct {
	MonthlyLimit float64
	UsedCredits  float64
}

// Data is the parsed response from Anthropic usage API.
type Data struct {
	FiveHour  *QuotaWindow
	SevenDay  *QuotaWindow
	Extra     *ExtraUsage
	ErrorType string
}

// apiResponse mirrors the JSON structure from the Anthropic usage API.
type apiResponse struct {
	FiveHour   *apiWindow `json:"five_hour"`
	SevenDay   *apiWindow `json:"seven_day"`
	ExtraUsage *struct {
		IsEnabled    bool    `json:"is_enabled"`
		MonthlyLimit float64 `json:"monthly_limit"`
		UsedCredits  float64 `json:"used_credits"`
	} `json:"extra_usage"`
	Error *struct {
		Type string `json:"type"`
	} `json:"error"`
}

// apiWindow represents a single window in the API response.
type apiWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}
