package chat

import (
	"context"

	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/chatdb"
)

// Profiles are public indexed membership, not a requirement for local login.
// Use the caller's existing owner fence; never hold a transaction over RPC I/O.
func (service *Service) profileDIDs(ctx context.Context, dids []string) (map[syntax.DID]bool, error) {
	profiles := make(map[syntax.DID]bool)
	err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := chatdb.New(tx).ChatProfileDIDs(ctx, dids)
		for _, did := range rows {
			profiles[syntax.DID(did)] = true
		}
		return err
	})
	return profiles, err
}

func (service *Service) requireProfiles(ctx context.Context, owner, peer syntax.DID) error {
	profiles, err := service.profileDIDs(ctx, []string{owner.String(), peer.String()})
	if err != nil {
		return err
	}
	if !profiles[owner] || !profiles[peer] {
		return ErrInaccessibleConversation
	}
	return nil
}

func (service *Service) profileConversations(ctx context.Context, owner syntax.DID, views []*upstream.ConvoDefs_ConvoView) ([]*upstream.ConvoDefs_ConvoView, error) {
	dids := []string{owner.String()}
	for _, view := range views {
		if peer, err := directPeer(owner, view); err == nil {
			dids = append(dids, peer.String())
		}
	}
	profiles, err := service.profileDIDs(ctx, dids)
	if err != nil {
		return nil, err
	}
	result := make([]*upstream.ConvoDefs_ConvoView, 0, len(views))
	if !profiles[owner] {
		return result, nil
	}
	for _, view := range views {
		if peer, err := directPeer(owner, view); err == nil && profiles[peer] {
			result = append(result, view)
		}
	}
	return result, nil
}
