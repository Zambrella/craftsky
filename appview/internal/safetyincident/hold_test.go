package safetyincident

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateHoldRequiresExplicitScopeBasisApproverAndExpiry(t *testing.T) {
	t.Parallel()
	now := time.Date(2030, 9, 22, 18, 0, 0, 0, time.UTC)
	valid := HoldRequest{
		IncidentID: uuid.New(), EvidenceIDs: []uuid.UUID{uuid.New()}, Basis: "preservation duty",
		ApprovedBy: "safety-admin-2", CreatedBy: "safety-admin-1", CreatedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	tests := []struct {
		name    string
		mutate  func(*HoldRequest)
		wantErr error
	}{
		{name: "valid"},
		{name: "missing incident", mutate: func(r *HoldRequest) { r.IncidentID = uuid.Nil }, wantErr: ErrInvalidHold},
		{name: "missing evidence scope", mutate: func(r *HoldRequest) { r.EvidenceIDs = nil }, wantErr: ErrInvalidHold},
		{name: "duplicate evidence scope", mutate: func(r *HoldRequest) { r.EvidenceIDs = append(r.EvidenceIDs, r.EvidenceIDs[0]) }, wantErr: ErrInvalidHold},
		{name: "missing basis", mutate: func(r *HoldRequest) { r.Basis = "" }, wantErr: ErrInvalidHold},
		{name: "missing approver", mutate: func(r *HoldRequest) { r.ApprovedBy = "" }, wantErr: ErrInvalidHold},
		{name: "missing creator", mutate: func(r *HoldRequest) { r.CreatedBy = "" }, wantErr: ErrInvalidHold},
		{name: "missing creation time", mutate: func(r *HoldRequest) { r.CreatedAt = time.Time{} }, wantErr: ErrInvalidHold},
		{name: "indefinite hold", mutate: func(r *HoldRequest) { r.ExpiresAt = time.Time{} }, wantErr: ErrInvalidHold},
		{name: "expired at creation", mutate: func(r *HoldRequest) { r.ExpiresAt = r.CreatedAt }, wantErr: ErrInvalidHold},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := valid
			request.EvidenceIDs = append([]uuid.UUID(nil), valid.EvidenceIDs...)
			if test.mutate != nil {
				test.mutate(&request)
			}
			if err := ValidateHold(request); !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateHold()=%v, want %v", err, test.wantErr)
			}
		})
	}
}
