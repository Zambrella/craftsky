package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrHandoffInvalid = errors.New("OAuth handoff invalid")

type HandoffServiceOptions struct {
	Pool              *pgxpool.Pool
	Owners            *ownerlifecycle.Store
	Sessions          *CraftskySessionStore
	ExchangeTTL       time.Duration
	ConfirmationTTL   time.Duration
	ReceiptKey        []byte
	ReceiptKeyVersion int
	Random            io.Reader
	Now               func() time.Time
}

type HandoffService struct {
	pool              *pgxpool.Pool
	owners            *ownerlifecycle.Store
	sessions          *CraftskySessionStore
	exchangeTTL       time.Duration
	confirmationTTL   time.Duration
	receiptAEAD       cipher.AEAD
	receiptKeyVersion int
	random            io.Reader
	now               func() time.Time
}

type HandoffExchangeResult struct {
	Token     string
	DID       syntax.DID
	Handle    syntax.Handle
	ReceiptID uuid.UUID
	ConfirmBy time.Time
}

func NewHandoffService(options HandoffServiceOptions) (*HandoffService, error) {
	if options.Pool == nil || options.Owners == nil || options.Sessions == nil {
		return nil, errors.New("handoff service requires database, owner lifecycle, and child sessions")
	}
	if options.ExchangeTTL <= 0 || options.ConfirmationTTL <= 0 || options.ConfirmationTTL >= options.ExchangeTTL {
		return nil, errors.New("handoff confirmation TTL must be positive and shorter than exchange TTL")
	}
	if len(options.ReceiptKey) != 32 || options.ReceiptKeyVersion <= 0 {
		return nil, errors.New("handoff receipt key must be a versioned 256-bit key")
	}
	block, err := aes.NewCipher(options.ReceiptKey)
	if err != nil {
		return nil, fmt.Errorf("construct handoff receipt cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("construct handoff receipt AEAD: %w", err)
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &HandoffService{
		pool: options.Pool, owners: options.Owners, sessions: options.Sessions,
		exchangeTTL: options.ExchangeTTL, confirmationTTL: options.ConfirmationTTL,
		receiptAEAD: aead, receiptKeyVersion: options.ReceiptKeyVersion,
		random: options.Random, now: options.Now,
	}, nil
}

func (service *HandoffService) CreateExchange(
	ctx context.Context,
	attempt CallbackAttempt,
	handle syntax.Handle,
	deviceID string,
) (string, error) {
	boundAttempt, ok := callbackAttemptFromContext(ctx)
	if !ok || boundAttempt != attempt || attempt.Purpose != LoginOAuthPurpose ||
		!attempt.validFor(attempt.Owner, attempt.State) || handle == "" || deviceID == "" {
		return "", ErrHandoffInvalid
	}
	codeBytes := make([]byte, 32)
	if _, err := io.ReadFull(service.random, codeBytes); err != nil {
		return "", fmt.Errorf("generate handoff code: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(codeBytes)
	codeHash := sha256.Sum256([]byte(code))
	exchangeID := uuid.New()
	now := service.now().UTC()
	err := service.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		var ownerState string
		var ownerGeneration, ownerEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,generation,auth_epoch
			FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
		`, attempt.Owner).Scan(&ownerState, &ownerGeneration, &ownerEpoch); err != nil {
			return err
		}
		if ownerState == "terminal" || ownerEpoch != attempt.AuthEpoch || ownerGeneration != attempt.OwnerGeneration {
			return ErrHandoffInvalid
		}
		var purpose OAuthPurpose
		var requestState AuthRequestState
		var requestAttempt uuid.UUID
		var storedDevice string
		if err := tx.QueryRow(ctx, `
			SELECT purpose,request_state,exchange_attempt_id,device_id
			FROM oauth_auth_requests WHERE state=$1 FOR UPDATE
		`, attempt.State).Scan(&purpose, &requestState, &requestAttempt, &storedDevice); err != nil {
			return err
		}
		if purpose != LoginOAuthPurpose || requestState != AuthRequestExchangeStarted ||
			requestAttempt != attempt.AttemptID || storedDevice != deviceID {
			return ErrHandoffInvalid
		}
		var parentState string
		var parentEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT lifecycle_state,auth_epoch
			FROM oauth_sessions WHERE account_did=$1 AND session_id=$2 FOR UPDATE
		`, attempt.Owner, attempt.State).Scan(&parentState, &parentEpoch); err != nil {
			return err
		}
		if parentState != "pending_handoff" || parentEpoch != ownerEpoch {
			return ErrHandoffInvalid
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO oauth_handoff_exchanges(
				id,code_hash,owner_did,owner_generation,auth_epoch,oauth_session_id,
				device_id,canonical_handle,state,expires_at,created_at,updated_at
			) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'ready',$9,$10,$10)
		`, exchangeID, codeHash[:], attempt.Owner, ownerGeneration, ownerEpoch,
			attempt.State, deviceID, handle, now.Add(service.exchangeTTL), now); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='consumed',exchange_finished_at=$3,consumed_at=COALESCE(consumed_at,$3)
			WHERE state=$1 AND exchange_attempt_id=$2 AND request_state='exchange_started'
		`, attempt.State, attempt.AttemptID, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrHandoffInvalid
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

type handoffDiscovery struct {
	ExchangeID     uuid.UUID
	Owner          syntax.DID
	OAuthSessionID string
	AuthEpoch      int64
	DeviceID       string
	Handle         syntax.Handle
	ReceiptID      *uuid.UUID
	ChildTokenHash []byte
}

func (service *HandoffService) Exchange(ctx context.Context, code, deviceID string) (HandoffExchangeResult, error) {
	if code == "" || deviceID == "" {
		return HandoffExchangeResult{}, ErrHandoffInvalid
	}
	hash := sha256.Sum256([]byte(code))
	var discovery handoffDiscovery
	err := service.pool.QueryRow(ctx, `
		SELECT exchange.id,exchange.owner_did,exchange.oauth_session_id,exchange.auth_epoch,
		       exchange.device_id,exchange.canonical_handle,receipt.id,receipt.child_token_hash
		FROM oauth_handoff_exchanges exchange
		LEFT JOIN oauth_handoff_receipts receipt ON receipt.exchange_id=exchange.id
		WHERE exchange.code_hash=$1 AND exchange.state IN ('ready','redeemed')
	`, hash[:]).Scan(
		&discovery.ExchangeID, &discovery.Owner, &discovery.OAuthSessionID,
		&discovery.AuthEpoch, &discovery.DeviceID, &discovery.Handle,
		&discovery.ReceiptID, &discovery.ChildTokenHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return HandoffExchangeResult{}, ErrHandoffInvalid
	}
	if err != nil {
		return HandoffExchangeResult{}, fmt.Errorf("discover handoff exchange: %w", err)
	}
	var result HandoffExchangeResult
	err = service.owners.WithExistingAuth(ctx, discovery.Owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		var outcomeErr error
		err := service.owners.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
			if authority.AuthEpoch != discovery.AuthEpoch || authority.State == ownerlifecycle.StateTerminal {
				return ErrHandoffInvalid
			}
			var parentState string
			var parentEpoch int64
			if err := tx.QueryRow(authCtx, `
				SELECT lifecycle_state,auth_epoch FROM oauth_sessions
				WHERE account_did=$1 AND session_id=$2 FOR UPDATE
			`, discovery.Owner, discovery.OAuthSessionID).Scan(&parentState, &parentEpoch); err != nil {
				return err
			}
			if parentState != "pending_handoff" || parentEpoch != authority.AuthEpoch {
				return ErrHandoffInvalid
			}
			if len(discovery.ChildTokenHash) > 0 {
				var locked int
				if err := tx.QueryRow(authCtx, `
					SELECT 1 FROM craftsky_sessions WHERE token_hash=$1 FOR UPDATE
				`, discovery.ChildTokenHash).Scan(&locked); err != nil {
					return err
				}
			}
			var state string
			var expiresAt time.Time
			var storedDevice string
			var handle syntax.Handle
			if err := tx.QueryRow(authCtx, `
				SELECT state,expires_at,device_id,canonical_handle
				FROM oauth_handoff_exchanges WHERE id=$1 AND code_hash=$2 FOR UPDATE
			`, discovery.ExchangeID, hash[:]).Scan(&state, &expiresAt, &storedDevice, &handle); err != nil {
				return err
			}
			now := service.now().UTC()
			if storedDevice != deviceID || !now.Before(expiresAt) {
				outcomeErr = ErrHandoffInvalid
				return nil
			}
			switch state {
			case "ready":
				token, childHash, err := service.sessions.createPendingTx(
					authCtx, tx, discovery.Owner, discovery.OAuthSessionID, deviceID,
				)
				if err != nil {
					return err
				}
				receiptID := uuid.New()
				confirmBy := now.Add(service.confirmationTTL)
				nonce := make([]byte, service.receiptAEAD.NonceSize())
				if _, err := io.ReadFull(service.random, nonce); err != nil {
					return fmt.Errorf("generate handoff receipt nonce: %w", err)
				}
				ciphertext := service.receiptAEAD.Seal(nil, nonce, []byte(token), receiptAAD(
					discovery.ExchangeID, receiptID, discovery.Owner, discovery.OAuthSessionID, deviceID, childHash,
				))
				if _, err := tx.Exec(authCtx, `
					UPDATE oauth_handoff_exchanges SET state='redeemed',updated_at=$2
					WHERE id=$1 AND state='ready'
				`, discovery.ExchangeID, now); err != nil {
					return err
				}
				if _, err := tx.Exec(authCtx, `
					INSERT INTO oauth_handoff_receipts(
						id,exchange_id,child_token_hash,ciphertext,nonce,key_version,
						state,confirm_by,created_at,updated_at
					) VALUES($1,$2,$3,$4,$5,$6,'pending',$7,$8,$8)
				`, receiptID, discovery.ExchangeID, childHash, ciphertext, nonce,
					service.receiptKeyVersion, confirmBy, now); err != nil {
					return err
				}
				result = HandoffExchangeResult{
					Token: token, DID: discovery.Owner, Handle: handle,
					ReceiptID: receiptID, ConfirmBy: confirmBy,
				}
				return nil
			case "redeemed":
				var receiptID uuid.UUID
				var childHash, ciphertext, nonce []byte
				var keyVersion int
				var confirmBy time.Time
				var receiptState string
				if err := tx.QueryRow(authCtx, `
					SELECT id,child_token_hash,ciphertext,nonce,key_version,confirm_by,state
					FROM oauth_handoff_receipts WHERE exchange_id=$1 FOR UPDATE
				`, discovery.ExchangeID).Scan(
					&receiptID, &childHash, &ciphertext, &nonce, &keyVersion, &confirmBy, &receiptState,
				); err != nil {
					return err
				}
				if receiptState != "pending" || keyVersion != service.receiptKeyVersion || !now.Before(confirmBy) {
					return ErrHandoffInvalid
				}
				plaintext, err := service.receiptAEAD.Open(nil, nonce, ciphertext, receiptAAD(
					discovery.ExchangeID, receiptID, discovery.Owner, discovery.OAuthSessionID, deviceID, childHash,
				))
				if err != nil {
					return ErrHandoffInvalid
				}
				result = HandoffExchangeResult{
					Token: string(plaintext), DID: discovery.Owner, Handle: handle,
					ReceiptID: receiptID, ConfirmBy: confirmBy.UTC(),
				}
				return nil
			default:
				return ErrHandoffInvalid
			}
		})
		if err != nil {
			return err
		}
		return outcomeErr
	})
	if err != nil {
		return HandoffExchangeResult{}, collapseHandoffError(err)
	}
	return result, nil
}

func (service *HandoffService) Confirm(
	ctx context.Context,
	token string,
	receiptID uuid.UUID,
	deviceID string,
) error {
	if token == "" || receiptID == uuid.Nil || deviceID == "" {
		return ErrHandoffInvalid
	}
	tokenHash := sha256.Sum256([]byte(token))
	var discovery handoffDiscovery
	err := service.pool.QueryRow(ctx, `
		SELECT exchange.id,exchange.owner_did,exchange.oauth_session_id,exchange.auth_epoch,
		       exchange.device_id,exchange.canonical_handle,receipt.id,receipt.child_token_hash
		FROM oauth_handoff_receipts receipt
		JOIN oauth_handoff_exchanges exchange ON exchange.id=receipt.exchange_id
		WHERE receipt.id=$1 AND receipt.child_token_hash=$2
	`, receiptID, tokenHash[:]).Scan(
		&discovery.ExchangeID, &discovery.Owner, &discovery.OAuthSessionID,
		&discovery.AuthEpoch, &discovery.DeviceID, &discovery.Handle,
		&discovery.ReceiptID, &discovery.ChildTokenHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrHandoffInvalid
	}
	if err != nil {
		return fmt.Errorf("discover handoff confirmation: %w", err)
	}
	err = service.owners.WithExistingAuth(ctx, discovery.Owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		return service.owners.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
			if authority.AuthEpoch != discovery.AuthEpoch || authority.State == ownerlifecycle.StateTerminal {
				return ErrHandoffInvalid
			}
			var parentState string
			var parentEpoch int64
			if err := tx.QueryRow(authCtx, `
				SELECT lifecycle_state,auth_epoch FROM oauth_sessions
				WHERE account_did=$1 AND session_id=$2 FOR UPDATE
			`, discovery.Owner, discovery.OAuthSessionID).Scan(&parentState, &parentEpoch); err != nil {
				return err
			}
			var childState string
			var childEpoch int64
			if err := tx.QueryRow(authCtx, `
				SELECT lifecycle_state,auth_epoch FROM craftsky_sessions
				WHERE token_hash=$1 FOR UPDATE
			`, tokenHash[:]).Scan(&childState, &childEpoch); err != nil {
				return err
			}
			var exchangeState, storedDevice string
			if err := tx.QueryRow(authCtx, `
				SELECT state,device_id FROM oauth_handoff_exchanges WHERE id=$1 FOR UPDATE
			`, discovery.ExchangeID).Scan(&exchangeState, &storedDevice); err != nil {
				return err
			}
			var receiptState string
			var confirmBy time.Time
			if err := tx.QueryRow(authCtx, `
				SELECT state,confirm_by FROM oauth_handoff_receipts WHERE id=$1 FOR UPDATE
			`, receiptID).Scan(&receiptState, &confirmBy); err != nil {
				return err
			}
			if storedDevice != deviceID || parentEpoch != authority.AuthEpoch || childEpoch != authority.AuthEpoch {
				return ErrHandoffInvalid
			}
			if receiptState == "confirmed" && exchangeState == "confirmed" &&
				parentState == "active" && childState == "active" {
				return nil
			}
			now := service.now().UTC()
			if receiptState != "pending" || exchangeState != "redeemed" ||
				parentState != "pending_handoff" || childState != "pending_confirmation" || !now.Before(confirmBy) {
				return ErrHandoffInvalid
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_sessions SET lifecycle_state='active',row_version=row_version+1,updated_at=$3
				WHERE account_did=$1 AND session_id=$2 AND lifecycle_state='pending_handoff'
			`, discovery.Owner, discovery.OAuthSessionID, now); err != nil {
				return err
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE craftsky_sessions SET lifecycle_state='active'
				WHERE token_hash=$1 AND lifecycle_state='pending_confirmation'
			`, tokenHash[:]); err != nil {
				return err
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_handoff_exchanges
				SET state='confirmed',code_hash=NULL,consumed_at=$2,updated_at=$2
				WHERE id=$1 AND state='redeemed'
			`, discovery.ExchangeID, now); err != nil {
				return err
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_handoff_receipts
				SET state='confirmed',ciphertext=NULL,nonce=NULL,consumed_at=$2,updated_at=$2
				WHERE id=$1 AND state='pending'
			`, receiptID, now); err != nil {
				return err
			}
			return nil
		})
	})
	return collapseHandoffError(err)
}

func receiptAAD(
	exchangeID uuid.UUID,
	receiptID uuid.UUID,
	owner syntax.DID,
	oauthSessionID string,
	deviceID string,
	childHash []byte,
) []byte {
	return []byte(fmt.Sprintf(
		"social.craftsky.oauth-handoff-receipt.v1\x00%s\x00%s\x00%s\x00%s\x00%s\x00%x",
		exchangeID, receiptID, owner, oauthSessionID, deviceID, childHash,
	))
}

func collapseHandoffError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrHandoffInvalid) || errors.Is(err, ownerlifecycle.ErrTerminalOwner) ||
		errors.Is(err, pgx.ErrNoRows) {
		return ErrHandoffInvalid
	}
	return err
}
