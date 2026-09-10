package revenuecat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/subscriptions"
)

type WebhookConfig struct {
	Authorization      string
	SigningSecret      string
	BodyLimit          int64
	IngressDeadline    time.Duration
	SignatureTolerance time.Duration
	Observer           WebhookObserver
}

type WebhookObserver interface {
	ObserveRevenueCatWebhook(context.Context, string, time.Duration)
}

type EventStore interface {
	AcceptRevenueCatEvent(context.Context, subscriptions.RevenueCatEvent, time.Time) (bool, error)
}

func NewWebhookHandler(config WebhookConfig, store EventStore, now func() time.Time) (http.Handler, error) {
	if config.Authorization == "" || config.SigningSecret == "" || config.BodyLimit <= 0 ||
		config.IngressDeadline <= 0 || config.IngressDeadline > 10*time.Second ||
		config.SignatureTolerance <= 0 || store == nil || now == nil {
		return nil, errors.New("invalid RevenueCat webhook configuration")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), config.IngressDeadline)
		defer cancel()
		r = r.WithContext(ctx)
		started := time.Now()
		observe := func(outcome string) {
			if config.Observer != nil {
				config.Observer.ObserveRevenueCatWebhook(r.Context(), outcome, time.Since(started))
			}
		}
		stopDeadlineClose := context.AfterFunc(ctx, func() { _ = r.Body.Close() })
		defer stopDeadlineClose()
		body, err := io.ReadAll(io.LimitReader(r.Body, config.BodyLimit+1))
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				observe("deadline_exceeded")
				http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
			observe("malformed")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if int64(len(body)) > config.BodyLimit {
			observe("oversized")
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			observe("deadline_exceeded")
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		acceptedAt := now()
		if err := VerifyWebhookAuthentication(r.Header, body, config.Authorization, config.SigningSecret, acceptedAt, config.SignatureTolerance); err != nil {
			observe("authentication_failed")
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}
		var envelope struct {
			APIVersion string `json:"api_version"`
			Event      struct {
				ID               string `json:"id"`
				Type             string `json:"type"`
				AppUserID        string `json:"app_user_id"`
				EventTimestampMS int64  `json:"event_timestamp_ms"`
			} `json:"event"`
		}
		if json.Unmarshal(body, &envelope) != nil || envelope.APIVersion == "" ||
			envelope.Event.ID == "" || envelope.Event.Type == "" || envelope.Event.AppUserID == "" ||
			envelope.Event.EventTimestampMS <= 0 {
			observe("malformed")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		customerID, err := uuid.Parse(envelope.Event.AppUserID)
		if err != nil {
			observe("malformed")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		accepted, err := store.AcceptRevenueCatEvent(ctx, subscriptions.RevenueCatEvent{
			ID: envelope.Event.ID, Type: envelope.Event.Type,
			RevenueCatAppUserID: customerID,
			OccurredAt:          time.UnixMilli(envelope.Event.EventTimestampMS).UTC(),
		}, acceptedAt)
		if err != nil {
			observe("store_error")
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if accepted {
			observe("accepted")
		} else {
			observe("duplicate")
		}
		w.WriteHeader(http.StatusOK)
	}), nil
}
