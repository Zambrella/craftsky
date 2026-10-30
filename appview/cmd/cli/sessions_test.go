package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestSessionsRevokeRequiresOneValidDID(t *testing.T) {
	for _, args := range [][]string{{"revoke"}, {"revoke", "--all", "did:plc:alice"}, {"revoke", "alice.example"}, {"revoke", "did:plc:"}, {"revoke", "did:plc:alice", "did:plc:bob"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			called := false
			cmd := newSessionsCmd(func(context.Context, syntax.DID) error { called = true; return nil }, nil)
			cmd.SetArgs(args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := cmd.Execute(); err == nil || called {
				t.Fatalf("invalid arguments reached revocation: called=%v error=%v", called, err)
			}
		})
	}
}

func TestSessionsRevokeInvalidatesOnlyRequestedUsersBearers(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	children := auth.NewCraftskySessionStore(pool, time.Minute)
	tokens := make(map[syntax.DID]string)
	for _, owner := range []syntax.DID{"did:plc:cli-revoke", "did:plc:cli-unaffected"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,
			    transitioned_at,created_at,updated_at)
			VALUES($1,'active',1,1,'test',now(),now(),now());
		`, owner); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,
			    auth_epoch,absolute_expires_at)
			VALUES($1,'cli-parent','{}','active',1,1,now()+interval '1 day')
		`, owner); err != nil {
			t.Fatal(err)
		}
		token, err := children.Create(ctx, owner.String(), "cli-parent", "fixture")
		if err != nil {
			t.Fatal(err)
		}
		tokens[owner] = token
	}
	connectionURL, err := url.Parse(pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	query := connectionURL.Query()
	query.Set("search_path", pool.Config().ConnConfig.RuntimeParams["search_path"])
	connectionURL.RawQuery = query.Encode()
	t.Setenv("DATABASE_URL", connectionURL.String())
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:8080")
	t.Setenv("TAP_WS_URL", "ws://localhost:2480/channel")
	t.Setenv("CRAFTSKY_DEV_DID", "did:plc:cli-revoke")
	previousEnv := envFlag
	envFlag = "dev"
	t.Cleanup(func() { envFlag = previousEnv })
	cmd := newSessionsCmd(revokeSessionsForDID, revokeSessionsForAll)
	cmd.SetArgs([]string{"revoke", "did:plc:cli-revoke"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := children.Lookup(ctx, tokens["did:plc:cli-revoke"]); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("revoked bearer lookup = %v", err)
	}
	if _, err := children.Lookup(ctx, tokens["did:plc:cli-unaffected"]); err != nil {
		t.Fatalf("another user's session was affected: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,terminal_at,created_at,updated_at) VALUES('did:plc:cli-terminal','terminal',1,1,'test',now(),now(),now(),now())`); err != nil {
		t.Fatal(err)
	}
	bulk := newSessionsCmd(revokeSessionsForDID, revokeSessionsForAll)
	var output bytes.Buffer
	bulk.SetArgs([]string{"revoke", "--all"})
	bulk.SetOut(&output)
	bulk.SetErr(&bytes.Buffer{})
	if err := bulk.Execute(); err != nil {
		t.Fatal(err)
	}
	for owner, token := range tokens {
		if _, err := children.Lookup(ctx, token); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
			t.Fatalf("bulk revocation left %s signed in: %v", owner, err)
		}
		var state string
		if err := pool.QueryRow(ctx, `SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1`, owner).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state != "revocation_pending" {
			t.Fatalf("OAuth cleanup not queued for %s: %s", owner, state)
		}
	}
	if !strings.Contains(output.String(), "all 2 accounts") {
		t.Fatalf("bulk output = %q", output.String())
	}
}

func TestSessionsRevokeRunsForRequestedDIDAndReportsQueuedCleanup(t *testing.T) {
	var got syntax.DID
	cmd := newSessionsCmd(func(ctx context.Context, did syntax.DID) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("revocation has no deadline")
		}
		got = did
		return nil
	}, nil)
	var output bytes.Buffer
	cmd.SetArgs([]string{"revoke", "did:plc:alice"})
	cmd.SetOut(&output)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got != "did:plc:alice" {
		t.Fatalf("revoked DID=%q", got)
	}
	if !strings.Contains(output.String(), "sessions revoked") || !strings.Contains(output.String(), "cleanup queued") {
		t.Fatalf("unexpected completion output: %q", output.String())
	}
}

func TestSessionsRevokeReturnsFailureWithoutClaimingSuccess(t *testing.T) {
	want := errors.New("revocation failed")
	cmd := newSessionsCmd(func(context.Context, syntax.DID) error { return want }, nil)
	var output bytes.Buffer
	cmd.SetArgs([]string{"revoke", "did:plc:alice"})
	cmd.SetOut(&output)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(output.String(), "sessions revoked") || strings.Contains(output.String(), "cleanup queued") {
		t.Fatalf("failure emitted success output: %q", output.String())
	}
}

func TestSessionsRevokeAllReportsCompletionAndFailures(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			want := errors.New("owner revocation failed")
			cmd := newSessionsCmd(func(context.Context, syntax.DID) error { t.Fatal("single runner called"); return nil }, func(context.Context) (int, error) {
				if fail {
					return 2, want
				}
				return 3, nil
			})
			var output bytes.Buffer
			cmd.SetArgs([]string{"revoke", "--all"})
			cmd.SetOut(&output)
			cmd.SetErr(&bytes.Buffer{})
			err := cmd.Execute()
			if fail {
				if !errors.Is(err, want) || !strings.Contains(output.String(), "2 accounts; bulk revocation incomplete") || strings.Contains(output.String(), "for all") {
					t.Fatalf("err=%v output=%q", err, output.String())
				}
			} else if err != nil || !strings.Contains(output.String(), "all 3 accounts; OAuth and push cleanup queued") {
				t.Fatalf("err=%v output=%q", err, output.String())
			}
		})
	}
}

func TestRevokeSessionOwnersContinuesAfterFailure(t *testing.T) {
	want := errors.New("failed owner")
	var called []syntax.DID
	count, err := revokeSessionOwners(context.Background(), []syntax.DID{"did:plc:a", "did:plc:b", "did:plc:c"}, func(ctx context.Context, owner syntax.DID) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("owner has no timeout")
		}
		called = append(called, owner)
		if owner == "did:plc:b" {
			return want
		}
		return nil
	})
	if count != 2 || !errors.Is(err, want) || len(called) != 3 {
		t.Fatalf("count=%d err=%v called=%v", count, err, called)
	}
}

func TestRevokeSessionOwnersHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	count, err := revokeSessionOwners(ctx, []syntax.DID{"did:plc:a"}, func(context.Context, syntax.DID) error { t.Fatal("revocation called after cancellation"); return nil })
	if count != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
