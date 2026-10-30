package imagesafety

import "testing"

func TestConfigReadinessRejectsUnsafeScannerModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		ready  bool
	}{
		{
			name: "development stub",
			config: Config{Environment: EnvironmentDevelopment, Mode: ScannerModeStub,
				ScannerID: "fixture-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1"},
			ready: true,
		},
		{
			name: "test stub",
			config: Config{Environment: EnvironmentTest, Mode: ScannerModeStub,
				ScannerID: "fixture-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1"},
			ready: true,
		},
		{
			name: "production manual moderation",
			config: Config{Environment: EnvironmentProduction, Mode: ScannerModeManual,
				ScannerID: "manual-moderation", PolicyVersion: "manual-v1", CorpusVersion: "none"},
			ready: true,
		},
		{
			name: "production stub",
			config: Config{Environment: EnvironmentProduction, Mode: ScannerModeStub,
				ScannerID: "fixture-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1"},
		},
		{
			name: "approved production adapter",
			config: Config{Environment: EnvironmentProduction, Mode: ScannerModeApproved,
				ScannerID: "approved-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1", AdapterReady: true},
			ready: true,
		},
		{
			name: "approved adapter with invalid credentials",
			config: Config{Environment: EnvironmentProduction, Mode: ScannerModeApproved,
				ScannerID: "approved-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1"},
		},
		{name: "production unconfigured", config: Config{Environment: EnvironmentProduction}},
		{name: "missing scanner id", config: Config{Environment: EnvironmentTest, Mode: ScannerModeStub, PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1"}},
		{name: "missing policy", config: Config{Environment: EnvironmentTest, Mode: ScannerModeStub, ScannerID: "fixture-scanner", CorpusVersion: "corpus-v1"}},
		{name: "missing corpus", config: Config{Environment: EnvironmentTest, Mode: ScannerModeStub, ScannerID: "fixture-scanner", PolicyVersion: "policy-v1"}},
		{name: "unknown mode", config: Config{Environment: EnvironmentTest, Mode: ScannerMode("unknown"), ScannerID: "fixture-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.config.Ready(); got != test.ready {
				t.Errorf("Config.Ready() = %t, want %t", got, test.ready)
			}
			if got := test.config.CanMarkClear(); got != test.ready {
				t.Errorf("Config.CanMarkClear() = %t, want %t", got, test.ready)
			}
		})
	}
}

func TestDevelopmentApprovalCapabilityRequiresValidatedConfiguration(t *testing.T) {
	config := Config{Environment: EnvironmentDevelopment, Mode: ScannerModeDevelopmentAutoApprove,
		ScannerID: DevelopmentApprovalScannerID, PolicyVersion: DevelopmentApprovalPolicyVersion, CorpusVersion: DevelopmentApprovalCorpusVersion}
	key := config.ProjectionKey()
	if !config.Ready() || !key.AutoApprovesDevelopmentImages() || config.AutomatedReady() {
		t.Fatal("development approval must be ready without claiming automated detection")
	}
	reconstructed := ScanKey{ScannerID: key.ScannerID, PolicyVersion: key.PolicyVersion, CorpusVersion: key.CorpusVersion}
	if reconstructed.AutoApprovesDevelopmentImages() {
		t.Fatal("stored scan identity must not grant an approval capability")
	}
	for _, environment := range []Environment{EnvironmentProduction, "unknown"} {
		invalid := config
		invalid.Environment = environment
		if invalid.Ready() || invalid.ProjectionKey().AutoApprovesDevelopmentImages() {
			t.Fatalf("accepted %s", environment)
		}
	}
	key.ScannerID = "manual-moderation"
	if key.AutoApprovesDevelopmentImages() {
		t.Fatal("approval capability must not apply to another scan identity")
	}
}
