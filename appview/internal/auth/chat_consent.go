package auth

import (
	"context"
	"crypto/sha256"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/chatdb"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// StartChatAuthorization is an authenticated, same-owner opt-in. The bearer
// is hashed immediately and never persisted in an authorization request.
func (service *OAuthFlowService) StartChatAuthorization(ctx context.Context, info AuthInfo, bearer string, mode HandoffMode, loopback, device string, push *bool) (string, error) {
	ctx, cancel := oauthOperationContext(ctx, service.startOperationTimeout)
	defer cancel()
	if info.DID == "" || info.SessionID == "" || bearer == "" || device == "" ||
		(mode != HandoffVerifiedLink && mode != HandoffLoopback && mode != HandoffDevScheme) ||
		(mode == HandoffLoopback) != (loopback != "") {
		return "", ErrOAuthFlowInvalid
	}
	hash := sha256.Sum256([]byte(bearer))
	var redirect string
	err := service.owners.WithExistingAuth(ctx, info.DID, func(ctx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.State != ownerlifecycle.StateActive {
			return ErrOAuthFlowInvalid
		}
		metadata := AuthRequestMetadata{Purpose: ChatOAuthPurpose, Owner: info.DID, OwnerGeneration: authority.Generation,
			AuthEpoch: authority.AuthEpoch, HandoffMode: mode, LoopbackURI: loopback, DeviceID: device,
			ChatParentSessionID: info.SessionID, ChatChildTokenHash: hash[:], ChatPushEnabled: push}
		if err := service.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `INSERT INTO chat_accounts(owner_did) VALUES($1) ON CONFLICT DO NOTHING`, info.DID); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT generation FROM chat_accounts WHERE owner_did=$1`, info.DID).Scan(&metadata.ChatGeneration); err != nil {
				return err
			}
			return validateChatConsentTx(ctx, tx, metadata)
		}); err != nil {
			return err
		}
		identity, err := service.app.Dir.LookupDID(ctx, info.DID)
		if err != nil {
			return err
		}
		if identity == nil || identity.DID != info.DID || identity.PDSEndpoint() == "" {
			return ErrOAuthFlowInvalid
		}
		issuer, err := service.app.Resolver.ResolveAuthServerURL(ctx, identity.PDSEndpoint())
		if err != nil {
			return err
		}
		server, err := service.app.Resolver.ResolveAuthServerMetadata(ctx, issuer)
		if err != nil {
			return err
		}
		request, err := service.app.SendAuthRequest(ctx, server, advertisedConsumerScopes(service.app.Config.Scopes), info.DID.String())
		if err != nil {
			return err
		}
		request.AccountDID = &info.DID
		metadata.ResourceServerOrigin = identity.PDSEndpoint()
		metadata.AuthorizationServerIssuer = server.Issuer
		if err := service.store.SaveAuthRequestInfo(context.WithValue(ctx, authRequestMetadataContextKey{}, metadata), *request); err != nil {
			return err
		}
		redirect = strings.TrimSuffix(server.AuthorizationEndpoint, "?") + "?" + url.Values{"client_id": {service.app.Config.ClientID}, "request_uri": {request.RequestURI}}.Encode()
		return nil
	})
	return redirect, err
}

// Reuses the caller's exclusive owner fence and its transaction. There is no
// nested advisory lock or pool checkout on callback/confirmation paths.
func validateChatConsentTx(ctx context.Context, tx pgx.Tx, metadata AuthRequestMetadata) error {
	var valid bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM chat_accounts chat
		JOIN craftsky_sessions child ON child.account_did=chat.owner_did
		JOIN oauth_sessions parent ON parent.account_did=child.account_did AND parent.session_id=child.oauth_session_id
		JOIN owner_lifecycles owner ON owner.owner_did=chat.owner_did
		WHERE chat.owner_did=$1 AND chat.generation=$2
		AND child.token_hash=$3 AND child.oauth_session_id=$4 AND child.last_device_id=$5
		AND child.lifecycle_state='active' AND child.idle_expires_at>now() AND child.auth_epoch=$6
		AND parent.lifecycle_state='active' AND parent.absolute_expires_at>now()
		AND parent.owner_generation=$7 AND parent.auth_epoch=$6
		AND owner.state='active' AND owner.generation=$7 AND owner.auth_epoch=$6
	)`, metadata.Owner, metadata.ChatGeneration, metadata.ChatChildTokenHash,
		metadata.ChatParentSessionID, metadata.DeviceID, metadata.AuthEpoch, metadata.OwnerGeneration).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return ErrOAuthFlowInvalid
	}
	return nil
}

func confirmChatConsentTx(ctx context.Context, tx pgx.Tx, owner syntax.DID, state string) (bool, error) {
	var purpose OAuthPurpose
	if err := tx.QueryRow(ctx, `SELECT purpose FROM oauth_auth_requests WHERE state=$1 FOR UPDATE`, state).Scan(&purpose); err != nil {
		if err == pgx.ErrNoRows {
			return false, ErrHandoffInvalid
		}
		return false, err
	}
	if purpose != ChatOAuthPurpose {
		if purpose != LoginOAuthPurpose && purpose != RegistrationOAuthPurpose {
			return false, ErrHandoffInvalid
		}
		return false, nil
	}
	var metadata AuthRequestMetadata
	err := tx.QueryRow(ctx, `SELECT purpose,owner_generation,auth_epoch,device_id,
		chat_parent_session_id,chat_child_token_hash,chat_generation,chat_push_enabled
		FROM oauth_auth_requests WHERE state=$1 AND purpose='chat'`, state).Scan(
		&metadata.Purpose, &metadata.OwnerGeneration, &metadata.AuthEpoch, &metadata.DeviceID,
		&metadata.ChatParentSessionID, &metadata.ChatChildTokenHash, &metadata.ChatGeneration, &metadata.ChatPushEnabled)
	if err == pgx.ErrNoRows {
		return false, ErrHandoffInvalid
	}
	if err != nil {
		return false, err
	}
	metadata.Owner = owner
	if err := validateChatConsentTx(ctx, tx, metadata); err != nil {
		return true, err
	}
	command, err := tx.Exec(ctx, `UPDATE chat_accounts SET enabled=true,generation=generation+1,
		push_enabled=COALESCE($3,push_enabled) WHERE owner_did=$1 AND generation=$2`, owner, metadata.ChatGeneration, metadata.ChatPushEnabled)
	if err != nil {
		return true, err
	}
	if command.RowsAffected() != 1 {
		return true, ErrOAuthFlowInvalid
	}
	return true, chatdb.New(tx).ResumeChatSync(ctx, owner.String())
}
