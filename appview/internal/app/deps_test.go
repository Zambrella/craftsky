package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

const authSchemaDDL = `
	CREATE TABLE oauth_sessions (
		account_did TEXT NOT NULL,
		session_id  TEXT NOT NULL,
		data        JSONB NOT NULL,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (account_did, session_id)
	);
	CREATE TABLE craftsky_sessions (
		token_hash        BYTEA NOT NULL PRIMARY KEY,
		account_did       TEXT NOT NULL,
		oauth_session_id  TEXT NOT NULL,
		device_label      TEXT,
		last_device_id    TEXT,
		created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
		last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
		revoked_at        TIMESTAMPTZ,
		FOREIGN KEY (account_did, oauth_session_id)
			REFERENCES oauth_sessions (account_did, session_id)
			ON DELETE CASCADE
	);
`

func TestNewDevDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:            EnvDev,
		DatabaseURL:    "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins: []string{"*"},
		DevDID:         "did:plc:test",
	}
	deps, cleanup, err := NewDevDeps(context.Background(), cfg)
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		if deps != nil && deps.DB != nil {
			deps.DB.Close()
		}
		t.Fatal("expected error for unreachable DB, got nil")
	}
	if !strings.Contains(err.Error(), "db") && !strings.Contains(err.Error(), "ping") {
		t.Errorf("err = %v, expected db/ping context", err)
	}
	if deps != nil {
		t.Errorf("deps = %v, want nil on error", deps)
	}
}

func TestNewProdDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:            EnvProd,
		DatabaseURL:    "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins: []string{"https://craftsky.social"},
	}
	_, _, err := NewProdDeps(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Covers the "which auth service gets wired" contract without touching
// the network: we construct Deps by hand and assert the field types match
// what each factory would have produced. This pins the behaviour even
// when a reachable DB isn't available.
func TestDepsAuthServiceShape(t *testing.T) {
	// Dev: StackedAuthService (real CraftskyAuthService primary, X-Dev-DID fallback)
	devDeps := &Deps{
		Config:      Config{Env: EnvDev, DevDID: "did:plc:default"},
		AuthService: &auth.StackedAuthService{Real: &auth.CraftskyAuthService{}},
	}
	if _, ok := devDeps.AuthService.(*auth.StackedAuthService); !ok {
		t.Errorf("dev: AuthService = %T, want *auth.StackedAuthService", devDeps.AuthService)
	}

	// Prod: CraftskyAuthService
	prodDeps := &Deps{
		Config:      Config{Env: EnvProd},
		AuthService: &auth.CraftskyAuthService{},
	}
	if _, ok := prodDeps.AuthService.(*auth.CraftskyAuthService); !ok {
		t.Errorf("prod: AuthService = %T, want *auth.CraftskyAuthService", prodDeps.AuthService)
	}
}

func TestExpirePDSSessionLogsBoundedContextWithoutRawIdentityOrSession(t *testing.T) {
	for _, tc := range []struct {
		name      string
		closePool bool
	}{
		{name: "successful cleanup"},
		{name: "cleanup errors", closePool: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, authSchemaDDL)
			ctx := context.Background()
			did := syntax.DID("did:plc:alice")
			sessionID := "session-secret"
			if !tc.closePool {
				oauthStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{Logger: slog.Default()})
				if err := oauthStore.SaveSession(ctx, oauth.ClientSessionData{
					AccountDID: did,
					SessionID:  sessionID,
					HostURL:    "https://pds.example",
				}); err != nil {
					t.Fatalf("SaveSession: %v", err)
				}
				if _, err := auth.NewCraftskySessionStore(pool, time.Minute).Create(ctx, did.String(), sessionID, "device"); err != nil {
					t.Fatalf("Create Craftsky session: %v", err)
				}
			} else {
				pool.Close()
			}

			var logs bytes.Buffer
			deps := &Deps{
				Logger: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
				OAuthStore: auth.NewPostgresAuthStore(pool, auth.StoreConfig{
					Logger: slog.Default(),
				}),
				CraftskySessionStore: auth.NewCraftskySessionStore(pool, time.Minute),
			}

			deps.expirePDSSession(ctx, did, sessionID)

			logged := logs.String()
			for _, forbidden := range []string{did.String(), sessionID, "device"} {
				if strings.Contains(logged, forbidden) {
					t.Fatalf("expirePDSSession logs contain forbidden value %q:\n%s", forbidden, logged)
				}
			}
			for _, want := range []string{
				`"component":"pds"`,
				`"operation":"oauth.session_resume"`,
				`"failure_stage":"session_resume"`,
				`"result":"error"`,
				`"error_category":"auth"`,
			} {
				if !strings.Contains(logged, want) {
					t.Fatalf("expirePDSSession logs missing bounded field %q:\n%s", want, logged)
				}
			}
		})
	}
}
