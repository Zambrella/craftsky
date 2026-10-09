package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/testdb"
)

func TestSearchImprovementsEnglishWordForms(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:author")
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	post := seedPost(t, pool, "did:plc:author", "english", "Finished knitting socks", now)
	project := seedSearchProject(t, pool, "did:plc:author", "project", "", "knitting", "Finished knitting socks", now)
	unknown := seedPost(t, pool, "did:plc:author", "unknown", "Finished knitting socks", now)
	if _, err := pool.Exec(ctx, "UPDATE craftsky_posts SET langs = ARRAY['en'] WHERE uri = ANY($1)", []string{post, project}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	store := api.NewSearchStore(pool, nil)
	for _, query := range []string{"sock", "knit"} {
		t.Run(query, func(t *testing.T) {
			posts, _, err := store.SearchPostsWithLanguages(ctx, "did:plc:viewer", nil, api.PostSearchRequest{Query: query, Limit: 10}, now)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(searchURIs(posts), post) {
				t.Fatalf("IT-001 English post missing for %q: %v", query, searchURIs(posts))
			}
			if query == "knit" && slices.Contains(searchURIs(posts), unknown) {
				t.Fatal("unknown-language post received English stemming")
			}
			projects, _, err := store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", nil, api.ProjectSearchRequest{Query: query, Limit: 10}, now)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(searchURIs(projects), project) {
				t.Fatalf("IT-001 English project missing for %q: %v", query, searchURIs(projects))
			}
		})
	}
}

// UT-001 / FR-001 / AC-002–003: filler removal must not erase craft terms.
func TestSearchImprovementsNaturalWording(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:author")
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	post := seedPost(t, pool, "did:plc:author", "post", "Finished knitting socks", now)
	project := seedSearchProject(t, pool, "did:plc:author", "project", "", "knitting", "Finished knitting socks", now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET langs = ARRAY['en']; INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	store := api.NewSearchStore(pool, nil)
	for _, query := range []string{"how to knit socks", "how to", "how", "DK socks"} {
		t.Run(query, func(t *testing.T) {
			posts, _, err := store.SearchPostsWithLanguages(ctx, "did:plc:viewer", nil, api.PostSearchRequest{Query: query, Limit: 10}, now)
			if err != nil {
				t.Fatal(err)
			}
			projects, _, err := store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", nil, api.ProjectSearchRequest{Query: query, Limit: 10}, now)
			if err != nil {
				t.Fatal(err)
			}
			if query == "how to knit socks" {
				if !slices.Equal(searchURIs(posts), []string{post}) || !slices.Equal(searchURIs(projects), []string{project}) {
					t.Fatalf("topic showcase missing: %v / %v", searchURIs(posts), searchURIs(projects))
				}
			} else if len(posts)+len(projects) != 0 {
				t.Fatalf("empty/filler or missing DK concept admitted results: %v / %v", searchURIs(posts), searchURIs(projects))
			}
		})
	}
}

func searchFixture(t *testing.T) (*pgxpool.Pool, *api.SearchStore, time.Time) {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:author")
	return pool, api.NewSearchStore(pool, nil), time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
}
func searchFixturePost(t *testing.T, pool *pgxpool.Pool, project bool, key, text string, langs []string, now time.Time) string {
	t.Helper()
	var uri string
	if project {
		uri = seedSearchProject(t, pool, "did:plc:author", key, "", "knitting", text, now)
	} else {
		uri = seedPost(t, pool, "did:plc:author", key, text, now)
	}
	if langs == nil {
		langs = []string{}
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_posts SET langs=$2 WHERE uri=$1`, uri, langs); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES($1,'post','bafycid','clear')`, uri); err != nil {
		t.Fatal(err)
	}
	return uri
}
func submittedSearch(t *testing.T, store *api.SearchStore, project bool, q string, limit int, cursor string, now time.Time) ([]api.SearchPostRow, string) {
	t.Helper()
	var rows []api.SearchPostRow
	var next string
	var err error
	if project {
		rows, next, err = store.SearchProjectsWithLanguages(context.Background(), "did:plc:viewer", nil, api.ProjectSearchRequest{Query: q, Limit: limit, Cursor: cursor}, now)
	} else {
		rows, next, err = store.SearchPostsWithLanguages(context.Background(), "did:plc:viewer", nil, api.PostSearchRequest{Query: q, Limit: limit, Cursor: cursor}, now)
	}
	if err != nil {
		t.Fatal(err)
	}
	return rows, next
}

// UT-002 / FR-001 / AC-003.
func TestSearchImprovementsLanguageAndLiteralNames(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprint(project), func(t *testing.T) {
			pool, store, now := searchFixture(t)
			want := []string{}
			for i, langs := range [][]string{{"en"}, {"en-GB"}, {"fr", "en"}, {"fr"}, nil} {
				uri := searchFixturePost(t, pool, project, fmt.Sprint(i), "Flax Light DK K2 knitting", langs, now)
				if i < 3 {
					want = append(want, uri)
				}
			}
			rows, _ := submittedSearch(t, store, project, "knit", 10, "", now)
			got := searchURIs(rows)
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("language stemming = %v want %v", got, want)
			}
			for _, q := range []string{"Flax Light", "DK", "K2", "KNITTING!"} {
				rows, _ := submittedSearch(t, store, project, q, 10, "", now)
				if len(rows) != 5 {
					t.Fatalf("literal %q got %d", q, len(rows))
				}
			}
		})
	}
}

