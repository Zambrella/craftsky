package auth

import (
	"context"
	"errors"
)

// ErrAuthTokenInvalid is returned by CraftskyAuthService.Authenticate
// when the presented bearer token is empty, unknown, or revoked.
// Middleware surfaces it as 401.
var ErrAuthTokenInvalid = errors.New("invalid craftsky session token")

// CraftskyAuthService is the real AuthService used in production. It
// resolves a bearer token to (DID, oauth_session_id) by looking it up
// in the craftsky_sessions table via CraftskySessionStore.
type CraftskyAuthService struct {
	Store *CraftskySessionStore
}

var _ AuthService = (*CraftskyAuthService)(nil)

func (s *CraftskyAuthService) Authenticate(ctx context.Context, token string) (AuthInfo, error) {
	if token == "" {
		return AuthInfo{}, ErrAuthTokenInvalid
	}
	info, err := s.Store.Lookup(ctx, token)
	if errors.Is(err, ErrCraftskySessionNotFound) {
		return AuthInfo{}, ErrAuthTokenInvalid
	}
	if err != nil {
		return AuthInfo{}, err
	}
	return info, nil
}
