package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"social.craftsky/appview/internal/observability"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/subscriptions"
)

type BillingAccountStore interface {
	EnsureAccount(context.Context, syntax.DID) (subscriptions.BillingAccount, bool, error)
	OwnerAccount(context.Context, syntax.DID) (subscriptions.BillingAccount, error)
	OwnerState(context.Context, syntax.DID, time.Time) (subscriptions.BillingState, error)
	RequestReconciliation(context.Context, uuid.UUID, time.Time) error
}

type SubscriptionAccessStore interface {
	SelfAccess(context.Context, syntax.DID, time.Time) (subscriptions.SelfAccess, error)
}

func EnsureBillingAccountHandler(store BillingAccountStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		_, created, err := store.EnsureAccount(r.Context(), owner)
		if err != nil {
			writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		state, err := store.OwnerState(r.Context(), owner, now())
		if err != nil {
			writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		envelope.WriteJSON(w, status, state)
	})
}

func GetBillingAccountHandler(store BillingAccountStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		account, err := store.OwnerState(r.Context(), owner, now())
		if err != nil {
			writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, account)
	})
}

func GetSubscriptionAccessHandler(store SubscriptionAccessStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		access, err := store.SelfAccess(r.Context(), did, now())
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetSubscriptionAccessHandler", "handler")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "subscription access unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, access)
	})
}

func RequestSubscriptionReconciliationHandler(store BillingAccountStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		account, err := store.OwnerAccount(r.Context(), owner)
		if err == nil {
			if resolver, ok := store.(interface {
				ResolveWebCheckoutForDeletion(context.Context, syntax.DID, time.Time) error
			}); ok {
				if resolveErr := resolver.ResolveWebCheckoutForDeletion(r.Context(), owner, now()); resolveErr != nil {
					writeBillingAccountError(w, middleware.GetRunID(r.Context()), resolveErr)
					return
				}
			}
			err = store.RequestReconciliation(r.Context(), account.ID, now())
		}
		if err != nil {
			writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		envelope.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
	})
}

func subscriptionOwnerScope(w http.ResponseWriter, r *http.Request) (syntax.DID, bool) {
	owner, ok := middleware.GetDID(r.Context())
	if !ok || owner == "" {
		envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_scope", "authenticated account scope missing", middleware.GetRunID(r.Context()), nil)
		return "", false
	}
	return owner, true
}

func writeBillingAccountError(w http.ResponseWriter, runID string, err error) {
	if errors.Is(err, subscriptions.ErrBillingAccountNotFound) {
		envelope.WriteError(w, http.StatusNotFound, "billing_account_not_found", "billing account not found", runID, nil)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "billing operation failed", runID, nil)
}

type WebSubscriptionCheckoutStore interface {
	WebCheckoutURL(context.Context, syntax.DID, subscriptions.Tier, time.Time) (string, error)
}

func CreateWebSubscriptionCheckoutHandler(store WebSubscriptionCheckoutStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		var body struct {
			Tier subscriptions.Tier `json:"tier"`
		}
		if err := decodeStrictJSONObject(r, &body); err != nil || !body.Tier.Paid() {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "choose a valid subscription tier", middleware.GetRunID(r.Context()), nil)
			return
		}
		destination, err := store.WebCheckoutURL(r.Context(), owner, body.Tier, now())
		if err != nil {
			switch {
			case errors.Is(err, subscriptions.ErrWebCheckoutUnavailable):
				envelope.WriteError(w, http.StatusServiceUnavailable, "web_checkout_unavailable", "web checkout is unavailable", middleware.GetRunID(r.Context()), nil)
			case errors.Is(err, subscriptions.ErrSubscriptionReconciliationRequired):
				envelope.WriteError(w, http.StatusConflict, "subscription_reconciliation_required", "refresh subscription status before checkout", middleware.GetRunID(r.Context()), nil)
			case errors.Is(err, subscriptions.ErrWebCheckoutPending):
				envelope.WriteError(w, http.StatusConflict, "web_checkout_pending", "finish or wait for the existing subscription checkout", middleware.GetRunID(r.Context()), nil)
			case errors.Is(err, subscriptions.ErrSubscriptionAlreadyExists):
				envelope.WriteError(w, http.StatusConflict, "subscription_already_exists", "this account already has a subscription", middleware.GetRunID(r.Context()), nil)
			default:
				writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			}
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteJSON(w, http.StatusOK, map[string]string{"url": destination})
	})
}

type WebSubscriptionManagementStore interface {
	WebManagementURL(context.Context, syntax.DID, time.Time) (string, error)
}

func CreateWebSubscriptionManagementHandler(store WebSubscriptionManagementStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		// The route accepts no body; server-side identity is the only authority.
		var probe [1]byte
		if r.Body != nil {
			n, err := r.Body.Read(probe[:])
			if n != 0 || (err != nil && err != io.EOF) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "billing management accepts no request body", middleware.GetRunID(r.Context()), nil)
				return
			}
		}
		destination, err := store.WebManagementURL(r.Context(), owner, now())
		if err != nil {
			if errors.Is(err, subscriptions.ErrWebManagementUnavailable) || errors.Is(err, subscriptions.ErrWebCheckoutUnavailable) {
				envelope.WriteError(w, http.StatusServiceUnavailable, "web_management_unavailable", "subscription management is unavailable", middleware.GetRunID(r.Context()), nil)
			} else {
				writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			}
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteJSON(w, http.StatusOK, map[string]string{"url": destination})
	})
}
