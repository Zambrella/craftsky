package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/tap"
)

const RepositoryRepairBatchSize = 100

type RepositoryCollectionRegistry interface {
	Collections() []syntax.NSID
}

type RepositoryRepairAction string

const (
	RepositoryRepairCreate RepositoryRepairAction = "create"
	RepositoryRepairUpdate RepositoryRepairAction = "update"
	RepositoryRepairDelete RepositoryRepairAction = "delete"
	RepositoryRepairNoop   RepositoryRepairAction = "no-op"
)

// RepositoryRepairDescription describes one deterministic comparison result.
type RepositoryRepairDescription struct {
	Collection syntax.NSID
	URI        syntax.ATURI
	Action     RepositoryRepairAction
	CID        syntax.CID
	Record     []byte
}

type RepositoryRepairIngestor interface {
	IngestRecord(context.Context, tap.Event) (tap.Outcome, error)
	ReconcileSource(context.Context, ReconciledSource) (tap.Outcome, error)
}

type RepositoryRepairConfig struct {
	Store     *Store
	Ingestor  RepositoryRepairIngestor
	Projector Projector
}

type RepositoryRepair struct {
	store     *Store
	ingestor  RepositoryRepairIngestor
	projector Projector
}

func NewRepositoryRepair(config RepositoryRepairConfig) (*RepositoryRepair, error) {
	if config.Store == nil || config.Ingestor == nil || config.Projector == nil {
		return nil, errors.New("repository repair requires store, ingestor, and projector")
	}
	return &RepositoryRepair{store: config.Store, ingestor: config.Ingestor, projector: config.Projector}, nil
}

// Apply compares one fully verified repository and runs every winning source
// change through the normal durable ingestion and transactional projector path.
func (repair *RepositoryRepair) Apply(
	ctx context.Context,
	snapshot VerifiedRepositorySnapshot,
	registry RepositoryCollectionRegistry,
) (string, error) {
	if snapshot.verified == nil {
		return "", ErrRepositorySnapshotUnverified
	}
	indexed, err := repair.store.repositoryRepairSources(ctx, snapshot.verified.did, registry)
	if err != nil {
		return "", err
	}
	ownerGeneration, err := repair.store.repositoryRepairOwnerGeneration(ctx, snapshot.verified.did)
	if err != nil {
		return "", err
	}
	descriptions, err := DescribeRepositoryRepair(snapshot, indexed, registry)
	if err != nil {
		return "", err
	}
	current := make(map[syntax.ATURI]SourceRecord, len(indexed))
	for _, source := range indexed {
		current[source.URI] = source
	}

	for start := 0; start < len(descriptions); start += RepositoryRepairBatchSize {
		end := min(start+RepositoryRepairBatchSize, len(descriptions))
		for _, description := range descriptions[start:end] {
			source, exists := current[description.URI]
			if exists && source.Revision > snapshot.verified.revision {
				continue
			}
			generationCurrent := isIndependentBusinessCollection(source.Collection) ||
				(source.ProjectionGeneration != nil && *source.ProjectionGeneration == ownerGeneration)
			if description.Action == RepositoryRepairNoop && source.OrderingStatus == "authoritative" && generationCurrent {
				continue
			}
			if exists {
				present := description.Action != RepositoryRepairDelete
				outcome, err := repair.ingestor.ReconcileSource(ctx, ReconciledSource{
					URI: description.URI, DID: snapshot.verified.did,
					ExpectedEventID: source.SourceEventID, ExpectedFingerprint: source.SourceFingerprint,
					Revision: snapshot.verified.revision, CID: description.CID,
					Record: description.Record, Present: present,
				})
				if err != nil || outcome.Kind != tap.OutcomeApplied {
					if err == nil {
						err = fmt.Errorf("repository repair source %s returned %s/%s", description.URI, outcome.Kind, outcome.Reason)
					}
					return "", err
				}
				continue
			}
			if description.Action != RepositoryRepairCreate {
				continue
			}
			event := tap.Event{
				ID:  deterministicRepairEventID(snapshot.verified, description),
				URI: description.URI, DID: snapshot.verified.did,
				Collection: description.Collection, Rkey: description.URI.RecordKey(),
				Rev: snapshot.verified.revision, CID: description.CID,
				Action: "create", Record: append([]byte(nil), description.Record...),
			}
			outcome, err := repair.ingestor.IngestRecord(ctx, event)
			if err != nil || outcome.Kind != tap.OutcomeApplied {
				if err == nil {
					err = fmt.Errorf("repository repair source %s returned %s/%s", description.URI, outcome.Kind, outcome.Reason)
				}
				return "", err
			}
		}
	}

	// Re-read after durable application so a retry also resumes projection jobs
	// created by a prior attempt that stopped between ingestion and projection.
	indexed, err = repair.store.repositoryRepairSources(ctx, snapshot.verified.did, registry)
	if err != nil {
		return "", err
	}
	for start := 0; start < len(indexed); start += RepositoryRepairBatchSize {
		end := min(start+RepositoryRepairBatchSize, len(indexed))
		for _, source := range indexed[start:end] {
			if source.Revision > snapshot.verified.revision {
				continue
			}
			claim, ok, err := repair.store.claimProjectionSource(ctx, source.URI)
			if err != nil {
				return "", err
			}
			if ok {
				if err := repair.store.Project(ctx, claim, repair.projector); err != nil {
					return "", fmt.Errorf("project repository repair source %s: %w", source.URI, err)
				}
			}
		}
	}
	return snapshot.verified.revision.String(), nil
}

