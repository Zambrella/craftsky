package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileCustomisationStoreOptions struct {
	Now func() time.Time
}

type ProfileCustomisationStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

// ProfileCustomisationBatchQuery is kept as one statement so response
// hydration remains bounded and its PostgreSQL plan can be regression-tested.
const ProfileCustomisationBatchQuery = `
	SELECT p.did, c.colour, c.profile_border, c.profile_background
	FROM craftsky_profiles p
	LEFT JOIN profile_customisations c ON c.owner_did = p.did
	WHERE p.did = ANY($1::text[])
`

func NewProfileCustomisationStore(
	pool *pgxpool.Pool,
	options ...ProfileCustomisationStoreOptions,
) *ProfileCustomisationStore {
	now := time.Now
	if len(options) > 0 && options[0].Now != nil {
		now = options[0].Now
	}
	return &ProfileCustomisationStore{pool: pool, now: now}
}

func (s *ProfileCustomisationStore) Read(
	ctx context.Context,
	owner syntax.DID,
) (ProfileCustomisation, error) {
	value := ProfileCustomisation{}
	err := s.pool.QueryRow(ctx, `
		SELECT colour, profile_border, profile_background
		FROM profile_customisations
		WHERE owner_did = $1
	`, owner).Scan(&value.Colour, &value.Border, &value.Background)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultProfileCustomisation, nil
	}
	if err != nil {
		return ProfileCustomisation{}, fmt.Errorf("profile customisation read: %w", err)
	}
	return EffectiveProfileCustomisation(value), nil
}

func (s *ProfileCustomisationStore) Put(
	ctx context.Context,
	owner syntax.DID,
	value ProfileCustomisation,
) (ProfileCustomisation, error) {
	now := s.now().UTC()
	stored := ProfileCustomisation{}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (owner_did) DO UPDATE SET
			colour = EXCLUDED.colour,
			profile_border = EXCLUDED.profile_border,
			profile_background = EXCLUDED.profile_background,
			updated_at = EXCLUDED.updated_at
		RETURNING colour, profile_border, profile_background
	`, owner, value.Colour, value.Border, value.Background, now).Scan(
		&stored.Colour,
		&stored.Border,
		&stored.Background,
	)
	if err != nil {
		return ProfileCustomisation{}, fmt.Errorf("profile customisation put: %w", err)
	}
	return EffectiveProfileCustomisation(stored), nil
}

func (s *ProfileCustomisationStore) ReadBatch(
	ctx context.Context,
	owners []syntax.DID,
) (map[syntax.DID]ProfileCustomisation, error) {
	unique := make(map[syntax.DID]struct{}, len(owners))
	dids := make([]string, 0, len(owners))
	for _, owner := range owners {
		if owner == "" {
			continue
		}
		if _, exists := unique[owner]; exists {
			continue
		}
		unique[owner] = struct{}{}
		dids = append(dids, owner.String())
	}
	result := make(map[syntax.DID]ProfileCustomisation, len(dids))
	if len(dids) == 0 {
		return result, nil
	}

	rows, err := s.pool.Query(ctx, ProfileCustomisationBatchQuery, dids)
	if err != nil {
		return nil, fmt.Errorf("profile customisation batch query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			owner                      string
			colour, border, background *string
		)
		if err := rows.Scan(&owner, &colour, &border, &background); err != nil {
			return nil, fmt.Errorf("profile customisation batch scan: %w", err)
		}
		value := DefaultProfileCustomisation
		if colour != nil {
			value.Colour = *colour
		}
		if border != nil {
			value.Border = *border
		}
		if background != nil {
			value.Background = *background
		}
		result[syntax.DID(owner)] = EffectiveProfileCustomisation(value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("profile customisation batch rows: %w", err)
	}
	return result, nil
}
