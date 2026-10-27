package api_test

import (
	"context"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

// IT-001 / REG-004: reindexing an edited future-dated post cannot bump it.
func TestPostEditFutureDatePreservesTimelineActivity(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:author")
	firstSeen := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:author','active',1,1,'test',$1,$1,$1)`, firstSeen); err != nil {
		t.Fatal(err)
	}
	uri := seedPost(t, pool, "did:plc:author", "future", "before", firstSeen)
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES($1,'post','bafycid','clear')`, uri); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at='2099-01-01T00:00:00Z',indexed_at=$2,profile_sort_at=$3,text='after' WHERE uri=$1`, uri, firstSeen.Add(24*time.Hour), firstSeen); err != nil {
		t.Fatal(err)
	}
	rows, _, err := api.NewPostStore(pool).ListTimeline(ctx, "did:plc:author", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("timeline count=%d", len(rows))
	}
	if !rows[0].ActivityAt.Equal(firstSeen) || !rows[0].Post.SortAt().Equal(firstSeen) {
		t.Fatalf("edited post moved: activity=%s cursor=%s want=%s", rows[0].ActivityAt, rows[0].Post.SortAt(), firstSeen)
	}
}

// IT-001 / REG-004: search results must use the same stable chronology as cursors.
func TestPostEditFutureDatePreservesSearchPosition(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:author")
	firstSeen := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:author','active',1,1,'test',$1,$1,$1)`, firstSeen); err != nil {
		t.Fatal(err)
	}
	older := seedPost(t, pool, "did:plc:author", "future", "before", firstSeen)
	newer := seedPost(t, pool, "did:plc:author", "newer", "newer", firstSeen.Add(time.Hour))
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at='2099-01-01T00:00:00Z',indexed_at=$2,text='after' WHERE uri=$1`, older, firstSeen.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET tags=ARRAY['sewing']`); err != nil {
		t.Fatal(err)
	}
	store := api.NewSearchStore(pool, nil)
	for _, sort := range []api.SearchSort{api.SearchSortChronological, api.SearchSortPopular} {
		t.Run(string(sort), func(t *testing.T) {
			rows, _, err := store.SearchHashtagPostsWithLanguages(ctx, "did:plc:author", nil, "sewing", sort, 10, "", firstSeen.Add(48*time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 || rows[0].Post.URI != newer || rows[1].Post.URI != older {
				t.Fatalf("edited post moved in %s search: %+v", sort, rows)
			}
		})
	}
}

// IT-001 / REG-004: comment ordering and its seek cursor share the same anchor.
func TestPostEditFutureDatePreservesCommentPosition(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:author")
	firstSeen := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:author','active',1,1,'test',$1,$1,$1)`, firstSeen); err != nil {
		t.Fatal(err)
	}
	root := seedPost(t, pool, "did:plc:author", "root", "root", firstSeen.Add(-time.Hour))
	older := seedReplyPost(t, pool, "did:plc:author", "future", "before", root, root, firstSeen)
	newer := seedReplyPost(t, pool, "did:plc:author", "newer", "newer", root, root, firstSeen.Add(time.Hour))
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at='2099-01-01T00:00:00Z',indexed_at=$2,text='after' WHERE uri=$1`, older, firstSeen.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	rows, _, err := api.NewPostStore(pool).ListRootComments(ctx, root, "did:plc:author", "newest", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].URI != newer || rows[1].URI != older {
		t.Fatalf("edited comment moved: %+v", rows)
	}
}

// IT-001 / REG-004: project and relevance search also retain chronology.
func TestPostEditFutureDatePreservesProjectSearchPosition(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:author")
	firstSeen := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:author','active',1,1,'test',$1,$1,$1)`, firstSeen); err != nil {
		t.Fatal(err)
	}
	older := seedPost(t, pool, "did:plc:author", "future", "project", firstSeen)
	newer := seedPost(t, pool, "did:plc:author", "newer", "project", firstSeen.Add(time.Hour))
	for _, uri := range []string{older, newer} {
		seedProjectMaterialization(t, pool, uri, api.ProjectCraftTypeSewing, "Coat")
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at='2099-01-01T00:00:00Z',indexed_at=$2 WHERE uri=$1`, older, firstSeen.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	store := api.NewSearchStore(pool, nil)
	for _, tc := range []struct {
		name, query string
		sort        api.SearchSort
	}{
		{"chronological", "", api.SearchSortChronological},
		{"popular", "", api.SearchSortPopular},
		{"relevance", "project", api.SearchSortChronological},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, _, err := store.SearchProjectsWithLanguages(ctx, "did:plc:author", nil, api.ProjectSearchRequest{Query: tc.query, Sort: tc.sort, Limit: 10}, firstSeen.Add(48*time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 || rows[0].Post.URI != newer || rows[1].Post.URI != older {
				t.Fatalf("edited project moved in %s search: %+v", tc.name, rows)
			}
		})
	}
}
