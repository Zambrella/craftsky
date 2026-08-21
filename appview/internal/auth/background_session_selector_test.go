package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

func TestBackgroundSessionSelector_SelectsMostRecentlyActiveOwnerSession(t *testing.T) {
	pool := withAuthSchema(t)
	ctx := context.Background()
	selector := auth.NewBackgroundSessionSelector(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	seedActiveAuthOwner(t, pool, alice)
	seedActiveAuthOwner(t, pool, bob)
	now := time.Now().UTC()

	for _, session := range []struct {
		did       syntax.DID
		id        string
		updatedAt time.Time
		lastSeen  time.Time
		revokedAt *time.Time
	}{
		{alice, "alice-old", now.Add(-2 * time.Hour), now.Add(-2 * time.Hour), nil},
		{alice, "alice-current", now.Add(-time.Hour), now.Add(-time.Minute), nil},
		{alice, "alice-revoked", now, now, &now},
		{bob, "bob-newer", now.Add(time.Hour), now.Add(time.Hour), nil},
	} {
		childState := "active"
		if session.revokedAt != nil {
			childState = "revoked"
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO oauth_sessions (
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				created_at,updated_at
			) VALUES ($1,$2,'{}','active',1,1,$3,$3)
		`, session.did, session.id, session.updatedAt); err != nil {
			t.Fatalf("insert OAuth session %s: %v", session.id, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_sessions (
				token_hash, account_did, oauth_session_id,
				lifecycle_state,auth_epoch,last_seen_at,revoked_at
			) VALUES (convert_to($1, 'UTF8'),$2,$3,$4,1,$5,$6)
		`, session.id, session.did, session.id, childState, session.lastSeen, session.revokedAt); err != nil {
			t.Fatalf("insert CraftSky session %s: %v", session.id, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			created_at,updated_at
		) VALUES($1,'alice-pending','{}','pending_handoff',1,1,$2,$2)
	`, alice, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("insert pending OAuth handoff: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,last_seen_at
		) VALUES(convert_to('alice-pending','UTF8'),$1,'alice-pending','pending_confirmation',1,$2)
	`, alice, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("insert pending child handoff: %v", err)
	}

	sessionID, err := selector.Select(ctx, alice)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sessionID != "alice-current" {
		t.Fatalf("session ID = %q, want alice-current", sessionID)
	}
}

func TestBackgroundSessionSelector_UsesStableSessionIDTieBreak(t *testing.T) {
	pool := withAuthSchema(t)
	ctx := context.Background()
	selector := auth.NewBackgroundSessionSelector(pool)
	alice := syntax.DID("did:plc:alice")
	seedActiveAuthOwner(t, pool, alice)
	at := time.Now().UTC()

	for _, sessionID := range []string{"session-z", "session-a"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO oauth_sessions (
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				created_at,updated_at
			) VALUES ($1,$2,'{}','active',1,1,$3,$3)
		`, alice, sessionID, at); err != nil {
			t.Fatalf("insert OAuth session %s: %v", sessionID, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_sessions (
				token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,last_seen_at
			) VALUES (convert_to($2, 'UTF8'),$1,$2,'active',1,$3)
		`, alice, sessionID, at); err != nil {
			t.Fatalf("insert CraftSky session %s: %v", sessionID, err)
		}
	}

	sessionID, err := selector.Select(ctx, alice)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sessionID != "session-a" {
		t.Fatalf("session ID = %q, want stable session-a tie-break", sessionID)
	}
}

func TestBackgroundSessionSelector_NoUsableOwnerSessionIsRetryable(t *testing.T) {
	pool := withAuthSchema(t)
	ctx := context.Background()
	selector := auth.NewBackgroundSessionSelector(pool)

	if _, err := selector.Select(ctx, syntax.DID("did:plc:alice")); !errors.Is(err, auth.ErrNoUsableBackgroundSession) {
		t.Fatalf("Select error = %v, want ErrNoUsableBackgroundSession", err)
	}
}

func TestBackgroundSessionSelector_InvalidatesOnlyExactOwnerSession(t *testing.T) {
	pool := withAuthSchema(t)
	ctx := context.Background()
	selector := auth.NewBackgroundSessionSelector(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	seedActiveAuthOwner(t, pool, alice)
	seedActiveAuthOwner(t, pool, bob)

	for _, did := range []syntax.DID{alice, bob} {
		for _, sessionID := range []string{"shared", "other"} {
			if _, err := pool.Exec(ctx, `
				INSERT INTO oauth_sessions(
					account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch
				) VALUES($1,$2,'{}','active',1,1)
			`, did, sessionID); err != nil {
				t.Fatalf("insert %s/%s: %v", did, sessionID, err)
			}
		}
	}

	if err := selector.Invalidate(ctx, alice, "shared"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	var remaining []string
	rows, err := pool.Query(ctx, `
		SELECT account_did || '/' || session_id
		FROM oauth_sessions
		ORDER BY account_did, session_id
	`)
	if err != nil {
		t.Fatalf("query remaining sessions: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("scan remaining session: %v", err)
		}
		remaining = append(remaining, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate remaining sessions: %v", err)
	}

	want := []string{
		"did:plc:alice/other",
		"did:plc:bob/other",
		"did:plc:bob/shared",
	}
	if len(remaining) != len(want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
	for i := range want {
		if remaining[i] != want[i] {
			t.Fatalf("remaining = %v, want %v", remaining, want)
		}
	}
}
