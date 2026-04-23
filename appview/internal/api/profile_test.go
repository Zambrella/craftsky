// appview/internal/api/profile_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// fakeStore implements the subset of ProfileStore that handlers call.
type fakeStore struct {
	row *api.ProfileRow
	err error
}

func (f *fakeStore) Read(_ context.Context, _ string) (*api.ProfileRow, error) {
	return f.row, f.err
}

// fakeResolver implements api.HandleResolver.
type fakeResolver struct {
	handleFor syntax.Handle
	didFor    syntax.DID
	err       error
}

func (f fakeResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return f.handleFor, f.err
}
func (f fakeResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return f.didFor, f.err
}

func nilLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetProfile_ByDIDHappyPath(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID: "did:plc:xyz", Crafts: []string{"sewing"},
		CreatedAt: time.Now(),
	}
	h := api.GetProfileHandler(
		&fakeStore{row: row},
		fakeResolver{handleFor: "alice.example"},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@did:plc:xyz", nil)
	req.SetPathValue("handleOrDid", "did:plc:xyz")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body api.ProfileResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body.DID != "did:plc:xyz" || body.Handle != "alice.example" {
		t.Errorf("%+v", body)
	}
}

func TestGetProfile_ByHandleHappyPath(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{DID: "did:plc:xyz", Crafts: []string{}, CreatedAt: time.Now()}
	resolver := fakeResolver{
		didFor:    syntax.DID("did:plc:xyz"),
		handleFor: syntax.Handle("alice.example"),
	}
	h := api.GetProfileHandler(&fakeStore{row: row}, resolver, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@alice.example", nil)
	req.SetPathValue("handleOrDid", "alice.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestGetProfile_InvalidIdentifier(t *testing.T) {
	t.Parallel()
	h := api.GetProfileHandler(&fakeStore{}, fakeResolver{}, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@NOT%20VALID", nil)
	req.SetPathValue("handleOrDid", "NOT VALID")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "invalid_identifier" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestGetProfile_NonMember(t *testing.T) {
	t.Parallel()
	h := api.GetProfileHandler(
		&fakeStore{err: api.ErrProfileNotFound},
		fakeResolver{},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@did:plc:gone", nil)
	req.SetPathValue("handleOrDid", "did:plc:gone")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "profile_not_found" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestGetProfile_ResolveDIDError(t *testing.T) {
	t.Parallel()
	h := api.GetProfileHandler(
		&fakeStore{},
		fakeResolver{err: errors.New("plc down")},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@alice.example", nil)
	req.SetPathValue("handleOrDid", "alice.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "identity_unavailable" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestGetMeProfile_HappyPath(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := api.GetMeProfileHandler(
		&fakeStore{row: row},
		fakeResolver{handleFor: "alice.example"},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestGetMeProfile_NoDIDInContext(t *testing.T) {
	t.Parallel()
	h := api.GetMeProfileHandler(&fakeStore{}, fakeResolver{}, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
}