// UT-003 / FR-002 / AC-004: all curated groups, both directions and tabs.
func TestSearchImprovementsCraftEquivalents(t *testing.T) {
	for _, project := range []bool{false, true} {
		for _, pair := range [][2]string{{"jumper", "sweater"}, {"wip", "work in progress"}, {"stockinette", "stocking stitch"}} {
			for direction := range 2 {
				q, source := pair[direction], pair[1-direction]
				t.Run(fmt.Sprintf("%t/%s", project, q), func(t *testing.T) {
					pool, store, now := searchFixture(t)
					useful := searchFixturePost(t, pool, project, "useful", source+" socks", []string{"en"}, now)
					searchFixturePost(t, pool, project, "missing", source+" hat", []string{"en"}, now)
					searchFixturePost(t, pool, project, "association", "cardigan yarn garter socks", []string{"en"}, now)
					rows, _ := submittedSearch(t, store, project, q+" socks", 10, "", now)
					if !slices.Equal(searchURIs(rows), []string{useful}) {
						t.Fatalf("equivalent missing or concepts dropped: %v", searchURIs(rows))
					}
					rows, _ = submittedSearch(t, store, project, q+" "+q+" socks", 10, "", now)
					if !slices.Equal(searchURIs(rows), []string{useful}) {
						t.Fatalf("repeated equivalence: %v", searchURIs(rows))
					}
				})
			}
		}
	}
}

// IT-002 / FR-002, FR-004 / AC-004,007,008.
func TestSearchImprovementsCrossFieldConcepts(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprint(project), func(t *testing.T) {
			pool, store, now := searchFixture(t)
			useful := searchFixturePost(t, pool, project, "useful", "blue", []string{"en"}, now)
			if project {
				seedProjectDetails(t, pool, useful, []string{"wool"}, nil, nil, nil)
				if _, err := pool.Exec(context.Background(), `WITH caption AS (UPDATE craftsky_posts SET text='blue' WHERE uri=$1) UPDATE craftsky_project_posts SET common_title='socks' WHERE uri=$1`, useful); err != nil {
					t.Fatal(err)
				}
			} else {
				seedPostTags(t, pool, useful, []string{"wool"})
				if _, err := pool.Exec(context.Background(), `UPDATE craftsky_posts SET images='[{"alt":"socks"}]' WHERE uri=$1`, useful); err != nil {
					t.Fatal(err)
				}
			}
			searchFixturePost(t, pool, project, "missing", "blue yarn socks", []string{"en"}, now)
			rows, _ := submittedSearch(t, store, project, "blue wool socks", 10, "", now)
			if !slices.Equal(searchURIs(rows), []string{useful}) {
				t.Fatalf("cross-field concepts: %v", searchURIs(rows))
			}
			split := searchFixturePost(t, pool, project, "split", "work socks", nil, now)
			seedPostTags(t, pool, split, []string{"in", "progress"})
			rows, _ = submittedSearch(t, store, project, "wip socks", 10, "", now)
			if slices.Contains(searchURIs(rows), split) {
				t.Fatal("approved phrase assembled across unrelated fields")
			}
		})
	}
}

