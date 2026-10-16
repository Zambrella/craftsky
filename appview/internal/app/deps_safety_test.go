package app

import (
	"context"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
	"testing"
	"time"
)

func TestSafetyDependenciesDoNotRequireUnconfiguredEvidenceStorage(t *testing.T) {
	deps, err := newSafetyDependencies(context.Background(), nil, Config{Env: EnvProd})
	if err != nil {
		t.Fatal(err)
	}
	if deps.objects != nil || deps.evidence != nil || deps.retention != nil {
		t.Fatal("unconfigured evidence capability must remain unavailable")
	}
	if deps.holds == nil || deps.workflows == nil || deps.csea == nil {
		t.Fatal("manual safety workflows must remain available")
	}
}

func TestSafetyDependenciesUseExplicitIntimateImageDeadline(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	received := time.Now().UTC().Truncate(time.Second)
	deps, err := newSafetyDependencies(t.Context(), pool, Config{Env: EnvDev, SafetyIntimateImageResponseWithin: 6 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	id, deadline, err := deps.workflows.OpenIntimateImage(t.Context(), safetyincident.IntimateImageCommand{ReceivedAt: received, Actor: safetyincident.Actor{ID: "operator:fixture", Role: safetyincident.RoleSafetyAdministrator}, StandingCode: "synthetic-standing", DeclarationsCode: "synthetic-declarations"})
	if err != nil || id == uuid.Nil || !deadline.DueAt.Equal(received.Add(6*time.Hour)) {
		t.Fatalf("configured workflow id=%s due=%v err=%v", id, deadline.DueAt, err)
	}
}
