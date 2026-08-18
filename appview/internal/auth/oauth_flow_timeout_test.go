package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNormalizeOAuthOperationTimeout(t *testing.T) {
	tests := []struct {
		name       string
		configured time.Duration
		fallback   time.Duration
		maximum    time.Duration
		want       time.Duration
		wantErr    bool
	}{
		{name: "default", fallback: 20 * time.Second, maximum: time.Minute, want: 20 * time.Second},
		{name: "lower deployment value", configured: 12 * time.Second, fallback: 20 * time.Second, maximum: time.Minute, want: 12 * time.Second},
		{name: "negative", configured: -time.Second, fallback: 20 * time.Second, maximum: time.Minute, wantErr: true},
		{name: "above security ceiling", configured: 61 * time.Second, fallback: 20 * time.Second, maximum: time.Minute, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeOAuthOperationTimeout(test.configured, test.fallback, test.maximum)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeOAuthOperationTimeout() error = %v, wantErr %t", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("normalizeOAuthOperationTimeout() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestOAuthOperationContextPreservesCancellationAndEarlierDeadline(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	child, cancelChild := oauthOperationContext(parent, time.Minute)
	cancelParent()
	select {
	case <-child.Done():
		if !errors.Is(child.Err(), context.Canceled) {
			t.Fatalf("canceled child error = %v", child.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("child did not preserve parent cancellation")
	}
	cancelChild()

	earlier, cancelEarlier := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelEarlier()
	bounded, cancelBounded := oauthOperationContext(earlier, time.Minute)
	defer cancelBounded()
	earlierDeadline, _ := earlier.Deadline()
	boundedDeadline, _ := bounded.Deadline()
	if !boundedDeadline.Equal(earlierDeadline) {
		t.Fatalf("bounded deadline = %v, want parent deadline %v", boundedDeadline, earlierDeadline)
	}
}