// IT-004 / FR-004 / AC-007–008: independent fields and absent optional values.
func TestSearchImprovementsScopedFields(t *testing.T) {
	cases := []struct {
		field   string
		project bool
		sql     string
	}{
		{"caption", false, `UPDATE craftsky_posts SET text=$2 WHERE uri=$1`},
		{"tag", false, `UPDATE craftsky_posts SET tags=ARRAY[$2::text] WHERE uri=$1`},
		{"alt", false, `UPDATE craftsky_posts SET images=jsonb_build_array(jsonb_build_object('alt',$2::text)) WHERE uri=$1`},
		{"caption", true, `UPDATE craftsky_posts SET text=$2 WHERE uri=$1`},
		{"title", true, `UPDATE craftsky_project_posts SET common_title=$2 WHERE uri=$1`},
		{"pattern", true, `UPDATE craftsky_project_posts SET pattern_name=$2 WHERE uri=$1`},
		{"materials", true, `UPDATE craftsky_project_posts SET materials=ARRAY[$2::text] WHERE uri=$1`},
		{"project-tag", true, `UPDATE craftsky_project_posts SET project_tags=ARRAY[$2::text] WHERE uri=$1`},
		{"design-tag", true, `UPDATE craftsky_project_posts SET design_tags=ARRAY[$2::text] WHERE uri=$1`},
		{"authored-tag", true, `UPDATE craftsky_posts SET tags=ARRAY[$2::text] WHERE uri=$1`},
		{"alt", true, `UPDATE craftsky_posts SET images=jsonb_build_array(jsonb_build_object('alt',$2::text),jsonb_build_object('alt',NULL)) WHERE uri=$1`},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%t/%s", tc.project, tc.field), func(t *testing.T) {
			pool, store, now := searchFixture(t)
			useful := searchFixturePost(t, pool, tc.project, "useful", "", nil, now)
			searchFixturePost(t, pool, tc.project, "empty", "", nil, now)
			if _, err := pool.Exec(context.Background(), tc.sql, useful, "shawl"); err != nil {
				t.Fatal(err)
			}
			rows, _ := submittedSearch(t, store, tc.project, "shawl", 10, "", now)
			if !slices.Equal(searchURIs(rows), []string{useful}) {
				t.Fatalf("%s field got %v", tc.field, searchURIs(rows))
			}
		})
	}
}

