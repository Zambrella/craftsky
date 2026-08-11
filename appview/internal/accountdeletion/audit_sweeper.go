package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditSweeper struct {
	pool      *pgxpool.Pool
	now       func() time.Time
	telemetry *DeletionTelemetry
}

func (sweeper *AuditSweeper) SetTelemetry(telemetry *DeletionTelemetry) {
	if sweeper != nil {
		sweeper.telemetry = telemetry
	}
}

func NewAuditSweeper(pool *pgxpool.Pool, now func() time.Time) *AuditSweeper {
	if now == nil {
		now = time.Now
	}
	return &AuditSweeper{pool: pool, now: now}
}

func (sweeper *AuditSweeper) Sweep(ctx context.Context, limit int) (int, error) {
	if sweeper == nil || sweeper.pool == nil || limit <= 0 {
		return 0, errors.New("invalid account deletion audit sweep")
	}
	result, err := sweeper.pool.Exec(ctx, `
		DELETE FROM account_deletion_audits
		WHERE job_id IN (
			SELECT job_id FROM account_deletion_audits
			WHERE expires_at<=$1
			ORDER BY expires_at,job_id
			LIMIT $2
		)
	`, sweeper.now().UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("sweep account deletion audits: %w", err)
	}
	deleted := int(result.RowsAffected())
	if sweeper.telemetry != nil {
		for range deleted {
			sweeper.telemetry.AuditExpired(ctx)
		}
	}
	return deleted, nil
}
