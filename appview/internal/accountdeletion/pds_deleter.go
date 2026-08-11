package accountdeletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type ExpectedRecordRegistrar interface {
	RegisterExpected(ctx context.Context, jobID string, owner syntax.DID, uri syntax.ATURI, collection syntax.NSID) error
}

type ExpectedDeleteRequestMarker interface {
	MarkDeleteRequested(ctx context.Context, jobID string, owner syntax.DID, uri syntax.ATURI) error
}

type PDSDeletionResult struct {
	Listed      int
	DeleteCalls int
}

type PDSDeleter struct {
	client    auth.DeletionPDSClient
	registrar ExpectedRecordRegistrar
	batchSize int
}

func NewPDSDeleter(client auth.DeletionPDSClient, registrar ExpectedRecordRegistrar, batchSize int) *PDSDeleter {
	if batchSize <= 0 {
		batchSize = 20
	}
	return &PDSDeleter{client: client, registrar: registrar, batchSize: batchSize}
}

// DeleteAll converges the authenticated owner's registered CraftSky record
// collections to empty. It restarts a full scan after each non-empty pass so
// cursor movement caused by deletion and concurrent record creation cannot
// silently skip records. Membership profile is scanned and deleted last.
func (deleter *PDSDeleter) DeleteAll(ctx context.Context, jobID string, owner syntax.DID) (PDSDeletionResult, error) {
	var result PDSDeletionResult
	for {
		listedThisPass := 0
		for _, collection := range CraftskyRecordCollections() {
			cursor := ""
			for {
				records, nextCursor, err := deleter.client.ListRecords(
					ctx,
					owner,
					collection.String(),
					cursor,
					deleter.batchSize,
				)
				if err != nil {
					return result, fmt.Errorf("list CraftSky deletion records: %w", err)
				}
				for _, record := range records {
					uri, err := syntax.ParseATURI(record.URI.String())
					if err != nil || uri.Authority().String() != owner.String() || uri.Collection() != collection {
						return result, errors.New("PDS returned a record outside the deletion owner or collection")
					}
					if err := deleter.registrar.RegisterExpected(ctx, jobID, owner, uri, collection); err != nil {
						return result, fmt.Errorf("register expected CraftSky record: %w", err)
					}
					listedThisPass++
					result.Listed++
					if marker, ok := deleter.registrar.(ExpectedDeleteRequestMarker); ok {
						if err := marker.MarkDeleteRequested(ctx, jobID, owner, uri); err != nil {
							return result, fmt.Errorf("mark CraftSky record delete request: %w", err)
						}
					}
					result.DeleteCalls++
					deleteErr := deleter.client.DeleteRecord(ctx, owner, collection.String(), uri.RecordKey().String())
					if deleteErr != nil && !errors.Is(deleteErr, auth.ErrRecordNotFound) {
						return result, fmt.Errorf("delete CraftSky record: %w", deleteErr)
					}
				}
				if nextCursor == "" {
					break
				}
				cursor = nextCursor
			}
		}
		if listedThisPass == 0 {
			return result, nil
		}
	}
}
