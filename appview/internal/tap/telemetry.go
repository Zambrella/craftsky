package tap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	telemetryPollInterval = 30 * time.Second
	telemetryStaleAfter   = 2 * time.Minute
	telemetryStallAfter   = 10 * time.Minute
)

// Telemetry is a cached, public-safe summary. Progress is not proof of being
// caught up: relay lag stays unknown until an independent reference exists.
type Telemetry struct {
	Status              string     `json:"status"`
	FirehoseCursor      *int64     `json:"firehoseCursor"`
	OutboxDepth         *int64     `json:"outboxDepth"`
	ResyncDepth         *int64     `json:"resyncDepth"`
	SampledAt           *time.Time `json:"sampledAt"`
	LastCursorAdvanceAt *time.Time `json:"lastCursorAdvanceAt"`
	RelayLagSeconds     *float64   `json:"relayLagSeconds"`
}

type telemetryConsumer struct {
	Consumer
	admin      *AdminClient
	mu         sync.RWMutex
	sample     Telemetry
	baselineAt time.Time
	failed     bool
}

// WithTelemetry polls private Tap statistics independently of health requests.
func WithTelemetry(consumer Consumer, admin *AdminClient) Consumer {
	return &telemetryConsumer{Consumer: consumer, admin: admin}
}

func (c *telemetryConsumer) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.poll(ctx)
		ticker := time.NewTicker(telemetryPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.poll(ctx)
			}
		}
	}()
	err := c.Consumer.Run(ctx)
	cancel()
	<-done
	return err
}

func (c *telemetryConsumer) poll(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sample, err := c.admin.telemetry(ctx)
	c.record(sample, err, time.Now().UTC())
}

func (c *telemetryConsumer) record(sample Telemetry, err error, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failed = err != nil
	if err != nil {
		return
	}
	previous := c.sample
	if previous.FirehoseCursor == nil || *sample.FirehoseCursor < *previous.FirehoseCursor {
		// Tap restart or cursor reset: establish a new observation baseline.
		c.baselineAt = now
	} else {
		sample.LastCursorAdvanceAt = previous.LastCursorAdvanceAt
		if *sample.FirehoseCursor > *previous.FirehoseCursor {
			sample.LastCursorAdvanceAt = &now
		}
	}
	sample.SampledAt = &now
	c.sample = sample
}

func (c *telemetryConsumer) Telemetry() Telemetry {
	return c.snapshot(time.Now().UTC())
}

func (c *telemetryConsumer) snapshot(now time.Time) Telemetry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := c.sample
	s.Status = "unknown"
	if c.failed || s.SampledAt == nil || now.Sub(*s.SampledAt) > telemetryStaleAfter {
		return s
	}
	last := c.baselineAt
	if s.LastCursorAdvanceAt != nil {
		last = *s.LastCursorAdvanceAt
	}
	if now.Sub(last) >= telemetryStallAfter {
		s.Status = "stalled"
	} else if s.LastCursorAdvanceAt != nil {
		s.Status = "progressing"
	}
	return s
}

func (c *AdminClient) telemetry(ctx context.Context) (Telemetry, error) {
	var cursor struct {
		Firehose *int64 `json:"firehose"`
	}
	var outbox struct {
		Depth *int64 `json:"outbox_buffer"`
	}
	var resync struct {
		Depth *int64 `json:"resync_buffer"`
	}
	for _, request := range []struct {
		path   string
		target any
	}{
		{"/stats/cursors", &cursor},
		{"/stats/outbox-buffer", &outbox},
		{"/stats/resync-buffer", &resync},
	} {
		if err := c.getStat(ctx, request.path, request.target); err != nil {
			return Telemetry{}, err
		}
	}
	if cursor.Firehose == nil || outbox.Depth == nil || resync.Depth == nil || *cursor.Firehose < 0 || *outbox.Depth < 0 || *resync.Depth < 0 {
		return Telemetry{}, fmt.Errorf("invalid tap statistics")
	}
	return Telemetry{FirehoseCursor: cursor.Firehose, OutboxDepth: outbox.Depth, ResyncDepth: resync.Depth}, nil
}

func (c *AdminClient) getStat(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tap statistics HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(target)
}
