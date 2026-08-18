package languages

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const languageStorePreStateDDL = `
CREATE TABLE craftsky_posts (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    tags        TEXT[]      NOT NULL DEFAULT '{}',
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
CREATE TABLE owner_lifecycles (
	owner_did TEXT PRIMARY KEY,
	state TEXT NOT NULL,
	generation BIGINT NOT NULL,
	auth_epoch BIGINT NOT NULL DEFAULT 1,
	transition_reason TEXT NOT NULL DEFAULT 'test',
	transitioned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	terminal_at TIMESTAMPTZ,
	purge_completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO owner_lifecycles(owner_did,state,generation) VALUES
	('did:plc:alice','active',1),
	('did:plc:bob','active',1),
	('did:plc:unknown','active',1);
`

func newLanguageStoreTestStore(t *testing.T) *Store {
	t.Helper()
	up, err := os.ReadFile("../../migrations/000033_post_languages.up.sql")
	if err != nil {
		t.Fatalf("read language migration: %v", err)
	}
	pool := testdb.WithSchema(t, languageStorePreStateDDL)
	if _, err := pool.Exec(context.Background(), string(up)); err != nil {
		t.Fatalf("apply language migration: %v", err)
	}
	return NewStore(pool)
}

func TestStoreGetAndReplaceAreIsolatedByDID(t *testing.T) {
	store := newLanguageStoreTestStore(t)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")

	if _, err := store.Get(ctx, alice); !errors.Is(err, ErrPreferencesNotFound) {
		t.Fatalf("Get absent error = %v, want ErrPreferencesNotFound", err)
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES
			($1, 'en', ARRAY['en']::text[]),
			($2, 'fr', ARRAY['fr', 'cy']::text[])
	`, alice, bob); err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	gotAlice, err := store.Get(ctx, alice)
	if err != nil {
		t.Fatalf("Get Alice: %v", err)
	}
	assertPreferences(t, gotAlice, Preferences{
		PrimaryLanguage:  "en",
		ContentLanguages: []string{"en"},
	})

	replaced, err := store.Replace(ctx, alice, Preferences{
		PrimaryLanguage:  "es",
		ContentLanguages: []string{"es", "en"},
	})
	if err != nil {
		t.Fatalf("Replace Alice: %v", err)
	}
	assertPreferences(t, replaced, Preferences{
		PrimaryLanguage:  "es",
		ContentLanguages: []string{"es", "en"},
	})

	gotBob, err := store.Get(ctx, bob)
	if err != nil {
		t.Fatalf("Get Bob: %v", err)
	}
	assertPreferences(t, gotBob, Preferences{
		PrimaryLanguage:  "fr",
		ContentLanguages: []string{"fr", "cy"},
	})

	unknown := syntax.DID("did:plc:unknown")
	if _, err := store.Replace(ctx, unknown, Preferences{
		PrimaryLanguage:  "en",
		ContentLanguages: []string{"en"},
	}); !errors.Is(err, ErrPreferencesNotFound) {
		t.Fatalf("Replace absent error = %v, want ErrPreferencesNotFound", err)
	}
}

func TestStoreInitializeConcurrentlyReturnsOneAuthoritativeRow(t *testing.T) {
	store := newLanguageStoreTestStore(t)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	did := syntax.DID("did:plc:alice")
	proposals := []Preferences{
		{
			PrimaryLanguage:  "en",
			ContentLanguages: []string{"en"},
		},
		{
			PrimaryLanguage:  "fr",
			ContentLanguages: []string{"fr", "cy"},
		},
	}

	start := make(chan struct{})
	results := make([]Preferences, len(proposals))
	errs := make([]error, len(proposals))
	var wait sync.WaitGroup
	for index, proposal := range proposals {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			results[index], errs[index] = store.Initialize(ctx, did, proposal)
		}()
	}
	close(start)
	wait.Wait()

	for index, err := range errs {
		if err != nil {
			t.Fatalf("Initialize call %d: %v", index, err)
		}
	}
	assertPreferences(t, results[0], results[1])

	var rowCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM account_language_preferences
		WHERE account_did = $1
	`, did).Scan(&rowCount); err != nil {
		t.Fatalf("count authoritative rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("preference row count = %d, want 1", rowCount)
	}

	stored, err := store.Get(ctx, did)
	if err != nil {
		t.Fatalf("Get authoritative row: %v", err)
	}
	assertPreferences(t, stored, results[0])
}

func TestStoreReplaceRejectsInvalidPreferencesAtomically(t *testing.T) {
	store := newLanguageStoreTestStore(t)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	did := syntax.DID("did:plc:alice")
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES ($1, 'en', ARRAY['en', 'cy']::text[])
	`, did); err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	tests := []struct {
		name       string
		preference Preferences
	}{
		{
			name: "invalid primary",
			preference: Preferences{
				PrimaryLanguage:  "not_a_tag",
				ContentLanguages: []string{"en"},
			},
		},
		{
			name: "non selectable primary variant",
			preference: Preferences{
				PrimaryLanguage:  "fr-CA",
				ContentLanguages: []string{"fr"},
			},
		},
		{
			name: "unsupported content variant",
			preference: Preferences{
				PrimaryLanguage:  "fr",
				ContentLanguages: []string{"fr-CA"},
			},
		},
		{
			name: "duplicate content",
			preference: Preferences{
				PrimaryLanguage:  "fr",
				ContentLanguages: []string{"fr", "fr"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := store.Replace(ctx, did, test.preference); !errors.Is(err, ErrInvalidPreferences) {
				t.Fatalf("Replace error = %v, want ErrInvalidPreferences", err)
			}
			stored, err := store.Get(ctx, did)
			if err != nil {
				t.Fatalf("Get after rejected replacement: %v", err)
			}
			assertPreferences(t, stored, Preferences{
				PrimaryLanguage:  "en",
				ContentLanguages: []string{"en", "cy"},
			})
		})
	}

	replaced, err := store.Replace(ctx, did, Preferences{
		PrimaryLanguage:  "fr",
		ContentLanguages: []string{},
	})
	if err != nil {
		t.Fatalf("Replace with empty Content: %v", err)
	}
	assertPreferences(t, replaced, Preferences{
		PrimaryLanguage:  "fr",
		ContentLanguages: []string{},
	})
}

func TestStoreHandleIdentityDeletedIsIdempotent(t *testing.T) {
	store := newLanguageStoreTestStore(t)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	for did, primary := range map[syntax.DID]string{alice: "en", bob: "fr"} {
		if _, err := store.Initialize(ctx, did, Preferences{
			PrimaryLanguage:  primary,
			ContentLanguages: []string{primary},
		}); err != nil {
			t.Fatalf("Initialize %s: %v", did, err)
		}
	}

	if err := store.HandleIdentityDeleted(ctx, alice); err != nil {
		t.Fatalf("first HandleIdentityDeleted: %v", err)
	}
	if err := store.HandleIdentityDeleted(ctx, alice); err != nil {
		t.Fatalf("second HandleIdentityDeleted: %v", err)
	}
	if _, err := store.Get(ctx, alice); !errors.Is(err, ErrPreferencesNotFound) {
		t.Fatalf("deleted Alice Get error = %v, want ErrPreferencesNotFound", err)
	}
	if _, err := store.Get(ctx, bob); err != nil {
		t.Fatalf("Bob preferences were removed: %v", err)
	}
}

func assertPreferences(t *testing.T, got, want Preferences) {
	t.Helper()
	if got.PrimaryLanguage != want.PrimaryLanguage {
		t.Fatalf("PrimaryLanguage = %q, want %q", got.PrimaryLanguage, want.PrimaryLanguage)
	}
	if len(got.ContentLanguages) != len(want.ContentLanguages) {
		t.Fatalf("ContentLanguages = %v, want %v", got.ContentLanguages, want.ContentLanguages)
	}
	for index := range want.ContentLanguages {
		if got.ContentLanguages[index] != want.ContentLanguages[index] {
			t.Fatalf("ContentLanguages = %v, want %v", got.ContentLanguages, want.ContentLanguages)
		}
	}
}
