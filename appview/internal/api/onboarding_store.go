package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

type OnboardingStatusStore struct {
	pool *pgxpool.Pool
}

func NewOnboardingStatusStore(pool *pgxpool.Pool) *OnboardingStatusStore {
	return &OnboardingStatusStore{pool: pool}
}

func (s *OnboardingStatusStore) Status(ctx context.Context, did syntax.DID) (OnboardingStatus, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("begin onboarding status read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ownerlifecycle.GuardPrivateMutationTx(ctx, tx, did, nil); err != nil {
		return OnboardingStatus{}, fmt.Errorf("authorize onboarding status read: %w", err)
	}
	var completedAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT completed_at
		FROM account_onboarding_completions
		WHERE account_did = $1
	`, did).Scan(&completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return OnboardingStatus{}, fmt.Errorf("commit onboarding status read: %w", err)
		}
		return OnboardingStatus{Completed: false}, nil
	}
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("read onboarding status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OnboardingStatus{}, fmt.Errorf("commit onboarding status read: %w", err)
	}
	return OnboardingStatus{Completed: true, CompletedAt: &completedAt}, nil
}

func (s *OnboardingStatusStore) Complete(ctx context.Context, did syntax.DID) (OnboardingStatus, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("begin onboarding completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ownerlifecycle.GuardPrivateMutationTx(ctx, tx, did, nil); err != nil {
		return OnboardingStatus{}, fmt.Errorf("authorize onboarding completion: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_onboarding_completions (account_did)
		VALUES ($1)
		ON CONFLICT (account_did) DO NOTHING
	`, did); err != nil {
		return OnboardingStatus{}, fmt.Errorf("complete onboarding: %w", err)
	}
	var completedAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT completed_at
		FROM account_onboarding_completions
		WHERE account_did = $1
	`, did).Scan(&completedAt); err != nil {
		return OnboardingStatus{}, fmt.Errorf("read completed onboarding status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OnboardingStatus{}, fmt.Errorf("commit onboarding completion: %w", err)
	}
	return OnboardingStatus{Completed: true, CompletedAt: &completedAt}, nil
}
