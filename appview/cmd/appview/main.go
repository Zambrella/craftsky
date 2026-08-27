// Command appview runs the Craftsky App View HTTP server.
//
// Usage:
//
//	appview dev
//	appview prod
//
// The positional argument selects the environment file under
// environments/ and the dev/prod divergent wiring (log level, auth
// service, CORS permissiveness).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/instagram"
)

const (
	instagramWebhookPollInterval        = 500 * time.Millisecond
	instagramReconciliationPollInterval = 500 * time.Millisecond
	instagramReconciliationBatchSize    = 100
	instagramRetentionInterval          = time.Hour
	scheduledWorkerPollInterval         = 10 * time.Second
	accountDeletionWorkerPollInterval   = 2 * time.Second
	backgroundWorkerShutdownTimeout     = 10 * time.Second
)

type instagramWebhookBatchProcessor interface {
	ProcessBatch(context.Context) (int, error)
}

type scheduledBatchProcessor interface {
	ProcessBatch(context.Context) (int, error)
}

type instagramReconciliationBatchProcessor interface {
	ProcessBatch(context.Context, int) (int, error)
}

type instagramRetentionRunner interface {
	Run(context.Context, int) (instagram.RetentionStats, error)
}

type accountDeletionProcessor interface {
	ProcessOne(context.Context) (bool, error)
}

