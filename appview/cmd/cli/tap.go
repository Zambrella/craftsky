package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var tapCmd = &cobra.Command{
	Use:   "tap",
	Short: "Tap consumer operations",
}

var tapStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print Tap consumer status from the running appview",
	Run: func(cmd *cobra.Command, args []string) {
		url := os.Getenv("APPVIEW_URL")
		if url == "" {
			url = "http://localhost:8080"
		}
		os.Exit(tapStatus(url, cmd.OutOrStdout()))
	},
}

func init() {
	tapCmd.AddCommand(tapStatusCmd)
}

type healthTap struct {
	Connected        bool   `json:"connected"`
	LastEventAt      string `json:"last_event_at"`
	ReconnectAttempt int    `json:"reconnect_attempt"`
	LastError        string `json:"last_error"`
}

type healthDoc struct {
	Tap healthTap `json:"tap"`
}

// tapStatus fetches /healthz and prints the tap block. Returns the shell
// exit code: 0 connected, 1 disconnected, 2 transport/parse error.
func tapStatus(baseURL string, out io.Writer) int {
	if out == nil {
		out = os.Stdout
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/healthz")
	if err != nil {
		fmt.Fprintf(out, "transport error: %v\n", err)
		return 2
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(out, "read error: %v\n", err)
		return 2
	}
	var doc healthDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		fmt.Fprintf(out, "parse error: %v\n", err)
		return 2
	}

	fmt.Fprintf(out, "connected:         %t\n", doc.Tap.Connected)
	fmt.Fprintf(out, "last_event_at:     %s%s\n", doc.Tap.LastEventAt, relSuffix(doc.Tap.LastEventAt))
	fmt.Fprintf(out, "reconnect_attempt: %d\n", doc.Tap.ReconnectAttempt)
	if doc.Tap.LastError != "" {
		fmt.Fprintf(out, "last_error:        %s\n", doc.Tap.LastError)
	}

	if doc.Tap.Connected {
		return 0
	}
	return 1
}

func relSuffix(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(" (%s ago)", time.Since(t).Round(time.Second))
}
