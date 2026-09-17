// Package buildinfo exposes build metadata embedded by the production linker.
package buildinfo

import "regexp"

const developmentVersion = "dev"

var (
	semanticVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	version         = developmentVersion
)

// Version returns the embedded semantic version, or dev when it is absent or invalid.
func Version() string {
	if !semanticVersion.MatchString(version) {
		return developmentVersion
	}
	return version
}

// SentryRelease returns a stable release name only for versioned builds.
func SentryRelease() string {
	value := Version()
	if value == developmentVersion {
		return ""
	}
	return "craftsky-appview@" + value
}
