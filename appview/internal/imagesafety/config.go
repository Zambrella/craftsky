package imagesafety

import "strings"

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentProduction  Environment = "production"
)

type ScannerMode string

const (
	ScannerModeStub                   ScannerMode = "stub"
	ScannerModeDevelopmentAutoApprove ScannerMode = "dev-auto-approve"
	// Manual mode validates image format; content review is performed by moderators.
	ScannerModeManual   ScannerMode = "manual"
	ScannerModeApproved ScannerMode = "approved"
)

type Config struct {
	Environment   Environment
	Mode          ScannerMode
	ScannerID     string
	PolicyVersion string
	CorpusVersion string
	// AdapterReady is set only after validating a production adapter. It is
	// deliberately not an environment-controlled assertion.
	AdapterReady bool
}

func (config Config) Ready() bool {
	if strings.TrimSpace(config.ScannerID) == "" || strings.TrimSpace(config.PolicyVersion) == "" ||
		strings.TrimSpace(config.CorpusVersion) == "" {
		return false
	}
	switch config.Mode {
	case ScannerModeDevelopmentAutoApprove:
		return (config.Environment == EnvironmentDevelopment || config.Environment == EnvironmentTest) &&
			!config.AdapterReady && config.ScannerID == DevelopmentApprovalScannerID &&
			config.PolicyVersion == DevelopmentApprovalPolicyVersion && config.CorpusVersion == DevelopmentApprovalCorpusVersion
	case ScannerModeManual:
		return !config.AdapterReady
	case ScannerModeStub:
		return config.Environment != EnvironmentProduction && !config.AdapterReady
	case ScannerModeApproved:
		return config.Environment == EnvironmentProduction && config.AdapterReady
	default:
		return false
	}
}

func (config Config) CanMarkClear() bool {
	return config.Ready()
}

// AutomatedReady reports an actual validated production content scanner.
// Manual moderation does not provide automated content detection.
func (config Config) AutomatedReady() bool {
	return config.Mode == ScannerModeApproved && config.Ready()
}

// Development approvals have a reserved identity and never count as scan evidence.
const (
	DevelopmentApprovalScannerID     = "development-auto-approve"
	DevelopmentApprovalPolicyVersion = "development-auto-approve-v1"
	DevelopmentApprovalCorpusVersion = "not-scanned"
)

// ProjectionKey carries the dev-only capability only after configuration checks.
// A plain ScanKey constructed from database values cannot enable the bypass.
func (config Config) ProjectionKey() ScanKey {
	return ScanKey{
		ScannerID: config.ScannerID, PolicyVersion: config.PolicyVersion, CorpusVersion: config.CorpusVersion,
		developmentAutoApprove: config.Mode == ScannerModeDevelopmentAutoApprove && config.Ready(),
	}
}
