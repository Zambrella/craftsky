package auth_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

func TestBrowserHandoffDestinationSurvivesDatabaseReload(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	owner := syntax.DID("did:plc:browser-owner")
	seedAuthOwner(t, pool, owner)
	target := "http://127.0.0.1:25319/auth/browser/complete?state=" + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	ctx := auth.WithLoginAuthRequest(context.Background(), owner, 1, 1, "https://pds.example.com", "https://auth.example.test", auth.HandoffBrowser, "browser-device", target)
	request := oauth.AuthRequestData{State: "browser-request-state", RequestURI: "urn:request:browser", AuthServerURL: "https://auth.example.test"}
	if err := store.SaveAuthRequestInfo(ctx, request); err != nil {
		t.Fatal(err)
	}
	restarted := auth.NewPostgresAuthStore(pool, testStoreConfig())
	metadata, err := restarted.LoadAuthRequestMetadata(context.Background(), request.State)
	if err != nil || metadata.HandoffMode != auth.HandoffBrowser || metadata.LoopbackURI != target || metadata.DeviceID != "browser-device" {
		t.Fatalf("browser request binding lost: %v", err)
	}
	if _, err := restarted.BeginExchange(context.Background(), request.State); err != nil {
		t.Fatal(err)
	}
}
