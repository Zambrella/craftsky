package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/tap"
)

const maxWorkerBatchSize = 1000

type ProjectionWorkerConfig struct {
	Store         *Store
	Projector     Projector
	WorkerID      string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	BatchSize     int
	BackoffMin    time.Duration
	BackoffMax    time.Duration
	Logger        *slog.Logger
}

type ProjectionWorker struct {
	config ProjectionWorkerConfig
	logger *slog.Logger
}

func NewProjectionWorker(config ProjectionWorkerConfig) (*ProjectionWorker, error) {
	if err := validateWorkerConfig(config.Store, config.WorkerID, config.PollInterval,
		config.LeaseDuration, config.BatchSize, config.BackoffMin, config.BackoffMax); err != nil {
		return nil, fmt.Errorf("projection worker: %w", err)
	}
	if config.Projector == nil {
		return nil, errors.New("projection worker requires a projector")
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &ProjectionWorker{config: config, logger: logger}, nil
}

func (worker *ProjectionWorker) Run(ctx context.Context) error {
	return runWorkerLoop(ctx, worker.config.PollInterval, func(ctx context.Context) error {
		_, err := worker.RunOnce(ctx)
		if err != nil {
			worker.logger.Error("Tap projection batch failed",
				slog.String("component", "tap_projection"),
				slog.String("error_category", "batch"))
		}
		return nil
	})
}

func (worker *ProjectionWorker) RunOnce(ctx context.Context) (int, error) {
	claims, err := worker.config.Store.ClaimProjectionJobs(ctx, ProjectionClaimRequest{
		Worker: worker.config.WorkerID, LeaseToken: uuid.New(),
		LeaseDuration: worker.config.LeaseDuration, Limit: worker.config.BatchSize,
	})
	if err != nil {
		return 0, err
	}
	var batchErr error
	for _, claim := range claims {
		delay := exponentialBackoff(claim.Attempts, worker.config.BackoffMin, worker.config.BackoffMax)
		if err := worker.config.Store.project(ctx, claim, worker.config.Projector, delay); err != nil {
			batchErr = errors.Join(batchErr, err)
		}
	}
	return len(claims), batchErr
}

type RepositoryWorkerConfig struct {
	Store         *Store
	Handler       RepositoryJobHandler
	WorkerID      string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	BatchSize     int
	BackoffMin    time.Duration
	BackoffMax    time.Duration
	Logger        *slog.Logger
}

type RepositoryWorker struct {
	config RepositoryWorkerConfig
	logger *slog.Logger
}

// QuarantineReplayWorkerConfig controls the bounded worker that consumes only
// operator-requested quarantine replays. The operation timeout must leave time
// inside the lease for the final compare-and-set update.
type QuarantineReplayWorkerConfig struct {
	Store            *Store
	Handler          QuarantineReplayHandler
	WorkerID         string
	PollInterval     time.Duration
	LeaseDuration    time.Duration
	OperationTimeout time.Duration
	BatchSize        int
	Logger           *slog.Logger
}

type QuarantineReplayWorker struct {
	config QuarantineReplayWorkerConfig
	logger *slog.Logger
}

func NewQuarantineReplayWorker(config QuarantineReplayWorkerConfig) (*QuarantineReplayWorker, error) {
	if config.Store == nil {
		return nil, errors.New("quarantine replay worker requires a store")
	}
	if config.Handler == nil {
		return nil, errors.New("quarantine replay worker requires a handler")
	}
	if strings.TrimSpace(config.WorkerID) == "" || len(config.WorkerID) > 256 {
		return nil, errors.New("quarantine replay worker ID must contain 1 to 256 characters")
	}
	if config.PollInterval <= 0 || config.LeaseDuration <= config.PollInterval {
		return nil, errors.New("quarantine replay lease duration must exceed the positive poll interval")
	}
	if config.OperationTimeout <= 0 || config.OperationTimeout >= config.LeaseDuration {
		return nil, errors.New("quarantine replay operation timeout must be positive and shorter than the lease")
	}
	if config.BatchSize <= 0 || config.BatchSize > maxWorkerBatchSize {
		return nil, fmt.Errorf("quarantine replay batch size must be between 1 and %d", maxWorkerBatchSize)
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &QuarantineReplayWorker{config: config, logger: logger}, nil
}

func (worker *QuarantineReplayWorker) Run(ctx context.Context) error {
	return runWorkerLoop(ctx, worker.config.PollInterval, func(ctx context.Context) error {
		_, err := worker.RunOnce(ctx)
		if err != nil {
			worker.logger.Error("Tap quarantine replay batch failed",
				slog.String("component", "tap_quarantine"),
				slog.String("error_category", "batch"))
		}
		return nil
	})
}

func (worker *QuarantineReplayWorker) RunOnce(ctx context.Context) (int, error) {
	claims, err := worker.config.Store.ClaimQuarantine(ctx, QuarantineClaimRequest{
		Worker: worker.config.WorkerID, LeaseToken: uuid.New(),
		LeaseDuration: worker.config.LeaseDuration, Limit: worker.config.BatchSize,
	})
	if err != nil {
		return 0, err
	}
	var batchErr error
	for _, claim := range claims {
		replayErr := worker.config.Store.ReplayQuarantine(ctx, claim, func(_ context.Context, envelope []byte) (tap.Outcome, error) {
			operationCtx, cancel := context.WithTimeout(ctx, worker.config.OperationTimeout)
			defer cancel()
			return worker.config.Handler(operationCtx, envelope)
		})
		if replayErr != nil {
			batchErr = errors.Join(batchErr, replayErr)
		}
	}
	return len(claims), batchErr
}

func NewRepositoryWorker(config RepositoryWorkerConfig) (*RepositoryWorker, error) {
	if err := validateWorkerConfig(config.Store, config.WorkerID, config.PollInterval,
		config.LeaseDuration, config.BatchSize, config.BackoffMin, config.BackoffMax); err != nil {
		return nil, fmt.Errorf("repository worker: %w", err)
	}
	if config.Handler == nil {
		return nil, errors.New("repository worker requires a handler")
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &RepositoryWorker{config: config, logger: logger}, nil
}

func (worker *RepositoryWorker) Run(ctx context.Context) error {
	return runWorkerLoop(ctx, worker.config.PollInterval, func(ctx context.Context) error {
		_, err := worker.RunOnce(ctx)
		if err != nil {
			worker.logger.Error("Tap repository batch failed",
				slog.String("component", "tap_repository"),
				slog.String("error_category", "batch"))
		}
		return nil
	})
}

func (worker *RepositoryWorker) RunOnce(ctx context.Context) (int, error) {
	claims, err := worker.config.Store.ClaimRepositoryJobs(ctx, RepositoryClaimRequest{
		Worker: worker.config.WorkerID, LeaseToken: uuid.New(),
		LeaseDuration: worker.config.LeaseDuration, Limit: worker.config.BatchSize,
	})
	if err != nil {
		return 0, err
	}
	var batchErr error
	for _, claim := range claims {
		delay := exponentialBackoff(claim.Attempts, worker.config.BackoffMin, worker.config.BackoffMax)
		if err := worker.config.Store.runRepositoryJob(ctx, claim, worker.config.Handler, delay); err != nil {
			batchErr = errors.Join(batchErr, err)
		}
	}
	return len(claims), batchErr
}

func validateWorkerConfig(store *Store, workerID string, poll, lease time.Duration, batch int, backoffMin, backoffMax time.Duration) error {
	if store == nil {
		return errors.New("store is required")
	}
	if strings.TrimSpace(workerID) == "" || len(workerID) > 256 {
		return errors.New("worker ID must contain 1 to 256 characters")
	}
	if poll <= 0 || lease <= poll {
		return errors.New("lease duration must exceed the positive poll interval")
	}
	if batch <= 0 || batch > maxWorkerBatchSize {
		return fmt.Errorf("batch size must be between 1 and %d", maxWorkerBatchSize)
	}
	if backoffMin <= 0 || backoffMax < backoffMin {
		return errors.New("backoff durations must be positive and ordered")
	}
	return nil
}

func exponentialBackoff(attempt int, minimum, maximum time.Duration) time.Duration {
	if attempt <= 1 {
		return minimum
	}
	delay := minimum
	for i := 1; i < attempt; i++ {
		if delay >= maximum/2 {
			return maximum
		}
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
}

func runWorkerLoop(ctx context.Context, poll time.Duration, runOnce func(context.Context) error) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			_ = runOnce(ctx)
			timer.Reset(poll)
		}
	}
}
