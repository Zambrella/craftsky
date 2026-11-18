package instagram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func TestPostImportRecordKeyPreservesOriginalImporterAlgorithm(t *testing.T) {
	date := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, clock := range []int64{1, 2, 1023, 1024, 1025} {
		key := bigHexKey(clock)
		got, err := PostImportRecordKey(key, date)
		want := syntax.NewTID(date.Truncate(24*time.Hour).UnixMicro()+clock/1024, uint(clock%1024)).String()
		if err != nil || got.String() != want {
			t.Fatalf("key(%d)=%s %v want %s", clock, got, err, want)
		}
	}
}

func bigHexKey(value int64) string {
	const hex = "0123456789abcdef"
	bytes := []byte("00000000000000000000000000000000")
	for i := len(bytes) - 1; value > 0; i-- {
		bytes[i] = hex[value%16]
		value /= 16
	}
	return string(bytes)
}

func TestPostImportReservationsPinOneIdentityAndRejectWrongFinalization(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	const sourceKey = "abcdef0123456789abcdef0123456789"
	const owner syntax.DID = "did:plc:import-owner"
	proposals := []syntax.RecordKey{"3firstday", "3secondday"}
	selected := make([]syntax.RecordKey, 2)
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := range proposals {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tx, err := pool.Begin(ctx)
			if err != nil {
				errCh <- err
				return
			}
			defer tx.Rollback(ctx)
			selected[i], err = ReservePostImport(ctx, tx, owner, sourceKey, proposals[i])
			if err == nil {
				err = tx.Commit(ctx)
			}
			errCh <- err
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if selected[0] == "" || selected[0] != selected[1] {
		t.Fatalf("concurrent source date changes allocated two keys: %v", selected)
	}
	history := NewPostImportStore(pool)
	items, err := history.Lookup(ctx, owner, []string{sourceKey})
	if err != nil || len(items) != 0 {
		t.Fatalf("pending reservation marked posted=%+v %v", items, err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	item := PostImport{SourceKey: sourceKey, URI: syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/3wrongkey"), Kind: "standard", OperationKey: uuid.New()}
	if FinalizePostImport(ctx, tx, owner, item) == nil {
		t.Fatal("wrong source URI finalized")
	}
	item.URI = syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + selected[0].String())
	if err := FinalizePostImport(ctx, tx, owner, item); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := ReservePostImport(ctx, tx, owner, sourceKey, "3anotherkey"); !errors.Is(err, ErrPostAlreadyImported) {
		t.Fatalf("accepted source was reservable again: %v", err)
	}
	items, err = history.Lookup(ctx, owner, []string{sourceKey})
	if err != nil || len(items) != 1 || items[0].URI != item.URI {
		t.Fatalf("accepted history=%+v %v", items, err)
	}
}
