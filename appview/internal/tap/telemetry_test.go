package tap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTelemetryProgressFailureAndRestart(t *testing.T) {
	now := time.Now().UTC()
	c := &telemetryConsumer{}
	sample := func(cursor int64) Telemetry { return Telemetry{FirehoseCursor: &cursor} }
	c.record(sample(100), nil, now)
	if c.snapshot(now).Status != "unknown" {
		t.Fatal("first sample must not prove progress")
	}
	c.record(sample(110), nil, now.Add(time.Minute))
	if c.snapshot(now.Add(time.Minute)).Status != "progressing" {
		t.Fatal("missing progress")
	}
	// Successful sampling must not conceal a stuck cursor.
	c.record(sample(110), nil, now.Add(12*time.Minute))
	if c.snapshot(now.Add(12*time.Minute)).Status != "stalled" {
		t.Fatal("missing stall")
	}
	c.record(Telemetry{}, errors.New("private URL should not be public"), now.Add(13*time.Minute))
	s := c.snapshot(now.Add(13 * time.Minute))
	if s.Status != "unknown" || !s.SampledAt.Equal(now.Add(12*time.Minute)) || *s.FirehoseCursor != 110 {
		t.Fatal("failed poll must retain old sample and report unknown")
	}
	c.record(sample(2), nil, now.Add(14*time.Minute))
	if s := c.snapshot(now.Add(14 * time.Minute)); s.Status != "unknown" || s.LastCursorAdvanceAt != nil {
		t.Fatal("restart must reset progress baseline")
	}
	c.record(sample(3), nil, now.Add(15*time.Minute))
	if c.snapshot(now.Add(18*time.Minute)).Status != "unknown" {
		t.Fatal("stale sample must not be healthy")
	}
}

func TestTelemetryStatsContract(t *testing.T) {
	for _, bad := range []string{"", "/stats/cursors", "/stats/outbox-buffer", "/stats/resync-buffer"} {
		t.Run(bad, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == bad {
					_, _ = w.Write([]byte(`{}`))
					return
				}
				switch r.URL.Path {
				case "/stats/cursors":
					_, _ = w.Write([]byte(`{"firehose":123}`))
				case "/stats/outbox-buffer":
					_, _ = w.Write([]byte(`{"outbox_buffer":4}`))
				case "/stats/resync-buffer":
					_, _ = w.Write([]byte(`{"resync_buffer":5}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			admin, err := NewAdminClient(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			s, err := admin.telemetry(context.Background())
			if bad != "" {
				if err == nil {
					t.Fatal("missing statistic accepted as zero")
				}
				return
			}
			if err != nil || *s.FirehoseCursor != 123 || *s.OutboxDepth != 4 || *s.ResyncDepth != 5 || s.RelayLagSeconds != nil {
				t.Fatalf("stats: %+v, %v", s, err)
			}
		})
	}
}

type waitingConsumer struct{}

func (waitingConsumer) State() ConnState              { return ConnState{} }
func (waitingConsumer) Run(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }

func TestTelemetryRunCancelsInFlightPoll(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	admin, _ := NewAdminClient(server.URL, server.Client())
	consumer := WithTelemetry(waitingConsumer{}, admin)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("no initial poll")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("poll prevented shutdown")
	}
}
