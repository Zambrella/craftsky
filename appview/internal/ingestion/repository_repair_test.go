package ingestion_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
)

func TestRepositoryRepairComparisonRequiresVerifiedSnapshotAndUsesImmutableRegistry(t *testing.T) {
	dispatcher := index.NewTransactionalDispatcher()
	for _, collection := range []syntax.NSID{
		"social.craftsky.feed.repost",
		"social.craftsky.actor.profile",
		"social.craftsky.feed.like",
		"social.craftsky.feed.post",
		"social.craftsky.feed.post",
	} {
		dispatcher.Register(collection, repairTestIndexer{})
	}

	collections := dispatcher.Collections()
	if !slices.IsSorted(collections) || len(collections) != 4 {
		t.Fatalf("collections = %v, want four sorted unique NSIDs", collections)
	}
	collections[0] = "social.craftsky.test.mutated"
	if dispatcher.Collections()[0] == collections[0] {
		t.Fatal("Collections returned mutable dispatcher state")
	}

	did := syntax.DID("did:plc:migrating")
	indexed := make([]ingestion.SourceRecord, 0, len(dispatcher.Collections())+1)
	for position, collection := range dispatcher.Collections() {
		uri := syntax.ATURI("at://" + did.String() + "/" + collection.String() + "/record")
		switch position {
		case 0:
		case 1:
			indexed = append(indexed, ingestion.SourceRecord{
				URI: uri, DID: did, Collection: collection, Rkey: "record", CID: "bafystale", Action: "create",
			})
		case 2:
			indexed = append(indexed, ingestion.SourceRecord{
				URI: uri, DID: did, Collection: collection, Rkey: "record", CID: "bafydelete", Action: "create",
			})
		case 3:
			indexed = append(indexed, ingestion.SourceRecord{
				URI: uri, DID: did, Collection: collection, Rkey: "record", CID: "bafyunchanged", Action: "create",
			})
		}
	}
	indexed = append(indexed, ingestion.SourceRecord{
		URI: "at://did:plc:migrating/social.example.unknown/stale",
		DID: did, Collection: "social.example.unknown", Rkey: "stale", CID: "bafyunknownstale", Action: "create",
	})

	var unverified ingestion.VerifiedRepositorySnapshot
	if descriptions, compareErr := ingestion.DescribeRepositoryRepair(unverified, indexed, dispatcher); !errors.Is(compareErr, ingestion.ErrRepositorySnapshotUnverified) || len(descriptions) != 0 {
		t.Fatalf("unverified comparison = (%v, %v), want no descriptions and unverified error", descriptions, compareErr)
	}
}

type repairTestIndexer struct{}

func (repairTestIndexer) Project(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error) {
	return tap.Applied(), nil
}
