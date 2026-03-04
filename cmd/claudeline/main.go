// Package main is the entry point for the claudeline CLI.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/lexfrei/claudeline/internal/fmtutil"
	"github.com/lexfrei/claudeline/internal/status"
	"github.com/lexfrei/claudeline/internal/usage"
)

var (
	version = "dev"
	commit  = "none"
)

type stdinData struct {
	Model struct {
		DisplayName string `json:"display_name"` //nolint:tagliatelle // External API format
	} `json:"model"`
	Cost struct {
		TotalCostUSD float64 `json:"total_cost_usd"` //nolint:tagliatelle // External API format
	} `json:"cost"`
	ContextWindow struct {
		UsedPercentage float64 `json:"used_percentage"` //nolint:tagliatelle // External API format
	} `json:"context_window"` //nolint:tagliatelle // External API format
	TranscriptPath string `json:"transcript_path"` //nolint:tagliatelle // External API format
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("claudeline %s (%s)\n", version, commit)

		return
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println("⚠️ stdin error")

		return
	}

	fmt.Print(buildStatusline(raw))
}

func buildStatusline(raw []byte) string {
	var data stdinData

	_ = json.Unmarshal(raw, &data)

	model := "Claude"
	if data.Model.DisplayName != "" {
		model = data.Model.DisplayName
	}

	segments := []string{
		"🤖 " + model,
		fmt.Sprintf("💰 $%.2f", data.Cost.TotalCostUSD),
	}

	if alert := status.FetchAlert(); alert != "" {
		segments = append(segments, alert)
	}

	if data.ContextWindow.UsedPercentage > 0 {
		segments = append(segments, fmtutil.ContextSegment(data.ContextWindow.UsedPercentage))
	}

	if compactions := usage.CountCompactions(data.TranscriptPath); compactions > 0 {
		segments = append(segments, fmt.Sprintf("🔄 %d", compactions))
	}

	segments = appendUsageSegments(segments)

	return fmtutil.JoinPipe(segments)
}

func appendUsageSegments(segments []string) []string {
	usageData, err := usage.Fetch()
	if err != nil {
		return append(segments, "⏳ 7d: ?% (?d)", "⏳ 5h: ?% (?h)")
	}

	if usageData.ErrorType != "" {
		return append(segments, "⚠️ /login needed")
	}

	if usageData.SevenDay != nil {
		segments = append(segments, usage.FormatQuotaWindow(usageData.SevenDay, "7d"))
	}

	if usageData.FiveHour != nil {
		segments = append(segments, usage.FormatQuotaWindow(usageData.FiveHour, "5h"))
	}

	if usageData.Extra != nil && usageData.Extra.UsedCredits > 0 {
		segments = append(segments, fmt.Sprintf("💳 $%.0f/$%.0f",
			usageData.Extra.UsedCredits, usageData.Extra.MonthlyLimit))
	}

	return segments
}
