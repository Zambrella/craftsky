package followergrowth

import (
	"context"
	"time"
)

const captureRetryDelay = 5 * time.Minute

type Capturer interface {
	Capture(ctx context.Context, snapshotDate, capturedAt time.Time) (CaptureResult, error)
}

type WorkerObserver interface {
	FollowerGrowthCapture(context.Context, string, string, time.Duration, int64, *time.Duration)
}

type workerTimer interface {
	Chan() <-chan time.Time
	Stop() bool
}

type realWorkerTimer struct {
	*time.Timer
}

func (t realWorkerTimer) Chan() <-chan time.Time { return t.C }

type Worker struct {
	capturer Capturer
	now      func() time.Time
	newTimer func(time.Duration) workerTimer
	observer WorkerObserver
}

type workerOption func(*Worker)

func withWorkerTime(now func() time.Time, newTimer func(time.Duration) workerTimer) workerOption {
	return func(worker *Worker) {
		worker.now = now
		worker.newTimer = newTimer
	}
}

func WithObserver(observer WorkerObserver) workerOption {
	return func(worker *Worker) {
		worker.observer = observer
	}
}

func NewWorker(capturer Capturer, options ...workerOption) *Worker {
	worker := &Worker{
		capturer: capturer,
		now:      time.Now,
		newTimer: func(delay time.Duration) workerTimer { return realWorkerTimer{time.NewTimer(delay)} },
	}
	for _, option := range options {
		option(worker)
	}
	return worker
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		now := w.now()
		utcNow := now.UTC()
		snapshotDate := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
		startedAt := w.now()
		result, err := w.capturer.Capture(ctx, snapshotDate, now)
		finishedAt := w.now()
		if ctx.Err() != nil {
			return nil
		}
		if w.observer != nil {
			outcome := "success"
			category := "none"
			if err != nil {
				outcome = "error"
				category = "capture"
			} else if result.AlreadyCompleted {
				outcome = "already_complete"
			}
			w.observer.FollowerGrowthCapture(
				ctx,
				outcome,
				category,
				nonNegativeDuration(finishedAt.Sub(startedAt)),
				result.CapturedProfileCount,
				result.LatestSuccessfulAge,
			)
		}

		delay := captureRetryDelay
		if err == nil {
			nextMidnight := snapshotDate.AddDate(0, 0, 1)
			delay = nextMidnight.Sub(w.now().UTC())
			if delay < 0 {
				delay = 0
			}
		}

		timer := w.newTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.Chan():
			timer.Stop()
		}
	}
}
