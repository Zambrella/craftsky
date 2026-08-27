package followergrowth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWorkerRunTimingAndCancellation(t *testing.T) {
	clock := &controlledClock{now: time.Date(2026, time.August, 25, 10, 0, 0, 0, time.UTC)}
	timers := &controlledTimerFactory{created: make(chan *controlledTimer, 4)}
	capturer := &scriptedCapturer{calls: make(chan captureCall, 4)}
	observer := &recordingWorkerObserver{calls: make(chan workerObservation, 4)}
	worker := NewWorker(capturer, WithObserver(observer), withWorkerTime(clock.Now, timers.New))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()

	startup := receiveCaptureCall(t, capturer.calls)
	if !startup.snapshotDate.Equal(growthDate(2026, time.August, 25)) {
		t.Fatalf("startup date = %s, want 2026-08-25 UTC", startup.snapshotDate)
	}
	startup.respond(CaptureResult{CapturedProfileCount: 3, LatestSuccessfulAge: durationPointer(0)}, nil)
	assertWorkerObservation(t, observer.calls, "success", "none", 3, durationPointer(0))

	midnightTimer := receiveControlledTimer(t, timers.created)
	if midnightTimer.duration != 14*time.Hour {
		t.Fatalf("midnight delay = %s, want 14h", midnightTimer.duration)
	}
	clock.Set(growthDate(2026, time.August, 26))
	midnightTimer.Fire()

	rollover := receiveCaptureCall(t, capturer.calls)
	if !rollover.snapshotDate.Equal(growthDate(2026, time.August, 26)) {
		t.Fatalf("rollover date = %s, want 2026-08-26 UTC", rollover.snapshotDate)
	}
	rollover.respond(CaptureResult{LatestSuccessfulAge: durationPointer(24 * time.Hour)}, errors.New("database unavailable"))
	assertWorkerObservation(t, observer.calls, "error", "capture", 0, durationPointer(24*time.Hour))

	retryTimer := receiveControlledTimer(t, timers.created)
	if retryTimer.duration != 5*time.Minute {
		t.Fatalf("retry delay = %s, want 5m", retryTimer.duration)
	}
	clock.Set(time.Date(2026, time.August, 26, 0, 5, 0, 0, time.UTC))
	retryTimer.Fire()

	retry := receiveCaptureCall(t, capturer.calls)
	if !retry.snapshotDate.Equal(growthDate(2026, time.August, 26)) {
		t.Fatalf("retry date = %s, want current UTC date", retry.snapshotDate)
	}
	retry.respond(CaptureResult{
		AlreadyCompleted: true, CapturedProfileCount: 3,
		LatestSuccessfulAge: durationPointer(24*time.Hour + 5*time.Minute),
	}, nil)
	assertWorkerObservation(t, observer.calls, "already_complete", "none", 3, durationPointer(24*time.Hour+5*time.Minute))
	_ = receiveControlledTimer(t, timers.created)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("worker cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
}

type workerObservation struct {
	result               string
	errorCategory        string
	capturedProfileCount int64
	latestSuccessfulAge  *time.Duration
}

type recordingWorkerObserver struct {
	calls chan workerObservation
}

func (o *recordingWorkerObserver) FollowerGrowthCapture(
	_ context.Context,
	result string,
	errorCategory string,
	_ time.Duration,
	capturedProfileCount int64,
	latestSuccessfulAge *time.Duration,
) {
	o.calls <- workerObservation{
		result:               result,
		errorCategory:        errorCategory,
		capturedProfileCount: capturedProfileCount,
		latestSuccessfulAge:  latestSuccessfulAge,
	}
}

func assertWorkerObservation(
	t *testing.T,
	calls <-chan workerObservation,
	wantResult string,
	wantCategory string,
	wantCount int64,
	wantLatestSuccessfulAge *time.Duration,
) {
	t.Helper()
	select {
	case got := <-calls:
		if got.result != wantResult || got.errorCategory != wantCategory ||
			got.capturedProfileCount != wantCount || !equalDurationPointers(got.latestSuccessfulAge, wantLatestSuccessfulAge) {
			t.Fatalf(
				"worker observation = %+v, want result=%s category=%s count=%d latestSuccessfulAge=%s",
				got, wantResult, wantCategory, wantCount, wantLatestSuccessfulAge,
			)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for worker observation")
	}
}

func durationPointer(value time.Duration) *time.Duration { return &value }

func equalDurationPointers(left, right *time.Duration) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

type captureCall struct {
	snapshotDate time.Time
	capturedAt   time.Time
	response     chan captureResponse
}

func (c captureCall) respond(result CaptureResult, err error) {
	c.response <- captureResponse{result: result, err: err}
}

type captureResponse struct {
	result CaptureResult
	err    error
}

type scriptedCapturer struct {
	calls chan captureCall
}

func (c *scriptedCapturer) Capture(ctx context.Context, snapshotDate, capturedAt time.Time) (CaptureResult, error) {
	call := captureCall{
		snapshotDate: snapshotDate,
		capturedAt:   capturedAt,
		response:     make(chan captureResponse, 1),
	}
	select {
	case c.calls <- call:
	case <-ctx.Done():
		return CaptureResult{}, ctx.Err()
	}
	select {
	case response := <-call.response:
		return response.result, response.err
	case <-ctx.Done():
		return CaptureResult{}, ctx.Err()
	}
}

type controlledClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *controlledClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *controlledClock) Set(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

type controlledTimerFactory struct {
	created chan *controlledTimer
}

func (f *controlledTimerFactory) New(duration time.Duration) workerTimer {
	timer := &controlledTimer{
		duration: duration,
		channel:  make(chan time.Time, 1),
	}
	f.created <- timer
	return timer
}

type controlledTimer struct {
	duration time.Duration
	channel  chan time.Time
}

func (t *controlledTimer) Chan() <-chan time.Time { return t.channel }
func (t *controlledTimer) Stop() bool             { return true }
func (t *controlledTimer) Fire()                  { t.channel <- time.Now() }

func receiveCaptureCall(t *testing.T, calls <-chan captureCall) captureCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for capture")
		return captureCall{}
	}
}

func receiveControlledTimer(t *testing.T, timers <-chan *controlledTimer) *controlledTimer {
	t.Helper()
	select {
	case timer := <-timers:
		return timer
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for timer")
		return nil
	}
}
