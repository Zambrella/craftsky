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
	"syscall"
	"time"

	"social.craftsky/appview/internal/app"
)

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

	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", "8080"),
		Handler: NewServer(ctx, deps),
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

	// listenErr receives the result of ListenAndServe. A non-nil,
	// non-ErrServerClosed error (e.g. port already in use) must unblock
	// the main goroutine so run() returns the error instead of hanging
	// on wg.Wait() forever.
	listenErr := make(chan error, 1)
	go func() {
		deps.Logger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			listenErr <- err
			return
		}
		listenErr <- nil
	}()

	select {
	case err := <-listenErr:
		// Listener died before any signal arrived. Usually bind failure.
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		// err == nil would mean ListenAndServe returned ErrServerClosed
		// without any signal — unexpected but benign. Fall through.
		return nil
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
	// Cancel the consumer explicitly and wait for it to exit, so the
	// shutdown log lines report in a predictable order.
	consumerCancel()
	<-consumerDone
	<-pushDone
	deps.Logger.Info("shutdown: tap consumer stopped")
	// Drain the listener goroutine's final send.
	<-listenErr
	return nil
}
