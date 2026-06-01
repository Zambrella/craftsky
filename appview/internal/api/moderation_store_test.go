// appview/internal/api/moderation_store_test.go
package api_test

import (
	"context"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestModerationStore_InsertOutput_PersistsPostAndAccountOutputs(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, moderationFlowMigrationDDL(t))
	store := api.NewModerationStore(pool)
	ctx := context.Background()
	expiresAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	internalReason := "private moderator reason"

	postRow, err := store.InsertOutput(ctx, api.ModerationOutputInput{
		SourceDID:         "did:plc:labeler",
		SubjectType:       api.ModerationSubjectPost,
		SubjectDID:        "did:plc:bob",
		SubjectCollection: ptrString("social.craftsky.feed.post"),
		SubjectRkey:       ptrString("3lf2abc"),
		SubjectURI:        ptrString("at://did:plc:bob/social.craftsky.feed.post/3lf2abc"),
		Value:             api.ModerationValueHide,
		Action:            api.ModerationActionApply,
		InternalReason:    &internalReason,
		ExpiresAt:         &expiresAt,
		CreatedAt:         createdAt,
	})
	if err != nil {
		t.Fatalf("InsertOutput post: %v", err)
	}
	if postRow.ID == "" {
		t.Fatal("post output ID is empty")
	}
	assertModerationOutputRow(t, postRow, api.ModerationOutputRow{
		SourceDID:         "did:plc:labeler",
		SubjectType:       api.ModerationSubjectPost,
		SubjectDID:        "did:plc:bob",
		SubjectCollection: ptrString("social.craftsky.feed.post"),
		SubjectRkey:       ptrString("3lf2abc"),
		SubjectURI:        ptrString("at://did:plc:bob/social.craftsky.feed.post/3lf2abc"),
		Value:             api.ModerationValueHide,
		Action:            api.ModerationActionApply,
		InternalReason:    &internalReason,
		ExpiresAt:         &expiresAt,
		CreatedAt:         createdAt,
	})

	accountRow, err := store.InsertOutput(ctx, api.ModerationOutputInput{
		SourceDID:   "did:plc:labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:bob",
		Value:       api.ModerationValueWarn,
		Action:      api.ModerationActionNegate,
		CreatedAt:   createdAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("InsertOutput account: %v", err)
	}
	if accountRow.ID == "" || accountRow.ID == postRow.ID {
		t.Fatalf("account output ID = %q, post ID = %q", accountRow.ID, postRow.ID)
	}
	assertModerationOutputRow(t, accountRow, api.ModerationOutputRow{
		SourceDID:   "did:plc:labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:bob",
		Value:       api.ModerationValueWarn,
		Action:      api.ModerationActionNegate,
		CreatedAt:   createdAt.Add(time.Minute),
	})

	var storedCount int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM moderation_outputs`).Scan(&storedCount); err != nil {
		t.Fatalf("count moderation outputs: %v", err)
	}
	if storedCount != 2 {
		t.Fatalf("stored outputs = %d, want 2", storedCount)
	}
}

func TestModerationStore_ActivePolicyForSubject_HandlesNegateAndExpiry(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, moderationFlowMigrationDDL(t))
	store := api.NewModerationStore(pool)
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	postURI := "at://did:plc:bob/social.craftsky.feed.post/3lf2abc"

	_, _ = store.InsertOutput(ctx, api.ModerationOutputInput{SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueHide, Action: api.ModerationActionApply, CreatedAt: now.Add(-4 * time.Minute)})
	_, _ = store.InsertOutput(ctx, api.ModerationOutputInput{SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueHide, Action: api.ModerationActionNegate, CreatedAt: now.Add(-3 * time.Minute)})
	_, _ = store.InsertOutput(ctx, api.ModerationOutputInput{SourceDID: "did:plc:ozone", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueWarn, Action: api.ModerationActionApply, CreatedAt: now.Add(-2 * time.Minute)})
	_, _ = store.InsertOutput(ctx, api.ModerationOutputInput{SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueTakedown, Action: api.ModerationActionApply, ExpiresAt: &past, CreatedAt: now.Add(-time.Minute)})

	policy, err := store.ActivePolicyForSubject(ctx, api.ModerationSubjectRef{Type: api.ModerationSubjectPost, DID: "did:plc:bob", URI: &postURI}, now)
	if err != nil {
		t.Fatalf("ActivePolicyForSubject: %v", err)
	}
	if policy.Hidden || !policy.Warning || policy.Value != api.ModerationValueWarn {
		t.Fatalf("policy = %+v, want visible warning", policy)
	}
}

func assertModerationOutputRow(t *testing.T, got *api.ModerationOutputRow, want api.ModerationOutputRow) {
	t.Helper()
	if got.SourceDID != want.SourceDID || got.SubjectType != want.SubjectType || got.SubjectDID != want.SubjectDID || got.Value != want.Value || got.Action != want.Action {
		t.Fatalf("row = %+v, want %+v", got, want)
	}
	assertStringPtr(t, "SubjectCollection", got.SubjectCollection, want.SubjectCollection)
	assertStringPtr(t, "SubjectRkey", got.SubjectRkey, want.SubjectRkey)
	assertStringPtr(t, "SubjectURI", got.SubjectURI, want.SubjectURI)
	assertStringPtr(t, "InternalReason", got.InternalReason, want.InternalReason)
	if got.ExpiresAt == nil || want.ExpiresAt == nil {
		if got.ExpiresAt != want.ExpiresAt {
			t.Fatalf("ExpiresAt = %v, want %v", got.ExpiresAt, want.ExpiresAt)
		}
	} else if !got.ExpiresAt.Equal(*want.ExpiresAt) {
		t.Fatalf("ExpiresAt = %s, want %s", *got.ExpiresAt, *want.ExpiresAt)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if got.IndexedAt.IsZero() {
		t.Fatal("IndexedAt is zero")
	}
}
