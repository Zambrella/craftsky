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
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
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
	handleFor    syntax.Handle
	handlesByDID map[string]syntax.Handle
	didFor       syntax.DID
	err          error
}

func (f fakeResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.handlesByDID != nil {
		return f.handlesByDID[did.String()], nil
	}
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

// fakePDSForPut is a lightweight mock scoped to PUT tests.
type fakePDSForPut struct {
	getBsky      func() (map[string]any, error)
	putBsky      func(body map[string]any) error
	putCraftsky  func(body map[string]any) error
	putBskyCalls []map[string]any
}

func (f *fakePDSForPut) GetRecord(_ context.Context, _ syntax.DID, collection, _ string, out any) (string, error) {
	if collection == "app.bsky.actor.profile" {
		rec, err := f.getBsky()
		if err != nil {
			return "", err
		}
		*(out.(*map[string]any)) = rec
		return "", nil
	}
	return "", errors.New("unexpected get collection: " + collection)
}
func (f *fakePDSForPut) PutRecord(_ context.Context, _ syntax.DID, collection, _ string, body any) error {
	m, _ := body.(map[string]any)
	switch collection {
	case "app.bsky.actor.profile":
		f.putBskyCalls = append(f.putBskyCalls, m)
		return f.putBsky(m)
	case "social.craftsky.actor.profile":
		return f.putCraftsky(m)
	}
	return errors.New("unexpected put collection: " + collection)
}
func (f *fakePDSForPut) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("CreateRecord: not implemented in fakePDSForPut")
}
func (f *fakePDSForPut) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return errors.New("DeleteRecord: not implemented in fakePDSForPut")
}

// newPutHandler wires a fake store, resolver, and PDS client.
func newPutHandler(
	t *testing.T,
	store *fakeStore,
	pds *fakePDSForPut,
	resolver fakeResolver,
) http.Handler {
	t.Helper()
	return api.PutMeProfileHandler(
		store,
		resolver,
		func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
			return pds, nil
		},
		nilLogger(),
	)
}

func TestPutProfile_HappyPathMergesBlueskyExtras(t *testing.T) {
	t.Parallel()
	captured := map[string]any{}
	pds := &fakePDSForPut{
		getBsky: func() (map[string]any, error) {
			return map[string]any{
				"displayName": "old",
				"avatar": map[string]any{
					"$type":    "blob",
					"ref":      map[string]any{"$link": "bafav"},
					"mimeType": "image/jpeg",
					"size":     1,
				},
			}, nil
		},
		putBsky: func(body map[string]any) error {
			for k, v := range body {
				captured[k] = v
			}
			return nil
		},
		putCraftsky: func(_ map[string]any) error { return nil },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{"sewing"}, CreatedAt: time.Now()}
	h := newPutHandler(t,
		&fakeStore{row: row},
		pds,
		fakeResolver{handleFor: "alice.example"},
	)
	body := `{"displayName":"new","crafts":["sewing","quilting"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(body))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if captured["displayName"] != "new" {
		t.Errorf("bluesky displayName = %v", captured["displayName"])
	}
	if _, ok := captured["avatar"]; !ok {
		t.Error("avatar must be preserved from existing record")
	}
}

func TestPutProfile_RejectsAvatar(t *testing.T) {
	t.Parallel()
	pds := &fakePDSForPut{}
	h := newPutHandler(t, &fakeStore{}, pds, fakeResolver{})
	body := `{"avatar":{"ref":{"$link":"x"}}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(body))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "unexpected_field" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestPutProfile_PartialSuccessReturns502(t *testing.T) {
	t.Parallel()
	pds := &fakePDSForPut{
		getBsky:     func() (map[string]any, error) { return map[string]any{}, nil },
		putBsky:     func(_ map[string]any) error { return nil },
		putCraftsky: func(_ map[string]any) error { return errors.New("pds down") },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := newPutHandler(t,
		&fakeStore{row: row},
		pds,
		fakeResolver{handleFor: "alice.example"},
	)
	body := `{"displayName":"x"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(body))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "pds_write_partial" {
		t.Errorf("code = %q", env.Error)
	}
	if env.Fields["craftsky"] != "failed" || env.Fields["bsky"] != "ok" {
		t.Errorf("fields = %v", env.Fields)
	}
}

func TestPutProfile_BothFailsReturns502(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	pds := &fakePDSForPut{
		getBsky:     func() (map[string]any, error) { return map[string]any{}, nil },
		putBsky:     func(_ map[string]any) error { return boom },
		putCraftsky: func(_ map[string]any) error { return boom },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := newPutHandler(t, &fakeStore{row: row}, pds, fakeResolver{handleFor: "alice.example"})
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "pds_write_failed" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestPutProfile_ReadBeforeWriteFailure(t *testing.T) {
	t.Parallel()
	pds := &fakePDSForPut{
		getBsky: func() (map[string]any, error) { return nil, errors.New("pds down") },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := newPutHandler(t, &fakeStore{row: row}, pds, fakeResolver{handleFor: "alice.example"})
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "pds_read_failed" {
		t.Errorf("code = %q", env.Error)
	}
}
