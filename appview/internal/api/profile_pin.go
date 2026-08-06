package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type ProfilePinReader interface {
	Read(context.Context, syntax.DID) (ProfilePinState, error)
}

type ProfilePinMutator interface {
	Pin(context.Context, syntax.DID, syntax.DID, syntax.RecordKey) (ProfilePinMutationResult, error)
	Unpin(context.Context, syntax.DID, syntax.ATURI) (ProfilePinMutationResult, error)
}

func GetProfilePinsHandler(reader ProfilePinReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := profilePinOwner(w, r)
		if !ok {
			return
		}
		state, err := reader.Read(r.Context(), owner)
		if err != nil {
			writeProfilePinError(w, r, http.StatusInternalServerError, "internal_error", "profile pin read failed")
			return
		}
		envelope.WriteJSON(w, http.StatusOK, state)
	})
}

func PinProfilePostHandler(store ProfilePinMutator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := profilePinOwner(w, r)
		if !ok {
			return
		}
		did, rkey, ok := profilePinPath(w, r)
		if !ok {
			return
		}
		if did != owner {
			writeProfilePinError(w, r, http.StatusForbidden, "forbidden", "cannot pin another member's post")
			return
		}
		result, err := store.Pin(r.Context(), owner, did, rkey)
		if err != nil {
			writeProfilePinMutationError(w, r, err)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, result.State)
	})
}

func UnpinProfilePostHandler(store ProfilePinMutator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := profilePinOwner(w, r)
		if !ok {
			return
		}
		did, rkey, ok := profilePinPath(w, r)
		if !ok {
			return
		}
		if did != owner {
			writeProfilePinError(w, r, http.StatusForbidden, "forbidden", "cannot unpin another member's post")
			return
		}
		uri := syntax.ATURI("at://" + did.String() + "/" + craftskyPostNSID + "/" + rkey.String())
		result, err := store.Unpin(r.Context(), owner, uri)
		if err != nil {
			writeProfilePinMutationError(w, r, err)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, result.State)
	})
}

func profilePinOwner(w http.ResponseWriter, r *http.Request) (syntax.DID, bool) {
	owner, ok := middleware.GetDID(r.Context())
	if !ok {
		writeProfilePinError(w, r, http.StatusInternalServerError, "internal_error", "no did in context")
	}
	return owner, ok
}

func profilePinPath(w http.ResponseWriter, r *http.Request) (syntax.DID, syntax.RecordKey, bool) {
	did, err := syntax.ParseDID(r.PathValue("did"))
	if err != nil {
		writeProfilePinError(w, r, http.StatusBadRequest, "invalid_identifier", "not a valid DID")
		return "", "", false
	}
	rkey, err := syntax.ParseRecordKey(r.PathValue("rkey"))
	if err != nil {
		writeProfilePinError(w, r, http.StatusBadRequest, "invalid_identifier", "not a valid record key")
		return "", "", false
	}
	return did, rkey, true
}

func writeProfilePinMutationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrProfilePinForbidden):
		writeProfilePinError(w, r, http.StatusForbidden, "forbidden", "cannot pin another member's post")
	case errors.Is(err, ErrProfilePinTargetNotFound), errors.Is(err, ErrPostNotFound):
		writeProfilePinError(w, r, http.StatusNotFound, "post_not_found", "post not found")
	case errors.Is(err, ErrProfilePinNotAllowed):
		writeProfilePinError(w, r, http.StatusUnprocessableEntity, "pin_not_allowed", "post cannot be pinned")
	default:
		writeProfilePinError(w, r, http.StatusInternalServerError, "internal_error", "profile pin mutation failed")
	}
}

func writeProfilePinError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	envelope.WriteError(w, status, code, message, middleware.GetRunID(r.Context()), nil)
}
