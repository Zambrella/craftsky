package app

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/push"
)

func newChatDependencies(pool *pgxpool.Pool, owners *ownerlifecycle.Store, sessions *auth.OAuthSessionCoordinator, federated *federatedClients, cfg Config) (*chat.Service, error) {
	store, err := chat.NewStore(pool, owners)
	if err != nil {
		return nil, err
	}
	profile := cfg.FederatedHTTP.PDSJSON
	profile.TotalTimeout = min(profile.TotalTimeout, 10*time.Second)
	profile.ResponseLimit = min(profile.ResponseLimit, 2<<20)
	profile.MaxRedirects = 0
	client, err := federated.boundary.Client(profile)
	if err != nil {
		return nil, err
	}
	var operations *chat.OperationLedger
	if cfg.ChatOperationHMACKey != "" {
		key, err := decodeChatOperationKey(cfg.ChatOperationHMACKey)
		if err != nil {
			return nil, err
		}
		operations, err = chat.NewOperationLedger(store, cfg.ChatOperationHMACKeyVersion, map[int][]byte{cfg.ChatOperationHMACKeyVersion: key})
		if err != nil {
			return nil, err
		}
	}
	return chat.NewService(chat.ServiceOptions{Store: store, Sessions: sessions, Operations: operations, Transport: func(session *oauth.ClientSession) (*chat.Transport, error) {
		api, err := auth.CapabilityAPIClient(session, client)
		if err != nil {
			return nil, err
		}
		return chat.NewTransport(api), nil
	}})
}

func newChatSynchronizer(pool *pgxpool.Pool, service *chat.Service, observer *observability.Observer) (*chat.Synchronizer, error) {
	if !service.OperationsAvailable() {
		return nil, nil
	}
	return chat.NewSynchronizer(service, auth.NewBackgroundSessionSelector(pool), chat.SyncOptions{Observer: observer})
}

func newChatDeliveryDispatcher(pool *pgxpool.Pool, service *chat.Service, sender push.Sender, observer *observability.Observer) (*chat.DeliveryDispatcher, error) {
	if !service.OperationsAvailable() {
		return nil, nil
	}
	return chat.NewDeliveryDispatcher(service, auth.NewBackgroundSessionSelector(pool), sender, chat.DeliveryOptions{Observer: observer})
}
