package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

func TestCraftskyLookupExpiresParentAndQueuesLocalRevocation(t *testing.T) {
	pool := withAuthSchema(t)
	token := "expired-parent-token"
	hash := sha256.Sum256([]byte(token))
	seedLifecycleSession(t, pool, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES('did:plc:parent-expired','active',1,1,'test',now()-interval '2 hours',now()-interval '2 hours',now()-interval '2 hours');
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			absolute_expires_at,created_at,updated_at
		) VALUES(
			'did:plc:parent-expired','parent-expired','{}','active',1,1,
			now()-interval '1 hour',now()-interval '2 hours',now()-interval '2 hours'
		);
	`, hash[:], `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			last_seen_at,idle_expires_at
		) VALUES($1,'did:plc:parent-expired','parent-expired','active',1,now(),now()+interval '1 day')
	`)
	store, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            24 * time.Hour,
		ActivityWriteInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Lookup(context.Background(), token); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("expired parent lookup = %v, want invalid", err)
	}
	var parentState, childState string
	if err := pool.QueryRow(context.Background(), `
		SELECT oauth.lifecycle_state,child.lifecycle_state
		FROM oauth_sessions oauth
		JOIN craftsky_sessions child
		  ON child.account_did=oauth.account_did AND child.oauth_session_id=oauth.session_id
		WHERE child.token_hash=$1
	`, hash[:]).Scan(&parentState, &childState); err != nil {
		t.Fatal(err)
	}
	if parentState != "revocation_pending" || childState != "revoked" {
		t.Fatalf("expired parent states = %s/%s", parentState, childState)
	}
}

func TestCraftskyLookupExpiresOnlyIdleChild(t *testing.T) {
	pool := withAuthSchema(t)
	token := "idle-child-token"
	hash := sha256.Sum256([]byte(token))
	seedLifecycleSession(t, pool, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES('did:plc:child-idle','active',1,1,'test',now()-interval '2 hours',now()-interval '2 hours',now()-interval '2 hours');
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			absolute_expires_at,created_at,updated_at
		) VALUES('did:plc:child-idle','parent-active','{}','active',1,1,
			now()+interval '10 days',now()-interval '2 hours',now()-interval '2 hours');
	`, hash[:], `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			created_at,last_seen_at,idle_expires_at
		) VALUES($1,'did:plc:child-idle','parent-active','active',1,
			now()-interval '2 hours',now()-interval '2 hours',now()-interval '1 hour')
	`)
	store, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            24 * time.Hour,
		ActivityWriteInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Lookup(context.Background(), token); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("idle child lookup = %v, want invalid", err)
	}
	var parentState, childState string
	if err := pool.QueryRow(context.Background(), `
		SELECT oauth.lifecycle_state,child.lifecycle_state
		FROM oauth_sessions oauth
		JOIN craftsky_sessions child
		  ON child.account_did=oauth.account_did AND child.oauth_session_id=oauth.session_id
		WHERE child.token_hash=$1
	`, hash[:]).Scan(&parentState, &childState); err != nil {
		t.Fatal(err)
	}
	if parentState != "active" || childState != "revoked" {
		t.Fatalf("idle child states = %s/%s", parentState, childState)
	}
}

func TestCraftskyActivityThrottleIsDatabaseBackedAcrossStores(t *testing.T) {
	pool := withAuthSchema(t)
	token := "database-throttle-token"
	hash := sha256.Sum256([]byte(token))
	seedLifecycleSession(t, pool, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES('did:plc:database-throttle','active',1,1,'test',now()-interval '2 hours',now()-interval '2 hours',now()-interval '2 hours');
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			absolute_expires_at,created_at,updated_at
		) VALUES('did:plc:database-throttle','parent','{}','active',1,1,
			now()+interval '10 days',now()-interval '2 hours',now()-interval '2 hours');
	`, hash[:], `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			created_at,last_seen_at,idle_expires_at
		) VALUES($1,'did:plc:database-throttle','parent','active',1,
			now()-interval '2 hours',now()-interval '2 hours',now()+interval '1 day')
	`)
	config := auth.CraftskySessionConfig{Inactivity: 24 * time.Hour, ActivityWriteInterval: time.Hour}
	storeA, err := auth.NewCraftskySessionStoreWithConfig(pool, config)
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := auth.NewCraftskySessionStoreWithConfig(pool, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storeA.Lookup(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	var first time.Time
	if err := pool.QueryRow(context.Background(), `SELECT last_seen_at FROM craftsky_sessions WHERE token_hash=$1`, hash[:]).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if _, err := storeB.Lookup(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	var second time.Time
	if err := pool.QueryRow(context.Background(), `SELECT last_seen_at FROM craftsky_sessions WHERE token_hash=$1`, hash[:]).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if !first.Equal(second) {
		t.Fatalf("second store bypassed throttle: %s -> %s", first, second)
	}
	if err := storeA.TouchDeviceID(context.Background(), "did:plc:database-throttle", "parent", "device-a"); err != nil {
		t.Fatal(err)
	}
	if err := storeB.TouchDeviceID(context.Background(), "did:plc:database-throttle", "parent", "device-b"); err != nil {
		t.Fatal(err)
	}
	var device string
	if err := pool.QueryRow(context.Background(), `SELECT last_device_id FROM craftsky_sessions WHERE token_hash=$1`, hash[:]).Scan(&device); err != nil {
		t.Fatal(err)
	}
	if device != "device-b" {
		t.Fatalf("changed device was throttled: %q", device)
	}

	typeOfStore := reflect.TypeOf(*storeA)
	for index := 0; index < typeOfStore.NumField(); index++ {
		if typeOfStore.Field(index).Type.Kind() == reflect.Map {
			t.Fatalf("CraftskySessionStore retains process map field %s", typeOfStore.Field(index).Name)
		}
	}
}

func seedLifecycleSession(t *testing.T, pool *pgxpool.Pool, parentSQL string, hash []byte, childSQL string) {
	t.Helper()
	// This generic-looking helper deliberately avoids hiding test authority:
	// every caller supplies the exact lifecycle fixture SQL above.
	if _, err := pool.Exec(context.Background(), parentSQL); err != nil {
		t.Fatalf("seed parent lifecycle: %v", err)
	}
	if _, err := pool.Exec(context.Background(), childSQL, hash); err != nil {
		t.Fatalf("seed child lifecycle: %v", err)
	}
}
