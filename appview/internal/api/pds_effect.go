package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdseffects"
)

// requirePDSEffectGeneration enforces the CurrentMember middleware contract at
// the handler boundary. A missing generation must fail before a PDS effect
// factory is invoked; the durable executor will recheck the exact generation
// again while its cross-process owner fence is held.
func requirePDSEffectGeneration(
	w http.ResponseWriter,
	r *http.Request,
	runID string,
) (int64, bool) {
	generation, ok := middleware.GetOwnerGeneration(r.Context())
	if !ok || generation <= 0 {
		envelope.WriteError(w, http.StatusServiceUnavailable,
			"lifecycle_unavailable", "membership unavailable", runID, nil)
		return 0, false
	}
	return generation, true
}

// immediateEffectIdentity namespaces a durable attempt to the one immediate
// HTTP mutation. Logging middleware supplies a unique run ID in production;
// direct handler tests and any future internal callers receive a UUID fallback.
// The same value is used as the stable mutation key for this request.
func immediateEffectIdentity(runID, operation string) (operationID, mutationKey string) {
	requestID := strings.TrimSpace(runID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	operationID = "http:" + operation + ":" + requestID
	return operationID, operationID
}

func newPDSEffectExecutor(
	ctx context.Context,
	factory pdseffects.ExecutorFactory,
	owner syntax.DID,
	sessionID string,
) (pdseffects.EffectExecutor, error) {
	if factory == nil {
		return nil, errors.New("durable PDS effect factory is unavailable")
	}
	executor, err := factory(ctx, owner, sessionID)
	if err != nil {
		return nil, err
	}
	if executor == nil {
		return nil, errors.New("durable PDS effect executor is unavailable")
	}
	return executor, nil
}
