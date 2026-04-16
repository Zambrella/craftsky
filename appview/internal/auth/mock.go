package auth

import "context"

// MockAuthService is the dev-only AuthService. It always authenticates.
// The returned DID comes from the request context (see WithDevDID) when
// present, otherwise DefaultDID.
type MockAuthService struct {
	DefaultDID string
}

var _ AuthService = (*MockAuthService)(nil)

func (m *MockAuthService) Authenticate(ctx context.Context, token string) (string, error) {
	if did, ok := DevDIDFromContext(ctx); ok {
		return did, nil
	}
	return m.DefaultDID, nil
}
