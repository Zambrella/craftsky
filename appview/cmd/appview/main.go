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
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
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

	go func() {
		deps.Logger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			deps.Logger.Error("server error", "err", err.Error())
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		deps.Logger.Info("shutdown: received signal")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			deps.Logger.Error("shutdown error", "err", err.Error())
		}
		deps.Logger.Info("shutdown: http server stopped")
	}()
	wg.Wait()
	return nil
}
