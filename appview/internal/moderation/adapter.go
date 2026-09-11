package moderation

import (
	"context"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type AdjudicationCommander interface {
	ResolveCase(context.Context, TrustedCommand) (CommandResult, error)
}

type AuthenticatedSource struct {
	SourceSystem string
	ActorID      string
	SourceDID    syntax.DID
}

type ResolveRequest struct {
	CaseID           uuid.UUID
	ExpectedRevision int64
	Decision         Decision

	// These fields are deliberately ignored. They make the trust boundary
	// explicit for adapters whose untrusted payloads contain actor-like data.
	ActorID      string
	SourceSystem string
	SourceDID    syntax.DID
}

type TrustedInputAdapter struct {
	commander AdjudicationCommander
}

func NewTrustedInputAdapter(commander AdjudicationCommander) *TrustedInputAdapter {
	return &TrustedInputAdapter{commander: commander}
}

func (a *TrustedInputAdapter) Normalize(source AuthenticatedSource, replayID string, request ResolveRequest) (TrustedCommand, error) {
	if strings.TrimSpace(source.SourceSystem) == "" ||
		strings.TrimSpace(source.ActorID) == "" || source.SourceDID == "" || strings.TrimSpace(replayID) == "" {
		return TrustedCommand{}, ErrInvalidCommand
	}
	return TrustedCommand{
		CaseID: request.CaseID, ExpectedRevision: request.ExpectedRevision,
		SourceSystem: source.SourceSystem, ReplayID: replayID,
		ActorID: source.ActorID, SourceDID: source.SourceDID, Decision: request.Decision,
	}, nil
}

func (a *TrustedInputAdapter) Invoke(ctx context.Context, command TrustedCommand) (CommandResult, error) {
	if a == nil || a.commander == nil {
		return CommandResult{}, ErrInvalidCommand
	}
	return a.commander.ResolveCase(ctx, command)
}
