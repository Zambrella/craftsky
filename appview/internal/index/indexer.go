// Package index defines the contract for writing atproto records into
// Postgres. Day one contains only the interface and a NotImplemented stub
// so the CLI's backfill subcommand compiles.
package index

import (
	"context"
	"errors"
)

// Indexer writes records into the application's Postgres store.
type Indexer interface {
	// Backfill re-indexes all records for the given DID from its PDS.
	Backfill(ctx context.Context, did string) error
}

// NotImplemented is the day-one stub.
type NotImplemented struct{}

var _ Indexer = NotImplemented{}

func (NotImplemented) Backfill(ctx context.Context, did string) error {
	return errors.New("indexer: not yet implemented")
}
