package auth

import (
	"context"
	"errors"
)

// StackedAuthService is the dev-mode AuthService. It tries the real
// CraftskyAuthService first so OAuth flows work end-to-end in dev; if
// the bearer token is invalid AND the request carries a non-empty
// X-Dev-DID header (surfaced via DevDIDFromContext), it falls back to
// the dev-DID for backwards compatibility with the existing dev
// shortcut. Without the header, the original ErrAuthTokenInvalid is
// returned so a stale or random token cannot silently authenticate.
//
// Prod uses CraftskyAuthService directly; the fallback path is dev-only.
type StackedAuthService struct {
	Real *CraftskyAuthService
}

var _ AuthService = (*StackedAuthService)(nil)

func (s *StackedAuthService) Authenticate(ctx context.Context, token string) (AuthInfo, error) {
	info, err := s.Real.Authenticate(ctx, token)
	if err == nil {
		return info, nil
	}
	if !errors.Is(err, ErrAuthTokenInvalid) {
		return AuthInfo{}, err
	}
	// Real lookup said "no such token" (or token was empty). Fall back to
	// the X-Dev-DID header if present.
	if did, ok := DevDIDFromContext(ctx); ok && did != "" {
		return AuthInfo{DID: did}, nil
	}
	return AuthInfo{}, err
}
