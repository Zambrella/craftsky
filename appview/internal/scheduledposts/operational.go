package scheduledposts

import (
	"context"
	"fmt"
	"time"
)

type OperationalObserver interface {
	ObserveScheduledQueue(status string, count, due, overdue int, oldestDueAge time.Duration)
	ObserveScheduledOperation(operation, result, errorClass string, duration time.Duration)
	ObserveScheduledPublication(attempt int, startLatency, duration time.Duration)
	ObserveScheduledCleanupQueue(pending int, oldestAge time.Duration)
}

type ScheduledQueueSnapshot struct {
	StatusCounts map[Status]int
	Due          int
	Overdue      int
	OldestDueAge time.Duration
}

type CleanupQueueSnapshot struct {
	Pending   int
	OldestAge time.Duration
}

type scheduledQueueSnapshotter interface {
	ScheduledQueueSnapshot(context.Context, time.Time) (ScheduledQueueSnapshot, error)
}

type cleanupQueueSnapshotter interface {
	CleanupQueueSnapshot(context.Context, time.Time) (CleanupQueueSnapshot, error)
}

func (s *Store) ScheduledQueueSnapshot(
	ctx context.Context,
	now time.Time,
) (ScheduledQueueSnapshot, error) {
	if s == nil || s.pool == nil || now.IsZero() {
		return ScheduledQueueSnapshot{}, fmt.Errorf("invalid scheduled queue snapshot")
	}
	snapshot := ScheduledQueueSnapshot{StatusCounts: map[Status]int{}}
	var scheduled, publishing, retrying, needsAttention int
	var oldestDueSeconds float64
	err := s.pool.QueryRow(ctx, scheduledQueueSnapshotSQL, now.UTC()).Scan(
		&scheduled,
		&publishing,
		&retrying,
		&needsAttention,
		&snapshot.Due,
		&snapshot.Overdue,
		&oldestDueSeconds,
	)
	if err != nil {
		return ScheduledQueueSnapshot{}, fmt.Errorf("read scheduled queue snapshot: %w", err)
	}
	snapshot.StatusCounts[StatusScheduled] = scheduled
	snapshot.StatusCounts[StatusPublishing] = publishing
	snapshot.StatusCounts[StatusRetrying] = retrying
	snapshot.StatusCounts[StatusNeedsAttention] = needsAttention
	snapshot.OldestDueAge = time.Duration(oldestDueSeconds * float64(time.Second))
	return snapshot, nil
}

func (s *Store) CleanupQueueSnapshot(
	ctx context.Context,
	now time.Time,
) (CleanupQueueSnapshot, error) {
	if s == nil || s.pool == nil || now.IsZero() {
		return CleanupQueueSnapshot{}, fmt.Errorf("invalid scheduled cleanup queue snapshot")
	}
	var snapshot CleanupQueueSnapshot
	var oldestSeconds float64
	if err := s.pool.QueryRow(ctx, cleanupQueueSnapshotSQL, now.UTC()).Scan(
		&snapshot.Pending,
		&oldestSeconds,
	); err != nil {
		return CleanupQueueSnapshot{}, fmt.Errorf("read scheduled cleanup queue snapshot: %w", err)
	}
	snapshot.OldestAge = time.Duration(oldestSeconds * float64(time.Second))
	return snapshot, nil
}

func observeScheduledQueue(observer OperationalObserver, snapshot ScheduledQueueSnapshot) {
	if observer == nil {
		return
	}
	for _, status := range []Status{
		StatusScheduled,
		StatusPublishing,
		StatusRetrying,
		StatusNeedsAttention,
	} {
		observer.ObserveScheduledQueue(
			string(status),
			snapshot.StatusCounts[status],
			snapshot.Due,
			snapshot.Overdue,
			snapshot.OldestDueAge,
		)
	}
}
