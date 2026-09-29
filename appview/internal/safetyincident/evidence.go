package safetyincident

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUnauthorized          = errors.New("restricted safety action unauthorized")
	ErrInvalidEvidence       = errors.New("invalid restricted evidence")
	ErrEvidenceStoreBoundary = errors.New("restricted evidence store cannot read objects")
)

type RestrictedObject struct {
	Key   string
	Bytes []byte
}

type ObjectRef struct {
	Key string
}

// EvidenceStore deliberately has no generic read or URL operation. Reads are
// available only through EvidenceService after permission and audit checks.
type EvidenceStore interface {
	Put(context.Context, RestrictedObject) (ObjectRef, error)
	Delete(context.Context, ObjectRef) error
}

type restrictedObjectReader interface {
	open(context.Context, ObjectRef) (io.ReadCloser, error)
}

type MemoryEvidenceStore struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryEvidenceStore() *MemoryEvidenceStore {
	return &MemoryEvidenceStore{objects: make(map[string][]byte)}
}

func (store *MemoryEvidenceStore) Put(_ context.Context, object RestrictedObject) (ObjectRef, error) {
	if store == nil || object.Key == "" || len(object.Bytes) == 0 {
		return ObjectRef{}, ErrInvalidEvidence
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.objects[object.Key] = append([]byte(nil), object.Bytes...)
	return ObjectRef{Key: object.Key}, nil
}

func (store *MemoryEvidenceStore) Delete(_ context.Context, ref ObjectRef) error {
	if store == nil || ref.Key == "" {
		return ErrInvalidEvidence
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.objects, ref.Key)
	return nil
}

func (store *MemoryEvidenceStore) open(_ context.Context, ref ObjectRef) (io.ReadCloser, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	value, ok := store.objects[ref.Key]
	if !ok {
		return nil, errors.New("restricted evidence object not found")
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), value...))), nil
}

func (store *MemoryEvidenceStore) ObjectCount() int {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return len(store.objects)
}

type PreserveCommand struct {
	IncidentID  uuid.UUID
	Bytes       []byte
	SHA256      [sha256.Size]byte
	ContentType string
	Reason      string
	ExpiresAt   time.Time
}

type EvidenceRef struct {
	ID        uuid.UUID
	ObjectKey string
}

type AccessCommand struct {
	EvidenceID uuid.UUID
	Reason     string
}

type EvidenceService struct {
	pool    *pgxpool.Pool
	objects EvidenceStore
	reader  restrictedObjectReader
	now     func() time.Time
}

func NewEvidenceService(pool *pgxpool.Pool, objects EvidenceStore, now func() time.Time) *EvidenceService {
	if now == nil {
		now = time.Now
	}
	reader, _ := objects.(restrictedObjectReader)
	return &EvidenceService{pool: pool, objects: objects, reader: reader, now: now}
}

func (service *EvidenceService) Preserve(ctx context.Context, actor Actor, command PreserveCommand) (EvidenceRef, error) {
	now := service.now().UTC().Truncate(time.Microsecond)
	if service == nil || service.pool == nil || service.objects == nil || command.IncidentID == uuid.Nil ||
		len(command.Bytes) == 0 || command.Reason == "" || !command.ExpiresAt.After(now) ||
		(command.ContentType != "image/jpeg" && command.ContentType != "image/png" && command.ContentType != "image/gif" && command.ContentType != "image/webp") ||
		sha256.Sum256(command.Bytes) != command.SHA256 {
		return EvidenceRef{}, ErrInvalidEvidence
	}
	if !actor.Allowed(PermissionEvidencePreserve, command.IncidentID) {
		return EvidenceRef{}, ErrUnauthorized
	}
	id := uuid.New()
	key := command.IncidentID.String() + "/" + id.String()
	object, err := service.objects.Put(ctx, RestrictedObject{Key: key, Bytes: command.Bytes})
	if err != nil {
		return EvidenceRef{}, fmt.Errorf("store restricted evidence: %w", err)
	}
	if _, err := service.pool.Exec(ctx, `INSERT INTO safety_evidence(
		id,incident_id,object_key,integrity_sha256,content_type,byte_size,preservation_reason,
		created_by,retention_expires_at,created_at
	) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, command.IncidentID, object.Key,
		command.SHA256[:], command.ContentType, len(command.Bytes), command.Reason, actor.ID,
		command.ExpiresAt.UTC(), now); err != nil {
		_ = service.objects.Delete(ctx, object)
		return EvidenceRef{}, fmt.Errorf("record restricted evidence: %w", err)
	}
	return EvidenceRef{ID: id, ObjectKey: object.Key}, nil
}

func (service *EvidenceService) Access(ctx context.Context, actor Actor, command AccessCommand) (io.ReadCloser, error) {
	if service == nil || service.pool == nil || command.EvidenceID == uuid.Nil || command.Reason == "" {
		return nil, ErrInvalidEvidence
	}
	var incidentID uuid.UUID
	var objectKey string
	if err := service.pool.QueryRow(ctx, `SELECT incident_id,object_key FROM safety_evidence WHERE id=$1 AND deleted_at IS NULL`, command.EvidenceID).Scan(&incidentID, &objectKey); err != nil {
		return nil, err
	}
	allowed := actor.Allowed(PermissionEvidenceAccess, incidentID)
	if _, err := service.pool.Exec(ctx, `INSERT INTO safety_evidence_accesses(
		id,evidence_id,actor_id,action,reason,allowed,created_at
	) VALUES($1,$2,$3,'access',$4,$5,$6)`, uuid.New(), command.EvidenceID,
		actor.ID, command.Reason, allowed, service.now().UTC().Truncate(time.Microsecond)); err != nil {
		return nil, fmt.Errorf("audit restricted evidence access: %w", err)
	}
	if !allowed {
		return nil, ErrUnauthorized
	}
	if service.reader == nil {
		return nil, ErrEvidenceStoreBoundary
	}
	return service.reader.open(ctx, ObjectRef{Key: objectKey})
}
