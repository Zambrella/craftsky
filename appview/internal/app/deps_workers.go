package app

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// authorityWorkers groups bounded background processors that consume the same
// auth/owner stores but have independent leases and polling schedules.
type authorityWorkers struct {
	oauthRevocation  *auth.OAuthRevocationProcessor
	auxiliaryCleanup *auth.AuxiliaryCleanupProcessor
	sessionExpiry    *auth.SessionExpiryProcessor
	terminalPurge    *ownerlifecycle.TerminalPurgeProcessor
}

func newAuthorityWorkers(
	pool *pgxpool.Pool,
	authCapability *authDependencies,
	owners *ownerDependencies,
	notificationStore *api.PostStore,
	observer *observability.Observer,
	cfg Config,
) (*authorityWorkers, error) {
	oauthCredentialRevoker, err := auth.NewIndigoOAuthCredentialRevoker(
		authCapability.app,
		authCapability.store,
	)
	if err != nil {
		return nil, fmt.Errorf("OAuth credential revoker: %w", err)
	}
	oauthRevocation, err := auth.NewOAuthRevocationProcessor(auth.OAuthRevocationProcessorOptions{
		Pool: pool, Revoker: oauthCredentialRevoker, Now: time.Now,
		BatchSize: cfg.OAuthRevocationBatchSize, LeaseDuration: cfg.OAuthRevocationLeaseDuration,
		OperationTimeout: cfg.OAuthRevocationOperationTimeout, MaxAttempts: cfg.OAuthRevocationMaxAttempts,
		BaseBackoff: cfg.OAuthRevocationBackoffMin, MaxBackoff: cfg.OAuthRevocationBackoffMax,
		MaxCredentialRetention: cfg.OAuthRevocationMaxCredentialRetention,
	})
	if err != nil {
		return nil, fmt.Errorf("OAuth revocation processor: %w", err)
	}
	authAuxiliaryCleanup, err := auth.NewAuxiliaryCleanupProcessor(auth.AuxiliaryCleanupProcessorOptions{
		Pool: pool, Cleaner: notificationStore, Now: time.Now,
		BatchSize: cfg.AuthAuxiliaryCleanupBatchSize, LeaseDuration: cfg.AuthAuxiliaryCleanupLeaseDuration,
		OperationTimeout: cfg.AuthAuxiliaryCleanupOperationTimeout, MaxAttempts: cfg.AuthAuxiliaryCleanupMaxAttempts,
		BaseBackoff: cfg.AuthAuxiliaryCleanupBackoffMin, MaxBackoff: cfg.AuthAuxiliaryCleanupBackoffMax,
	})
	if err != nil {
		return nil, fmt.Errorf("auth auxiliary cleanup processor: %w", err)
	}
	sessionExpiry, err := auth.NewSessionExpiryProcessor(auth.SessionExpiryProcessorOptions{
		Lifecycle: authCapability.sessionLifecycle,
		BatchSize: cfg.SessionExpirySweepBatch,
	})
	if err != nil {
		return nil, fmt.Errorf("session expiry processor: %w", err)
	}
	terminalPurge, err := ownerlifecycle.NewTerminalPurgeProcessor(ownerlifecycle.TerminalPurgeProcessorConfig{
		Store:          owners.lifecycles,
		WorkerID:       "appview-terminal-purge",
		PollInterval:   cfg.TerminalPurgePollInterval,
		ComponentLimit: cfg.TerminalPurgeComponentLimit,
		RowBatchSize:   cfg.TerminalPurgeRowBatchSize,
		LeaseDuration:  cfg.TerminalPurgeLeaseDuration,
		RetryDelay:     cfg.TerminalPurgeRetryDelay,
		Observer:       observer,
	})
	if err != nil {
		return nil, fmt.Errorf("terminal purge processor: %w", err)
	}
	return &authorityWorkers{
		oauthRevocation: oauthRevocation, auxiliaryCleanup: authAuxiliaryCleanup,
		sessionExpiry: sessionExpiry, terminalPurge: terminalPurge,
	}, nil
}