// UT-004 / FR-003 / AC-005–006: execute the actual SQL-owned edit predicate.
func TestSearchImprovementsOneEditBoundary(t *testing.T) {
	pool, _, _ := searchFixture(t)
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"crochett", "crochet", true}, {"crohet", "crochet", true}, {"croxhet", "crochet", true}, {"corchet", "crochet", true},
		{"flaxx", "flax", true}, {"chaussette", "chaussettes", true}, {"écharpe", "échrpe", true},
		{"crochett", "crochetzz", false}, {"crochet", "crochet", false}, {"abc", "cba", false}, {"", "", false},
	} {
		t.Run(tc.a+"/"+tc.b, func(t *testing.T) {
			var got bool
			if err := pool.QueryRow(context.Background(), `SELECT craftsky_search_one_edit($1,$2)`, tc.a, tc.b).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("one edit %q -> %q = %t want %t", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// UT-007 / NFR-002 / AC-012: deterministic eight-candidate vocabulary.
func TestSearchImprovementsCandidateCap(t *testing.T) {
	pool, store, now := searchFixture(t)
	words := []string{"socksj", "socksi", "socksh", "socksg", "socksf", "sockse", "socksd", "socksc", "socksb", "socksa"}
	want := []string{}
	for i, word := range words {
		uri := searchFixturePost(t, pool, false, fmt.Sprint(i), word+" wool", nil, now)
		if word <= "socksh" {
			want = append(want, uri)
		}
	}
	// Missing-other-concept candidates sort first but must not consume the cap.
	searchFixturePost(t, pool, false, "irrelevant", "asocks bsocks csocks dsocks", nil, now)
	rows, _ := submittedSearch(t, store, false, "socks wool", 100, "", now)
	got := searchURIs(rows)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("capped eligible vocabulary=%v want=%v", got, want)
	}
	again, _ := submittedSearch(t, store, false, "socks wool", 100, "", now)
	if !slices.Equal(searchURIs(rows), searchURIs(again)) {
		t.Fatal("selection depends on row order")
	}
}

// IT-003 / FR-003 / AC-005–006: correction preserves every other concept.
func TestSearchImprovementsTyposAndProtectedCodes(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprint(project), func(t *testing.T) {
			pool, store, now := searchFixture(t)
			corrected := searchFixturePost(t, pool, project, "corrected", "crochet blanket", nil, now)
			missing := searchFixturePost(t, pool, project, "missing", "crochet hat", nil, now)
			code := searchFixturePost(t, pool, project, "code", "DK K2 KAL socks", nil, now)
			searchFixturePost(t, pool, project, "code-near", "DL K3 KAM socks", nil, now)
			pattern := searchFixturePost(t, pool, project, "pattern", "Flax Light", nil, now)
			queries := []struct {
				q    string
				want []string
			}{
				{"crochett blanket", []string{corrected}}, {"crohett blankett", nil},
				{"crochett blankett", nil}, {"zzzzq blanket", nil},
				{"DK socks", []string{code}}, {"K2 socks", []string{code}}, {"KAL socks", []string{code}},
				{"Flaxx Light", []string{pattern}}, {"Flaxx Missing", nil},
				{"crocxet blanket", []string{corrected}}, {"corchet blanket", []string{corrected}}, {"crohet blanket", []string{corrected}},
			}
			for _, tc := range queries {
				rows, _ := submittedSearch(t, store, project, tc.q, 10, "", now)
				if !slices.Equal(searchURIs(rows), tc.want) {
					t.Fatalf("%q got %v want %v (missing %s)", tc.q, searchURIs(rows), tc.want, missing)
				}
			}
			reliable := searchFixturePost(t, pool, project, "reliable", "crochett blanket", nil, now.Add(-time.Hour))
			rows, _ := submittedSearch(t, store, project, "crochett blanket", 10, "", now)
			if !slices.Equal(searchURIs(rows), []string{reliable, corrected}) {
				t.Fatalf("reliable before correction: %v", searchURIs(rows))
			}
		})
	}
}

// IT-009 / NFR-002 / AC-012.
func TestSearchImprovementsRequestAndResultBounds(t *testing.T) {
	for _, route := range []string{"/v1/search/posts", "/v1/search/projects"} {
		for _, tc := range []struct {
			q     string
			limit string
			valid bool
		}{
			{strings.Repeat("é", 256), "100", true}, {strings.Repeat("é", 257), "100", false},
			{"socks", "101", false}, {"socks", "0", false}, {"socks", "100", true},
		} {
			req := httptest.NewRequest("GET", route+"?q="+url.QueryEscape(tc.q)+"&limit="+tc.limit, nil)
			var err error
			if strings.HasSuffix(route, "posts") {
				_, err = api.ParsePostSearchRequest(req)
			} else {
				_, err = api.ParseProjectSearchRequest(req)
			}
			if (err == nil) != tc.valid {
				t.Fatalf("%s query runes %d limit %s error=%v", route, len([]rune(tc.q)), tc.limit, err)
			}
		}
	}
	pool, store, now := searchFixture(t)
	for i := range 101 {
		searchFixturePost(t, pool, false, fmt.Sprint(i), "socks", nil, now)
	}
	rows, next := submittedSearch(t, store, false, "socks", 100, "", now)
	if len(rows) != 100 || next == "" {
		t.Fatalf("result bound %d next=%q", len(rows), next)
	}
	final, last := submittedSearch(t, store, false, "socks", 100, next, now)
	if len(final) != 1 || last != "" {
		t.Fatalf("remaining results %d", len(final))
	}
}

// IT-010 / RULE-002 / AC-015: real visibility predicates precede vocabulary caps.
func TestSearchImprovementsVisibilityBeforeCandidates(t *testing.T) {
	for _, project := range []bool{false, true} {
		for _, mode := range []string{"mute", "block-out", "block-in", "hide", "takedown", "terminal", "language", "image-hold"} {
			t.Run(fmt.Sprintf("%t/%s", project, mode), func(t *testing.T) {
				pool, store, now := searchFixture(t)
				ctx := context.Background()
				seedMember(t, pool, "did:plc:viewer")
				seedMember(t, pool, "did:plc:hidden")
				reliable := searchFixturePost(t, pool, project, "visible-reliable", "socks wool sweater", []string{"en"}, now.Add(-time.Hour))
				corrected := searchFixturePost(t, pool, project, "visible-corrected", "zsocks wool sweater", []string{"en"}, now)
				for i, word := range []string{"asocks", "bsocks", "csocks", "dsocks", "esocks", "fsocks", "gsocks", "hsocks", "isocks"} {
					uri := searchFixturePost(t, pool, project, fmt.Sprintf("hidden-%d", i), word+" wool sweater", []string{"en"}, now)
					if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET did='did:plc:hidden' WHERE uri=$1`, uri); err != nil {
						t.Fatal(err)
					}
					if mode == "hide" || mode == "takedown" {
						seedModerationOutput(t, pool, "post", "did:plc:hidden", uri, mode, now)
					}
				}
				switch mode {
				case "mute":
					seedMute(t, pool, "did:plc:viewer", "did:plc:hidden")
				case "block-out":
					seedSearchBlock(t, pool, "did:plc:viewer", "did:plc:hidden")
				case "block-in":
					seedSearchBlock(t, pool, "did:plc:hidden", "did:plc:viewer")
				case "terminal":
					if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,terminal_at) VALUES('did:plc:hidden','terminal',1,1,'test',now(),now())`); err != nil {
						t.Fatal(err)
					}
				case "language":
					if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET langs=ARRAY['fr'] WHERE did='did:plc:hidden'`); err != nil {
						t.Fatal(err)
					}
				case "image-hold":
					if _, err := pool.Exec(ctx, `UPDATE image_subject_states SET visibility_state='blocked' WHERE subject_uri IN(SELECT uri FROM craftsky_posts WHERE did='did:plc:hidden')`); err != nil {
						t.Fatal(err)
					}
				}
				for _, q := range []string{"socks wool", "jumper wool"} {
					var all []string
					cursor := ""
					for page := 0; page < 20; page++ {
						var rows []api.SearchPostRow
						var err error
						if project {
							rows, cursor, err = store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, api.ProjectSearchRequest{Query: q, Limit: 1, Cursor: cursor}, now)
						} else {
							rows, cursor, err = store.SearchPostsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, api.PostSearchRequest{Query: q, Limit: 1, Cursor: cursor}, now)
						}
						if err != nil {
							t.Fatal(err)
						}
						all = append(all, searchURIs(rows)...)
						if cursor == "" {
							break
						}
					}
					if len(all) != 2 || !slices.Contains(all, reliable) || !slices.Contains(all, corrected) {
						t.Fatalf("%s visibility got %v", q, all)
					}
					if q == "socks wool" && all[0] != reliable {
						t.Fatal("tier ordering")
					}
				}
				if mode == "language" {
					if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET did='did:plc:viewer' WHERE uri=$1`, corrected); err != nil {
						t.Fatal(err)
					}
					rows, _, err := store.SearchPostsWithLanguages(ctx, "did:plc:viewer", []string{"de"}, api.PostSearchRequest{Query: "socks wool", Limit: 10}, now)
					if project {
						rows, _, err = store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", []string{"de"}, api.ProjectSearchRequest{Query: "socks wool", Limit: 10}, now)
					}
					if err != nil || !slices.Equal(searchURIs(rows), []string{corrected}) {
						t.Fatalf("own-post language exception %v err=%v", searchURIs(rows), err)
					}
					rows, _ = submittedSearch(t, store, project, "socks wool", 100, "", now)
					if len(rows) < 3 {
						t.Fatal("no-preference exception lost")
					}
				}
			})
		}
	}
}

