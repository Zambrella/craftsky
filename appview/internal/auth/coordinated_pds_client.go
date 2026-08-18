package auth

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// OAuthSessionRunner holds the owner and parent-session fences for the full
// lifetime of one authenticated PDS operation.
type OAuthSessionRunner interface {
	WithActiveSession(context.Context, syntax.DID, string, OAuthSessionOperation) error
}

// OAuthEffectSessionRunner combines participant owner fences with the OAuth
// parent-session fence on one dedicated connection. It is deliberately
// separate from OAuthSessionRunner so non-effect read paths can retain the
// smaller ordinary session contract.
type OAuthEffectSessionRunner interface {
	WithActiveEffectSession(
		context.Context,
		[]ownerlifecycle.ExpectedOwner,
		syntax.DID,
		string,
		OAuthSessionOperation,
	) error
}

// ActiveEffectPDSOperation receives a purpose-bound client that is valid only
// for the duration of the callback.
type ActiveEffectPDSOperation func(context.Context, PDSClient) error

// ActiveEffectPDSBoundary is the narrow capability consumed by durable PDS
// effect executors. It cannot return or retain a resumed OAuth session.
type ActiveEffectPDSBoundary interface {
	WithActiveEffects(
		context.Context,
		[]ownerlifecycle.ExpectedOwner,
		ActiveEffectPDSOperation,
	) error
}

// SessionPDSClientBuilder builds a purpose client from the fenced OAuth
// session. The resulting client must never escape the operation callback.
type SessionPDSClientBuilder func(context.Context, *oauth.ClientSession) (PDSClient, error)

type DeletionOAuthSessionRunner interface {
	WithDeletionSession(
		context.Context,
		syntax.DID,
		DeletionSessionAuthority,
		OAuthSessionOperation,
	) error
}

// CoordinatedPDSClient is the ordinary authenticated PDS boundary. Every
// method resumes and persists the parent OAuth session while its database
// fences remain held; no mutable OAuth session is retained between calls.
type CoordinatedPDSClient struct {
	runner    OAuthSessionRunner
	owner     syntax.DID
	sessionID string
	build     SessionPDSClientBuilder
}

func NewCoordinatedPDSClient(
	runner OAuthSessionRunner,
	owner syntax.DID,
	sessionID string,
	build SessionPDSClientBuilder,
) (*CoordinatedPDSClient, error) {
	if runner == nil || owner == "" || sessionID == "" || build == nil {
		return nil, errors.New("coordinated PDS client dependencies are unavailable")
	}
	return &CoordinatedPDSClient{runner: runner, owner: owner, sessionID: sessionID, build: build}, nil
}

func (client *CoordinatedPDSClient) withClient(
	ctx context.Context,
	operation func(context.Context, PDSClient) error,
) error {
	if client == nil || client.runner == nil || client.build == nil || operation == nil {
		return errors.New("coordinated PDS client is unavailable")
	}
	return client.runner.WithActiveSession(ctx, client.owner, client.sessionID, func(
		operationCtx context.Context,
		session *oauth.ClientSession,
	) error {
		purposeClient, err := client.build(operationCtx, session)
		if err != nil {
			return err
		}
		if purposeClient == nil {
			return errors.New("coordinated PDS purpose client is unavailable")
		}
		return operation(operationCtx, purposeClient)
	})
}

// WithActiveEffects constructs the purpose PDS client only after every
// expected owner generation and the parent OAuth session have been fenced on
// the same connection. The purpose client never becomes a field and must not
// escape operation.
func (client *CoordinatedPDSClient) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation ActiveEffectPDSOperation,
) error {
	if client == nil || client.runner == nil || client.build == nil || operation == nil {
		return errors.New("coordinated PDS effect boundary is unavailable")
	}
	runner, ok := client.runner.(OAuthEffectSessionRunner)
	if !ok || runner == nil {
		return errors.New("coordinated PDS effect session runner is unavailable")
	}
	return runner.WithActiveEffectSession(
		ctx,
		expected,
		client.owner,
		client.sessionID,
		func(operationCtx context.Context, session *oauth.ClientSession) error {
			purposeClient, err := client.build(operationCtx, session)
			if err != nil {
				return err
			}
			if purposeClient == nil {
				return errors.New("coordinated PDS purpose client is unavailable")
			}
			return operation(operationCtx, purposeClient)
		},
	)
}

func (client *CoordinatedPDSClient) GetRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	out any,
) (string, error) {
	var cid string
	err := client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		var err error
		cid, err = purposeClient.GetRecord(operationCtx, repo, collection, rkey, out)
		return err
	})
	return cid, err
}

func (client *CoordinatedPDSClient) PutRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	record any,
) error {
	return client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		return purposeClient.PutRecord(operationCtx, repo, collection, rkey, record)
	})
}

func (client *CoordinatedPDSClient) PutRecordWithSwap(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	record any,
	expectedCID syntax.CID,
) error {
	return client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		putter, ok := purposeClient.(ConditionalPDSRecordPutter)
		if !ok || putter == nil {
			return ErrConditionalPutUnsupported
		}
		return putter.PutRecordWithSwap(
			operationCtx, repo, collection, rkey, record, expectedCID,
		)
	})
}

