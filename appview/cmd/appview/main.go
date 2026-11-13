// Command appview runs the Craftsky App View HTTP server.
//
// Usage:
//
//	appview dev
//	appview prod
//	appview version
//	appview --version
//
// The positional argument selects the environment file under
// config/ and the dev/prod divergent wiring (log level, auth
// service, CORS permissiveness).
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/buildinfo"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/observability"
)

const (
	instagramWebhookPollInterval        = 500 * time.Millisecond
	instagramReconciliationPollInterval = 500 * time.Millisecond
	instagramReconciliationBatchSize    = 100
	instagramRetentionInterval          = time.Hour
	scheduledWorkerPollInterval         = 10 * time.Second
	accountDeletionWorkerPollInterval   = 2 * time.Second
	safetyRetentionPollInterval         = time.Minute
	safetyRetentionBatchSize            = 100
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

type safetyRetentionProcessor interface {
	RunOnce(context.Context, int) (int, error)
}

type revenueCatProcessor interface {
	ScheduleActiveAccounts(context.Context) (int64, error)
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
	if err := run(context.Background(), os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) > 1 && (args[1] == "version" || args[1] == "--version") {
		_, err := fmt.Fprintln(output, buildinfo.Version())
		return err
	}

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

	cfg, err := app.LoadConfig(env, fmt.Sprintf("config/%s.env", env))
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
	httpServer.Addr, err = listenAddress(os.Getenv("PORT"))
	if err != nil {
		return err
	}
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
	consumerCtx = context.WithValue(consumerCtx, workerObserverKey{}, deps.Observability)
	defer consumerCancel()
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		runCtx := workerAttemptContext(consumerCtx)
		if err := deps.Consumer.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			reportWorkerFailure(runCtx, deps.Logger, err, "tap", "tap.consume", "run", "consumer")
		}
	}()
	tapProjectionDone := make(chan struct{})
	if deps.TapProjectionWorker != nil {
		go func() {
			defer close(tapProjectionDone)
			runCtx := workerAttemptContext(consumerCtx)
			if err := deps.TapProjectionWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				reportWorkerFailure(runCtx, deps.Logger, err, "tap_projection", "tap.projection.run", "run", "worker")
			}
		}()
	} else {
		close(tapProjectionDone)
	}
	tapRepositoryDone := make(chan struct{})
	if deps.TapRepositoryWorker != nil {
		go func() {
			defer close(tapRepositoryDone)
			runCtx := workerAttemptContext(consumerCtx)
			if err := deps.TapRepositoryWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				reportWorkerFailure(runCtx, deps.Logger, err, "tap_repository", "tap.repository.run", "run", "worker")
			}
		}()
	} else {
		close(tapRepositoryDone)
	}
	tapQuarantineDone := make(chan struct{})
	if deps.TapQuarantineWorker != nil {
		go func() {
			defer close(tapQuarantineDone)
			runCtx := workerAttemptContext(consumerCtx)
			if err := deps.TapQuarantineWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				reportWorkerFailure(runCtx, deps.Logger, err, "tap_quarantine", "tap.quarantine.run", "run", "worker")
			}
		}()
	} else {
		close(tapQuarantineDone)
	}
	imageSafetyDone := make(chan struct{})
	if deps.ImageSafetyWorker != nil {
		go func() {
			defer close(imageSafetyDone)
			runCtx := workerAttemptContext(consumerCtx)
			if err := deps.ImageSafetyWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				reportWorkerFailure(runCtx, deps.Logger, err, "image_safety", "image_safety.run", "run", "worker")
			}
		}()
	} else {
		close(imageSafetyDone)
	}
	followerGrowthDone := startFollowerGrowthWorker(
		consumerCtx,
		deps.FollowerGrowthWorker,
		deps.Logger,
	)
	pushDone := make(chan struct{})
	chatSyncDone := make(chan struct{})
	chatDeliveryDone := make(chan struct{})
	chatMaintenanceDone := make(chan struct{})
	if deps.ChatDispatcher != nil {
		go func() {
			defer close(chatDeliveryDone)
			runCtx := privateWorkerAttemptContext(consumerCtx)
			if err := deps.ChatDispatcher.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				deps.Observability.ObservePrivateFailure(runCtx, err, "", "", "chat.push", "worker", "error", 1)
			}
		}()
	} else {
		close(chatDeliveryDone)
	}
	if deps.Chat != nil {
		go func() {
			defer close(chatMaintenanceDone)
			runCtx := privateWorkerAttemptContext(consumerCtx)
			if err := deps.Chat.RunMaintenance(runCtx, deps.Observability); err != nil && !errors.Is(err, context.Canceled) {
				deps.Observability.ObservePrivateFailure(runCtx, err, "", "", "chat.retention", "worker", "error", 1)
			}
		}()
	} else {
		close(chatMaintenanceDone)
	}
	if deps.ChatSynchronizer != nil {
		go func() {
			defer close(chatSyncDone)
			runCtx := privateWorkerAttemptContext(consumerCtx)
			if err := deps.ChatSynchronizer.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				deps.Observability.ObservePrivateFailure(runCtx, err, "", "", "chat.sync", "scheduler", "error", 1)
			}
		}()
	} else {
		close(chatSyncDone)
	}
	if deps.PushDispatcher != nil {
		go func() {
			defer close(pushDone)
			runCtx := workerAttemptContext(consumerCtx)
			if err := deps.PushDispatcher.Run(runCtx, deps.Config.PushPollInterval, "appview"); err != nil && !errors.Is(err, context.Canceled) {
				reportWorkerFailure(runCtx, deps.Logger, err, "push", "push.dispatch", "run", "worker")
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
	safetyRetentionDone := make(chan struct{})
	if deps.SafetyRetention != nil {
		go func() {
			defer close(safetyRetentionDone)
			runSafetyRetentionWorker(
				consumerCtx, deps.SafetyRetention, deps.Logger,
				safetyRetentionBatchSize, safetyRetentionPollInterval,
			)
		}()
	} else {
		close(safetyRetentionDone)
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
	revenueCatDone := startRevenueCatProcessor(
		consumerCtx,
		deps.RevenueCatReconciler,
		deps.Logger,
		deps.Config.RevenueCat.ReconciliationPollInterval(),
		deps.Config.RevenueCat.ReconciliationScheduleInterval(),
	)
	moderationExpiryDone := startBatchWorker(
		consumerCtx,
		deps.ModerationExpiry,
		deps.Logger,
		deps.Config.ModerationExpiryPollInterval,
		"moderation",
		"expire_strikes",
	)
	pdsCommandCompactionDone := startBatchWorker(
		consumerCtx,
		deps.PDSCommandCompaction,
		deps.Logger,
		deps.Config.PDSCommandCompactionPollInterval,
		"pds_commands",
		"compact",
	)
	var bunnyCleanup scheduledBatchProcessor
	if deps.BunnyVideoCleanup != nil {
		bunnyCleanup = deps.BunnyVideoCleanup
	}
	bunnyCleanupDone := startBatchWorker(consumerCtx, bunnyCleanup, deps.Logger, 5*time.Second, "bunny_video_cleanup", "delete_and_purge")
	var safetyEmail scheduledBatchProcessor
	if deps.SafetyEmail != nil {
		safetyEmail = deps.SafetyEmail
	}
	safetyEmailDone := startBatchWorker(consumerCtx, safetyEmail, deps.Logger, 30*time.Second, "safety_email", "deliver")
	workerDone := []<-chan struct{}{
		safetyEmailDone,
		bunnyCleanupDone,
		consumerDone,
		tapProjectionDone,
		tapRepositoryDone,
		tapQuarantineDone,
		imageSafetyDone,
		followerGrowthDone,
		pushDone,
		chatSyncDone,
		chatDeliveryDone,
		chatMaintenanceDone,
		instagramWorkersDone,
		instagramReconciliationDone,
		instagramRetentionDone,
		scheduledPublicationDone,
		scheduledCleanupDone,
		accountDeletionDone,
		safetyRetentionDone,
		authRequestSweepDone,
		oauthRevocationDone,
		authAuxiliaryCleanupDone,
		sessionExpiryDone,
		accountDeletionIntentExpiryDone,
		terminalPurgeDone,
		identityCacheRefreshDone,
		revenueCatDone,
		moderationExpiryDone,
		pdsCommandCompactionDone,
	}

	// listenErr receives the result of Serve. A non-nil,
	// non-ErrServerClosed error (e.g. port already in use) must unblock
	// the main goroutine so run() returns the error instead of hanging
	// while background workers continue using dependencies being torn down.
	listenErr := make(chan error, 1)
	go func() {
		logListening(deps.Logger, httpServer.Addr)
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

func startRevenueCatProcessor(ctx context.Context, processor revenueCatProcessor, logger *slog.Logger, pollInterval, scheduleInterval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	if processor == nil || pollInterval <= 0 || scheduleInterval <= 0 {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		schedule := func() {
			attemptCtx := privateWorkerAttemptContext(ctx)
			if _, err := processor.ScheduleActiveAccounts(attemptCtx); err != nil && ctx.Err() == nil && logger != nil {
				reportWorkerFailure(attemptCtx, logger, err, "revenuecat_reconciliation", "schedule", "schedule", "worker")
			}
		}
		schedule()
		poll := time.NewTicker(pollInterval)
		periodic := time.NewTicker(scheduleInterval)
		defer poll.Stop()
		defer periodic.Stop()
		for {
			attemptCtx := privateWorkerAttemptContext(ctx)
			processed, err := processor.ProcessOne(attemptCtx)
			if ctx.Err() != nil {
				return
			}
			if err != nil && logger != nil {
				reportWorkerFailure(attemptCtx, logger, err, "revenuecat_reconciliation", "process", "process", "provider")
			}
			select {
			case <-periodic.C:
				schedule()
			default:
			}
			if err == nil && processed {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-periodic.C:
				schedule()
			case <-poll.C:
			}
		}
	}()
	return done
}

func logListening(logger *slog.Logger, addr string) {
	logger.Info("listening",
		slog.String("addr", addr),
		slog.String("app_version", buildinfo.Version()),
	)
}

func listenAddress(port string) (string, error) {
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	return net.JoinHostPort("0.0.0.0", port), nil
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
		attemptCtx := workerAttemptContext(ctx)
		if err := worker.Run(attemptCtx); err != nil && !errors.Is(err, context.Canceled) {
			reportWorkerFailure(attemptCtx, logger, err, "follower_growth", "capture", "run", "worker")
		}
	}()
	return done
}

func runSafetyRetentionWorker(
	ctx context.Context,
	worker safetyRetentionProcessor,
	logger *slog.Logger,
	batchSize int,
	pollInterval time.Duration,
) {
	if worker == nil || batchSize <= 0 || pollInterval <= 0 {
		return
	}
	for {
		attemptCtx := privateWorkerAttemptContext(ctx)
		processed, err := worker.RunOnce(attemptCtx, batchSize)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			reportWorkerFailure(attemptCtx, logger, err, "safety_retention", "process", "process", "worker")
		}
		if err == nil && processed == batchSize {
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

func runAccountDeletionWorker(ctx context.Context, worker accountDeletionProcessor, logger *slog.Logger, pollInterval time.Duration) {
	if worker == nil || pollInterval <= 0 {
		return
	}
	for {
		attemptCtx := privateWorkerAttemptContext(ctx)
		processed, err := worker.ProcessOne(attemptCtx)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			reportWorkerFailure(attemptCtx, logger, err, "account_deletion", "process", "process", "worker")
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
	runBatchWorker(ctx, worker, logger, pollInterval, "scheduled_posts", operation, false)
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
		runBatchWorker(ctx, worker, logger, pollInterval, component, operation, true)
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
	captureFailure bool,
) {
	if worker == nil || pollInterval <= 0 {
		return
	}
	for {
		attemptCtx := privateWorkerAttemptContext(ctx)
		processed, err := worker.ProcessBatch(attemptCtx)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			if captureFailure {
				reportWorkerFailure(attemptCtx, logger, err, component, operation, "process", "worker")
			} else {
				observability.LogDiagnostic(attemptCtx, logger, workerFailureInput(err, component, operation, "process", "worker"))
			}
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
		attemptCtx := privateWorkerAttemptContext(ctx)
		processed, err := worker.ProcessBatch(attemptCtx)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			reportWorkerFailure(attemptCtx, logger, err, "instagram_webhook", "instagram.webhook.process", "process", "worker")
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
		attemptCtx := privateWorkerAttemptContext(ctx)
		processed, err := worker.ProcessBatch(attemptCtx, batchSize)
		if ctx.Err() != nil {
			return
		}
		if err != nil && logger != nil {
			reportWorkerFailure(attemptCtx, logger, err, "instagram_reconciliation", "instagram.reconciliation.process", "process", "worker")
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
		attemptCtx := privateWorkerAttemptContext(ctx)
		if _, err := runner.Run(attemptCtx, batchSize); err != nil && ctx.Err() == nil && logger != nil {
			reportWorkerFailure(attemptCtx, logger, err, "instagram_retention", "instagram.retention.run", "process", "worker")
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

// Keep the observer on the worker tree, but clone its scope and ownership marker
// for each attempt so unrelated polls cannot share breadcrumbs or captures.
type workerObserverKey struct{}

func workerAttemptContext(ctx context.Context) context.Context {
	observer, _ := ctx.Value(workerObserverKey{}).(*observability.Observer)
	return observability.WithRequestObserver(observability.WithDiagnosticOccurrence(ctx), observer)
}

func privateWorkerAttemptContext(ctx context.Context) context.Context {
	return observability.WithPrivateDiagnosticContext(workerAttemptContext(ctx), "", "")
}

func reportWorkerFailure(ctx context.Context, logger *slog.Logger, cause error, component, operation, stage, category string) {
	input := workerFailureInput(cause, component, operation, stage, category)
	observability.LogDiagnostic(ctx, logger, input)
	observability.CaptureRequestDiagnostic(ctx, input)
}

func workerFailureInput(cause error, component, operation, stage, category string) observability.DiagnosticInput {
	result := "retry"
	if stage == "run" {
		result = "error"
	}
	return observability.DiagnosticInput{Error: cause, Context: observability.EventContext{
		"component": component, "operation": operation, "failure_stage": stage, "result": result, "error_category": category,
	}}
}
