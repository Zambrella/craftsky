package scheduledposts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestUnpublishedScheduleAndPreviewRemainOwnerPrivate(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	media, _ := newScheduledTestMediaService(t, store, objects)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.New()
	privateBytes := []byte("private-image-canary")
	staged, err := media.Put(ctx, PutPrivateMediaParams{
		ID: mediaID, OwnerDID: alice, OwnerGeneration: 1,
		MIMEType: "image/jpeg", Bytes: privateBytes, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePayload(Payload{
		Kind: PostKindStandard, Text: "private-text-canary",
		Facets: json.RawMessage(`[{"feature":"private-facet-canary"}]`),
		Media:  []PayloadMedia{{ID: mediaID.String(), Alt: "private-alt-canary"}},
	})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: alice, OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now.Add(time.Hour),
		PayloadBytes: payload, PayloadHash: [32]byte{2}, PayloadVersion: 1,
		MediaIDs: []uuid.UUID{mediaID},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(ctx, bob, created.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("foreign get error=%v", err)
	}
	if err := store.Delete(ctx, bob, created.ID, now); err != nil {
		t.Fatalf("foreign idempotent delete error=%v", err)
	}
	if _, err := media.Open(ctx, bob, mediaID); !errors.Is(err, ErrScheduledMediaNotFound) {
		t.Fatalf("foreign preview error=%v", err)
	}
	if err := media.Delete(ctx, bob, mediaID, now, 1); err != nil {
		t.Fatalf("foreign idempotent media delete disclosed state: %v", err)
	}
	owned, err := store.Get(ctx, alice, created.ID)
	if err != nil || !bytes.Equal(owned.PayloadBytes, payload) {
		t.Fatalf("foreign operations changed Alice's schedule: resource=%#v err=%v", owned, err)
	}
	preview, err := media.Open(ctx, alice, mediaID)
	if err != nil {
		t.Fatal(err)
	}
	defer preview.Body.Close()
	gotBytes, err := io.ReadAll(preview.Body)
	if err != nil || !bytes.Equal(gotBytes, privateBytes) {
		t.Fatalf("owner preview=%q err=%v", gotBytes, err)
	}

	diagnostics := fmt.Sprint(SafeDiagnosticFields(
		DiagnosticOperationPublish, DiagnosticResultFailure, "pds_unavailable",
	))
	output := staged.ObjectKey + diagnostics
	for _, canary := range []string{
		"private-text-canary", "private-facet-canary", "private-alt-canary",
		"private-image-canary", alice.String(), "owner-session-token",
	} {
		if strings.Contains(output, canary) {
			t.Fatalf("private canary %q leaked into operational metadata %q", canary, output)
		}
	}
}
