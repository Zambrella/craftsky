package imagesafety

import (
	"bytes"
	"context"
	"errors"
	"github.com/google/uuid"
	"log/slog"
	"strings"
	"time"

	"social.craftsky/appview/internal/observability"
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
	Observer       *observability.Observer
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
		_, _ = worker.ProcessOne(ctx) // Each attempt owns its diagnostics.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (worker *Worker) ProcessOne(ctx context.Context) (bool, error) {
	ctx = observability.WithRequestObserver(observability.WithDiagnosticOccurrence(ctx), worker.options.Observer)
	claim, found, err := worker.options.Store.Claim(ctx, worker.options.WorkerID, worker.options.LeaseDuration)
	if err != nil || !found {
		if err != nil {
			worker.reportFailure(ctx, claim, err, "claim", "retry")
		}
		return found, err
	}
	ctx = observability.WithPrivateDiagnosticContext(ctx, "", claim.JobID.String())
	attemptCtx, cancel := context.WithTimeout(ctx, worker.options.OperationLimit)
	defer cancel()

	body, err := worker.options.Fetcher.Fetch(attemptCtx, claim.Source)
	if err != nil {
		return true, worker.fail(ctx, claim, StateUnavailable, "fetch", err)
	}
	result, err := worker.options.Scanner.Scan(attemptCtx, ScanInput{
		BlobCID: claim.Source.BlobCID, MIMEType: claim.Source.DeclaredMIME,
		Size: int64(len(body)), Content: bytes.NewReader(body),
	})
	if err != nil || !result.State.Valid() || result.State == StatePending {
		if err == nil {
			err = &observability.DiagnosticError{Message: "image scanner returned an invalid result"}
		}
		return true, worker.fail(ctx, claim, StateError, "scanner", err)
	}
	if result.State == StateClear && !worker.options.Config.CanMarkClear() {
		return true, worker.fail(ctx, claim, StateError, "configuration", &observability.DiagnosticError{Message: "image scanner configuration cannot mark images clear"})
	}
	if result.State == StateUnavailable || result.State == StateError {
		return true, worker.fail(ctx, claim, result.State, "scanner", &observability.DiagnosticError{Message: "image scanner returned an unavailable or error result"})
	}
	if result.State == StateMatch && (worker.options.MatchRecorder == nil ||
		strings.TrimSpace(result.ProviderReference) == "" || len(result.ProviderReference) > 512 ||
		strings.TrimSpace(result.IntegrityMetadataReference) == "" || len(result.IntegrityMetadataReference) > 512) {
		return true, worker.fail(ctx, claim, StateError, "incident", &observability.DiagnosticError{Message: "image scanner match lacks required incident metadata"})
	}

	_, err = worker.options.Store.CompleteResult(ctx, claim, result, worker.options.MatchRecorder)
	if err != nil && result.State == StateMatch && !errors.Is(err, ErrLeaseLost) {
		return true, worker.fail(ctx, claim, StateError, "incident", err)
	}
	if err != nil {
		worker.reportFailure(ctx, claim, err, "completion", "error")
	}
	return true, err
}

func (worker *Worker) fail(ctx context.Context, claim Claim, state State, category string, cause error) error {
	delay, retry := worker.options.RetryPolicy.Next(claim.Attempt)
	outcome := "retry"
	if !retry {
		outcome = "exhausted"
	}
	var err error
	if retry {
		err = worker.options.Store.Retry(ctx, claim, state, category, worker.options.Now().UTC().Add(delay))
	} else {
		_, err = worker.options.Store.DeadLetter(ctx, claim, state, category)
	}
	if err != nil {
		cause = errors.Join(cause, err)
		outcome = "error"
	}
	worker.reportFailure(ctx, claim, cause, category, outcome)
	return err
}

func (worker *Worker) reportFailure(ctx context.Context, claim Claim, cause error, stage, outcome string) {
	workflowRef := ""
	if claim.JobID != uuid.Nil {
		workflowRef = claim.JobID.String()
	}
	input := observability.DiagnosticInput{Error: cause, Context: observability.EventContext{
		"component": "image_safety", "operation": "image_safety.scan", "failure_stage": stage, "result": outcome, "attempt": claim.Attempt,
	}, Workflow: observability.PrivateFailureContext(cause, "", workflowRef)}
	observability.LogDiagnostic(ctx, worker.options.Logger, input)
	worker.options.Observer.CaptureDiagnostic(ctx, input)
}
