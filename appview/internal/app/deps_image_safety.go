package app

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/safetyintake"
)

type imageSafetyDependencies struct {
	worker    *imagesafety.Worker
	store     *imagesafety.WorkerStore
	incidents *safetyincident.Store
	intake    *safetyintake.Store
}

func imageSafetyProjectionKey(config imagesafety.Config) imagesafety.ScanKey {
	key := imagesafety.ScanKey{
		ScannerID:     config.ScannerID,
		PolicyVersion: config.PolicyVersion,
		CorpusVersion: config.CorpusVersion,
	}
	if key.ScannerID != "" && key.PolicyVersion != "" && key.CorpusVersion != "" {
		return key
	}
	return imagesafety.ScanKey{
		ScannerID:     "unconfigured",
		PolicyVersion: "unconfigured",
		CorpusVersion: "unconfigured",
	}
}

func newImageSafetyDependencies(
	pool *pgxpool.Pool,
	federated *federatedClients,
	cfg Config,
) (*imageSafetyDependencies, error) {
	store := imagesafety.NewWorkerStore(pool, time.Now)
	incidents := safetyincident.NewStore(pool, time.Now)
	result := &imageSafetyDependencies{
		store: store, incidents: incidents, intake: safetyintake.NewStore(pool, time.Now),
	}
	if !cfg.ImageSafety.Ready() {
		return result, nil
	}
	if cfg.ImageSafety.Mode != imagesafety.ScannerModeStub {
		return nil, fmt.Errorf("approved image scanner adapter is not implemented")
	}
	fetcher, err := imagesafety.NewPDSBlobFetcher(
		federated.authoritativeDirectory,
		federated.pdsRepository,
		federated.boundary,
		cfg.MaxImageUploadBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("image safety PDS fetcher: %w", err)
	}
	worker, err := imagesafety.NewWorker(imagesafety.WorkerOptions{
		Store: store, Fetcher: fetcher, Scanner: imagesafety.DevelopmentScanner{}, MatchRecorder: incidents,
		Config: cfg.ImageSafety,
		RetryPolicy: imagesafety.RetryPolicy{
			MaxAttempts:    cfg.ImageSafetyMaxAttempts,
			InitialBackoff: cfg.ImageSafetyBackoffMin,
			MaxBackoff:     cfg.ImageSafetyBackoffMax,
		},
		WorkerID: "appview-image-safety", LeaseDuration: cfg.ImageSafetyLeaseDuration,
		OperationLimit: cfg.ImageSafetyOperationTimeout, PollInterval: cfg.ImageSafetyPollInterval,
		Now: time.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("image safety worker: %w", err)
	}
	result.worker = worker
	return result, nil
}
