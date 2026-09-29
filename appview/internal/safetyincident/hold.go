package safetyincident

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidHold = errors.New("invalid restricted evidence hold")

type HoldRequest struct {
	IncidentID  uuid.UUID
	EvidenceIDs []uuid.UUID
	Basis       string
	ApprovedBy  string
	CreatedBy   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

func ValidateHold(request HoldRequest) error {
	if request.IncidentID == uuid.Nil || len(request.EvidenceIDs) == 0 ||
		strings.TrimSpace(request.Basis) == "" || strings.TrimSpace(request.ApprovedBy) == "" ||
		strings.TrimSpace(request.CreatedBy) == "" || request.CreatedAt.IsZero() ||
		request.ExpiresAt.IsZero() || !request.ExpiresAt.After(request.CreatedAt) {
		return ErrInvalidHold
	}
	seen := make(map[uuid.UUID]struct{}, len(request.EvidenceIDs))
	for _, evidenceID := range request.EvidenceIDs {
		if evidenceID == uuid.Nil {
			return ErrInvalidHold
		}
		if _, duplicate := seen[evidenceID]; duplicate {
			return ErrInvalidHold
		}
		seen[evidenceID] = struct{}{}
	}
	return nil
}

type HoldService struct{ pool *pgxpool.Pool }

func NewHoldService(pool *pgxpool.Pool) *HoldService { return &HoldService{pool: pool} }

func (service *HoldService) Create(ctx context.Context, actor Actor, request HoldRequest) (uuid.UUID, error) {
	if err := ValidateHold(request); err != nil {
		return uuid.Nil, err
	}
	if service == nil || service.pool == nil || !actor.Allowed(PermissionHoldManage, request.IncidentID) || actor.ID != request.CreatedBy {
		return uuid.Nil, ErrUnauthorized
	}
	id := uuid.New()
	err := pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		evidenceIDs := slices.Clone(request.EvidenceIDs)
		slices.SortFunc(evidenceIDs, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		for _, evidenceID := range evidenceIDs {
			var lockedID uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT id FROM safety_evidence
				WHERE id=$1 AND incident_id=$2 AND deleted_at IS NULL
				FOR UPDATE`, evidenceID, request.IncidentID).Scan(&lockedID); errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidHold
			} else if err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_legal_holds(
			id,incident_id,basis,approved_by,created_by,expires_at,created_at
		) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, request.IncidentID, request.Basis,
			request.ApprovedBy, request.CreatedBy, request.ExpiresAt.UTC(), request.CreatedAt.UTC()); err != nil {
			return err
		}
		for _, evidenceID := range request.EvidenceIDs {
			result, err := tx.Exec(ctx, `INSERT INTO safety_legal_hold_evidence(hold_id,evidence_id,created_at)
				SELECT $1,id,$3 FROM safety_evidence WHERE id=$2 AND incident_id=$4`, id, evidenceID,
				request.CreatedAt.UTC(), request.IncidentID)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return ErrInvalidHold
			}
		}
		return nil
	})
	return id, err
}
