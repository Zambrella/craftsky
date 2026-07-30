package languages

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPreferencesNotFound = errors.New("language preferences not found")

type Preferences struct {
	PrimaryLanguage  string   `json:"primaryLanguage"`
	ContentLanguages []string `json:"contentLanguages"`
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Get(ctx context.Context, did syntax.DID) (Preferences, error) {
	return scanPreferences(s.pool.QueryRow(ctx, `
		SELECT primary_language, content_languages
		FROM account_language_preferences
		WHERE account_did = $1
	`, did))
}

func (s *Store) Replace(
	ctx context.Context,
	did syntax.DID,
	preferences Preferences,
) (Preferences, error) {
	if err := ValidatePreferences(preferences); err != nil {
		return Preferences{}, err
	}
	return scanPreferences(s.pool.QueryRow(ctx, `
		UPDATE account_language_preferences
		SET primary_language = $2,
		    content_languages = $3,
		    updated_at = now()
		WHERE account_did = $1
		RETURNING primary_language, content_languages
	`, did, preferences.PrimaryLanguage, preferences.ContentLanguages))
}

func (s *Store) Initialize(
	ctx context.Context,
	did syntax.DID,
	proposal Preferences,
) (Preferences, error) {
	if err := ValidatePreferences(proposal); err != nil {
		return Preferences{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Preferences{}, fmt.Errorf("begin language preferences initialization: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES ($1, $2, $3)
		ON CONFLICT (account_did) DO NOTHING
	`, did, proposal.PrimaryLanguage, proposal.ContentLanguages); err != nil {
		return Preferences{}, fmt.Errorf("initialize language preferences: %w", err)
	}

	preferences, err := scanPreferences(tx.QueryRow(ctx, `
		SELECT primary_language, content_languages
		FROM account_language_preferences
		WHERE account_did = $1
	`, did))
	if err != nil {
		return Preferences{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Preferences{}, fmt.Errorf("commit language preferences initialization: %w", err)
	}
	return preferences, nil
}

type preferenceRow interface {
	Scan(dest ...any) error
}

func scanPreferences(row preferenceRow) (Preferences, error) {
	var preferences Preferences
	if err := row.Scan(
		&preferences.PrimaryLanguage,
		&preferences.ContentLanguages,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, ErrPreferencesNotFound
		}
		return Preferences{}, fmt.Errorf("read language preferences: %w", err)
	}
	if preferences.ContentLanguages == nil {
		preferences.ContentLanguages = []string{}
	}
	return preferences, nil
}
