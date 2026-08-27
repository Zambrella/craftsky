// appview/internal/scheduledposts/store.go
package scheduledposts

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaximumActivePosts             = 3
	publishingEffectCleanupTimeout = 5 * time.Second
	tidClockIDCount                = 1 << 10
)

var (
	ErrCapacityReached           = errors.New("scheduled post capacity reached")
	ErrScheduleNotFound          = errors.New("scheduled post not found")
	ErrMutationLocked            = errors.New("scheduled post mutation locked")
	ErrWorkerLeaseLost           = errors.New("scheduled post worker lease lost")
	ErrOperationConflict         = errors.New("scheduled post operation conflict")
	ErrScheduledMediaUnavailable = errors.New("scheduled post media unavailable")
	ErrScheduledMediaInvalid     = errors.New("scheduled post media invalid")
)

type Store struct {
	pool     *pgxpool.Pool
	observer OperationalObserver
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) SetOperationalObserver(observer OperationalObserver) {
	if s != nil {
		s.observer = observer
	}
}

func hasDuplicateUUIDs(values []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return true
		}
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
