package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type RecentSearchRow struct {
	ID                string
	Type              string
	DisplayLabel      string
	NormalizedPayload []byte
	UpdatedAt         time.Time
}

func (s *SearchStore) ListRecentSearches(ctx context.Context, viewerDID string) ([]RecentSearchRow, error) {
	var out []RecentSearchRow
	err := s.observeDB(ctx, "search.recent.list", "/v1/search/recent", func(ctx context.Context) error {
		var err error
		out, err = s.listRecentSearchesObserved(ctx, viewerDID)
		return err
	})
	return out, err
}

func (s *SearchStore) listRecentSearchesObserved(ctx context.Context, viewerDID string) ([]RecentSearchRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, search_type, display_label, normalized_payload, updated_at
		FROM craftsky_recent_searches
		WHERE viewer_did = $1
		ORDER BY updated_at DESC, id DESC
		LIMIT 50`, viewerDID)
	if err != nil {
		return nil, fmt.Errorf("list recent searches: %w", err)
	}
	defer rows.Close()
	out := []RecentSearchRow{}
	for rows.Next() {
		var row RecentSearchRow
		if err := rows.Scan(&row.ID, &row.Type, &row.DisplayLabel, &row.NormalizedPayload, &row.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *SearchStore) SaveRecentSearch(ctx context.Context, viewerDID string, req SaveRecentSearchRequest, now time.Time) (RecentSearchRow, error) {
	var row RecentSearchRow
	err := s.observeDB(ctx, "search.recent.save", "/v1/search/recent", func(ctx context.Context) error {
		var err error
		row, err = s.saveRecentSearchObserved(ctx, viewerDID, req, now)
		return err
	})
	return row, err
}

func (s *SearchStore) saveRecentSearchObserved(ctx context.Context, viewerDID string, req SaveRecentSearchRequest, now time.Time) (RecentSearchRow, error) {
	id, err := newRecentSearchID()
	if err != nil {
		return RecentSearchRow{}, err
	}
	var row RecentSearchRow
	err = s.pool.QueryRow(ctx, `
		INSERT INTO craftsky_recent_searches (id, viewer_did, search_type, display_label, normalized_payload, normalized_payload_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (viewer_did, search_type, normalized_payload_hash)
		DO UPDATE SET updated_at = excluded.updated_at
		RETURNING id, search_type, display_label, normalized_payload, updated_at`,
		id, viewerDID, req.Type, req.DisplayLabel, req.NormalizedPayload, req.PayloadHash, now,
	).Scan(&row.ID, &row.Type, &row.DisplayLabel, &row.NormalizedPayload, &row.UpdatedAt)
	if err != nil {
		return RecentSearchRow{}, fmt.Errorf("save recent search: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM craftsky_recent_searches
		WHERE viewer_did = $1 AND id IN (
			SELECT id FROM (
				SELECT id, row_number() OVER (ORDER BY updated_at DESC, id DESC) AS rn
				FROM craftsky_recent_searches
				WHERE viewer_did = $1
			) ranked WHERE rn > 50
		)`, viewerDID); err != nil {
		return RecentSearchRow{}, fmt.Errorf("prune recent searches: %w", err)
	}
	return row, nil
}

func (s *SearchStore) DeleteRecentSearch(ctx context.Context, viewerDID, id string) error {
	return s.observeDB(ctx, "search.recent.delete", "/v1/search/recent/{id}", func(ctx context.Context) error {
		return s.deleteRecentSearchObserved(ctx, viewerDID, id)
	})
}

func (s *SearchStore) deleteRecentSearchObserved(ctx context.Context, viewerDID, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM craftsky_recent_searches WHERE viewer_did = $1 AND id = $2`, viewerDID, id)
	if err != nil {
		return fmt.Errorf("delete recent search: %w", err)
	}
	return nil
}

func newRecentSearchID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "recent_" + hex.EncodeToString(buf), nil
}
