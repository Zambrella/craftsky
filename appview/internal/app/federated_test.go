package app

import (
	"context"
	"errors"
	"testing"

	"social.craftsky/appview/internal/federatedhttp"
)

func TestNewFederatedClientsBuildsDistinctPurposeClients(t *testing.T) {
	t.Parallel()

	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		t.Fatalf("federatedHTTPConfigFromEnv: %v", err)
	}
	clients, err := newFederatedClients(config)
	if err != nil {
		t.Fatalf("newFederatedClients: %v", err)
	}
	defer clients.boundary.CloseIdleConnections()
	if clients.metadata == nil || clients.oauth == nil || clients.pdsJSON == nil ||
		clients.pdsBlob == nil || clients.directory == nil {
		t.Fatal("federated client set is incomplete")
	}
	if clients.metadata == clients.oauth || clients.oauth == clients.pdsJSON ||
		clients.pdsJSON == clients.pdsBlob {
		t.Fatal("purpose clients unexpectedly share an http.Client wrapper")
	}
}

func TestFederatedClientsRejectStoredPrivateSessionDestination(t *testing.T) {
	t.Parallel()

	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		t.Fatalf("federatedHTTPConfigFromEnv: %v", err)
	}
	clients, err := newFederatedClients(config)
	if err != nil {
		t.Fatalf("newFederatedClients: %v", err)
	}
	defer clients.boundary.CloseIdleConnections()
	err = clients.validateSessionDestinations(
		context.Background(),
		"https://127.0.0.1",
		"https://auth.example",
		"https://auth.example/token",
		"",
	)
	if !errors.Is(err, federatedhttp.ErrDestinationRejected) {
		t.Fatalf("error = %v, want destination rejection", err)
	}
}