func (store *Store) repositoryRepairOwnerGeneration(ctx context.Context, did syntax.DID) (int64, error) {
	var generation int64
	if err := store.pool.QueryRow(ctx, `
		SELECT generation FROM owner_lifecycles WHERE owner_did=$1
	`, did).Scan(&generation); err != nil {
		return 0, fmt.Errorf("read repository repair owner generation: %w", err)
	}
	return generation, nil
}

func deterministicRepairEventID(snapshot *verifiedRepositorySnapshot, description RepositoryRepairDescription) uint64 {
	digest := sha256.Sum256([]byte(snapshot.did.String() + "\x00" + snapshot.root.String() + "\x00" +
		snapshot.revision.String() + "\x00" + string(description.Action) + "\x00" +
		description.URI.String() + "\x00" + description.CID.String()))
	return (binary.BigEndian.Uint64(digest[:8]) & ((1 << 63) - 2)) + 1
}

func (store *Store) repositoryRepairSources(ctx context.Context, did syntax.DID, registry RepositoryCollectionRegistry) ([]SourceRecord, error) {
	if registry == nil {
		return nil, errors.New("repository collection registry is required")
	}
	collections := registry.Collections()
	slices.Sort(collections)
	collections = slices.Compact(collections)
	rows, err := store.pool.Query(ctx, sourceSelect+` WHERE did=$1 AND collection=ANY($2) ORDER BY uri`, did, collections)
	if err != nil {
		return nil, fmt.Errorf("list repository repair sources: %w", err)
	}
	defer rows.Close()
	sources := make([]SourceRecord, 0)
	for rows.Next() {
		source, err := sourceRow(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repository repair sources: %w", err)
	}
	return sources, nil
}

func (store *Store) claimProjectionSource(ctx context.Context, uri syntax.ATURI) (ProjectionClaim, bool, error) {
	now := store.now().UTC().Truncate(time.Microsecond)
	claim := ProjectionClaim{ProjectionJob: ProjectionJob{LeaseOwner: "repository-repair", LeaseToken: uuid.New(), LeaseExpiresAt: now.Add(time.Minute)}}
	err := store.pool.QueryRow(ctx, `
		UPDATE tap_projection_jobs AS job
		SET state='processing',attempts=job.attempts+1,dependency_kind=NULL,dependency_key=NULL,
		    lease_owner=$2,lease_token=$3,lease_expires_at=$4,updated_at=$1
		FROM tap_source_records AS source
		WHERE job.source_uri=$5 AND source.uri=job.source_uri
		  AND source.source_event_id=job.source_event_id
		  AND (job.state IN ('pending','blocked')
		       OR (job.state='processing' AND job.lease_expires_at<=$1))
		RETURNING job.id,job.source_uri,job.projection_kind,job.source_event_id,
		          job.state,job.attempts,job.lease_owner,job.lease_token,job.lease_expires_at
	`, now, claim.LeaseOwner, claim.LeaseToken, claim.LeaseExpiresAt, uri).Scan(
		&claim.ID, &claim.SourceURI, &claim.ProjectionKind, &claim.SourceEventID,
		&claim.State, &claim.Attempts, &claim.LeaseOwner, &claim.LeaseToken, &claim.LeaseExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		var state string
		var currentEventID uint64
		if readErr := store.pool.QueryRow(ctx, `
			SELECT job.state,source.source_event_id
			FROM tap_projection_jobs AS job
			JOIN tap_source_records AS source ON source.uri=job.source_uri
			WHERE job.source_uri=$1
		`, uri).Scan(&state, &currentEventID); readErr != nil {
			return ProjectionClaim{}, false, fmt.Errorf("read repository repair projection %s: %w", uri, readErr)
		}
		if state == "complete" || state == "permanent_denied" {
			return ProjectionClaim{}, false, nil
		}
		return ProjectionClaim{}, false, fmt.Errorf("repository repair projection %s is not settled", uri)
	}
	if err != nil {
		return ProjectionClaim{}, false, fmt.Errorf("claim repository repair projection %s: %w", uri, err)
	}
	return claim, true, nil
}

// DescribeRepositoryRepair performs no I/O and mutates no projection state.
func DescribeRepositoryRepair(
	snapshot VerifiedRepositorySnapshot,
	indexed []SourceRecord,
	registry RepositoryCollectionRegistry,
) ([]RepositoryRepairDescription, error) {
	if snapshot.verified == nil {
		return nil, ErrRepositorySnapshotUnverified
	}
	if registry == nil {
		return nil, fmt.Errorf("repository collection registry is required")
	}

	collections := registry.Collections()
	slices.Sort(collections)
	collections = slices.Compact(collections)
	registered := make(map[syntax.NSID]struct{}, len(collections))
	for _, collection := range collections {
		registered[collection] = struct{}{}
	}

	authoritative := make(map[syntax.ATURI]RepositorySnapshotRecord)
	for _, record := range snapshot.verified.records {
		uri, err := syntax.ParseATURI(record.URI.String())
		if err != nil || uri.Authority().DID() != snapshot.verified.did {
			continue
		}
		if _, ok := registered[uri.Collection()]; !ok {
			continue
		}
		authoritative[uri] = record
	}

	current := make(map[syntax.ATURI]SourceRecord)
	for _, source := range indexed {
		if source.DID != snapshot.verified.did || source.Action == "delete" {
			continue
		}
		if _, ok := registered[source.Collection]; !ok {
			continue
		}
		current[source.URI] = source
	}

	uris := make([]syntax.ATURI, 0, len(authoritative)+len(current))
	for uri := range authoritative {
		uris = append(uris, uri)
	}
	for uri := range current {
		if _, exists := authoritative[uri]; !exists {
			uris = append(uris, uri)
		}
	}
	slices.Sort(uris)

	descriptions := make([]RepositoryRepairDescription, 0, len(uris))
	for _, uri := range uris {
		record, inSnapshot := authoritative[uri]
		source, inAppView := current[uri]
		description := RepositoryRepairDescription{URI: uri}
		switch {
		case inSnapshot:
			description.Collection = uri.Collection()
			description.CID = record.CID
			description.Record = append([]byte(nil), record.Record...)
		case inAppView:
			description.Collection = source.Collection
		}
		switch {
		case inSnapshot && !inAppView:
			description.Action = RepositoryRepairCreate
		case inSnapshot && source.CID != record.CID:
			description.Action = RepositoryRepairUpdate
		case !inSnapshot && inAppView:
			description.Action = RepositoryRepairDelete
		default:
			description.Action = RepositoryRepairNoop
		}
		descriptions = append(descriptions, description)
	}
	return descriptions, nil
}
