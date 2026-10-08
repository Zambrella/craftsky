package middleware

import (
	"context"
	"crypto/sha256"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestPostgresModeratorAuthenticatorUsesActiveDigestPermissionsAndAssignments(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000087_safety_operators.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `CREATE TABLE safety_incidents (id UUID PRIMARY KEY);`+string(migration))
	ctx := context.Background()
	operatorID := uuid.New()
	incidentID := uuid.New()
	digest := sha256.Sum256([]byte("operator-secret"))
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id) VALUES ($1)`, incidentID); err != nil {
		t.Fatalf("seed incident: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_operators(id, actor_id, role, created_at, updated_at)
		VALUES ($1, 'helper-1', 'helper', now(), now())
	`, operatorID); err != nil {
		t.Fatalf("seed operator: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_operator_tokens(id, operator_id, token_digest, expires_at, created_at)
		VALUES ($1, $2, $3, now()+interval '1 hour', now())
	`, uuid.New(), operatorID, digest[:]); err != nil {
		t.Fatalf("seed operator token: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_operator_permissions(operator_id, permission, granted_by, granted_at)
		VALUES ($1, 'incident.readSafe', 'bootstrap', now())
	`, operatorID); err != nil {
		t.Fatalf("seed operator permission: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_operator_assignments(id, incident_id, operator_id, assignment_role, assigned_by, assigned_at)
		VALUES ($1, $2, $3, 'helper', 'bootstrap', now())
	`, uuid.New(), incidentID, operatorID); err != nil {
		t.Fatalf("seed operator assignment: %v", err)
	}

	authenticator := NewPostgresModeratorAuthenticator(pool, "admin-api", time.Now)
	moderator, err := authenticator.Authenticate(ctx, "operator-secret")
	if err != nil {
		t.Fatalf("authenticate active operator: %v", err)
	}
	if moderator.ActorID != "helper-1" || moderator.Role != "helper" || moderator.SourceSystem != "admin-api" {
		t.Fatalf("moderator identity = %+v", moderator)
	}
	if !moderator.Permissions["incident.readSafe"] || !moderator.AssignedIncidentReferences[incidentID.String()] {
		t.Fatalf("moderator authorization = %+v", moderator)
	}
	if _, err := authenticator.Authenticate(ctx, "wrong-secret"); err == nil {
		t.Fatal("wrong token authenticated")
	}
	if _, err := pool.Exec(ctx, `UPDATE safety_operator_tokens SET active=false, revoked_at=now() WHERE operator_id=$1`, operatorID); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticator.Authenticate(ctx, "operator-secret"); err == nil {
		t.Fatal("revoked token authenticated")
	}
}
