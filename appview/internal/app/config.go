// Package app wires configuration and dependencies for both the server and
// CLI binaries. It is the single source of truth for how a Craftsky App View
// process is assembled.
package app

import "fmt"

// Env identifies which deployment environment the process is running in.
// It drives which .env file is loaded and which real/mock dependencies are
// wired by the Deps factories.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// ParseEnv converts the string form of an environment (as supplied by the
// server's positional arg or the CLI's --env flag) into an Env.
// Case-sensitive: "DEV" returns an error.
func ParseEnv(s string) (Env, error) {
	switch s {
	case string(EnvDev):
		return EnvDev, nil
	case string(EnvProd):
		return EnvProd, nil
	default:
		return "", fmt.Errorf("unknown env %q: expected %q or %q", s, EnvDev, EnvProd)
	}
}
