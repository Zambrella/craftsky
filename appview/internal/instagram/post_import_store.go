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
// algorithm. All edits and attempts for one source address one create-only
// record; a collision is rejected by the command executor, never overwritten.
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
	rows, err := s.pool.Query(ctx, `SELECT source_key,post_uri,kind,operation_key FROM instagram_post_imports
		WHERE owner_did=$1 AND source_key=ANY($2::text[]) AND post_uri IS NOT NULL ORDER BY source_key`, owner, sourceKeys)
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
// preparation transaction. Rejected attempts keep that identity available for
// corrected content; ambiguous attempts cannot allocate a second identity.
func ReservePostImport(ctx context.Context, tx pgx.Tx, owner syntax.DID, sourceKey string, proposed syntax.RecordKey) (syntax.RecordKey, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO instagram_post_imports(owner_did,source_key,selected_rkey)
		VALUES ($1,$2,$3) ON CONFLICT(owner_did,source_key) DO NOTHING`, owner, sourceKey, proposed); err != nil {
		return "", err
	}
	var selected syntax.RecordKey
	var uri *string
	if err := tx.QueryRow(ctx, `SELECT selected_rkey,post_uri FROM instagram_post_imports
		WHERE owner_did=$1 AND source_key=$2 FOR UPDATE`, owner, sourceKey).Scan(&selected, &uri); err != nil {
		return "", err
	}
	if uri != nil {
		return "", ErrPostAlreadyImported
	}
	return selected, nil
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
