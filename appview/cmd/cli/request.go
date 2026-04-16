package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// requestArgs is the testable surface of the request subcommand. Passing
// the dependencies in as a struct keeps doRequest pure — no reading from
// cobra flags or environment inside the function.
type requestArgs struct {
	Method  string
	Path    string
	BaseURL string    // e.g. "http://localhost:8080"
	DevDID  string    // empty disables the X-Dev-DID header
	Body    []byte    // nil = no body
	Headers []string  // extra headers as "Key: Value"
	Out     io.Writer // stdout in real runs; bytes.Buffer in tests
	ErrOut  io.Writer // stderr in real runs; bytes.Buffer in tests
}

// doRequest sends one HTTP request using args and writes the status line
// + body to args.Out. Returns (exitCode, internalErr). exitCode follows
// the contract:
//
//	0 — 2xx response
//	1 — 4xx/5xx response
//	2 — transport error (couldn't reach server)
//
// internalErr is non-nil only for bugs (bad args, write failures).
func doRequest(args requestArgs) (int, error) {
	body := io.Reader(nil)
	if args.Body != nil {
		body = bytes.NewReader(args.Body)
	}

	req, err := http.NewRequest(args.Method, args.BaseURL+args.Path, body)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	if args.DevDID != "" {
		req.Header.Set("Authorization", "Bearer dev")
		req.Header.Set("X-Dev-DID", args.DevDID)
	}
	for _, h := range args.Headers {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			return 0, fmt.Errorf("bad header %q (want 'Key: Value')", h)
		}
		req.Header.Add(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Transport errors are non-success outcomes; they go to stderr
		// so scripts piping stdout (`cli request ... | jq`) don't mix
		// body bytes with error diagnostics.
		fmt.Fprintf(args.ErrOut, "transport error: %s\n", err)
		return 2, nil
	}
	defer resp.Body.Close()

	// First line: "<code> <text>\n"
	if _, err := fmt.Fprintf(args.Out, "%d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode)); err != nil {
		return 0, err
	}
	// Body verbatim.
	if _, err := io.Copy(args.Out, resp.Body); err != nil {
		return 0, err
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return 0, nil
	default:
		return 1, nil
	}
}

var (
	reqHeaderFlag []string
	reqBodyFlag   string
	reqDIDFlag    string
)

var requestCmd = &cobra.Command{
	Use:   "request METHOD PATH",
	Short: "Send an HTTP request to the running appview server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := parseEnvFlag()
		if err != nil {
			return err
		}
		cfg, err := loadCfgLight(env)
		if err != nil {
			return err
		}

		base, err := resolveBaseURL(env)
		if err != nil {
			return err
		}

		did := reqDIDFlag
		if did == "" && env == devEnvMarker {
			did = cfg.DevDID
		}

		var body []byte
		if reqBodyFlag != "" {
			body = []byte(reqBodyFlag)
		}

		code, err := doRequest(requestArgs{
			Method:  strings.ToUpper(args[0]),
			Path:    args[1],
			BaseURL: base,
			DevDID:  did,
			Body:    body,
			Headers: reqHeaderFlag,
			Out:     os.Stdout,
			ErrOut:  os.Stderr,
		})
		if err != nil {
			return err
		}
		// Cobra's RunE-to-exit-code mapping is binary (nil→0, non-nil→1).
		// This subcommand needs a tri-state exit: 0 = 2xx, 1 = 4xx/5xx,
		// 2 = transport error. os.Exit is safe here because requestCmd
		// holds no resources — no DB pool, no open files. Any deferred
		// cleanup in this goroutine has already run by the time
		// doRequest returns.
		if code != 0 {
			_ = os.Stdout.Sync()
			_ = os.Stderr.Sync()
			os.Exit(code)
		}
		return nil
	},
}

func init() {
	requestCmd.Flags().StringArrayVarP(&reqHeaderFlag, "header", "H", nil, "extra header 'Key: Value' (may repeat)")
	requestCmd.Flags().StringVarP(&reqBodyFlag, "data", "d", "", "request body")
	requestCmd.Flags().StringVar(&reqDIDFlag, "did", "", "override the dev DID sent in X-Dev-DID (dev env only)")
	rootCmd.AddCommand(requestCmd)
}