func (client *CoordinatedPDSClient) CreateRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	record any,
) (syntax.ATURI, syntax.CID, error) {
	var uri syntax.ATURI
	var cid syntax.CID
	err := client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		var err error
		uri, cid, err = purposeClient.CreateRecord(operationCtx, repo, collection, record)
		return err
	})
	return uri, cid, err
}

func (client *CoordinatedPDSClient) DeleteRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
) error {
	return client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		return purposeClient.DeleteRecord(operationCtx, repo, collection, rkey)
	})
}

func (client *CoordinatedPDSClient) DeleteRecordWithSwap(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	expectedCID syntax.CID,
) error {
	return client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		deleter, ok := purposeClient.(ConditionalPDSRecordDeleter)
		if !ok || deleter == nil {
			return ErrConditionalDeleteUnsupported
		}
		return deleter.DeleteRecordWithSwap(
			operationCtx, repo, collection, rkey, expectedCID,
		)
	})
}

func (client *CoordinatedPDSClient) ListRecords(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	cursor string,
	limit int,
) ([]PDSRecord, string, error) {
	var records []PDSRecord
	var nextCursor string
	err := client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		lister, ok := purposeClient.(PDSRecordLister)
		if !ok {
			return errors.New("coordinated PDS purpose client cannot list records")
		}
		var err error
		records, nextCursor, err = lister.ListRecords(operationCtx, repo, collection, cursor, limit)
		return err
	})
	return records, nextCursor, err
}

func (client *CoordinatedPDSClient) UploadBlob(
	ctx context.Context,
	contentType string,
	body []byte,
) (*UploadedBlob, error) {
	var blob *UploadedBlob
	err := client.withClient(ctx, func(operationCtx context.Context, purposeClient PDSClient) error {
		var err error
		blob, err = purposeClient.UploadBlob(operationCtx, contentType, body)
		return err
	})
	return blob, err
}

var _ PDSClient = (*CoordinatedPDSClient)(nil)
var _ PDSRecordLister = (*CoordinatedPDSClient)(nil)
var _ ActiveEffectPDSBoundary = (*CoordinatedPDSClient)(nil)
var _ ConditionalPDSRecordPutter = (*CoordinatedPDSClient)(nil)
var _ ConditionalPDSRecordDeleter = (*CoordinatedPDSClient)(nil)

// CoordinatedDeletionPDSClient exposes only the list/delete surface and binds
// every call to one accepted deletion operation and credential generation.
type CoordinatedDeletionPDSClient struct {
	runner    DeletionOAuthSessionRunner
	owner     syntax.DID
	authority DeletionSessionAuthority
	build     SessionPDSClientBuilder
}

func NewCoordinatedDeletionPDSClient(
	runner DeletionOAuthSessionRunner,
	owner syntax.DID,
	authority DeletionSessionAuthority,
	build SessionPDSClientBuilder,
) (*CoordinatedDeletionPDSClient, error) {
	if runner == nil || owner == "" || !authority.valid() || build == nil {
		return nil, errors.New("coordinated deletion PDS client dependencies are unavailable")
	}
	return &CoordinatedDeletionPDSClient{
		runner: runner, owner: owner, authority: authority, build: build,
	}, nil
}

func (client *CoordinatedDeletionPDSClient) withClient(
	ctx context.Context,
	operation func(context.Context, DeletionPDSClient) error,
) error {
	if client == nil || client.runner == nil || client.build == nil || operation == nil {
		return errors.New("coordinated deletion PDS client is unavailable")
	}
	return client.runner.WithDeletionSession(
		ctx,
		client.owner,
		client.authority,
		func(operationCtx context.Context, session *oauth.ClientSession) error {
			purposeClient, err := client.build(operationCtx, session)
			if err != nil {
				return err
			}
			deletionClient, ok := purposeClient.(DeletionPDSClient)
			if !ok || deletionClient == nil {
				return errors.New("coordinated PDS purpose client lacks deletion capability")
			}
			return operation(operationCtx, deletionClient)
		},
	)
}

func (client *CoordinatedDeletionPDSClient) ListRecords(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	cursor string,
	limit int,
) ([]PDSRecord, string, error) {
	var records []PDSRecord
	var nextCursor string
	err := client.withClient(ctx, func(operationCtx context.Context, purposeClient DeletionPDSClient) error {
		var err error
		records, nextCursor, err = purposeClient.ListRecords(
			operationCtx, repo, collection, cursor, limit,
		)
		return err
	})
	return records, nextCursor, err
}

func (client *CoordinatedDeletionPDSClient) DeleteRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
) error {
	return client.withClient(ctx, func(operationCtx context.Context, purposeClient DeletionPDSClient) error {
		return purposeClient.DeleteRecord(operationCtx, repo, collection, rkey)
	})
}

var _ DeletionPDSClient = (*CoordinatedDeletionPDSClient)(nil)
