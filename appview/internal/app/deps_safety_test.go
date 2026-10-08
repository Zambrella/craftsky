package app

import (
	"context"
	"testing"
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
