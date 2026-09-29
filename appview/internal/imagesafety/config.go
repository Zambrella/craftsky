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
	ScannerModeStub     ScannerMode = "stub"
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
