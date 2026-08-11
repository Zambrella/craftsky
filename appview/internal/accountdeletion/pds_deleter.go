package accountdeletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type PDSDeletionResult struct {
	Listed      int
	DeleteCalls int
}

type PDSDeleter struct {
	client    auth.DeletionPDSClient
	batchSize int
}

func NewPDSDeleter(client auth.DeletionPDSClient, batchSize int) *PDSDeleter {
	if batchSize <= 0 {
		batchSize = 20
	}
	return &PDSDeleter{client: client, batchSize: batchSize}
}

// DeleteAll converges the authenticated owner's registered CraftSky record
// collections to empty. It first converges every non-membership collection,
// then deletes membership, and finally proves the complete registry empty.
// Repeated passes recover records skipped when deletion mutates pagination.
func (deleter *PDSDeleter) DeleteAll(ctx context.Context, owner syntax.DID) (PDSDeletionResult, error) {
	var result PDSDeletionResult
	collections := CraftskyRecordCollections()
	if len(collections) == 0 {
		return result, nil
	}
	nonMembership := collections[:len(collections)-1]
	membership := collections[len(collections)-1:]

	for {
		for {
			listed, err := deleter.deletePass(ctx, owner, nonMembership, &result)
			if err != nil {
				return result, err
			}
			if listed == 0 {
				break
			}
		}
		for {
			listed, err := deleter.deletePass(ctx, owner, membership, &result)
			if err != nil {
				return result, err
			}
			if listed == 0 {
				break
			}
		}
		empty, err := deleter.scanEmpty(ctx, owner, collections)
		if err != nil {
			return result, err
		}
		if empty {
			return result, nil
		}
	}
}

func (deleter *PDSDeleter) deletePass(
	ctx context.Context,
	owner syntax.DID,
	collections []syntax.NSID,
	result *PDSDeletionResult,
) (int, error) {
	listed := 0
	for _, collection := range collections {
		cursor := ""
		for {
			records, nextCursor, err := deleter.client.ListRecords(ctx, owner, collection.String(), cursor, deleter.batchSize)
			if err != nil {
				return listed, fmt.Errorf("list CraftSky deletion records: %w", err)
			}
			for _, record := range records {
				uri, err := validateDeletionRecord(record, owner, collection)
				if err != nil {
					return listed, err
				}
				listed++
				result.Listed++
				result.DeleteCalls++
				deleteErr := deleter.client.DeleteRecord(ctx, owner, collection.String(), uri.RecordKey().String())
				if deleteErr != nil && !errors.Is(deleteErr, auth.ErrRecordNotFound) {
					return listed, fmt.Errorf("delete CraftSky record: %w", deleteErr)
				}
			}
			if nextCursor == "" {
				break
			}
			cursor = nextCursor
		}
	}
	return listed, nil
}

func (deleter *PDSDeleter) scanEmpty(ctx context.Context, owner syntax.DID, collections []syntax.NSID) (bool, error) {
	empty := true
	for _, collection := range collections {
		records, _, err := deleter.client.ListRecords(ctx, owner, collection.String(), "", deleter.batchSize)
		if err != nil {
			return false, fmt.Errorf("scan final CraftSky deletion records: %w", err)
		}
		for _, record := range records {
			if _, err := validateDeletionRecord(record, owner, collection); err != nil {
				return false, err
			}
			empty = false
		}
	}
	return empty, nil
}

func validateDeletionRecord(record auth.PDSRecord, owner syntax.DID, collection syntax.NSID) (syntax.ATURI, error) {
	uri, err := syntax.ParseATURI(record.URI.String())
	if err != nil || uri.Authority().String() != owner.String() || uri.Collection() != collection {
		return "", errors.New("PDS returned a record outside the deletion owner or collection")
	}
	return uri, nil
}
