package api_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/puddle/v2"

	"social.craftsky/appview/internal/api"
)

// This is the AV-026 acquisition-error seam for the shared chronological and
// popular post query. It must fail before any method is called on invalid rows.
func TestSearchStorePostQueryAcquisitionErrorDoesNotUseInvalidRows(t *testing.T) {
	config, err := pgxpool.ParseConfig("postgres://test:test@127.0.0.1:1/test")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	store := api.NewSearchStore(pool, nil)

	for _, sort := range []api.SearchSort{api.SearchSortChronological, api.SearchSortPopular} {
		t.Run(string(sort), func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("search used rows after query acquisition failed: %v", recovered)
				}
			}()

			_, _, err := store.SearchHashtagPostsWithLanguages(
				context.Background(),
				"did:plc:viewer",
				[]string{},
				"knitting",
				sort,
				10,
				"",
				time.Now().UTC(),
			)
			if !errors.Is(err, puddle.ErrClosedPool) {
				t.Fatalf("error = %v, want closed pool", err)
			}
			if !strings.Contains(err.Error(), "search hashtag posts: closed pool") {
				t.Fatalf("error = %q, want query acquisition context", err)
			}
		})
	}
}
