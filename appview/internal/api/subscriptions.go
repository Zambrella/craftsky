package api

import (
	"context"
	"errors"
	"net/http"
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

type SubscriptionAssignmentStore interface {
	Assign(context.Context, subscriptions.AssignParams) (subscriptions.Assignment, error)
	Unassign(context.Context, subscriptions.UnassignParams) error
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
			err = store.RequestReconciliation(r.Context(), account.ID, now())
		}
		if err != nil {
			writeBillingAccountError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		envelope.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
	})
}

func AssignSubscriptionLicenseHandler(store SubscriptionAssignmentStore, now func() time.Time) http.Handler {
	type requestBody struct {
		TargetDID string `json:"targetDid"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		licenseID, err := uuid.Parse(r.PathValue("licenseId"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid assignment request", middleware.GetRunID(r.Context()), nil)
			return
		}
		var body requestBody
		if err := decodeStrictJSONObject(r, &body); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid assignment request", middleware.GetRunID(r.Context()), nil)
			return
		}
		target, err := syntax.ParseDID(body.TargetDID)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid assignment request", middleware.GetRunID(r.Context()), nil)
			return
		}
		deviceID, ok := middleware.GetDeviceID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_scope", "authenticated device scope missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		assignment, err := store.Assign(r.Context(), subscriptions.AssignParams{
			OwnerDID: owner, LicenseID: licenseID, TargetDID: target, DeviceID: deviceID, Now: now(),
		})
		if err != nil {
			writeAssignmentError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, assignment)
	})
}

func UnassignSubscriptionLicenseHandler(store SubscriptionAssignmentStore, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := subscriptionOwnerScope(w, r)
		if !ok {
			return
		}
		licenseID, err := uuid.Parse(r.PathValue("licenseId"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid assignment request", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err := store.Unassign(r.Context(), subscriptions.UnassignParams{OwnerDID: owner, LicenseID: licenseID, Now: now()}); err != nil {
			writeAssignmentError(w, middleware.GetRunID(r.Context()), err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func writeAssignmentError(w http.ResponseWriter, runID string, err error) {
	switch {
	case errors.Is(err, subscriptions.ErrLicenseNotFound):
		envelope.WriteError(w, http.StatusNotFound, "billing_license_not_found", "billing license not found", runID, nil)
	case errors.Is(err, subscriptions.ErrAssignmentTargetIneligible):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "assignment_target_ineligible", "assignment target is ineligible", runID, nil)
	case errors.Is(err, subscriptions.ErrDIDAlreadyAssigned):
		envelope.WriteError(w, http.StatusConflict, "assignment_conflict", "target already has a paid assignment", runID, nil)
	case errors.Is(err, subscriptions.ErrAssignmentCooldown):
		envelope.WriteError(w, http.StatusConflict, "assignment_cooldown", "license assignment cannot be changed yet", runID, nil)
	default:
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "billing operation failed", runID, nil)
	}
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