func seedSearchBlock(t *testing.T, pool *pgxpool.Pool, actor, subject string) {
	t.Helper()
	seedSearchSet(t, pool, "block", actor, subject, true)
}

func seedSearchSet(t *testing.T, pool *pgxpool.Pool, kind, actor, scope string, account bool) {
	t.Helper()
	collection := "social.craftsky.feed." + kind
	if account {
		collection = "app.bsky.graph." + kind
	}
	uri := "at://" + actor + "/" + collection + "/test"
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
 INSERT INTO tap_source_records(uri,did,collection,rkey,source_event_id,source_fingerprint,revision,cid,action,record,record_bytes,live,ordering_status,projection_disposition,structural_validation_status,semantic_validation_status,observed_at,updated_at)
 VALUES($1,$2,$3,'test',900001,decode(repeat('00',32),'hex'),'3aaaaaaaaaaa2','set-cid','create','{}'::json,2,true,'authoritative','eligible','valid','valid',now(),now())`, uri, actor, collection); err != nil {
		t.Fatal(err)
	}
	var did, target any
	if account {
		did = scope
	} else {
		target = scope
	}
	if _, err := pool.Exec(ctx, `INSERT INTO pds_set_sources(source_uri,kind,actor_did,scope_key,subject_did,subject_uri,activity_at,eligible) VALUES($1,$2,$3,$4,$5,$6,now(),true)`, uri, kind, actor, scope, did, target); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO pds_set_aggregates(kind,actor_did,scope_key,subject_did,subject_uri,representative_source_uri,activated_at,eligible_source_count) VALUES($1,$2,$3,$4,$5,$6,now(),1)`, kind, actor, scope, did, target, uri); err != nil {
		t.Fatal(err)
	}
}

// UT-005, IT-005 / FR-005 / AC-009: database-owned ranking.
func TestSearchImprovementsRanking(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprint(project), func(t *testing.T) {
			pool, store, now := searchFixture(t)
			primaryOld := searchFixturePost(t, pool, project, "primary-old", "crochett blanket", nil, now.Add(-time.Hour))
			primaryA := searchFixturePost(t, pool, project, "primary-a", "crochett blanket", nil, now)
			primaryZ := searchFixturePost(t, pool, project, "primary-z", "crochett blanket", nil, now)
			tag := searchFixturePost(t, pool, project, "tag", "", nil, now.Add(time.Hour))
			seedPostTags(t, pool, tag, []string{"crochett", "blanket"})
			alt := searchFixturePost(t, pool, project, "alt", "", nil, now.Add(2*time.Hour))
			if _, err := pool.Exec(context.Background(), `UPDATE craftsky_posts SET images='[{"alt":"crochett blanket"}]' WHERE uri=$1`, alt); err != nil {
				t.Fatal(err)
			}
			typo := searchFixturePost(t, pool, project, "typo", "crochet crochet crochet blanket blanket", nil, now.Add(3*time.Hour))
			typoTag := searchFixturePost(t, pool, project, "typo-tag", "", nil, now)
			seedPostTags(t, pool, typoTag, []string{"crochet", "blanket"})
			want := []string{primaryZ, primaryA, primaryOld, tag, alt, typo, typoTag}
			rows, _ := submittedSearch(t, store, project, "crochett blanket", 100, "", now)
			if !slices.Equal(searchURIs(rows), want) {
				t.Fatalf("ranking=%v want=%v", searchURIs(rows), want)
			}
			originalScore := rows[3].Score
			seedPostTags(t, pool, tag, []string{"crochett", "CROCHETT", "blanket", "blanket", "crochett"})
			for i := range 3 {
				actor := fmt.Sprintf("did:plc:fan%d", i)
				seedSearchSet(t, pool, "like", actor, tag, false)
				seedSearchSet(t, pool, "repost", actor, tag, false)
			}
			rows, _ = submittedSearch(t, store, project, "crochett blanket", 100, "", now)
			if !slices.Equal(searchURIs(rows), want) || rows[3].Score != originalScore {
				t.Fatalf("duplicate tags/popularity changed ranking: %v", searchURIs(rows))
			}
			if project {
				// Pattern names share primary-field weight with titles and captions.
				pattern := searchFixturePost(t, pool, true, "pattern", "", nil, now)
				if _, err := pool.Exec(context.Background(), `UPDATE craftsky_project_posts SET pattern_name='crochett blanket' WHERE uri=$1`, pattern); err != nil {
					t.Fatal(err)
				}
				rows, _ = submittedSearch(t, store, true, "crochett blanket", 100, "", now)
				if slices.Index(searchURIs(rows), pattern) > slices.Index(searchURIs(rows), tag) {
					t.Fatal("pattern below supporting tag")
				}
			}
		})
	}
}

