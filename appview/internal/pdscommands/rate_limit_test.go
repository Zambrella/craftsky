package pdscommands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/auth"
)

func TestRateLimitJournalFailureRemainsAStorageFailure(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://unused@127.0.0.1/craftsky_test")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close() // fail before networking or writes
	store, err := NewStore(StoreConfig{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	err = store.CompleteRateLimitedDispatch(context.Background(), uuid.New(), &auth.RateLimitError{RetryAfter: time.Minute})
	var limited *auth.RateLimitError
	if err == nil || errors.As(err, &limited) {
		t.Fatalf("journal failure was hidden by throttle guidance: %v", err)
	}
}
