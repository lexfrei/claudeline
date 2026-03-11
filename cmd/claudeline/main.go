// Package main is the entry point for the claudeline CLI.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lexfrei/claudeline/internal/config"
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

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".claudelinerc.toml")
}

func newRootCmd() *cobra.Command {
	var configPath string

	cfg := config.Defaults()

	rootCmd := &cobra.Command{
		Use:     "claudeline",
		Short:   "Real-time statusline for Claude Code",
		Version: fmt.Sprintf("%s (%s)", version, commit),
		Args:    cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Println("⚠️ stdin error")

				return
			}

			fmt.Print(buildStatusline(raw, &cfg))
		},
	}

	rootCmd.PersistentPreRun = func(_ *cobra.Command, _ []string) {
		cfg = config.Load(configPath)

		applyFlagOverrides(rootCmd, &cfg)

		status.CacheTTL = cfg.Cache.StatusTTL
		usage.CacheTTL = cfg.Cache.UsageTTL
	}

	rootCmd.SetVersionTemplate("claudeline {{.Version}}\n")

	flags := rootCmd.PersistentFlags()
	flags.StringVar(&configPath, "config", defaultConfigPath(), "config file path")
	flags.Bool("no-model", false, "disable model segment")
	flags.Bool("no-cost", false, "disable cost segment")
	flags.Bool("no-status", false, "disable status segment")
	flags.Bool("no-context", false, "disable context segment")
	flags.Bool("no-compactions", false, "disable compactions segment")
	flags.Bool("no-quota", false, "disable quota segment")
	flags.Bool("no-credits", false, "disable credits segment")

	return rootCmd
}

func applyFlagOverrides(cmd *cobra.Command, cfg *config.Config) {
	if flagSet(cmd, "no-model") {
		cfg.Segments.Model = false
	}

	if flagSet(cmd, "no-cost") {
		cfg.Segments.Cost = false
	}

	if flagSet(cmd, "no-status") {
		cfg.Segments.Status = false
	}

	if flagSet(cmd, "no-context") {
		cfg.Segments.Context = false
	}

	if flagSet(cmd, "no-compactions") {
		cfg.Segments.Compactions = false
	}

	if flagSet(cmd, "no-quota") {
		cfg.Segments.Quota = false
	}

	if flagSet(cmd, "no-credits") {
		cfg.Segments.Credits = false
	}
}

func flagSet(cmd *cobra.Command, name string) bool {
	flag := cmd.PersistentFlags().Lookup(name)

	return flag != nil && flag.Changed
}

func main() {
	err := newRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func buildStatusline(raw []byte, cfg *config.Config) string {
	var data stdinData

	_ = json.Unmarshal(raw, &data)

	var segments []string

	if cfg.Segments.Model {
		model := "Claude"
		if data.Model.DisplayName != "" {
			model = data.Model.DisplayName
		}

		segments = append(segments, "🤖 "+model)
	}

	if cfg.Segments.Cost {
		segments = append(segments, fmt.Sprintf("💰 $%.2f", data.Cost.TotalCostUSD))
	}

	if cfg.Segments.Status {
		if alert := status.FetchAlert(); alert != "" {
			segments = append(segments, alert)
		}
	}

	if cfg.Segments.Context && data.ContextWindow.UsedPercentage > 0 {
		segments = append(segments, fmtutil.ContextSegment(data.ContextWindow.UsedPercentage))
	}

	if cfg.Segments.Compactions {
		if compactions := usage.CountCompactions(data.TranscriptPath); compactions > 0 {
			segments = append(segments, fmt.Sprintf("🔄 %d", compactions))
		}
	}

	if cfg.Segments.Quota || cfg.Segments.Credits {
		segments = appendUsageSegments(segments, cfg)
	}

	return fmtutil.JoinPipe(segments)
}

func appendStaleQuotaSegments(segments []string) []string {
	lastGood := usage.FetchLastGood()
	if lastGood == nil {
		return append(segments, "⏳ 7d: ?% (?d)", "⏳ 5h: ?% (?h)")
	}

	if lastGood.SevenDay != nil {
		segments = append(segments, usage.FormatStaleQuotaWindow(lastGood.SevenDay, "7d"))
	}

	if lastGood.FiveHour != nil {
		segments = append(segments, usage.FormatStaleQuotaWindow(lastGood.FiveHour, "5h"))
	}

	return segments
}

func appendRateLimitSegments(segments []string, cfg *config.Config) []string {
	lastGood := usage.FetchLastGood()
	segments = append(segments, usage.FormatRateLimitSegment(usage.FindExhaustedWindow(lastGood)))

	if cfg.Segments.Quota {
		segments = appendStaleQuotaSegments(segments)
	}

	return segments
}

func appendUsageSegments(segments []string, cfg *config.Config) []string {
	usageData, err := usage.Fetch()
	if err != nil {
		if cfg.Segments.Quota {
			segments = appendStaleQuotaSegments(segments)
		}

		return segments
	}

	switch usageData.ErrorType {
	case "":
		// no error, continue
	case "rate_limit_error":
		return appendRateLimitSegments(segments, cfg)
	default:
		return append(segments, "⚠️ /login needed")
	}

	if cfg.Segments.Quota {
		if usageData.SevenDay != nil {
			segments = append(segments, usage.FormatQuotaWindow(usageData.SevenDay, "7d"))
		}

		if usageData.FiveHour != nil {
			segments = append(segments, usage.FormatQuotaWindow(usageData.FiveHour, "5h"))
		}
	}

	if cfg.Segments.Credits && usageData.Extra != nil && usageData.Extra.UsedCredits > 0 {
		segments = append(segments, fmt.Sprintf("💳 $%.0f/$%.0f",
			usageData.Extra.UsedCredits, usageData.Extra.MonthlyLimit))
	}

	return segments
}
