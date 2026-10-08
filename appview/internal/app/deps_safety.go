package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/retention"
	"social.craftsky/appview/internal/safetyincident"
)

type safetyDependencies struct {
	objects   *safetyincident.S3EvidenceStore
	evidence  *safetyincident.EvidenceService
	holds     *safetyincident.HoldService
	workflows *safetyincident.WorkflowService
	csea      *safetyincident.CSEAWorkflow
	retention *retention.Worker
}

func newSafetyDependencies(ctx context.Context, pool *pgxpool.Pool, cfg Config) (*safetyDependencies, error) {
	deadlines, err := safetyincident.NewDeadlinePolicy(nil)
	if err != nil {
		return nil, fmt.Errorf("safety workflow deadlines: %w", err)
	}
	result := &safetyDependencies{
		holds:     safetyincident.NewHoldService(pool),
		workflows: safetyincident.NewWorkflowService(pool, deadlines, time.Now),
		csea:      safetyincident.NewCSEAWorkflow(pool),
	}
	if cfg.SafetyEvidenceS3.AccessKeyID == "" || cfg.SafetyEvidenceS3.AccessKeyID == "not-configured" ||
		cfg.SafetyEvidenceS3.SecretAccessKey == "" || cfg.SafetyEvidenceS3.SecretAccessKey == "not-configured" {
		return result, nil
	}
	objects, err := safetyincident.NewS3EvidenceStore(ctx, cfg.SafetyEvidenceS3)
	if err != nil {
		return nil, fmt.Errorf("restricted evidence object store: %w", err)
	}
	if err := objects.Check(ctx); err != nil {
		return nil, fmt.Errorf("restricted evidence object store check: %w", err)
	}
	result.objects = objects
	result.evidence = safetyincident.NewEvidenceService(pool, objects, time.Now)
	result.retention = retention.NewWorker(pool, objects, time.Now)
	return result, nil
}
