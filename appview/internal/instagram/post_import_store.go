package instagram

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var postImportSourceKeyPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func ValidPostImportSourceKey(key string) bool { return postImportSourceKeyPattern.MatchString(key) }

// PostImportRecordKey matches the original local importer's deterministic TID
// algorithm for the first publication. A collision is rejected by the command
// executor, never overwritten.
func PostImportRecordKey(sourceKey string, createdAt time.Time) (syntax.RecordKey, error) {
	if !ValidPostImportSourceKey(sourceKey) {
		return "", errors.New("invalid Instagram source key")
	}
	identity, _ := new(big.Int).SetString(sourceKey, 16)
	const microsecondsPerDay int64 = 86_400_000_000
	const clockSlots int64 = 1024
	slot := new(big.Int).Mod(identity, big.NewInt(microsecondsPerDay*clockSlots)).Int64()
	day := createdAt.UTC().Truncate(24 * time.Hour)
	return syntax.RecordKey(syntax.NewTID(day.UnixMicro()+slot/clockSlots, uint(slot%clockSlots)).String()), nil
}

type PostImport struct {
	OperationKey uuid.UUID    `json:"-"`
	SourceKey    string       `json:"sourceKey"`
	URI          syntax.ATURI `json:"uri"`
	Kind         string       `json:"kind"`
}

type PostImportStore struct{ pool *pgxpool.Pool }

func NewPostImportStore(pool *pgxpool.Pool) *PostImportStore { return &PostImportStore{pool: pool} }

func (s *PostImportStore) Lookup(ctx context.Context, owner syntax.DID, sourceKeys []string) ([]PostImport, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("instagram post import store unavailable")
	}
	rows, err := s.pool.Query(ctx, `SELECT i.source_key,i.post_uri,i.kind,i.operation_key FROM instagram_post_imports i
		WHERE i.owner_did=$1 AND i.source_key=ANY($2::text[]) AND i.post_uri IS NOT NULL
		AND NOT EXISTS (SELECT 1 FROM post_publication_state p
			WHERE p.uri=i.post_uri AND p.owner_did=i.owner_did AND p.origin_status='deleted')
		ORDER BY i.source_key`, owner, sourceKeys)
	if err != nil {
		return nil, fmt.Errorf("lookup Instagram post imports: %w", err)
	}
	defer rows.Close()
	items := []PostImport{}
	for rows.Next() {
		var item PostImport
		if err := rows.Scan(&item.SourceKey, &item.URI, &item.Kind, &item.OperationKey); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var ErrPostAlreadyImported = errors.New("instagram source was already imported")

// ReservePostImport pins one create-only URI per source inside the append
// preparation transaction. A confirmed deletion retires that URI, even if the
// original PDS write was never acknowledged.
func ReservePostImport(ctx context.Context, tx pgx.Tx, owner syntax.DID, sourceKey string, proposed syntax.RecordKey, newRecordKey func() (syntax.RecordKey, error)) (syntax.RecordKey, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO instagram_post_imports(owner_did,source_key,selected_rkey)
		VALUES ($1,$2,$3) ON CONFLICT(owner_did,source_key) DO NOTHING`, owner, sourceKey, proposed); err != nil {
		return "", err
	}
	var selected syntax.RecordKey
	var publishedURI *syntax.ATURI
	if err := tx.QueryRow(ctx, `SELECT selected_rkey,post_uri FROM instagram_post_imports
		WHERE owner_did=$1 AND source_key=$2 FOR UPDATE`, owner, sourceKey).Scan(&selected, &publishedURI); err != nil {
		return "", err
	}
	selectedURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + selected.String())
	if publishedURI != nil {
		selectedURI = *publishedURI
	}
	var deleted bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM post_publication_state
		WHERE uri=$1 AND owner_did=$2 AND origin_status='deleted')`, selectedURI, owner).Scan(&deleted); err != nil {
		return "", err
	}
	if !deleted {
		if publishedURI != nil {
			return "", ErrPostAlreadyImported
		}
		return selected, nil
	}
	next, err := newRecordKey()
	if err != nil {
		return "", err
	}
	if next == selected {
		return "", errors.New("Instagram reimport requires a fresh record key")
	}
	_, err = tx.Exec(ctx, `UPDATE instagram_post_imports SET selected_rkey=$3,post_uri=NULL,kind=NULL,operation_key=NULL
		WHERE owner_did=$1 AND source_key=$2`, owner, sourceKey, next)
	return next, err
}

// PostImportReservationActive runs under the chosen URI's publication fence.
// Attempts prepared before a deletion can never recreate the retired record.
func PostImportReservationActive(ctx context.Context, tx pgx.Tx, owner syntax.DID, sourceKey string, uri syntax.ATURI) (bool, error) {
	var active bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM instagram_post_imports i
		WHERE i.owner_did=$1 AND i.source_key=$2 AND i.selected_rkey=$3 AND i.post_uri IS NULL
		AND NOT EXISTS (SELECT 1 FROM post_publication_state p
			WHERE p.uri=$4 AND p.owner_did=i.owner_did AND p.origin_status='deleted')
	)`, owner, sourceKey, uri.RecordKey(), uri).Scan(&active)
	return active, err
}

// FinalizePostImport runs only within the accepted command transaction, under
// its owner-lifecycle fence. Never create history for an unconfirmed PDS write.
func FinalizePostImport(ctx context.Context, tx pgx.Tx, owner syntax.DID, item PostImport) error {
	result, err := tx.Exec(ctx, `UPDATE instagram_post_imports SET post_uri=$3,kind=$4,
		operation_key=COALESCE(operation_key,$5) WHERE owner_did=$1 AND source_key=$2
		AND selected_rkey=$6 AND (post_uri IS NULL OR post_uri=$3)`, owner, item.SourceKey, item.URI, item.Kind, item.OperationKey, item.URI.RecordKey())
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("instagram import identity conflicts with accepted record")
	}
	return nil
}
