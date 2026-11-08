package api

import (
	"context"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

type GuestVisibilitySettings interface {
	ReadGuestVisibility(context.Context, syntax.DID, int64, string) (pdscommands.GuestVisibilitySetting, error)
	PutGuestVisibility(context.Context, pdscommands.GuestVisibilityPutRequest) (pdscommands.CommandResult, error)
}

func LoggedOutVisibilityHandler(settings GuestVisibilitySettings) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx := r.Context()
		runID := middleware.GetRunID(ctx)
		fail := func(status int, code, message string) { envelope.WriteError(w, status, code, message, runID, nil) }
		owner, ok := middleware.GetDID(ctx)
		if !ok {
			fail(401, "authentication_required", "Sign in to manage your preferences.")
			return
		}
		generation, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		session, _ := middleware.GetOAuthSessionID(ctx)
		if session == "" || settings == nil {
			fail(503, "pds_unavailable", "Preferences are temporarily unavailable.")
			return
		}
		switch r.Method {
		case http.MethodGet:
			value, err := settings.ReadGuestVisibility(ctx, owner, generation, session)
			if err != nil {
				WriteCommandError(w, runID, err, ctx)
				return
			}
			writeJSON(w, 200, value)
		case http.MethodPut:
			key, ok := requireCommandOperationKey(w, r, runID)
			if !ok {
				return
			}
			var body struct {
				Hide        *bool   `json:"hideFromLoggedOut"`
				ExpectedCID *string `json:"expectedCid"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1024)
			if err := decodeStrictJSONObject(r, &body); err != nil || body.Hide == nil || body.ExpectedCID == nil || len(*body.ExpectedCID) > 512 {
				fail(400, "malformed_body", "Provide hideFromLoggedOut and expectedCid.")
				return
			}
			result, err := settings.PutGuestVisibility(ctx, pdscommands.GuestVisibilityPutRequest{
				Owner: owner, OwnerGeneration: generation, SessionID: session, OperationKey: key, ExpectedCID: syntax.CID(*body.ExpectedCID), HideFromLoggedOut: *body.Hide,
				Rejected: func(cause error) pdscommands.TerminalResult { return RejectedCommandResult(runID, cause, ctx) },
			})
			if err != nil {
				WriteCommandError(w, runID, err, ctx)
				return
			}
			WriteCommandResponse(w, CommandResultFromStored(result))
		default:
			w.Header().Set("Allow", "GET, PUT")
			fail(405, "method_not_allowed", "Method not allowed.")
		}
	})
}
