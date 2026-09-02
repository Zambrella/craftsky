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

type AccountTypeDeleter interface {
	DeleteAccountType(context.Context, syntax.DID) error
}

func isDeletionCollection(collection syntax.NSID) bool {
	for _, candidate := range CraftskyRecordCollections() {
		if collection == candidate {
			return true
		}
	}
	return false
}

func NewPDSDeleter(client auth.DeletionPDSClient, batchSize int) *PDSDeleter {
	if batchSize <= 0 {
		batchSize = 20
	}
	return &PDSDeleter{client: client, batchSize: batchSize}
}

// DeleteExact removes one already-authorized safety URI. The URI is
// revalidated against the authenticated owner and the closed CraftSky
// collection registry before the narrow PDS capability is invoked.
func (deleter *PDSDeleter) DeleteExact(
	ctx context.Context,
	owner syntax.DID,
	uri syntax.ATURI,
) error {
	if deleter == nil || deleter.client == nil {
		return errors.New("PDS deletion client is unavailable")
	}
	parsed, err := parsePDSSafetyURI(owner, uri.String())
	if err != nil {
		return err
	}
	err = deleter.client.DeleteRecord(
		ctx,
		owner,
		parsed.Collection().String(),
		parsed.RecordKey().String(),
	)
	if err != nil && !errors.Is(err, auth.ErrRecordNotFound) {
		return fmt.Errorf("delete exact CraftSky safety record: %w", err)
	}
	return nil
}

// DeleteAll converges the authenticated owner's registered PDS collections in
// fixed stages and proves the complete registry empty. Repeated passes recover
// records skipped when deletion mutates pagination.
func (deleter *PDSDeleter) DeleteAll(ctx context.Context, owner syntax.DID) (PDSDeletionResult, error) {
	return deleter.deleteAll(ctx, owner, nil)
}

// DeleteAllWithAccountType preserves the permanent-deletion boundary between
// public business records and the membership-defining profile.
func (deleter *PDSDeleter) DeleteAllWithAccountType(
	ctx context.Context,
	owner syntax.DID,
	accountTypes AccountTypeDeleter,
) (PDSDeletionResult, error) {
	if accountTypes == nil {
		return PDSDeletionResult{}, errors.New("account type deletion is unavailable")
	}
	return deleter.deleteAll(ctx, owner, accountTypes)
}

func (deleter *PDSDeleter) deleteAll(
	ctx context.Context,
	owner syntax.DID,
	accountTypes AccountTypeDeleter,
) (PDSDeletionResult, error) {
	var result PDSDeletionResult
	collections := CraftskyRecordCollections()
	if len(collections) == 0 {
		return result, nil
	}
	if len(collections) < 3 {
		return result, errors.New("CraftSky deletion registry is missing business stages")
	}
	stages := [][]syntax.NSID{
		collections[:len(collections)-3],
		collections[len(collections)-3 : len(collections)-2],
		collections[len(collections)-2 : len(collections)-1],
	}
	membership := collections[len(collections)-1:]

	for {
		for _, stage := range stages {
			if err := deleter.convergeCollections(ctx, owner, stage, &result); err != nil {
				return result, err
			}
		}
		if accountTypes != nil {
			if err := accountTypes.DeleteAccountType(ctx, owner); err != nil {
				return result, fmt.Errorf("delete CraftSky account type: %w", err)
			}
		}
		if err := deleter.convergeCollections(ctx, owner, membership, &result); err != nil {
			return result, err
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

func (deleter *PDSDeleter) convergeCollections(
	ctx context.Context,
	owner syntax.DID,
	collections []syntax.NSID,
	result *PDSDeletionResult,
) error {
	for {
		listed, err := deleter.deletePass(ctx, owner, collections, result)
		if err != nil {
			return err
		}
		if listed == 0 {
			return nil
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
