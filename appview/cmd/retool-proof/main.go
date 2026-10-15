// retool-proof is an isolated synthetic fixture server. The production Docker
// image does not include it, and it accepts only its dedicated proof database.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/retoolproof"
)

func main() {
	if err := run(); err != nil {
		logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(os.Stderr, nil)))
		logStartupFailure(logger, err)
		os.Exit(1)
	}
}

func logStartupFailure(logger *slog.Logger, cause error) {
	observability.LogDiagnostic(context.Background(), logger, observability.DiagnosticInput{
		Error: cause,
		Context: observability.EventContext{
			"component": "retool-proof", "operation": "proof.start", "failure_stage": "startup", "result": "error",
		},
	})
}

func run() error {
	databaseURL := os.Getenv("RETOOL_PROOF_DATABASE_URL")
	u, err := url.Parse(databaseURL)
	if err != nil || u.Path != "/craftsky_retool_proof" || (u.Hostname() != "host.docker.internal" && u.Hostname() != "127.0.0.1") {
		return errors.New("dedicated local proof database required")
	}
	m, err := migrate.New("file:///app/migrations", databaseURL)
	if err != nil {
		return err
	}
	err = m.Up()
	_, _ = m.Close()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	credential := func() string {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	alpha, beta := credential(), credential()
	if existing, err := os.ReadFile("/run/proof/credentials.json"); err == nil {
		var credentials struct {
			Alpha string `json:"alpha"`
			Beta  string `json:"beta"`
		}
		if json.Unmarshal(existing, &credentials) != nil || credentials.Alpha == "" || credentials.Beta == "" {
			return errors.New("invalid proof credential file")
		}
		alpha, beta = credentials.Alpha, credentials.Beta
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fixture, err := retoolproof.Seed(ctx, pool, alpha, beta)
	if err != nil {
		return err
	}
	// This file is outside the repository and mounted into the container. Never
	// print credentials or include them in the OpenAPI specification/app source.
	data, _ := json.Marshal(map[string]any{"alpha": alpha, "beta": beta, "caseReference": fixture.CaseReference, "evidenceId": fixture.EvidenceID})
	if err := os.WriteFile("/run/proof/credentials.json", data, 0600); err != nil {
		return err
	}
	origin := os.Getenv("RETOOL_PROOF_PUBLIC_ORIGIN")
	if origin == "" {
		origin = "http://127.0.0.1:18147"
	}
	handler, err := retoolproof.New(pool, fixture, origin)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: ":8080", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Fprintln(os.Stdout, "synthetic moderation proof ready")
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