func stopBackgroundWorkers(cancel context.CancelFunc, timeout time.Duration, done ...<-chan struct{}) error {
	if cancel == nil {
		return errors.New("background worker cancellation is unavailable")
	}
	if timeout <= 0 {
		return errors.New("background worker shutdown timeout must be positive")
	}
	cancel()
	ctx, release := context.WithTimeout(context.Background(), timeout)
	defer release()
	for _, workerDone := range done {
		if workerDone == nil {
			continue
		}
		select {
		case <-workerDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	// Signal handling wraps the whole run so Ctrl-C during deps init
	// (e.g. slow DB connect) exits cleanly.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// SIGPIPE is sent when a client disconnects mid-write. Go's net/http
	// already surfaces this as an error return; we don't need the signal.
	signal.Ignore(syscall.SIGPIPE)

	if len(args) <= 1 {
		return fmt.Errorf("expected argument of either 'dev' or 'prod'")
	}
	env, err := app.ParseEnv(args[1])
	if err != nil {
		return err
	}

	cfg, err := app.LoadConfig(env, fmt.Sprintf("environments/%s.env", env))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var deps *app.Deps
	var cleanup func()
	switch env {
	case app.EnvDev:
		deps, cleanup, err = app.NewDevDeps(ctx, cfg)
	case app.EnvProd:
		deps, cleanup, err = app.NewProdDeps(ctx, cfg)
	default:
		// ParseEnv should have rejected anything else, but defense in depth.
		return fmt.Errorf("unreachable: unknown env %q after ParseEnv", env)
	}
	if err != nil {
		return fmt.Errorf("build deps: %w", err)
	}
	defer cleanup()

	handler, err := NewServerWithAdmission(ctx, deps, handlerAdmissionConfigFromApp(cfg))
	if err != nil {
		return fmt.Errorf("configure HTTP admission: %w", err)
	}
	httpAdmission := httpAdmissionConfigFromApp(cfg)
	httpServer, err := NewHTTPServer(handler, httpAdmission)
	if err != nil {
		return fmt.Errorf("configure HTTP server: %w", err)
	}
	httpServer.Addr = net.JoinHostPort("0.0.0.0", "8080")
	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	limitedListener, err := NewConnectionLimitListener(listener, httpAdmission.MaxConnections)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("configure HTTP listener: %w", err)
	}

	// Start the Tap consumer alongside the HTTP server. It runs until
	// consumerCtx is cancelled, which happens on signal or if the HTTP
	// listener dies (we cancel explicitly below in both paths).
	consumerCtx, consumerCancel := context.WithCancel(ctx)
	defer consumerCancel()
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		if err := deps.Consumer.Run(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
			deps.Logger.Error("tap consumer exited",
				slog.String("component", "tap"),
				slog.String("operation", "tap.consume"),
				slog.String("result", "error"),
				slog.String("error_category", "consumer"))
		}
	}()
	tapProjectionDone := make(chan struct{})
	if deps.TapProjectionWorker != nil {
		go func() {
			defer close(tapProjectionDone)
			if err := deps.TapProjectionWorker.Run(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
				deps.Logger.Error("tap projection worker exited",
					slog.String("component", "tap_projection"),
					slog.String("result", "error"))
			}
		}()
	} else {
		close(tapProjectionDone)
	}
	tapRepositoryDone := make(chan struct{})
	if deps.TapRepositoryWorker != nil {
		go func() {
			defer close(tapRepositoryDone)
			if err := deps.TapRepositoryWorker.Run(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
				deps.Logger.Error("tap repository worker exited",
					slog.String("component", "tap_repository"),
					slog.String("result", "error"))
			}
		}()
	} else {
		close(tapRepositoryDone)
	}
	tapQuarantineDone := make(chan struct{})
	if deps.TapQuarantineWorker != nil {
		go func() {
			defer close(tapQuarantineDone)
			if err := deps.TapQuarantineWorker.Run(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
				deps.Logger.Error("tap quarantine replay worker exited",
					slog.String("component", "tap_quarantine"),
					slog.String("result", "error"))
			}
		}()
	} else {
		close(tapQuarantineDone)
	}
	followerGrowthDone := startFollowerGrowthWorker(
		consumerCtx,
		deps.FollowerGrowthWorker,
		deps.Logger,
	)
	pushDone := make(chan struct{})
	if deps.PushDispatcher != nil {
		go func() {
			defer close(pushDone)
			if err := deps.PushDispatcher.Run(consumerCtx, deps.Config.PushPollInterval, "appview"); err != nil && !errors.Is(err, context.Canceled) {
				deps.Logger.Error("push dispatcher exited", slog.String("result", "error"))
			}
		}()
	} else {
		close(pushDone)
	}
	instagramWorkersDone := make(chan struct{})
	if deps.InstagramWebhookWorker != nil {
		workerCount := deps.Config.InstagramLimits.WorkerConcurrency
		if workerCount < 1 {
			workerCount = 1
		}
		go func() {
			defer close(instagramWorkersDone)
			var workers sync.WaitGroup
			workers.Add(workerCount)
			for workerID := 0; workerID < workerCount; workerID++ {
				go func() {
					defer workers.Done()
					runInstagramWebhookWorker(consumerCtx, deps.InstagramWebhookWorker, deps.Logger, instagramWebhookPollInterval)
				}()
			}
			workers.Wait()
		}()
	} else {
		close(instagramWorkersDone)
	}
	instagramReconciliationDone := make(chan struct{})
	if deps.InstagramReconciliation != nil {
		go func() {
			defer close(instagramReconciliationDone)
			runInstagramReconciliationWorker(
				consumerCtx,
				deps.InstagramReconciliation,
				deps.Logger,
				instagramReconciliationBatchSize,
				instagramReconciliationPollInterval,
			)
		}()
	} else {
		close(instagramReconciliationDone)
	}
	instagramRetentionDone := make(chan struct{})
	if deps.InstagramRetention != nil {
		batchSize := deps.Config.InstagramLimits.OperatorBatchMax
		if batchSize < 1 || batchSize > instagram.MaxRetentionBatch {
			batchSize = instagram.MaxRetentionBatch
		}
		go func() {
			defer close(instagramRetentionDone)
			runInstagramRetention(
				consumerCtx,
				deps.InstagramRetention,
				deps.Logger,
				batchSize,
				instagramRetentionInterval,
			)
		}()
	} else {
		close(instagramRetentionDone)
	}
	scheduledPublicationDone := make(chan struct{})
	if deps.ScheduledPublisher != nil {
		go func() {
			defer close(scheduledPublicationDone)
			runScheduledWorker(consumerCtx, deps.ScheduledPublisher, deps.Logger, scheduledWorkerPollInterval, "publication")
		}()
	} else {
		close(scheduledPublicationDone)
	}
	scheduledCleanupDone := make(chan struct{})
	if deps.ScheduledCleanup != nil {
		go func() {
			defer close(scheduledCleanupDone)
			runScheduledWorker(consumerCtx, deps.ScheduledCleanup, deps.Logger, scheduledWorkerPollInterval, "cleanup")
		}()
	} else {
		close(scheduledCleanupDone)
	}
	accountDeletionDone := make(chan struct{})
	if deps.AccountDeletionWorker != nil {
		go func() {
			defer close(accountDeletionDone)
			runAccountDeletionWorker(
				consumerCtx,
				deps.AccountDeletionWorker,
				deps.Logger,
				accountDeletionWorkerPollInterval,
			)
		}()
	} else {
		close(accountDeletionDone)
	}
	authRequestSweepDone := make(chan struct{})
	if deps.OAuthStore != nil {
		go func() {
			defer close(authRequestSweepDone)
			runAuthRequestSweeper(
				consumerCtx,
				deps.OAuthStore,
				deps.Logger,
				deps.Observability,
				deps.Config.OAuthAuthRequestSweepBatch,
				deps.Config.OAuthAuthRequestSweepInterval,
			)
		}()
	} else {
		close(authRequestSweepDone)
	}
	oauthRevocationDone := startBatchWorker(
		consumerCtx,
		deps.OAuthRevocation,
		deps.Logger,
		deps.Config.OAuthRevocationPollInterval,
		"oauth",
		"credential_revocation",
	)
	authAuxiliaryCleanupDone := startBatchWorker(
		consumerCtx,
		deps.AuthAuxiliaryCleanup,
		deps.Logger,
		deps.Config.AuthAuxiliaryCleanupPollInterval,
		"auth_auxiliary_cleanup",
		"notification_subscription_cleanup",
	)
	sessionExpiryDone := startBatchWorker(
		consumerCtx,
		deps.SessionExpiry,
		deps.Logger,
		deps.Config.SessionExpirySweepInterval,
		"auth_session_expiry",
		"expire",
	)
	accountDeletionIntentExpiryDone := startBatchWorker(
		consumerCtx,
		deps.AccountDeletionIntentExpiry,
		deps.Logger,
		deps.Config.AccountDeletionIntentSweepInterval,
		"account_deletion",
		"expire_intent",
	)
	terminalPurgeDone := startBatchWorker(
		consumerCtx,
		deps.TerminalPurge,
		deps.Logger,
		deps.Config.TerminalPurgePollInterval,
		"owner_lifecycle",
		"terminal_purge",
	)
	identityCacheRefreshDone := startBatchWorker(
		consumerCtx,
		deps.IdentityCacheRefresh,
		deps.Logger,
		deps.Config.IdentityCacheRefreshPollInterval,
		"identity_cache",
		"refresh",
	)
	workerDone := []<-chan struct{}{
		consumerDone,
		tapProjectionDone,
		tapRepositoryDone,
		tapQuarantineDone,
		followerGrowthDone,
		pushDone,
		instagramWorkersDone,
		instagramReconciliationDone,
		instagramRetentionDone,
		scheduledPublicationDone,
		scheduledCleanupDone,
		accountDeletionDone,
		authRequestSweepDone,
		oauthRevocationDone,
		authAuxiliaryCleanupDone,
		sessionExpiryDone,
		accountDeletionIntentExpiryDone,
		terminalPurgeDone,
		identityCacheRefreshDone,
	}

	// listenErr receives the result of Serve. A non-nil,
	// non-ErrServerClosed error (e.g. port already in use) must unblock
	// the main goroutine so run() returns the error instead of hanging
	// while background workers continue using dependencies being torn down.
	listenErr := make(chan error, 1)
	go func() {
		deps.Logger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.Serve(limitedListener); err != nil && err != http.ErrServerClosed {
			listenErr <- err
			return
		}
		listenErr <- nil
	}()

	select {
	case err := <-listenErr:
		// Listener died before any signal arrived. Usually bind failure.
		workerErr := stopBackgroundWorkers(
			consumerCancel,
			backgroundWorkerShutdownTimeout,
			workerDone...,
		)
		if err != nil {
			return errors.Join(fmt.Errorf("listen: %w", err), workerErr)
		}
		// err == nil would mean ListenAndServe returned ErrServerClosed
		// without any signal — unexpected but benign. Fall through.
		return workerErr
	case <-ctx.Done():
		// Signal path: drain the listener via Shutdown then wait for
		// the listenErr goroutine to finish.
	}

	deps.Logger.Info("shutdown: received signal")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		deps.Logger.Error("shutdown error",
			slog.String("component", "http"),
			slog.String("operation", "shutdown"),
			slog.String("result", "error"),
			slog.String("error_category", "shutdown"))
	}
	deps.Logger.Info("shutdown: http server stopped")
	// Cancel every worker and wait only for the configured shutdown budget.
	// This happens before deferred dependency cleanup on every server-exit path.
	if err := stopBackgroundWorkers(
		consumerCancel,
		backgroundWorkerShutdownTimeout,
		workerDone...,
	); err != nil {
		deps.Logger.Error("background worker shutdown timed out",
			slog.String("component", "workers"),
			slog.String("operation", "shutdown"),
			slog.String("result", "error"),
			slog.String("error_category", "timeout"))
	}
	deps.Logger.Info("shutdown: tap consumer stopped")
	// Drain the listener goroutine's final send without creating another
	// unbounded shutdown wait if a server implementation regresses.
	select {
	case <-listenErr:
	case <-shutdownCtx.Done():
		_ = httpServer.Close()
	}
	return nil
}

