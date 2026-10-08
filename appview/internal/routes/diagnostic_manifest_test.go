package routes

import (
	"encoding/json"
	"os"
	"testing"
)

// IT-008: every actual catalogue policy requires explicit diagnostic metadata;
// adding/removing routes fails until both runtime fixtures are reviewed.
func TestDiagnosticManifestCoversActualCatalogue(t *testing.T) {
	var manifest []struct {
		Method   string `json:"method"`
		Path     string `json:"path"`
		Category string `json:"category"`
		Public   bool   `json:"public"`
	}
	data, err := os.ReadFile("diagnostic_manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	metadata := map[string]bool{}
	for _, item := range manifest {
		key := policyKey(item.Method, item.Path)
		if metadata[key] || item.Category == "" {
			t.Fatalf("invalid metadata %q", key)
		}
		metadata[key] = true
	}
	policies := V1RoutePolicies(EnvDev, Config{ModerationAdminEnabled: true, EnableDevModeration: true, DevModerationToken: "fixture"})
	for _, policy := range policies {
		key := policyKey(policy.Method, policy.PathPattern)
		if !metadata[key] {
			t.Errorf("missing diagnostic metadata %s", key)
		}
		delete(metadata, key)
	}
	for key := range metadata {
		t.Errorf("orphan diagnostic metadata %s", key)
	}
}
