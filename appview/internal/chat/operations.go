package chat

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"social.craftsky/appview/internal/chatdb"
)

const operationFinalizationTimeout = 5 * time.Second

var ErrOperationConflict = errors.New("messaging operation ID has a different request")
var ErrOperationUnavailable = errors.New("messaging operation unavailable")

type OperationReceipt struct {
	OperationID uuid.UUID `json:"operationId"`
	Kind        string    `json:"kind"`
	State       string    `json:"state"`
	MessageID   MessageID `json:"messageId,omitempty"`
	FailureCode string    `json:"failureCode,omitempty"`
}

func (OperationReceipt) String() string   { return "ChatOperationReceipt([REDACTED])" }
func (OperationReceipt) GoString() string { return "ChatOperationReceipt([REDACTED])" }

type OperationResult struct{ ID, Revision string }
type OperationLedger struct {
	store   *Store
	current int
	keys    map[int][]byte
}

func (*OperationLedger) String() string   { return "messaging operation ledger [REDACTED]" }
func (*OperationLedger) GoString() string { return "messaging operation ledger [REDACTED]" }

func NewOperationLedger(store *Store, current int, keys map[int][]byte) (*OperationLedger, error) {
	if store == nil || current <= 0 || current > 1_000_000 || len(keys[current]) != 32 {
		return nil, ErrOperationUnavailable
	}
	copy := make(map[int][]byte, len(keys))
	for version, key := range keys {
		if version <= 0 || len(key) != 32 {
			return nil, ErrOperationUnavailable
		}
		copy[version] = slices.Clone(key)
	}
	return &OperationLedger{store: store, current: current, keys: copy}, nil
}

func operationDigest(key []byte, owner syntax.DID, id uuid.UUID, kind string, payload any) ([]byte, error) {
	// JSON supplies unambiguous framing and stable map-key order. Owner,
	// operation and kind are part of the keyed digest's domain separation.
	encoded, err := json.Marshal([]any{"craftsky-chat-operation-v1", owner, id, kind, payload})
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(encoded)
	return mac.Sum(nil), nil
}

func receiptFor(row chatdb.ChatOperation) OperationReceipt {
	state := row.State
	if state == "prepared" || state == "attempting" {
		state = "uncertain"
	}
	result := OperationReceipt{OperationID: uuid.UUID(row.OperationID.Bytes), Kind: row.Kind, State: state, FailureCode: row.ErrorCode}
	if row.Kind == "send" {
		result.MessageID = MessageID(row.UpstreamID)
	}
	return result
}

// Execute requires the caller's existing owner/session fence. A duplicate
// request only reads a receipt; it never takes over an abandoned write.
// The two local commits finish before the remote call starts.
func (ledger *OperationLedger) Execute(ctx context.Context, owner syntax.DID, id uuid.UUID, kind string, payload any, call func(context.Context) (OperationResult, error)) (OperationReceipt, error) {
	if owner == "" || id == uuid.Nil || (kind != "send" && kind != "report") || call == nil {
		return OperationReceipt{}, ErrOperationUnavailable
	}
	digest, err := operationDigest(ledger.keys[ledger.current], owner, id, kind, payload)
	if err != nil {
		return OperationReceipt{}, err
	}
	operationID := pgtype.UUID{Bytes: id, Valid: true}
	var row chatdb.ChatOperation
	var created bool
	err = ledger.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		q := chatdb.New(tx)
		connection, err := q.ChatConnection(ctx, owner.String())
		if err != nil {
			return err
		}
		if !connection.Enabled {
			return ErrDisabled
		}
		count, err := q.PrepareChatOperation(ctx, chatdb.PrepareChatOperationParams{OwnerDid: owner.String(), OperationID: operationID, Kind: kind, KeyVersion: int32(ledger.current), RequestDigest: digest, MessagingGeneration: connection.Generation})
		if err != nil {
			return err
		}
		created = count == 1
		if !created {
			if _, err := q.ExpirePreparedChatOperation(ctx, chatdb.ExpirePreparedChatOperationParams{OwnerDid: owner.String(), OperationID: operationID}); err != nil {
				return err
			}
		}
		row, err = q.GetChatOperation(ctx, chatdb.GetChatOperationParams{OwnerDid: owner.String(), OperationID: operationID})
		if err != nil {
			return err
		}
		if row.Kind != kind {
			return ErrOperationConflict
		}
		if key, ok := ledger.keys[int(row.KeyVersion)]; ok {
			want, err := operationDigest(key, owner, id, kind, payload)
			if err != nil {
				return err
			}
			if !hmac.Equal(want, row.RequestDigest) {
				return ErrOperationConflict
			}
		}
		return nil
	})
	if err != nil {
		return OperationReceipt{}, err
	}
	if !created {
		return receiptFor(row), nil
	}
	err = ledger.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		count, err := chatdb.New(tx).AttemptChatOperation(ctx, chatdb.AttemptChatOperationParams{OwnerDid: owner.String(), OperationID: operationID})
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrOperationUnavailable
		}
		return nil
	})
	if err != nil {
		return receiptFor(row), err
	}
	result, callErr := call(ctx)
	state, code := "confirmed", ""
	if callErr != nil || result.ID == "" || len(result.ID) > 512 || len(result.Revision) > 512 {
		state, code = "uncertain", "uncertain"
		var apiError *atclient.APIError
		if errors.As(callErr, &apiError) && (apiError.StatusCode == 400 || apiError.StatusCode == 401 || apiError.StatusCode == 403 || apiError.StatusCode == 404 || apiError.StatusCode == 409 || apiError.StatusCode == 413 || apiError.StatusCode == 422 || apiError.StatusCode == 429) {
			state, code = "failed", "rejected"
		}
		result = OperationResult{}
	}
	row.State, row.UpstreamID, row.UpstreamRevision, row.ErrorCode = state, result.ID, result.Revision, code
	// Persist uncertainty even when the HTTP request was cancelled. The owner
	// fence remains held by the surrounding coordinator until this returns.
	finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), operationFinalizationTimeout)
	defer cancel()
	err = ledger.store.owners.WithAuthTransaction(finalCtx, func(tx pgx.Tx) error {
		count, err := chatdb.New(tx).FinishChatOperation(finalCtx, chatdb.FinishChatOperationParams{OwnerDid: owner.String(), OperationID: operationID, State: state, UpstreamID: result.ID, UpstreamRevision: result.Revision, ErrorCode: code})
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrOperationUnavailable
		}
		return nil
	})
	if err != nil {
		row.State = "uncertain"
		row.UpstreamID = ""
		row.ErrorCode = "uncertain"
	}
	return receiptFor(row), errors.Join(callErr, err)
}
