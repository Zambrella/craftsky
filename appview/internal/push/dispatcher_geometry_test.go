package push

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

type geometrySender struct{}

func (geometrySender) Send(context.Context, SendRequest) (ProviderResult, error) {
	return ProviderResult{Class: ResultSuccess}, nil
}

func TestNewDispatcherValidatedRejectsUnsafeLeaseGeometry(t *testing.T) {
	tests := []struct {
		name    string
		options DispatcherOptions
		wantKey string
	}{
		{
			name: "concurrency is positive",
			options: DispatcherOptions{
				BatchSize: 1, Concurrency: -1, LeaseDuration: time.Minute,
				SendTimeout: time.Second, FinalizationMargin: time.Second,
			},
			wantKey: "PUSH_CONCURRENCY",
		},
		{
			name: "concurrency cannot exceed the poll budget",
			options: DispatcherOptions{
				BatchSize: 1, Concurrency: 2, LeaseDuration: time.Minute,
				SendTimeout: time.Second, FinalizationMargin: time.Second,
			},
			wantKey: "PUSH_CONCURRENCY",
		},
		{
			name: "margin is positive",
			options: DispatcherOptions{
				BatchSize: 1, Concurrency: 1, LeaseDuration: time.Minute,
				SendTimeout: time.Second, FinalizationMargin: -time.Second,
			},
			wantKey: "PUSH_FINALIZATION_MARGIN",
		},
		{
			name: "lease contains send and finalization windows",
			options: DispatcherOptions{
				BatchSize: 1, Concurrency: 1, LeaseDuration: 11 * time.Second,
				SendTimeout: 10 * time.Second, FinalizationMargin: time.Second,
			},
			wantKey: "PUSH_LEASE_DURATION",
		},
		{
			name: "duration addition cannot overflow validation",
			options: DispatcherOptions{
				BatchSize: 1, Concurrency: 1,
				LeaseDuration:      time.Duration(1<<63 - 1),
				SendTimeout:        time.Duration(1<<63-1) - time.Second,
				FinalizationMargin: 2 * time.Second,
			},
			wantKey: "PUSH_LEASE_DURATION",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDispatcherValidated(nil, geometrySender{}, test.options)
			if err == nil || !strings.Contains(err.Error(), test.wantKey) {
				t.Fatalf("error = %v, want key %s", err, test.wantKey)
			}
		})
	}
}

func TestNewDispatcherValidatedDefaultsToSafeBoundedGeometry(t *testing.T) {
	dispatcher, err := NewDispatcherValidated(nil, geometrySender{}, DispatcherOptions{
		LifecycleFence: permissiveLifecycleFence{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dispatcher.options.Concurrency < 1 ||
		dispatcher.options.Concurrency > dispatcher.options.BatchSize {
		t.Fatalf("unsafe default concurrency geometry: %+v", dispatcher.options)
	}
	if dispatcher.options.FinalizationMargin <= 0 ||
		dispatcher.options.LeaseDuration <=
			dispatcher.options.SendTimeout+dispatcher.options.FinalizationMargin {
		t.Fatalf("unsafe default lease geometry: %+v", dispatcher.options)
	}
}

func TestAttemptDeadlineUsesEarliestBoundAndRequiresUsefulWindow(t *testing.T) {
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		delivery   time.Time
		lease      time.Time
		timeout    time.Duration
		margin     time.Duration
		want       time.Time
		wantUsable bool
	}{
		{
			name:     "send timeout",
			delivery: now.Add(time.Hour), lease: now.Add(time.Minute),
			timeout: 10 * time.Second, margin: 5 * time.Second,
			want: now.Add(10 * time.Second), wantUsable: true,
		},
		{
			name:     "delivery deadline",
			delivery: now.Add(3 * time.Second), lease: now.Add(time.Minute),
			timeout: 10 * time.Second, margin: 5 * time.Second,
			want: now.Add(3 * time.Second), wantUsable: true,
		},
		{
			name:     "remaining lease window after elapsed claim time",
			delivery: now.Add(time.Hour), lease: now.Add(7 * time.Second),
			timeout: 10 * time.Second, margin: 5 * time.Second,
			want: now.Add(2 * time.Second), wantUsable: true,
		},
		{
			name:     "no finalization window",
			delivery: now.Add(time.Hour), lease: now.Add(5 * time.Second),
			timeout: 10 * time.Second, margin: 5 * time.Second,
			want: now, wantUsable: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, usable := attemptDeadline(
				now,
				test.delivery,
				test.lease,
				test.timeout,
				test.margin,
			)
			if !got.Equal(test.want) || usable != test.wantUsable {
				t.Fatalf("deadline=%s usable=%v, want %s/%v", got, usable, test.want, test.wantUsable)
			}
		})
	}
}

func TestNewDispatcherValidatedRejectsMissingLifecycleFence(t *testing.T) {
	_, err := NewDispatcherValidated(nil, geometrySender{}, DispatcherOptions{})
	if err == nil || !strings.Contains(err.Error(), "owner lifecycle fence") {
		t.Fatalf("error = %v, want owner lifecycle fence", err)
	}
}

func TestDecodeClaimDIDsPreservesOptionalActor(t *testing.T) {
	recipient, actor, err := decodeClaimDIDs("did:plc:viewer", sql.NullString{})
	if err != nil {
		t.Fatal(err)
	}
	if recipient.String() != "did:plc:viewer" || actor != nil {
		t.Fatalf("recipient=%q actor=%v", recipient, actor)
	}

	_, actor, err = decodeClaimDIDs(
		"did:plc:viewer",
		sql.NullString{String: "did:plc:actor", Valid: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if actor == nil || actor.String() != "did:plc:actor" {
		t.Fatalf("actor=%v", actor)
	}
}

func TestDecodeClaimDIDsRejectsMalformedIdentifier(t *testing.T) {
	if _, _, err := decodeClaimDIDs("not-a-did", sql.NullString{}); err == nil {
		t.Fatal("malformed recipient DID was accepted")
	}
	if _, _, err := decodeClaimDIDs(
		"did:plc:viewer",
		sql.NullString{String: "not-a-did", Valid: true},
	); err == nil {
		t.Fatal("malformed actor DID was accepted")
	}
}
