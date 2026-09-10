package subscriptions

import (
	"context"
	"sync"
)

type recordingBillingObserver struct {
	mu          sync.Mutex
	assignments [][2]string
	anomalies   []int
	closures    []string
}

func (observer *recordingBillingObserver) ObserveSubscriptionAssignment(_ context.Context, operation, outcome string) {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.assignments = append(observer.assignments, [2]string{operation, outcome})
}

func (observer *recordingBillingObserver) ObserveSubscriptionAnomalies(_ context.Context, count int) {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.anomalies = append(observer.anomalies, count)
}

func (observer *recordingBillingObserver) ObserveBillingClosure(_ context.Context, outcome string) {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.closures = append(observer.closures, outcome)
}
