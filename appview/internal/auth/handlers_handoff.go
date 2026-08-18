package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type HandoffCoordinator interface {
	CreateExchange(context.Context, CallbackAttempt, syntax.Handle, string) (string, error)
	Exchange(context.Context, string, string) (HandoffExchangeResult, error)
	Confirm(context.Context, string, uuid.UUID, string) error
}

type handoffExchangeRequest struct {
	Code string `json:"code"`
}

type handoffExchangeResponse struct {
	Token     string    `json:"token"`
	DID       string    `json:"did"`
	Handle    string    `json:"handle"`
	ReceiptID uuid.UUID `json:"receiptId"`
	ConfirmBy string    `json:"confirmBy"`
}

type handoffConfirmRequest struct {
	ReceiptID string `json:"receiptId"`
}

func (h *HTTPHandlers) HandoffExchangeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		if h.Handoffs == nil {
			writeHandoffUnavailable(w, runID)
			return
		}
		var request handoffExchangeRequest
		if err := decodeSingleJSON(r, &request); err != nil || request.Code == "" {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
			return
		}
		deviceID, ok := ctxkeys.GetDeviceID(r.Context())
		if !ok || deviceID == "" {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
			return
		}
		result, err := h.Handoffs.Exchange(r.Context(), request.Code, deviceID)
		if err != nil {
			if errors.Is(err, ErrHandoffInvalid) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
				return
			}
			if h.Logger != nil {
				h.Logger.Error("OAuth handoff exchange unavailable", authLogErrorAttrs(runID, "oauth.handoff.exchange", "infrastructure")...)
			}
			writeHandoffUnavailable(w, runID)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(handoffExchangeResponse{
			Token: result.Token, DID: result.DID.String(), Handle: result.Handle.String(),
			ReceiptID: result.ReceiptID, ConfirmBy: result.ConfirmBy.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		})
	})
}

func (h *HTTPHandlers) HandoffConfirmHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		if h.Handoffs == nil {
			writeHandoffUnavailable(w, runID)
			return
		}
		var request handoffConfirmRequest
		if err := decodeSingleJSON(r, &request); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
			return
		}
		receiptID, err := uuid.Parse(request.ReceiptID)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
			return
		}
		deviceID, ok := ctxkeys.GetDeviceID(r.Context())
		token := bearerToken(r)
		if !ok || deviceID == "" || token == "" {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
			return
		}
		if err := h.Handoffs.Confirm(r.Context(), token, receiptID, deviceID); err != nil {
			if errors.Is(err, ErrHandoffInvalid) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff", "handoff is invalid or expired", runID, nil)
				return
			}
			if h.Logger != nil {
				h.Logger.Error("OAuth handoff confirmation unavailable", authLogErrorAttrs(runID, "oauth.handoff.confirm", "infrastructure")...)
			}
			writeHandoffUnavailable(w, runID)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func decodeSingleJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeHandoffUnavailable(w http.ResponseWriter, runID string) {
	w.Header().Set("Retry-After", "5")
	envelope.WriteError(w, http.StatusServiceUnavailable, "handoff_unavailable", "sign-in handoff is temporarily unavailable", runID, nil)
}
