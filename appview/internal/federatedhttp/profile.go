package federatedhttp

import (
	"fmt"
	"time"
)

// Purpose selects a hard response ceiling and a finite default deadline for a
// class of federated request.
type Purpose string

const (
	PurposeOAuthMetadata Purpose = "oauth_metadata"
	PurposeOAuthRequest  Purpose = "oauth_request"
	PurposePDSJSON       Purpose = "pds_json"
	PurposePDSUpload     Purpose = "pds_upload"
)

const (
	MaxOAuthMetadataResponseBytes int64 = 64 << 10
	MaxOAuthResponseBytes         int64 = 256 << 10
	MaxPDSJSONResponseBytes       int64 = 4 << 20
	MaxPDSUploadResponseBytes     int64 = 256 << 10

	maxTotalTimeout = 30 * time.Second
	maxRedirects    = 3
)

// Profile contains the tunable values for one purpose. Callers may lower the
// secure defaults before construction; Boundary.Client rejects disabled or
// enlarged budgets.
type Profile struct {
	Purpose       Purpose
	TotalTimeout  time.Duration
	ResponseLimit int64
	MaxRedirects  int
}

// DefaultProfile returns the secure operational defaults for purpose.
func DefaultProfile(purpose Purpose) (Profile, error) {
	profile := Profile{
		Purpose:      purpose,
		MaxRedirects: 2,
	}
	switch purpose {
	case PurposeOAuthMetadata:
		profile.TotalTimeout = 10 * time.Second
		profile.ResponseLimit = MaxOAuthMetadataResponseBytes
	case PurposeOAuthRequest:
		profile.TotalTimeout = 15 * time.Second
		profile.ResponseLimit = MaxOAuthResponseBytes
	case PurposePDSJSON:
		profile.TotalTimeout = 15 * time.Second
		profile.ResponseLimit = MaxPDSJSONResponseBytes
	case PurposePDSUpload:
		profile.TotalTimeout = 20 * time.Second
		profile.ResponseLimit = MaxPDSUploadResponseBytes
	default:
		return Profile{}, fmt.Errorf("federated http: unknown purpose")
	}
	return profile, nil
}

func validateProfile(profile Profile) error {
	defaults, err := DefaultProfile(profile.Purpose)
	if err != nil {
		return err
	}
	if profile.TotalTimeout <= 0 || profile.TotalTimeout > maxTotalTimeout {
		return fmt.Errorf("federated http: total timeout must be positive and at most %s", maxTotalTimeout)
	}
	if profile.ResponseLimit <= 0 || profile.ResponseLimit > defaults.ResponseLimit {
		return fmt.Errorf("federated http: response limit exceeds purpose ceiling")
	}
	if profile.MaxRedirects < 0 || profile.MaxRedirects > maxRedirects {
		return fmt.Errorf("federated http: redirect limit must be between 0 and %d", maxRedirects)
	}
	return nil
}
