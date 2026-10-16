package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/operatorconsole"
	"social.craftsky/appview/internal/testdb"
)

func TestOperatorDIDRegistryCLI(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	output, err := operatorCLI(t, pool, "allow", "did:plc:namedconsoleadmin", "--permission-set", "admin")
	if err != nil {
		t.Fatal(err)
	}
	var op operatorconsole.Operator
	if err := json.Unmarshal([]byte(output), &op); err != nil {
		t.Fatal(err)
	}
	if op.DID != "did:plc:namedconsoleadmin" || !op.Permissions["operator.manage"] || strings.Contains(output, "token") {
		t.Fatal("incorrect allowlist result")
	}
	if _, err := operatorCLI(t, pool, "set", op.ID.String(), "--permission-set", "mod"); err != nil {
		t.Fatal(err)
	}
	current, err := operatorconsole.NewStore(pool, time.Now).Lookup(t.Context(), syntax.DID("did:plc:namedconsoleadmin"))
	if err != nil || current.Permissions["operator.manage"] {
		t.Fatal("set change not effective")
	}
	if _, err := operatorCLI(t, pool, "disable", op.ID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := operatorconsole.NewStore(pool, time.Now).Lookup(t.Context(), op.DID); err == nil {
		t.Fatal("disable not effective")
	}
}

func operatorCLI(t *testing.T, pool *pgxpool.Pool, args ...string) (string, error) {
	t.Helper()
	cmd := newOperatorsCmd(func(context.Context) (*pgxpool.Pool, func(), error) { return pool, func() {}, nil })
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(t.Context())
	return output.String(), err
}