// UT-006 / FR-006 / AC-010.
func TestSearchImprovementsCursorValidation(t *testing.T) {
	base := map[string]any{"kind": "searchPostsRelevance", "query": "socks", "tier": 1, "score": 0.2, "createdAt": "2026-10-09T12:00:00Z", "uri": "at://did:plc:author/social.craftsky.feed.post/test"}
	encoded, err := envelope.EncodeCursor(base)
	if err != nil {
		t.Fatal(err)
	}
	cur, err := api.DecodeRelevanceSearchCursor(encoded, "searchPostsRelevance", "socks")
	if err != nil || cur.Tier != 1 || cur.Score != 0.2 {
		t.Fatalf("roundtrip=%+v error=%v", cur, err)
	}
	for _, tc := range []struct {
		key   string
		value any
	}{
		{"tier", nil}, {"tier", 2}, {"tier", 0.5}, {"score", -1}, {"score", "NaN"}, {"createdAt", "yesterday"}, {"uri", "not-an-aturi"}, {"query", "hats"}, {"kind", "searchProjectsRelevance"},
	} {
		copy := map[string]any{}
		for k, v := range base {
			copy[k] = v
		}
		copy[tc.key] = tc.value
		encoded, err := envelope.EncodeCursor(copy)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := api.DecodeRelevanceSearchCursor(encoded, "searchPostsRelevance", "socks"); err != envelope.ErrInvalidCursor {
			t.Fatalf("invalid %s=%v error=%v", tc.key, tc.value, err)
		}
	}
	if _, err := api.DecodeRelevanceSearchCursor("bad@@", "searchPostsRelevance", "socks"); err != envelope.ErrInvalidCursor {
		t.Fatal(err)
	}
	if _, err := api.DecodeRelevanceSearchCursor("", "searchPostsRelevance", "socks"); err != nil {
		t.Fatal(err)
	}
}

