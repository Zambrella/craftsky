package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordingRevenueCatProcessor struct {
	mu        sync.Mutex
	schedules int
	processed int
	called    chan struct{}
	err       error
}

func (processor *recordingRevenueCatProcessor) ScheduleActiveAccounts(context.Context) (int64, error) {
	processor.mu.Lock()
	processor.schedules++
	processor.mu.Unlock()
	return 1, nil
}

func (processor *recordingRevenueCatProcessor) ProcessOne(context.Context) (bool, error) {
	processor.mu.Lock()
	processor.processed++
	processor.mu.Unlock()
	select {
	case processor.called <- struct{}{}:
	default:
	}
	return false, processor.err
}

func TestRevenueCatProcessorLogsOnlyBoundedProviderFailure(t *testing.T) {
	const canary = "provider-private-customer-secret-canary"
	ctx, cancel := context.WithCancel(context.Background())
	processor := &recordingRevenueCatProcessor{called: make(chan struct{}, 1), err: errors.New(canary)}
	var output bytes.Buffer
	done := startRevenueCatProcessor(ctx, processor, slog.New(slog.NewJSONHandler(&output, nil)), time.Hour, time.Hour)
	select {
	case <-processor.called:
	case <-time.After(time.Second):
		t.Fatal("RevenueCat processor did not report provider failure")
	}
	cancel()
	<-done
	if strings.Contains(output.String(), canary) {
		t.Fatalf("worker log leaked provider error: %s", output.String())
	}
	for _, want := range []string{`"component":"revenuecat_reconciliation"`, `"result":"error"`, `"error_category":"provider"`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("worker log missing %s: %s", want, output.String())
		}
	}
}

func TestRevenueCatProcessorStartsOnceSchedulesProcessesAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := &recordingRevenueCatProcessor{called: make(chan struct{}, 4)}
	done := startRevenueCatProcessor(ctx, processor, nil, time.Millisecond, time.Millisecond)

	select {
	case <-processor.called:
	case <-time.After(time.Second):
		t.Fatal("RevenueCat processor did not process owner refresh/retry work")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RevenueCat processor did not stop after cancellation")
	}
	processor.mu.Lock()
	defer processor.mu.Unlock()
	if processor.schedules == 0 || processor.processed == 0 {
		t.Fatalf("RevenueCat processor calls = schedules %d processed %d", processor.schedules, processor.processed)
	}
}
