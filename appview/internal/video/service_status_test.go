package video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicJobStatusClientUsesExactTokenlessBoundedRequest(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/xrpc/app.bsky.video.getJobStatus" || r.URL.Query().Get("jobId") != "job-1" {
			t.Fatalf("request = %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		_, _ = w.Write([]byte(`{"jobStatus":{"jobId":"job-1","did":"did:plc:alice","state":"JOB_STATE_PROCESSING"}}`))
	}))
	t.Cleanup(server.Close)

	client, err := NewPublicJobStatusClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewPublicJobStatusClient: %v", err)
	}
	status, err := client.GetJobStatus(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if status.JobId != "job-1" || status.Did != "did:plc:alice" {
		t.Fatalf("status = %+v", status)
	}
}

func TestPublicJobStatusClientFailsClosedOnRedirectAndOversizedResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{name: "redirect", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://example.com/leak", http.StatusFound)
		})},
		{name: "oversized", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", maxJobStatusResponseBytes+1)))
		})},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewTLSServer(test.handler)
			t.Cleanup(server.Close)
			client, err := NewPublicJobStatusClient(server.URL, server.Client())
			if err != nil {
				t.Fatalf("NewPublicJobStatusClient: %v", err)
			}
			if _, err := client.GetJobStatus(context.Background(), "job-1"); err == nil || strings.Contains(err.Error(), "job-1") {
				t.Fatalf("error = %v, want redacted failure", err)
			}
		})
	}
}
