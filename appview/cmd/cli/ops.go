package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/ops"
	"social.craftsky/appview/internal/tap"
)

type opsReport struct {
	ops.Snapshot
	Health opsHealth `json:"health"`
}

// Deliberately excludes healthz's raw last_error and all row-level diagnostics.
type opsHealth struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Tap    struct {
		Connected *bool          `json:"connected"`
		Telemetry *tap.Telemetry `json:"telemetry,omitempty"`
	} `json:"tap"`
	ImageSafety *struct {
		Ready    bool  `json:"ready"`
		Required *bool `json:"required,omitempty"`
	} `json:"imageSafety,omitempty"`
}

type opsStatusLoader func(context.Context, string, int) (opsReport, error)

func newOpsCmd(load opsStatusLoader) *cobra.Command {
	command := &cobra.Command{Use: "ops", Short: "Read-only operational summaries"}
	var jsonOutput bool
	var baseURL string
	var timeout time.Duration
	var retryThreshold int
	status := &cobra.Command{
		Use: "status", Short: "Summarize AppView health and unfinished background work",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if timeout <= 0 || timeout > time.Minute {
				return errors.New("timeout must be positive and at most 1m")
			}
			if retryThreshold < 1 || retryThreshold > 1000 {
				return errors.New("retry-threshold must be between 1 and 1000")
			}
			if baseURL == "" {
				baseURL = os.Getenv("APPVIEW_URL")
			}
			if baseURL != "" {
				if err := validateOpsURL(baseURL); err != nil {
					return err
				}
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			report, err := load(ctx, baseURL, retryThreshold)
			if err != nil {
				return err
			}
			return writeOpsReport(cmd.OutOrStdout(), report, jsonOutput)
		},
	}
	status.Flags().BoolVar(&jsonOutput, "json", false, "emit the aggregate snapshot as JSON")
	status.Flags().StringVar(&baseURL, "url", "", "AppView origin (default APPVIEW_URL, configured public origin, or local PORT)")
	status.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "total database and health request budget (up to 1m)")
	status.Flags().IntVar(&retryThreshold, "retry-threshold", 5, "unfinished rows at or above this attempt count are highlighted")
	command.AddCommand(status)
	return command
}

func loadOpsStatus(ctx context.Context, baseURL string, retryThreshold int) (opsReport, error) {
	env, err := parseEnvFlag()
	if err != nil {
		return opsReport{}, err
	}
	cfg, err := loadCfgLight(env)
	if err != nil {
		return opsReport{}, err
	}
	if baseURL == "" {
		baseURL = defaultOpsOrigin(cfg.OAuth.PublicOrigin.String())
	}
	if err := validateOpsURL(baseURL); err != nil {
		return opsReport{}, err
	}
	// Construct only a database pool; dependency construction can initialize
	// external providers and is unnecessary for this read-only operation.
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return opsReport{}, err
	}
	defer pool.Close()
	snapshot, err := ops.ReadSnapshot(ctx, pool, retryThreshold)
	if err != nil {
		return opsReport{}, err
	}
	return opsReport{Snapshot: snapshot, Health: readOpsHealth(ctx, baseURL)}, nil
}

func defaultOpsOrigin(publicOrigin string) string {
	// Production rejects loopback Host headers for /healthz. Use the configured
	// canonical origin instead; localhost defaults apply only to local OAuth.
	if publicOrigin != "" {
		return publicOrigin
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return "http://" + net.JoinHostPort("127.0.0.1", port)
}

func validateOpsURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return errors.New("url must be an HTTP(S) origin without credentials, path, query or fragment")
	}
	return nil
}

func readOpsHealth(ctx context.Context, baseURL string) opsHealth {
	unavailable := opsHealth{Status: "unavailable", DB: "unknown"}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/healthz", nil)
	if err != nil {
		return unavailable
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Do(request)
	if err != nil {
		return unavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return unavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(body) > 65536 {
		return unavailable
	}
	var health opsHealth
	if json.Unmarshal(body, &health) != nil || (health.Status != "ok" && health.Status != "degraded") ||
		(health.DB != "ok" && health.DB != "error") || health.Tap.Connected == nil {
		return unavailable
	}
	if telemetry := health.Tap.Telemetry; telemetry != nil {
		switch telemetry.Status {
		case "progressing", "stalled", "unknown":
		default:
			telemetry.Status = "unknown"
		}
	}
	return health
}

func writeOpsReport(out io.Writer, report opsReport, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "Snapshot: %s (database read-only)\n", report.CollectedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(writer, "AppView: %s; reported DB: %s\n", report.Health.Status, report.Health.DB)
	if report.Health.Tap.Connected != nil {
		fmt.Fprintf(writer, "Tap connected: %t\n", *report.Health.Tap.Connected)
	}
	if telemetry := report.Health.Tap.Telemetry; telemetry != nil {
		fmt.Fprintf(writer, "Tap progress: %s (cursor progress does not prove relay-head freshness)\n", telemetry.Status)
		fmt.Fprintf(writer, "Tap cursor: %s; outbox: %s; resync: %s\n",
			opsCounter(telemetry.FirehoseCursor), opsCounter(telemetry.OutboxDepth), opsCounter(telemetry.ResyncDepth))
		if telemetry.SampledAt != nil {
			fmt.Fprintf(writer, "Tap sample: %s\n", telemetry.SampledAt.UTC().Format(time.RFC3339))
		}
	}
	if report.Health.ImageSafety != nil {
		fmt.Fprintf(writer, "Image safety ready: %t\n", report.Health.ImageSafety.Ready)
	}
	fmt.Fprintf(writer, "\nQUEUE\tPENDING\tDUE\tOVERDUE\tOLDEST PENDING\tOLDEST DUE\tATTEMPTS >= %d\tMAX ATTEMPTS\tSTATES\n", report.RetryThreshold)
	for _, queue := range report.Queues {
		states := make([]string, 0, len(queue.States))
		for state, count := range queue.States {
			states = append(states, fmt.Sprintf("%s=%d", state, count))
		}
		sort.Strings(states)
		fmt.Fprintf(writer, "%s\t%d\t%d\t%d\t%s\t%s\t%d\t%d\t%s\n", queue.Name,
			queue.Pending, queue.Due, queue.Overdue, opsAge(report.CollectedAt, queue.OldestPendingAt),
			opsAge(report.CollectedAt, queue.OldestDueAt), queue.RepeatedAttempts, queue.MaxAttempts, strings.Join(states, ", "))
	}
	fmt.Fprintln(writer, "\nOverdue: due for at least 60s. Repeated attempts are not a failure verdict. No work is retried or changed.")
	return writer.Flush()
}

func opsCounter(value *int64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%d", *value)
}

func opsAge(now time.Time, value *time.Time) string {
	if value == nil {
		return "-"
	}
	age := now.Sub(*value)
	if age < 0 {
		age = 0
	}
	return age.Round(time.Second).String()
}

func init() {
	rootCmd.AddCommand(newOpsCmd(loadOpsStatus))
}
