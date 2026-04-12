// Package main is the entry point for the claudeline CLI.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lexfrei/claudeline/internal/compaction"
	"github.com/lexfrei/claudeline/internal/config"
	"github.com/lexfrei/claudeline/internal/fmtutil"
	"github.com/lexfrei/claudeline/internal/promotion"
	"github.com/lexfrei/claudeline/internal/status"
	"github.com/lexfrei/claudeline/internal/usage"
)

var (
	version = "dev"
	commit  = "none"
)

type stdinRateWindow struct {
	UsedPercentage float64 `json:"used_percentage"` //nolint:tagliatelle // External API format
	ResetsAt       float64 `json:"resets_at"`       //nolint:tagliatelle // External API format
}

type stdinData struct {
	Model struct {
		DisplayName string `json:"display_name"` //nolint:tagliatelle // External API format
	} `json:"model"`
	Workspace struct {
		GitWorktree string `json:"git_worktree"` //nolint:tagliatelle // External API format
	} `json:"workspace"`
	Cost struct {
		TotalCostUSD float64 `json:"total_cost_usd"` //nolint:tagliatelle // External API format
	} `json:"cost"`
	ContextWindow struct {
		UsedPercentage float64 `json:"used_percentage"` //nolint:tagliatelle // External API format
	} `json:"context_window"` //nolint:tagliatelle // External API format
	TranscriptPath string `json:"transcript_path"` //nolint:tagliatelle // External API format
	RateLimits     struct {
		FiveHour *stdinRateWindow `json:"five_hour"` //nolint:tagliatelle // External API format
		SevenDay *stdinRateWindow `json:"seven_day"` //nolint:tagliatelle // External API format
	} `json:"rate_limits"` //nolint:tagliatelle // External API format
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

		if cfg.MacInsecure {
			usage.CacheTTL = cfg.Cache.UsageTTL
		}
	}

	rootCmd.SetVersionTemplate("claudeline {{.Version}}\n")

	flags := rootCmd.PersistentFlags()
	flags.StringVar(&configPath, "config", defaultConfigPath(), "config file path")
	flags.Bool("no-model", false, "disable model segment")
	flags.Bool("no-worktree", false, "disable worktree segment")
	flags.String("cost", "", "cost segment mode: auto (default), true, false")
	flags.Bool("no-status", false, "disable status segment")
	flags.Bool("no-context", false, "disable context segment")
	flags.Bool("no-compactions", false, "disable compactions segment")
	flags.Bool("no-quota", false, "disable quota segment")
	flags.Bool("mac-insecure", false, "use macOS Keychain + Anthropic API for per-model quotas and credits")
	flags.Bool("per-model-quota", false, "enable per-model quota segments (requires --mac-insecure)")
	flags.Bool("no-credits", false, "disable credits segment (only with --mac-insecure)")
	flags.Bool("no-offpeak", false, "disable off-peak promotion indicators")

	rootCmd.AddCommand(newValidateCmd(&configPath))

	return rootCmd
}

func newValidateCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate config file",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			problems := config.Validate(*configPath)
			if len(problems) == 0 {
				fmt.Println("config ok")

				return
			}

			for _, p := range problems {
				fmt.Fprintf(os.Stderr, "error: %s\n", p)
			}

			os.Exit(1)
		},
	}
}