// IT-006, AT-005 / FR-006 / AC-010.
func TestSearchImprovementsPagination(t *testing.T) {
	for _, project := range []bool{false, true} {
		for _, mode := range []string{"reliable", "synonym", "typo", "mixed"} {
			t.Run(fmt.Sprintf("%t/%s", project, mode), func(t *testing.T) {
				pool, store, now := searchFixture(t)
				q := "crochett blanket"
				reliable := []string{}
				corrected := []string{}
				for i := range 6 {
					source := "crochett blanket"
					if mode == "synonym" {
						q = "jumper blanket"
						source = "sweater blanket"
					}
					if mode == "typo" {
						source = "crochet blanket"
					}
					uri := searchFixturePost(t, pool, project, fmt.Sprintf("r%d", i), source, nil, now)
					if mode == "typo" {
						corrected = append(corrected, uri)
					} else {
						reliable = append(reliable, uri)
					}
				}
				if mode == "mixed" {
					for i := range 3 {
						corrected = append(corrected, searchFixturePost(t, pool, project, fmt.Sprintf("c%d", i), "crochet blanket", nil, now.Add(time.Hour)))
					}
					reliable = append(reliable, searchFixturePost(t, pool, project, "dual", "crochett crochet blanket", nil, now))
				}
				slices.Sort(reliable)
				slices.Reverse(reliable)
				slices.Sort(corrected)
				slices.Reverse(corrected)
				want := append(reliable, corrected...)
				for _, limit := range []int{1, 2, 3} {
					for repeat := range 2 {
						all := []string{}
						cursor := ""
						first := ""
						for page := 0; page < 20; page++ {
							rows, next := submittedSearch(t, store, project, q, limit, cursor, now)
							all = append(all, searchURIs(rows)...)
							cursor = next
							if page == 0 {
								first = cursor
							}
							if cursor == "" {
								break
							}
						}
						if !slices.Equal(all, want) {
							t.Fatalf("limit=%d repeat=%d got=%v want=%v", limit, repeat, all, want)
						}
						for _, other := range []struct{ route, q string }{{"/v1/search/posts", "different"}, {"/v1/search/projects", "different"}} {
							request := authedReq("GET", other.route+"?q="+url.QueryEscape(other.q)+"&cursor="+url.QueryEscape(first), "", "did:plc:viewer")
							response := httptest.NewRecorder()
							if strings.HasSuffix(other.route, "posts") {
								api.SearchPostsHandler(store, nil, nilLogger()).ServeHTTP(response, request)
							} else {
								api.SearchProjectsHandler(store, nil, nilLogger()).ServeHTTP(response, request)
							}
							if response.Code != 400 || !strings.Contains(response.Body.String(), "invalid_cursor") {
								t.Fatalf("wrong-query cursor %d %s", response.Code, response.Body.String())
							}
						}
						otherRoute := "/v1/search/projects"
						if project {
							otherRoute = "/v1/search/posts"
						}
						request := authedReq("GET", otherRoute+"?q="+url.QueryEscape(q)+"&cursor="+url.QueryEscape(first), "", "did:plc:viewer")
						response := httptest.NewRecorder()
						if project {
							api.SearchPostsHandler(store, nil, nilLogger()).ServeHTTP(response, request)
						} else {
							api.SearchProjectsHandler(store, nil, nilLogger()).ServeHTTP(response, request)
						}
						if response.Code != 400 || !strings.Contains(response.Body.String(), "invalid_cursor") {
							t.Fatalf("wrong-tab cursor %d %s", response.Code, response.Body.String())
						}
					}
				}
			})
		}
	}
}

// IT-008 / NFR-001, RULE-003 / AC-011,016: search is a local read.
func TestSearchImprovementsReadOnlyRetrieval(t *testing.T) {
	pool, _, now := searchFixture(t)
	searchFixturePost(t, pool, false, "post", "crochet socks sweater", []string{"en"}, now)
	searchFixturePost(t, pool, true, "project", "crochet socks sweater", []string{"en"}, now)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_recent_searches(id,viewer_did,search_type,display_label,normalized_payload,normalized_payload_hash) VALUES('existing','did:plc:viewer','post','saved history','{"q":"saved history"}','hash')`); err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		var raw string
		if err := pool.QueryRow(ctx, `SELECT jsonb_build_object('posts',(SELECT jsonb_agg(to_jsonb(p) ORDER BY uri) FROM craftsky_posts p),'history',(SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM craftsky_recent_searches h))::text`).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := snapshot()
	config := pool.Config()
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	readonly, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer readonly.Close()
	store := api.NewSearchStore(readonly, nil)
	for _, project := range []bool{false, true} {
		for _, q := range []string{"socks", "sock", "jumper", "crochett socks", "zzzzq", "shawl"} {
			submittedSearch(t, store, project, q, 1, "", now)
		}
	}
	if after := snapshot(); after != before {
		t.Fatal("search changed records/CIDs or recent history")
	}
	// SearchStore and its shared matching owner take only PostgreSQL and the
	// existing observer; no search/model/PDS client is wired into retrieval.
}

// IT-011 / RULE-003 / AC-016.
func TestSearchImprovementsPrivateAndQuoteBoundaries(t *testing.T) {
	pool, store, now := searchFixture(t)
	ctx := context.Background()
	target := searchFixturePost(t, pool, false, "target", "shawl", nil, now)
	quote := searchFixturePost(t, pool, false, "quote", "unrelated", nil, now)
	ownQuote := searchFixturePost(t, pool, false, "own-quote", "shawl", nil, now)
	project := searchFixturePost(t, pool, true, "project", "shawl", nil, now)
	quotedProject := searchFixturePost(t, pool, true, "project-quote", "shawl", nil, now)
	reply := searchFixturePost(t, pool, false, "reply", "shawl", nil, now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET quote_uri=$1,quote_cid='bafycid' WHERE uri=ANY($2)`, target, []string{quote, ownQuote, quotedProject}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET reply_root_uri=$1,reply_parent_uri=$1,reply_root_cid='bafycid',reply_parent_cid='bafycid' WHERE uri=$2`, target, reply); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot) VALUES('11111111-1111-4111-8111-111111111111','test','account','did:plc:author','did:plc:author','{"text":"privatequiltcanary"}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES('did:plc:author','active',1,1,'test',now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO scheduled_posts(id,owner_did,owner_generation,operation_id,request_hash,status,scheduled_at,next_attempt_at,payload_bytes,payload_hash) VALUES('22222222-2222-4222-8222-222222222222','did:plc:author',1,'33333333-3333-4333-8333-333333333333',decode(repeat('00',32),'hex'),'scheduled',now()+interval '1 day',now()+interval '1 day',convert_to('{"text":"privateschedulecanary"}','UTF8'),decode(repeat('00',32),'hex'))`); err != nil {
		t.Fatal(err)
	}
	for _, isProject := range []bool{false, true} {
		rows, _ := submittedSearch(t, store, isProject, "shawl", 100, "", now)
		want := []string{ownQuote, target}
		if isProject {
			want = []string{project}
		}
		got := searchURIs(rows)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("tab/quote/reply boundary=%v want=%v", got, want)
		}
		for _, q := range []string{"privatequiltcanary", "privateschedulecanary"} {
			rows, _ := submittedSearch(t, store, isProject, q, 100, "", now)
			if len(rows) != 0 {
				t.Fatalf("private marker returned %v", searchURIs(rows))
			}
		}
	}
}

