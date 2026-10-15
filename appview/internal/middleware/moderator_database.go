package middleware

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ctxkeys"
)

var ErrModeratorAuthentication = errors.New("moderator authentication failed")

type ModeratorAuthenticator interface {
	Authenticate(context.Context, string) (ctxkeys.Moderator, error)
}

type PostgresModeratorAuthenticator struct {
	pool         *pgxpool.Pool
	sourceSystem string
	now          func() time.Time
}

func NewPostgresModeratorAuthenticator(pool *pgxpool.Pool, sourceSystem string, now func() time.Time) *PostgresModeratorAuthenticator {
	if now == nil {
		now = time.Now
	}
	return &PostgresModeratorAuthenticator{pool: pool, sourceSystem: sourceSystem, now: now}
}

func (authenticator *PostgresModeratorAuthenticator) Authenticate(ctx context.Context, token string) (ctxkeys.Moderator, error) {
	if authenticator == nil || authenticator.pool == nil || authenticator.sourceSystem == "" || token == "" {
		return ctxkeys.Moderator{}, ErrModeratorAuthentication
	}
	digest := sha256.Sum256([]byte(token))
	now := authenticator.now().UTC()
	var operatorID uuid.UUID
	var moderator ctxkeys.Moderator
	err := authenticator.pool.QueryRow(ctx, `
		SELECT operator.id, operator.actor_id, operator.role,operator.retool_user_id,token.id,token.expires_at
		FROM safety_operator_tokens token
		JOIN safety_operators operator ON operator.id=token.operator_id
		WHERE token.token_digest=$1
		  AND token.active AND token.revoked_at IS NULL AND token.expires_at > $2
		  AND operator.active AND operator.deactivated_at IS NULL
	`, digest[:], now).Scan(&operatorID, &moderator.ActorID, &moderator.Role, &moderator.RetoolUserID, &moderator.TokenID, &moderator.TokenExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ctxkeys.Moderator{}, ErrModeratorAuthentication
	}
	if err != nil {
		return ctxkeys.Moderator{}, fmt.Errorf("authenticate moderator: %w", err)
	}
	moderator.SourceSystem = authenticator.sourceSystem
	moderator.OperatorID = operatorID
	moderator.Permissions = map[string]bool{}
	moderator.AssignedIncidentReferences = map[string]bool{}

	permissionRows, err := authenticator.pool.Query(ctx, `
		SELECT permission
		FROM safety_operator_permissions
		WHERE operator_id=$1 AND revoked_at IS NULL
	`, operatorID)
	if err != nil {
		return ctxkeys.Moderator{}, fmt.Errorf("read moderator permissions: %w", err)
	}
	for permissionRows.Next() {
		var permission string
		if err := permissionRows.Scan(&permission); err != nil {
			permissionRows.Close()
			return ctxkeys.Moderator{}, fmt.Errorf("scan moderator permission: %w", err)
		}
		moderator.Permissions[permission] = true
	}
	if err := permissionRows.Err(); err != nil {
		permissionRows.Close()
		return ctxkeys.Moderator{}, fmt.Errorf("iterate moderator permissions: %w", err)
	}
	permissionRows.Close()

	assignmentRows, err := authenticator.pool.Query(ctx, `
		SELECT incident_id
		FROM safety_operator_assignments
		WHERE operator_id=$1 AND released_at IS NULL
	`, operatorID)
	if err != nil {
		return ctxkeys.Moderator{}, fmt.Errorf("read moderator assignments: %w", err)
	}
	for assignmentRows.Next() {
		var incidentID uuid.UUID
		if err := assignmentRows.Scan(&incidentID); err != nil {
			assignmentRows.Close()
			return ctxkeys.Moderator{}, fmt.Errorf("scan moderator assignment: %w", err)
		}
		moderator.AssignedIncidentReferences[incidentID.String()] = true
	}
	if err := assignmentRows.Err(); err != nil {
		assignmentRows.Close()
		return ctxkeys.Moderator{}, fmt.Errorf("iterate moderator assignments: %w", err)
	}
	assignmentRows.Close()

	tag, err := authenticator.pool.Exec(ctx, `
		UPDATE safety_operator_tokens AS token SET last_used_at=$2
		FROM safety_operators AS operator
		WHERE token.operator_id=$1 AND token.token_digest=$3 AND token.active AND token.revoked_at IS NULL AND token.expires_at>$2
		AND operator.id=token.operator_id AND operator.active AND operator.deactivated_at IS NULL
	`, operatorID, now, digest[:])
	if err != nil {
		return ctxkeys.Moderator{}, fmt.Errorf("record moderator token use: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ctxkeys.Moderator{}, ErrModeratorAuthentication
	}
	return moderator, nil
}