func applyFlagOverrides(cmd *cobra.Command, cfg *config.Config) {
	if flagSet(cmd, "no-model") {
		cfg.Segments.Model = false
	}

	if flagSet(cmd, "no-worktree") {
		cfg.Segments.Worktree = false
	}

	if flagSet(cmd, "cost") {
		if val, _ := cmd.PersistentFlags().GetString("cost"); val != "" {
			cfg.Segments.Cost = val
		}
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

	if flagSet(cmd, "mac-insecure") {
		cfg.MacInsecure = true
	}

	if flagSet(cmd, "per-model-quota") {
		cfg.Segments.PerModelQuota = true
	}

	if flagSet(cmd, "no-credits") {
		cfg.Segments.Credits = false
	}

	if flagSet(cmd, "no-offpeak") {
		cfg.Segments.OffPeak = false
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

	unmarshalErr := json.Unmarshal(raw, &data)
	if unmarshalErr != nil && len(raw) > 0 {
		fmt.Fprintf(os.Stderr, "claudeline: stdin parse error: %v\n", unmarshalErr)
	}

	var segments []string

	segments = appendIdentitySegments(segments, &data, cfg)
	segments = appendCostAndStatusSegments(segments, &data, cfg)
	segments = appendContextSegments(segments, &data, cfg)

	if cfg.Segments.Worktree && data.Workspace.GitWorktree != "" {
		segments = append(segments, "🌿 "+data.Workspace.GitWorktree)
	}

	if cfg.Segments.Quota || cfg.Segments.Credits {
		segments = appendUsageSegments(segments, &data, cfg)
	}

	return fmtutil.JoinPipe(segments)
}

// appendIdentitySegments adds model segment.
func appendIdentitySegments(segments []string, data *stdinData, cfg *config.Config) []string {
	if cfg.Segments.Model {
		model := "Claude"
		if data.Model.DisplayName != "" {
			model = data.Model.DisplayName
		}

		segments = append(segments, "🤖 "+model)
	}

	return segments
}

// appendCostAndStatusSegments adds cost and platform status segments.
func appendCostAndStatusSegments(segments []string, data *stdinData, cfg *config.Config) []string {
	if shouldShowCost(cfg.Segments.Cost, data.RateLimits.FiveHour != nil || data.RateLimits.SevenDay != nil) {
		segments = append(segments, fmt.Sprintf("💰 $%.2f", data.Cost.TotalCostUSD))
	}

	if cfg.Segments.Status {
		if alert := status.FetchAlert(); alert != "" {
			segments = append(segments, alert)
		}
	}

	return segments
}

// appendContextSegments adds context window and compaction segments.
func appendContextSegments(segments []string, data *stdinData, cfg *config.Config) []string {
	if cfg.Segments.Context && data.ContextWindow.UsedPercentage > 0 {
		segments = append(segments, fmtutil.ContextSegment(data.ContextWindow.UsedPercentage))
	}

	if cfg.Segments.Compactions {
		if compactions := compaction.CountCompactions(data.TranscriptPath); compactions > 0 {
			segments = append(segments, fmt.Sprintf("🔄 %d", compactions))
		}
	}

	return segments
}

// shouldShowCost determines whether to display the cost segment.
// In "auto" mode, cost is hidden for subscribers (who have rate_limits).
func shouldShowCost(mode string, isSubscriber bool) bool {
	switch mode {
	case config.CostOn:
		return true
	case config.CostOff:
		return false
	default: // auto
		return !isSubscriber
	}
}

// stdinRateWindowToQuota converts a stdin rate limit window to a fmtutil.QuotaWindow.
func stdinRateWindowToQuota(win *stdinRateWindow, totalMinutes int) *fmtutil.QuotaWindow {
	if win == nil {
		return nil
	}

	resetsAt := time.Unix(int64(win.ResetsAt), 0)
	remaining := max(int(time.Until(resetsAt).Minutes()), 0)

	return &fmtutil.QuotaWindow{
		Utilization:      win.UsedPercentage,
		ResetsAt:         resetsAt,
		TotalMinutes:     totalMinutes,
		RemainingMinutes: remaining,
	}
}

// buildQuotaFromStdin builds quota data from stdin rate_limits (no API call).
func buildQuotaFromStdin(data *stdinData) *fmtutil.Data {
	return &fmtutil.Data{
		FiveHour: stdinRateWindowToQuota(data.RateLimits.FiveHour, fmtutil.FiveHourWindowMinutes),
		SevenDay: stdinRateWindowToQuota(data.RateLimits.SevenDay, fmtutil.SevenDayWindowMinutes),
	}
}

func appendQuotaWindows(segments []string, data *fmtutil.Data, perModel bool, promo promotion.Status) []string {
	type labeledWindow struct {
		win   *fmtutil.QuotaWindow
		label string
	}

	windows := []labeledWindow{
		{data.SevenDay, "7d"},
		{data.FiveHour, "5h"},
	}

	if perModel {
		windows = append(windows,
			labeledWindow{data.SevenDayOpus, "7d-opus"},
			labeledWindow{data.SevenDaySonnet, "7d-sonnet"},
			labeledWindow{data.SevenDayCowork, "7d-cowork"},
			labeledWindow{data.SevenDayOAuthApps, "7d-oauth"},
		)
	}

	for _, w := range windows {
		if w.win != nil {
			segments = append(segments, fmtutil.FormatQuotaWindow(w.win, w.label, promoIndicator(w.label, promo)))
		}
	}

	return segments
}

func appendUsageSegments(segments []string, data *stdinData, cfg *config.Config) []string {
	var promo promotion.Status
	if cfg.Segments.OffPeak {
		promo = promotion.Current()
	}

	if cfg.MacInsecure {
		return appendInsecureUsageSegments(segments, cfg, promo)
	}

	return appendStdinUsageSegments(segments, data, cfg, promo)
}

// appendStdinUsageSegments builds quota segments from stdin rate_limits (default, secure path).
func appendStdinUsageSegments(segments []string, data *stdinData, cfg *config.Config, promo promotion.Status) []string {
	quotaData := buildQuotaFromStdin(data)

	if cfg.Segments.Quota {
		segments = appendQuotaWindows(segments, quotaData, false, promo)
	}

	return segments
}

// appendInsecureUsageSegments builds quota segments from Anthropic API via macOS Keychain (--mac-insecure).
func appendInsecureUsageSegments(segments []string, cfg *config.Config, promo promotion.Status) []string {
	usageData, err := usage.Fetch()
	if err != nil {
		if cfg.Segments.Quota {
			segments = appendStaleQuotaSegments(segments, cfg.Segments.PerModelQuota, promo)
		}

		return segments
	}

	switch usageData.ErrorType {
	case "":
		// no error, continue
	case "rate_limit_error":
		return appendRateLimitSegments(segments, cfg, promo)
	default:
		return append(segments, "⚠️ /login needed")
	}

	if cfg.Segments.Quota {
		segments = appendQuotaWindows(segments, usageData, cfg.Segments.PerModelQuota, promo)
	}

	if cfg.Segments.Credits && usageData.Extra != nil && usageData.Extra.UsedCredits > 0 {
		segments = append(segments, fmt.Sprintf("💳 $%.0f/$%.0f",
			usageData.Extra.UsedCredits, usageData.Extra.MonthlyLimit))
	}

	return segments
}

func appendStaleQuotaSegments(segments []string, perModel bool, promo promotion.Status) []string {
	lastGood := usage.FetchLastGood()
	if lastGood == nil {
		return append(segments, "⏳ 7d: ?% (?d)", "⏳ 5h: ?% (?h)")
	}

	type labeledWindow struct {
		win   *fmtutil.QuotaWindow
		label string
	}

	windows := []labeledWindow{
		{lastGood.SevenDay, "7d"},
		{lastGood.FiveHour, "5h"},
	}

	if perModel {
		windows = append(windows,
			labeledWindow{lastGood.SevenDayOpus, "7d-opus"},
			labeledWindow{lastGood.SevenDaySonnet, "7d-sonnet"},
			labeledWindow{lastGood.SevenDayCowork, "7d-cowork"},
			labeledWindow{lastGood.SevenDayOAuthApps, "7d-oauth"},
		)
	}

	for _, w := range windows {
		if w.win != nil {
			segments = append(segments, fmtutil.FormatStaleQuotaWindow(w.win, w.label, promoIndicator(w.label, promo)))
		}
	}

	return segments
}

func appendRateLimitSegments(segments []string, cfg *config.Config, promo promotion.Status) []string {
	lastGood := usage.FetchLastGood()
	segments = append(segments, fmtutil.FormatRateLimitSegment(fmtutil.FindExhaustedWindow(lastGood, cfg.Segments.PerModelQuota)))

	if cfg.Segments.Quota {
		segments = appendStaleQuotaSegments(segments, cfg.Segments.PerModelQuota, promo)
	}

	return segments
}

func promoIndicator(label string, promo promotion.Status) string {
	if !promo.Active {
		return ""
	}

	switch {
	case label == "5h" || strings.HasPrefix(label, "5h-"):
		return promo.FiveHour
	case label == "7d" || strings.HasPrefix(label, "7d-"):
		return promo.SevenDay
	default:
		return ""
	}
}
