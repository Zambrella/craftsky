package app

import (
	"net/http"

	"github.com/bluesky-social/indigo/atproto/identity"

	"social.craftsky/appview/internal/ingestion"
)

func newRepositorySnapshotFetcher(
	directory identity.Directory,
	client *http.Client,
) (*ingestion.RepositorySnapshotFetcher, error) {
	return ingestion.NewRepositorySnapshotFetcher(directory, client)
}
