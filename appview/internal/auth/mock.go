package auth

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// MockAuthService is the dev-only AuthService. It always authenticates.
// The returned DID comes from the request context (see WithDevDID) when
// present, otherwise DefaultDID.
type MockAuthService struct {
	DefaultDID syntax.DID
}

var _ AuthService = (*MockAuthService)(nil)

func (m *MockAuthService) Authenticate(ctx context.Context, token string) (AuthInfo, error) {
	if did, ok := DevDIDFromContext(ctx); ok {
		return AuthInfo{DID: did}, nil
	}
	return AuthInfo{DID: m.DefaultDID}, nil
}
