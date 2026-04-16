package auth

import (
	"context"
	"errors"
)

// NotImplementedAuthService is the prod AuthService until real atproto
// OAuth lands. It always returns an error. Wiring /whoami behind
// Authenticated in prod deliberately produces 401s, exercising the
// middleware path.
type NotImplementedAuthService struct{}

var _ AuthService = (*NotImplementedAuthService)(nil)

// ErrAuthNotImplemented is returned by NotImplementedAuthService.Authenticate
// so callers can type-check for it if they care (middleware doesn't — it
// returns 401 regardless).
var ErrAuthNotImplemented = errors.New("atproto OAuth not implemented yet")

func (NotImplementedAuthService) Authenticate(ctx context.Context, token string) (string, error) {
	return "", ErrAuthNotImplemented
}