type followerGrowthRunner interface {
	Run(context.Context) error
}

func startFollowerGrowthWorker(
	ctx context.Context,
	worker followerGrowthRunner,
	logger *slog.Logger,
) <-chan struct{} {
	done := make(chan struct{})
	if worker == nil {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("follower growth worker exited",
				slog.String("component", "follower_growth"),
				slog.String("operation", "capture"),
				slog.String("result", "error"),
				slog.String("error_category", "worker"))
		}
	}()
	return done
}

func runAccountDeletionWorker(ctx context.Context, worker accountDeletionProcessor, logger *slog.Logger, pollInterval time.Duration) {
	if worker == nil || pollInterval <= 0 {
		return
	}
	for {
		processed, err := worker.ProcessOne(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			logger.Error("account deletion worker failed",
				slog.String("component", "account_deletion"),
				slog.String("operation", "process"),
				slog.String("result", "error"),
				slog.String("error_category", "worker"))
		}
		if err == nil && processed {
			continue
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func runScheduledWorker(ctx context.Context, worker scheduledBatchProcessor, logger *slog.Logger, pollInterval time.Duration, operation string) {
	runBatchWorker(ctx, worker, logger, pollInterval, "scheduled_posts", operation)
}

func startBatchWorker(
	ctx context.Context,
	worker scheduledBatchProcessor,
	logger *slog.Logger,
	pollInterval time.Duration,
	component string,
	operation string,
) <-chan struct{} {
	done := make(chan struct{})
	if worker == nil {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		runBatchWorker(ctx, worker, logger, pollInterval, component, operation)
	}()
	return done
}

func runBatchWorker(
	ctx context.Context,
	worker scheduledBatchProcessor,
	logger *slog.Logger,
	pollInterval time.Duration,
	component string,
	operation string,
) {
	if worker == nil || pollInterval <= 0 {
		return
	}
	for {
		processed, err := worker.ProcessBatch(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			logger.Error("background worker batch failed",
				slog.String("component", component),
				slog.String("operation", operation),
				slog.String("result", "error"),
				slog.String("error_category", "worker"))
		}
		if err == nil && processed > 0 {
			continue
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func runInstagramWebhookWorker(ctx context.Context, worker instagramWebhookBatchProcessor, logger *slog.Logger, pollInterval time.Duration) {
	if worker == nil || pollInterval <= 0 {
		return
	}
	for {
		processed, err := worker.ProcessBatch(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			logger.Error("Instagram webhook worker batch failed",
				slog.String("component", "instagram_webhook"),
				slog.String("operation", "instagram.webhook.process"),
				slog.String("result", "error"),
				slog.String("error_category", "worker"))
		}
		if err == nil && processed > 0 {
			continue
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func runInstagramReconciliationWorker(
	ctx context.Context,
	worker instagramReconciliationBatchProcessor,
	logger *slog.Logger,
	batchSize int,
	pollInterval time.Duration,
) {
	if worker == nil || batchSize < 1 || batchSize > 500 || pollInterval <= 0 {
		return
	}
	for {
		processed, err := worker.ProcessBatch(ctx, batchSize)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			logger.Error("Instagram reconciliation worker batch failed",
				slog.String("component", "instagram_reconciliation"),
				slog.String("operation", "instagram.reconciliation.process"),
				slog.String("result", "error"),
				slog.String("error_category", "worker"))
		}
		if err == nil && processed > 0 {
			continue
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func runInstagramRetention(
	ctx context.Context,
	runner instagramRetentionRunner,
	logger *slog.Logger,
	batchSize int,
	interval time.Duration,
) {
	if runner == nil || batchSize < 1 || batchSize > instagram.MaxRetentionBatch || interval <= 0 {
		return
	}
	for {
		if _, err := runner.Run(ctx, batchSize); err != nil && ctx.Err() == nil && logger != nil {
			logger.Error("Instagram retention batch failed",
				slog.String("component", "instagram_retention"),
				slog.String("operation", "instagram.retention.run"),
				slog.String("result", "error"),
				slog.String("error_category", "worker"))
		}
		if ctx.Err() != nil {
			return
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}