// IT-013 / RULE-004 / AC-017: real handler responses and read-only history.
func TestSearchImprovementsHandlerContract(t *testing.T) {
	pool, store, now := searchFixture(t)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:viewer")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES('did:plc:viewer','active',1,1,'test',now())`); err != nil {
		t.Fatal(err)
	}
	for _, project := range []bool{false, true} {
		key := "post"
		route := "/v1/search/posts"
		if project {
			key = "project"
			route = "/v1/search/projects"
		}
		useful := searchFixturePost(t, pool, project, key, "crochet blanket", nil, now)
		seedSavedPost(t, pool, "did:plc:viewer", useful, nil, now)
		seedMember(t, pool, "did:plc:viewer"+key)
		if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, "did:plc:viewer"+key); err != nil {
			t.Fatal(err)
		}
		seedSearchSet(t, pool, "like", "did:plc:viewer"+key, useful, false)
		handler := api.SearchPostsHandler(store, fakeResolver{}, nilLogger())
		if project {
			handler = api.SearchProjectsHandler(store, fakeResolver{}, nilLogger())
		}
		for _, tc := range []struct {
			suffix string
			status int
			code   string
		}{
			{"?q=crochett+blanket", 200, ""}, {"?q=crochett+blanket&limit=101", 400, "validation_error"},
			{"?q=crochett+blanket&cursor=bad@@", 400, "invalid_cursor"}, {"?q=how+to", 200, ""},
		} {
			response := httptest.NewRecorder()
			request := authedReq("GET", route+tc.suffix, "", "did:plc:viewer")
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.status != 200 {
				for _, key := range []string{"error", "message", "requestId"} {
					if _, ok := body[key]; !ok {
						t.Fatalf("missing error field %s: %s", key, response.Body.String())
					}
				}
				if !strings.Contains(response.Body.String(), tc.code) {
					t.Fatal(response.Body.String())
				}
			} else {
				if _, ok := body["items"]; !ok {
					t.Fatal("missing items")
				}
				for _, forbidden := range []string{"matchTier", "tier", "relevanceScore", "score", "data"} {
					if _, ok := body[forbidden]; ok {
						t.Fatalf("unexpected %s", forbidden)
					}
				}
				var page api.SearchPostPageResponse
				if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(tc.suffix, "crochett") && (len(page.Items) != 1 || page.Items[0].URI != useful || !page.Items[0].ViewerHasSaved || page.Items[0].LikeCount != 1) {
					t.Fatalf("hydrated response=%s", response.Body.String())
				}
			}
		}
		// Unauthenticated requests remain owned by route middleware; handler's
		// missing-DID contract is retained when directly invoked without it.
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", route+"?q=socks", nil))
		if response.Code != 500 || !strings.Contains(response.Body.String(), "missing_authenticated_did") {
			t.Fatal("missing-DID handler contract changed")
		}
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM craftsky_recent_searches").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("result fetching wrote search history")
	}
}

// UT-007 / FR-002, NFR-002 / AC-004,012: phrase components cannot consume
// correction slots unless the complete reconstructed phrase qualifies.
func TestSearchImprovementsPhraseCandidateCap(t *testing.T) {
	pool, store, now := searchFixture(t)
	want := searchFixturePost(t, pool, false, "useful", "zork in progress socks", nil, now)
	for i, word := range []string{"fork", "pork", "word", "wore", "worf", "worg", "worh", "wori", "worj"} {
		searchFixturePost(t, pool, false, fmt.Sprint(i), word+" socks progress", nil, now)
	}
	rows, _ := submittedSearch(t, store, false, "work in progress socks", 100, "", now)
	if !slices.Equal(searchURIs(rows), []string{want}) {
		t.Fatalf("incomplete phrases consumed vocabulary cap: %v", searchURIs(rows))
	}
}
