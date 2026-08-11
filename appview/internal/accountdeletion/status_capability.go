package accountdeletion

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type StatusCapabilitySigner struct {
	key    []byte
	now    func() time.Time
	random io.Reader
}

type statusCapabilityClaims struct {
	Version   int        `json:"v"`
	JobID     uuid.UUID  `json:"j"`
	Owner     syntax.DID `json:"o"`
	ExpiresAt int64      `json:"e"`
	Nonce     string     `json:"n"`
}

type GeneratedStatusCapability struct {
	Token string
	Hash  []byte
}

func NewStatusCapabilitySigner(key []byte, now func() time.Time, random io.Reader) (*StatusCapabilitySigner, error) {
	if len(key) < 32 || random == nil {
		return nil, errors.New("status capability signer requires a 32-byte key and randomness")
	}
	if now == nil {
		now = time.Now
	}
	return &StatusCapabilitySigner{key: append([]byte(nil), key...), now: now, random: random}, nil
}

func (signer *StatusCapabilitySigner) Generate(jobID uuid.UUID, owner syntax.DID, expiresAt time.Time) (GeneratedStatusCapability, error) {
	if signer == nil || jobID == uuid.Nil || owner == "" || !signer.now().Before(expiresAt) {
		return GeneratedStatusCapability{}, ErrStatusUnauthorized
	}
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(signer.random, nonce); err != nil {
		return GeneratedStatusCapability{}, err
	}
	payload, err := json.Marshal(statusCapabilityClaims{
		Version: 1, JobID: jobID, Owner: owner, ExpiresAt: expiresAt.Unix(),
		Nonce: base64.RawURLEncoding.EncodeToString(nonce),
	})
	if err != nil {
		return GeneratedStatusCapability{}, err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signer.signature(encodedPayload)
	token := encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature)
	return GeneratedStatusCapability{Token: token, Hash: HashSecret(token)}, nil
}

func (signer *StatusCapabilitySigner) Verify(token string) (StatusGrant, error) {
	parts := strings.Split(token, ".")
	if signer == nil || len(parts) != 2 {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, signer.signature(parts[0])) {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	var claims statusCapabilityClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Version != 1 || claims.JobID == uuid.Nil || claims.Owner == "" || claims.Nonce == "" {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	expiresAt := time.Unix(claims.ExpiresAt, 0).UTC()
	if !signer.now().Before(expiresAt) {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	return StatusGrant{JobID: claims.JobID.String(), Owner: claims.Owner, ExpiresAt: expiresAt}, nil
}

func (signer *StatusCapabilitySigner) signature(payload string) []byte {
	mac := hmac.New(sha256.New, signer.key)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (store *Store) AuthorizeStatusCapability(
	ctx context.Context,
	signer *StatusCapabilitySigner,
	token string,
	jobID uuid.UUID,
	owner syntax.DID,
	deviceID string,
	action StatusAction,
) (StatusGrant, error) {
	grant, err := signer.Verify(token)
	if err != nil || grant.JobID != jobID.String() || grant.Owner != owner ||
		AuthorizeStatus(grant, store.now().UTC(), jobID.String(), owner, action) != nil {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	var exists bool
	if err := store.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM account_deletion_status_credentials
			WHERE token_hash=$1 AND job_id=$2 AND owner_did=$3 AND device_id=$4
			  AND revoked_at IS NULL AND expires_at>$5
		)
	`, HashSecret(token), jobID, owner, deviceID, store.now().UTC()).Scan(&exists); err != nil {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	if !exists && action == StatusRead {
		if err := store.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM account_deletion_audits
				WHERE job_id=$1 AND did=$2 AND expires_at>$3
			)
		`, jobID, owner, store.now().UTC()).Scan(&exists); err != nil {
			return StatusGrant{}, ErrStatusUnauthorized
		}
	}
	if !exists {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	return grant, nil
}
