package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
)

const tapRepositoryLeaseMargin = 5 * time.Second

func newTapRepositoryJobHandler(
	store *ingestion.Store,
	service *ingestion.Service,
	repositoryTracker auth.RepositoryTracker,
	anonymousPDS auth.PDSClient,
) ingestion.RepositoryJobHandler {
	return func(ctx context.Context, claim ingestion.RepositoryClaim) (string, error) {
		if store == nil || service == nil || repositoryTracker == nil || anonymousPDS == nil {
			return "", errors.New("tap repository reconciliation dependencies are unavailable")
		}
		remaining := time.Until(claim.LeaseExpiresAt) - tapRepositoryLeaseMargin
		if remaining <= 0 {
			return "", ingestion.ErrProjectionLeaseLost
		}
		jobCtx, cancel := context.WithTimeout(ctx, remaining)
		defer cancel()

		switch claim.Kind {
		case ingestion.RepositoryJobTapAddRepo:
			return "", repositoryTracker.AddRepo(jobCtx, claim.DID)
		case ingestion.RepositoryJobPDSReconcile:
			return reconcileTapRepository(jobCtx, store, service, anonymousPDS, claim.DID)
		default:
			return "", fmt.Errorf("unsupported Tap repository job kind %q", claim.Kind)
		}
	}
}

func reconcileTapRepository(
	ctx context.Context,
	store *ingestion.Store,
	service *ingestion.Service,
	anonymousPDS auth.PDSClient,
	did syntax.DID,
) (string, error) {
	var lastRevision string
	for {
		sources, err := store.UncertainSources(ctx, did, 100)
		if err != nil {
			return "", err
		}
		if len(sources) == 0 {
			remaining, err := service.ReconcileUnresolvedPDSAttempts(
				ctx, did, anonymousPDS, 100,
			)
			if err != nil {
				return "", err
			}
			if remaining {
				return "", errors.New("unresolved PDS effect attempts remain")
			}
			return lastRevision, nil
		}
		for _, source := range sources {
			observation := ingestion.ReconciledSource{
				URI: source.URI, DID: source.DID, ExpectedEventID: source.SourceEventID,
				ExpectedFingerprint: source.SourceFingerprint, Revision: source.Revision,
			}
			var record map[string]any
			cid, getErr := anonymousPDS.GetRecord(
				ctx, source.DID, source.Collection.String(), source.Rkey.String(), &record,
			)
			switch {
			case getErr == nil:
				encoded, err := json.Marshal(record)
				if err != nil {
					return "", fmt.Errorf("encode reconciled PDS record: %w", err)
				}
				observation.Present = true
				observation.CID = syntax.CID(cid)
				observation.Record = encoded
			case errors.Is(getErr, auth.ErrRecordNotFound):
				observation.Present = false
			default:
				return "", getErr
			}
			outcome, err := service.ReconcileSource(ctx, observation)
			if err != nil {
				return "", err
			}
			if !outcome.Acknowledgable() {
				return "", fmt.Errorf("tap source reconciliation remained retryable: %s", outcome.Reason)
			}
			lastRevision = observation.Revision.String()
		}
	}
}
