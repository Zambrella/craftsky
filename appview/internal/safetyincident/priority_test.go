package safetyincident

import (
	"testing"
	"time"
)

func TestPriorityForUsesCredibleUrgencyAndDeadlineNotReportVolume(t *testing.T) {
	now := time.Date(2030, 9, 22, 12, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Minute)
	future := now.Add(time.Hour)

	tests := []struct {
		name  string
		input PriorityInput
		want  int
	}{
		{name: "credible immediate threat", input: PriorityInput{CredibleImmediateThreat: true}, want: PriorityImmediate},
		{name: "overdue", input: PriorityInput{Deadline: &overdue}, want: PriorityDeadline},
		{name: "detection", input: PriorityInput{Detected: true}, want: PriorityDetected},
		{name: "appeal", input: PriorityInput{Appealed: true}, want: PriorityAppeal},
		{name: "upcoming deadline", input: PriorityInput{Deadline: &future}, want: PriorityDeadline},
		{name: "retention expiry", input: PriorityInput{RetentionExpiry: true}, want: PriorityDeadline},
		{name: "awaiting information", input: PriorityInput{AwaitingInformation: true}, want: PriorityRoutine},
		{name: "many reports are not guilt", input: PriorityInput{ReportCount: 1000000}, want: PriorityRoutine},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PriorityFor(test.input, now); got != test.want {
				t.Fatalf("PriorityFor()=%d want %d", got, test.want)
			}
		})
	}
}

func TestPriorityForReportCountCannotRaiseOtherwiseIdenticalWork(t *testing.T) {
	now := time.Now()
	low := PriorityFor(PriorityInput{ReportCount: 1}, now)
	high := PriorityFor(PriorityInput{ReportCount: 9999}, now)
	if low != high {
		t.Fatalf("report volume changed priority: one=%d many=%d", low, high)
	}
}
