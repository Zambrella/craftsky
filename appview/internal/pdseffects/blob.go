package pdseffects

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type UploadBlobRequest struct {
	// OperationID and MutationKey have the same contract as PutRecordRequest.
	OperationID     string
	MutationKey     string
	Owner           syntax.DID
	OwnerGeneration int64
	ExpectedOwners  []ownerlifecycle.ExpectedOwner
	MIME            string
	Bytes           []byte
}

// PredictBlobCID returns the content-addressed raw CID expected from
// com.atproto.repo.uploadBlob and the exact SHA-256 of the uploaded bytes.
func PredictBlobCID(body []byte) (syntax.CID, [sha256.Size]byte, error) {
	if len(body) == 0 {
		return "", [sha256.Size]byte{}, errors.New("PDS blob body is empty")
	}
	digest := sha256.Sum256(body)
	multihashValue, err := multihash.Encode(digest[:], multihash.SHA2_256)
	if err != nil {
		return "", [sha256.Size]byte{}, fmt.Errorf("encode PDS blob multihash: %w", err)
	}
	return syntax.CID(cid.NewCidV1(cid.Raw, multihashValue).String()), digest, nil
}

func blobDeterministicKey(
	owner syntax.DID,
	contentCID syntax.CID,
	digest [sha256.Size]byte,
	mime string,
	size int,
) string {
	return fmt.Sprintf(
		"pds-blob://%s/%s?sha256=%s&mime=%s&size=%d",
		owner,
		contentCID,
		hex.EncodeToString(digest[:]),
		url.QueryEscape(mime),
		size,
	)
}

func (executor *Executor) UploadBlob(
	ctx context.Context,
	request UploadBlobRequest,
) (*auth.UploadedBlob, error) {
	if executor == nil || executor.attempts == nil || executor.boundary == nil {
		return nil, errors.New("durable PDS effect executor is unavailable")
	}
	if err := validateEffectOwner(
		executor.owner,
		request.OperationID,
		request.MutationKey,
		request.Owner,
		request.OwnerGeneration,
		request.ExpectedOwners,
	); err != nil {
		return nil, err
	}
	request.MIME = strings.TrimSpace(request.MIME)
	if request.MIME == "" || strings.ContainsAny(request.MIME, "\r\n") {
		return nil, errors.New("PDS blob MIME type is invalid")
	}
	predictedCID, digest, err := PredictBlobCID(request.Bytes)
	if err != nil {
		return nil, err
	}
	exactKey := blobDeterministicKey(
		request.Owner,
		predictedCID,
		digest,
		request.MIME,
		len(request.Bytes),
	)
	var uploaded *auth.UploadedBlob
	err = executor.boundary.WithActiveEffects(
		ctx,
		request.ExpectedOwners,
		func(effectCtx context.Context, client auth.PDSClient) error {
			attempt, err := executor.attempts.CreateEffectAttempt(
				effectCtx,
				ownerlifecycle.NewEffectAttempt{
					OperationID:        request.OperationID,
					MutationKey:        request.MutationKey,
					Owner:              request.Owner,
					OwnerGeneration:    request.OwnerGeneration,
					Kind:               ownerlifecycle.EffectObjectPut,
					Action:             ownerlifecycle.EffectActionUploadBlob,
					DeterministicKey:   exactKey,
					RequestFingerprint: digest,
					ExpectedCID:        predictedCID.String(),
					RemoteDeadline:     executor.now().UTC().Add(executor.timeout),
				},
			)
			if errors.Is(err, ownerlifecycle.ErrAttemptConflict) {
				return &ConflictError{
					OperationID: request.OperationID,
					ExactKey:    exactKey,
					Cause:       err,
				}
			}
			if err != nil {
				return err
			}
			switch attempt.Outcome {
			case ownerlifecycle.OutcomeAccepted, ownerlifecycle.OutcomeReconciledAccepted:
				if attempt.ResultCID == "" {
					return &OutcomeAmbiguousError{
						OperationID: request.OperationID,
						ExactKey:    exactKey,
						Cause:       errors.New("accepted PDS blob has no authoritative CID"),
					}
				}
				uploaded = reconstructedBlob(
					syntax.CID(attempt.ResultCID),
					request.MIME,
					int64(len(request.Bytes)),
				)
				return nil
			case ownerlifecycle.OutcomePrepared:
				// Continue to the one permitted upload below.
			case ownerlifecycle.OutcomeDispatched, ownerlifecycle.OutcomeUnknownPreTransition:
				return &OutcomeAmbiguousError{
					OperationID: request.OperationID,
					ExactKey:    exactKey,
				}
			default:
				return ErrEffectRejected
			}

			attempt, err = executor.attempts.MarkAttemptDispatched(
				effectCtx,
				attempt.OperationID,
				request.Owner,
				request.OwnerGeneration,
			)
			if err != nil {
				return err
			}
			callCtx, cancel := context.WithTimeout(effectCtx, executor.timeout)
			uploaded, err = client.UploadBlob(callCtx, request.MIME, request.Bytes)
			cancel()
			if err != nil {
				return &OutcomeAmbiguousError{
					OperationID: request.OperationID,
					ExactKey:    exactKey,
					Cause:       err,
				}
			}
			if uploaded == nil || uploaded.Raw == nil || uploaded.CID != predictedCID.String() ||
				uploaded.MIME != request.MIME || uploaded.Size != int64(len(request.Bytes)) {
				return &ConflictError{
					OperationID: request.OperationID,
					ExactKey:    exactKey,
					Cause:       errors.New("PDS returned blob metadata that differs from content identity"),
				}
			}
			if _, completeErr := executor.attempts.CompleteEffectAttempt(
				effectCtx,
				attempt.OperationID,
				request.Owner,
				request.OwnerGeneration,
				ownerlifecycle.OutcomeAccepted,
				uploaded.CID,
			); completeErr != nil {
				return completeErr
			}
			return nil
		},
	)
	return uploaded, err
}

func reconstructedBlob(contentCID syntax.CID, mime string, size int64) *auth.UploadedBlob {
	if contentCID == "" {
		return nil
	}
	return &auth.UploadedBlob{
		CID:  contentCID.String(),
		MIME: mime,
		Size: size,
		Raw: map[string]any{
			"$type":    "blob",
			"ref":      map[string]any{"$link": contentCID.String()},
			"mimeType": mime,
			"size":     size,
		},
	}
}
