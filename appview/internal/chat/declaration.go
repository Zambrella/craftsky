package chat

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"social.craftsky/appview/internal/auth"
)

type IncomingPolicy struct {
	AllowIncoming string `json:"allowIncoming"`
	Persisted     bool   `json:"persisted"`
}

func (service *Service) IncomingPolicy(ctx context.Context, info auth.AuthInfo) (IncomingPolicy, error) {
	var result IncomingPolicy
	err := service.withRPC(ctx, info, nil, func(ctx context.Context, transport *Transport) error {
		current, err := transport.declaration(ctx, info.DID)
		if err != nil {
			return err
		}
		result, err = policyFromDeclaration(current)
		return err
	})
	return result, err
}

func incomingFromWire(value string) (string, error) {
	switch value {
	case "all":
		return "everyone", nil
	case "following":
		return "following", nil
	case "none":
		return "nobody", nil
	}
	return "", ErrProviderResponse
}
func incomingToWire(value string) (string, error) {
	switch value {
	case "everyone":
		return "all", nil
	case "following":
		return "following", nil
	case "nobody":
		return "none", nil
	}
	return "", ErrInvalidIdentifier
}

type declaration struct {
	CID   string                     `json:"cid"`
	Value map[string]json.RawMessage `json:"value"`
}

func (transport *Transport) declaration(ctx context.Context, owner syntax.DID) (declaration, error) {
	var result declaration
	err := transport.callFixed(ctx, http.MethodGet, "com.atproto.repo.getRecord", "", map[string]any{"repo": owner.String(), "collection": "chat.bsky.actor.declaration", "rkey": "self"}, nil, &result)
	var apiError *atclient.APIError
	if errors.As(err, &apiError) && apiError.Name == "RecordNotFound" {
		return declaration{Value: map[string]json.RawMessage{"$type": json.RawMessage(`"chat.bsky.actor.declaration"`), "allowIncoming": json.RawMessage(`"following"`)}}, nil
	}
	if err != nil {
		return result, err
	}
	if result.CID == "" || len(result.CID) > 512 || result.Value == nil {
		return result, ErrProviderResponse
	}
	var recordType string
	if err := json.Unmarshal(result.Value["$type"], &recordType); err != nil || recordType != "chat.bsky.actor.declaration" {
		return result, ErrProviderResponse
	}
	return result, nil
}
func policyFromDeclaration(record declaration) (IncomingPolicy, error) {
	var incoming string
	if err := json.Unmarshal(record.Value["allowIncoming"], &incoming); err != nil {
		return IncomingPolicy{}, errorsProviderCause(err)
	}
	value, err := incomingFromWire(incoming)
	return IncomingPolicy{AllowIncoming: value, Persisted: record.CID != ""}, err
}

func (service *Service) SetIncomingPolicy(ctx context.Context, info auth.AuthInfo, value string) (IncomingPolicy, error) {
	var result IncomingPolicy
	incoming, err := incomingToWire(value)
	if err != nil {
		return result, err
	}
	required := append(rpcScopes("chat.bsky.actor.getStatus"), "repo:chat.bsky.actor.declaration?action=create&action=update")
	err = service.withRequired(ctx, info, required, func(ctx context.Context, transport *Transport) error {
		for range 3 {
			current, err := transport.declaration(ctx, info.DID)
			if err != nil {
				return err
			}
			encoded, _ := json.Marshal(incoming)
			current.Value["allowIncoming"] = encoded
			var expected *string
			if current.CID != "" {
				expected = &current.CID
			}
			body := struct {
				Repo       syntax.DID                 `json:"repo"`
				Collection string                     `json:"collection"`
				Rkey       string                     `json:"rkey"`
				Record     map[string]json.RawMessage `json:"record"`
				SwapRecord *string                    `json:"swapRecord"`
			}{info.DID, "chat.bsky.actor.declaration", "self", current.Value, expected}
			var response struct {
				CID string `json:"cid"`
				URI string `json:"uri"`
			}
			err = transport.callFixed(ctx, http.MethodPost, "com.atproto.repo.putRecord", "", nil, body, &response)
			var apiError *atclient.APIError
			if errors.As(err, &apiError) && apiError.Name == "InvalidSwap" {
				continue
			}
			if err != nil {
				return err
			}
			if response.CID == "" || response.URI != "at://"+info.DID.String()+"/chat.bsky.actor.declaration/self" {
				return ErrProviderResponse
			}
			confirmed, err := transport.declaration(ctx, info.DID)
			if err != nil {
				return err
			}
			result, err = policyFromDeclaration(confirmed)
			if err != nil {
				return err
			}
			if result.AllowIncoming != value || !result.Persisted {
				return ErrProviderResponse
			}
			return nil
		}
		return ErrConnectionChanged
	})
	return result, err
}
