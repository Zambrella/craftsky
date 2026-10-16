package operatorconsole_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/operatorconsole"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

type consoleDirectory struct {
	identity.Directory
	account *identity.Identity
}

func (d consoleDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return d.account, nil
}
func (d consoleDirectory) LookupDID(context.Context, syntax.DID) (*identity.Identity, error) {
	return d.account, nil
}

type providerTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (p providerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "identity.console.test" {
		return nil, fmt.Errorf("unexpected fixture destination")
	}
	copy := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = p.target.Scheme
	u.Host = p.target.Host
	copy.URL = &u
	return p.base.RoundTrip(copy)
}
func runConsoleFlow(t *testing.T, scenario string) {
	did := syntax.DID("did:plc:consolelogin")
	const providerOrigin = "https://identity.console.test"
	var state, scope string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource":
			json.NewEncoder(w).Encode(map[string]any{"resource": providerOrigin, "authorization_servers": []string{providerOrigin}})
		case "/.well-known/oauth-authorization-server":
			json.NewEncoder(w).Encode(map[string]any{"issuer": providerOrigin, "authorization_endpoint": providerOrigin + "/authorize", "token_endpoint": providerOrigin + "/token", "pushed_authorization_request_endpoint": providerOrigin + "/par", "scopes_supported": []string{"atproto"}, "response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"}, "code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"none", "private_key_jwt"}, "authorization_response_iss_parameter_supported": true, "client_id_metadata_document_supported": true, "token_endpoint_auth_signing_alg_values_supported": []string{"ES256"}, "dpop_signing_alg_values_supported": []string{"ES256"}, "require_pushed_authorization_requests": true})
		case "/par":
			r.ParseForm()
			state = r.Form.Get("state")
			scope = r.Form.Get("scope")
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{"request_uri": "urn:ietf:params:oauth:request_uri:test", "expires_in": 90})
		case "/token":
			subject, granted := did.String(), "atproto"
			tokenType := "DPoP"
			if scenario == "wrong subject" {
				subject = "did:plc:anotheraccount"
			}
			if scenario == "missing scope" {
				granted = ""
			}
			if scenario == "extra scope" {
				granted = "atproto transition:generic"
			}
			json.NewEncoder(w).Encode(map[string]any{"sub": subject, "scope": granted, "access_token": "oauth-access-protected-canary", "refresh_token": "oauth-refresh-protected-canary", "token_type": tokenType, "expires_in": 300})
		default:
			t.Errorf("unexpected upstream operation %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	client := upstream.Client()
	client.Transport = providerTransport{base: client.Transport, target: target}
	clock := time.Now().UTC()
	pool := testdb.WithMigratedSchema(t)
	store := operatorconsole.NewStore(pool, func() time.Time { return clock })
	op, err := store.Allow(t.Context(), did, "mod")
	if err != nil {
		t.Fatal(err)
	}
	cfg := oauth.NewLocalhostConfig("http://127.0.0.1:18080/oauth/admin/callback", []string{"atproto"})
	app := oauth.NewClientApp(&cfg, nil)
	app.Client = client
	app.Resolver.Client = client
	app.Dir = consoleDirectory{account: &identity.Identity{DID: did, Handle: syntax.Handle("reviewer.test"), Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: providerOrigin}}}}
	h, err := operatorconsole.NewHandlers(store, app, "http://127.0.0.1:18080", nil)
	if err != nil {
		t.Fatal(err)
	}

	if scenario == "legacy credentials" {
		for _, credential := range []string{"legacy-human-token", "consumer-session-token", "external-resource-token"} {
			r := httptest.NewRequest("GET", "http://127.0.0.1:18080/v1/admin/identity", nil)
			r.Header.Set("Authorization", "Bearer "+credential)
			r.Header.Set("X-Dev-DID", did.String())
			w := httptest.NewRecorder()
			h.Authentication()(api.OperatorIdentityHandler()).ServeHTTP(w, r)
			if w.Code != 401 {
				t.Fatal("non-operator credential admitted")
			}
		}
		return
	}
	if scenario == "handle transferred" {
		app.Dir.(consoleDirectory).account.DID = syntax.DID("did:plc:handleacquirer")
	}
	if scenario == "handle changed" {
		app.Dir.(consoleDirectory).account.Handle = syntax.Handle("renamed.test")
	}
	bootstrap := httptest.NewRecorder()
	h.Bootstrap().ServeHTTP(bootstrap, httptest.NewRequest("GET", "http://127.0.0.1:18080/v1/admin/auth/csrf", nil))
	var csrf struct {
		Token string `json:"csrfToken"`
	}
	json.Unmarshal(bootstrap.Body.Bytes(), &csrf)
	if csrf.Token == "" {
		t.Fatal("no browser binding")
	}
	login := httptest.NewRequest("POST", "http://127.0.0.1:18080/v1/admin/auth/login", strings.NewReader(`{"identifier":"reviewer.test"}`))
	if scenario == "handle changed" {
		login.Body = io.NopCloser(strings.NewReader(`{"identifier":"renamed.test"}`))
	}
	login.Header.Set("Origin", "http://127.0.0.1:18080")
	login.Header.Set("X-CSRF-Token", csrf.Token)
	login.Header.Set("Content-Type", "application/json")
	for _, c := range bootstrap.Result().Cookies() {
		login.AddCookie(c)
	}
	if scenario == "forged body" {
		login.Body = http.NoBody
		login = httptest.NewRequest("POST", "http://127.0.0.1:18080/v1/admin/auth/login", strings.NewReader(`{"identifier":"reviewer.test","role":"admin"}`))
		login.Header.Set("Origin", "http://127.0.0.1:18080")
		login.Header.Set("X-CSRF-Token", csrf.Token)
		for _, c := range bootstrap.Result().Cookies() {
			login.AddCookie(c)
		}
	}
	if scenario == "unlisted" {
		if err := store.Disable(t.Context(), op.ID); err != nil {
			t.Fatal(err)
		}
	}
	if scenario == "login CSRF" {
		login.Header.Del("X-CSRF-Token")
	}
	start := httptest.NewRecorder()
	h.Login().ServeHTTP(start, login)
	if scenario == "forged body" {
		if start.Code != 400 || state != "" {
			t.Fatal("forged login body accepted")
		}
		return
	}
	if scenario == "unlisted" || scenario == "login CSRF" || scenario == "handle transferred" {
		if start.Code != 403 || state != "" {
			t.Fatal("denied login reached provider")
		}
		return
	}
	if start.Code != 200 || state == "" || scope != "atproto" {
		t.Fatalf("login %d %s scope=%s", start.Code, start.Body, scope)
	}
	callback := httptest.NewRequest("GET", "http://127.0.0.1:18080/oauth/admin/callback?"+url.Values{"state": {state}, "iss": {providerOrigin}, "code": {"fixture-code"}}.Encode(), nil)
	for _, c := range bootstrap.Result().Cookies() {
		callback.AddCookie(c)
	}
	switch scenario {
	case "disabled during flow":
		if err := store.Disable(t.Context(), op.ID); err != nil {
			t.Fatal(err)
		}
	case "wrong browser":
		callback.Header.Del("Cookie")
	case "wrong issuer":
		q := callback.URL.Query()
		q.Set("iss", "https://evil.test")
		callback.URL.RawQuery = q.Encode()
	case "duplicate state":
		callback.URL.RawQuery += "&state=forged"
	case "expired attempt":
		clock = clock.Add(11 * time.Minute)
	}
	complete := httptest.NewRecorder()
	h.Callback().ServeHTTP(complete, callback)
	switch scenario {
	case "disabled during flow", "wrong subject", "missing scope", "extra scope", "wrong browser", "wrong issuer", "duplicate state", "expired attempt":
		if complete.Code < 400 || len(complete.Result().Cookies()) != 0 {
			t.Fatal("invalid callback issued session")
		}
		return
	}
	if complete.Code != 303 || complete.Header().Get("Location") != "/admin/" {
		t.Fatalf("callback %d %s", complete.Code, complete.Body)
	}
	cookies := append(bootstrap.Result().Cookies(), complete.Result().Cookies()...)
	call := func(method, path string, handler http.Handler) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:18080"+path, nil)
		r.Header.Set("Origin", "http://127.0.0.1:18080")
		r.Header.Set("X-CSRF-Token", csrf.Token)
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if scenario == "terminal DID" {
		fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		owners, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := owners.EnsureOnboardingOwner(t.Context(), did); err != nil {
			t.Fatal(err)
		}
		if _, err := owners.Terminalize(t.Context(), ownerlifecycle.TerminalizeRequest{Owner: did, Reason: "identityDeleted"}); err != nil {
			t.Fatal(err)
		}
	}
	switch scenario {
	case "set change":
		if err := store.SetPermissionSet(t.Context(), op.ID, "helper"); err != nil {
			t.Fatal(err)
		}
	case "session disabled":
		if err := store.Disable(t.Context(), op.ID); err != nil {
			t.Fatal(err)
		}
	case "session revoked":
		if err := store.RevokeSessions(t.Context(), op.ID); err != nil {
			t.Fatal(err)
		}
	case "idle expiry":
		clock = clock.Add(31 * time.Minute)
	case "absolute expiry":
		for range 33 {
			clock = clock.Add(15 * time.Minute)
			_ = call("GET", "/v1/admin/identity", h.Authentication()(api.OperatorIdentityHandler()))
		}
	}
	if scenario == "helper assignment" {
		if err := store.SetPermissionSet(t.Context(), op.ID, "helper"); err != nil {
			t.Fatal(err)
		}
		incidentID, scanID := uuid.New(), uuid.New()
		if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'synthetic-helper-image','fixture','fixture','fixture','match')`, scanID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(t.Context(), `INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'synthetic-helper','synthetic-helper',now(),now())`, incidentID, scanID); err != nil {
			t.Fatal(err)
		}
		assignmentID := uuid.New()
		if _, err := pool.Exec(t.Context(), `INSERT INTO safety_operator_assignments(id,incident_id,operator_id,assignment_role,assigned_by,assigned_at) VALUES($1,$2,$3,'helper','test-fixture',now())`, assignmentID, incidentID, op.ID); err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		mux.Handle("GET /v1/admin/safety/incidents/{incidentReference}", h.Authentication()(api.RequireOperatorPermission(safetyincident.PermissionIncidentReadSafe)(api.RequireSafetyPermission(safetyincident.PermissionIncidentReadSafe)(api.SafetyIncidentDetailHandler(safetyincident.NewStore(pool, time.Now))))))
		path := "/v1/admin/safety/incidents/" + incidentID.String()
		if w := call("GET", path, mux); w.Code != 200 {
			t.Fatalf("assigned incident: %d %s", w.Code, w.Body)
		}
		if w := call("GET", "/v1/admin/safety/incidents/"+uuid.New().String(), mux); w.Code != 403 {
			t.Fatal("unassigned incident accepted")
		}
		if _, err := pool.Exec(t.Context(), `UPDATE safety_operator_assignments SET released_at=now() WHERE id=$1`, assignmentID); err != nil {
			t.Fatal(err)
		}
		if w := call("GET", path, mux); w.Code != 403 {
			t.Fatal("released assignment retained access")
		}
		if w := call("GET", "/v1/admin/moderation/cases", h.Authentication()(api.RequireOperatorPermission(safetyincident.PermissionModerationRead)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })))); w.Code != 403 {
			t.Fatal("helper browsed global cases")
		}
	}
	if scenario == "database unavailable" {
		pool.Close()
	}
	identity := call("GET", "/v1/admin/identity", h.Authentication()(api.OperatorIdentityHandler()))
	switch scenario {
	case "database unavailable", "terminal DID", "session disabled", "session revoked", "idle expiry", "absolute expiry":
		if identity.Code != 401 {
			t.Fatal("invalid session accepted")
		}
		return
	}
	if scenario == "set change" && (!strings.Contains(identity.Body.String(), `"permissionSet":"helper"`) || !strings.Contains(identity.Body.String(), `"moderation.manage":false`)) {
		t.Fatal("session retained old grants")
	}
	if identity.Code != 200 || !strings.Contains(identity.Body.String(), `"did":"did:plc:consolelogin"`) || strings.Contains(identity.Body.String(), "canary") {
		t.Fatalf("identity %d %s", identity.Code, identity.Body)
	}
	replay := httptest.NewRecorder()
	h.Callback().ServeHTTP(replay, callback)
	if replay.Code < 400 {
		t.Fatal("callback replay accepted")
	}
	unsafe := httptest.NewRequest("POST", "http://127.0.0.1:18080/v1/admin/auth/logout", nil)
	for _, c := range cookies {
		unsafe.AddCookie(c)
	}
	unsafe.Header.Set("Origin", "https://evil.test")
	unsafe.Header.Set("X-CSRF-Token", csrf.Token)
	denied := httptest.NewRecorder()
	h.Logout().ServeHTTP(denied, unsafe)
	if denied.Code != 403 {
		t.Fatal("cross-origin logout accepted")
	}
	if w := call("POST", "/v1/admin/auth/logout", h.Logout()); w.Code != 204 {
		t.Fatalf("logout %d", w.Code)
	}
	if w := call("GET", "/v1/admin/identity", h.Authentication()(api.OperatorIdentityHandler())); w.Code != 401 {
		t.Fatal("logged-out session accepted")
	}
}

func TestOperatorOAuthAndSessionBoundaries(t *testing.T) {
	for _, scenario := range []string{"legacy credentials", "handle transferred", "handle changed", "database unavailable", "terminal DID", "forged body", "helper assignment", "success", "unlisted", "login CSRF", "disabled during flow", "wrong subject", "missing scope", "extra scope", "wrong browser", "wrong issuer", "duplicate state", "expired attempt", "set change", "session disabled", "session revoked", "idle expiry", "absolute expiry"} {
		t.Run(scenario, func(t *testing.T) { runConsoleFlow(t, scenario) })
	}
}
