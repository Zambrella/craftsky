package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/tap"
)

const tapRepositoryLeaseMargin = 5 * time.Second

func newTapRepositoryJobHandler(
	repositoryTracker tap.RepositoryTracker,
	snapshotFetcher *ingestion.RepositorySnapshotFetcher,
	registry ingestion.RepositoryCollectionRegistry,
	repair *ingestion.RepositoryRepair,
	observer *observability.Observer,
) ingestion.RepositoryJobHandler {
	return func(ctx context.Context, claim ingestion.RepositoryClaim) (string, error) {
		if repositoryTracker == nil || snapshotFetcher == nil || registry == nil || repair == nil {
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
			started := time.Now()
			snapshot, err := snapshotFetcher.Fetch(jobCtx, claim.DID, registry)
			if err != nil {
				reason := "remote_unavailable"
				var reasoned interface{ ReasonCode() string }
				if errors.As(err, &reasoned) {
					reason = reasoned.ReasonCode()
				}
				observer.ObserveRepositorySnapshotVerification("error", reason, time.Since(started))
				return "", err
			}
			observer.ObserveRepositorySnapshotVerification("success", "none", time.Since(started))
			return repair.Apply(jobCtx, snapshot, registry)
		default:
			return "", fmt.Errorf("unsupported Tap repository job kind %q", claim.Kind)
		}
	}
}
