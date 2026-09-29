package imagesafety

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
)

type BlobFetcher interface {
	Fetch(context.Context, BlobSource) ([]byte, error)
}

type WorkerOptions struct {
	Store          *WorkerStore
	Fetcher        BlobFetcher
	Scanner        Scanner
	MatchRecorder  MatchRecorder
	Config         Config
	RetryPolicy    RetryPolicy
	WorkerID       string
	LeaseDuration  time.Duration
	OperationLimit time.Duration
	PollInterval   time.Duration
	Now            func() time.Time
	Logger         *slog.Logger
}

type Worker struct {
	options WorkerOptions
}

func NewWorker(options WorkerOptions) (*Worker, error) {
	if options.Store == nil || options.Fetcher == nil || options.Scanner == nil {
		return nil, errors.New("image scan worker dependencies are incomplete")
	}
	if !options.Config.Ready() || options.WorkerID == "" || options.LeaseDuration <= 0 ||
		options.OperationLimit <= 0 || options.OperationLimit >= options.LeaseDuration ||
		options.PollInterval <= 0 || options.PollInterval >= options.LeaseDuration {
		return nil, errors.New("image scan worker configuration is unsafe")
	}
	if options.RetryPolicy.MaxAttempts <= 0 || options.RetryPolicy.InitialBackoff <= 0 ||
		options.RetryPolicy.MaxBackoff < options.RetryPolicy.InitialBackoff {
		return nil, errors.New("image scan retry policy is invalid")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	return &Worker{options: options}, nil
}

func (worker *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(worker.options.PollInterval)
	defer ticker.Stop()
	for {
		if _, err := worker.ProcessOne(ctx); err != nil && !errors.Is(err, context.Canceled) {
			worker.options.Logger.ErrorContext(ctx, "image scan attempt failed",
				slog.String("component", "image_safety"),
				slog.String("error_category", "worker"))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (worker *Worker) ProcessOne(ctx context.Context) (bool, error) {
	claim, found, err := worker.options.Store.Claim(ctx, worker.options.WorkerID, worker.options.LeaseDuration)
	if err != nil || !found {
		return found, err
	}
	attemptCtx, cancel := context.WithTimeout(ctx, worker.options.OperationLimit)
	defer cancel()

	body, err := worker.options.Fetcher.Fetch(attemptCtx, claim.Source)
	if err != nil {
		return true, worker.fail(ctx, claim, StateUnavailable, "fetch")
	}
	result, err := worker.options.Scanner.Scan(attemptCtx, ScanInput{
		BlobCID: claim.Source.BlobCID, MIMEType: claim.Source.DeclaredMIME,
		Size: int64(len(body)), Content: bytes.NewReader(body),
	})
	if err != nil || !result.State.Valid() || result.State == StatePending {
		return true, worker.fail(ctx, claim, StateError, "scanner")
	}
	if result.State == StateClear && !worker.options.Config.CanMarkClear() {
		return true, worker.fail(ctx, claim, StateError, "configuration")
	}
	if result.State == StateUnavailable || result.State == StateError {
		return true, worker.fail(ctx, claim, result.State, "scanner")
	}
	if result.State == StateMatch && (worker.options.MatchRecorder == nil ||
		strings.TrimSpace(result.ProviderReference) == "" || len(result.ProviderReference) > 512 ||
		strings.TrimSpace(result.IntegrityMetadataReference) == "" || len(result.IntegrityMetadataReference) > 512) {
		return true, worker.fail(ctx, claim, StateError, "incident")
	}

	_, err = worker.options.Store.CompleteResult(ctx, claim, result, worker.options.MatchRecorder)
	if err != nil && result.State == StateMatch && !errors.Is(err, ErrLeaseLost) {
		return true, worker.fail(ctx, claim, StateError, "incident")
	}
	return true, err
}

func (worker *Worker) fail(ctx context.Context, claim Claim, state State, category string) error {
	delay, retry := worker.options.RetryPolicy.Next(claim.Attempt)
	if retry {
		return worker.options.Store.Retry(ctx, claim, state, category, worker.options.Now().UTC().Add(delay))
	}
	_, err := worker.options.Store.DeadLetter(ctx, claim, state, category)
	return err
}
