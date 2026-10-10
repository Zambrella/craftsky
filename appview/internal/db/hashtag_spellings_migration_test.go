package db_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/postutil"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const hashtagAllSourcesRecord = `{"text":"#MeMadeMay", "facets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"},{"$type":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}],"project":{"common":{"tags":["memademay"],"pattern":{"name":"#Pattern","nameFacets":[{"index":{"byteStart":0,"byteEnd":8},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Pattern"}]}],"designer":"#Designer","designerFacets":[{"index":{"byteStart":0,"byteEnd":9},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Designer"}]}],"publisher":"#Publisher","publisherFacets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Publisher"}]}]},"materials":[{"text":"#Material","facets":[{"index":{"byteStart":0,"byteEnd":9},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Material"}]}]}]}}}`

// UT-006 / IR-001: migration normalization follows the deployed Go Unicode rules.
func TestHashtagSpellingsUnicodeNormalization(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	var source strings.Builder
	source.WriteString("ASCII abc #Été 編み物 123")
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if unicode.ToLower(r) != r {
			source.WriteRune(r)
			source.WriteRune(unicode.ToLower(r))
		}
	}
	var got string
	if err := pool.QueryRow(context.Background(), `SELECT craftsky_tag_lower($1)`, source.String()).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := strings.ToLower(source.String()); got != want {
		t.Fatalf("SQL lowercase disagrees with Go Unicode %s; update the migration mapping with the Go toolchain", unicode.Version)
	}
}

// UT-003 / FR-004, RULE-002 / AC-006, AC-008.
func TestHashtagSpellingsRetainsEligibleSources(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	record := hashtagAllSourcesRecord
	for _, tc := range []struct {
		project bool
		want    []string
	}{
		{true, []string{"Designer", "Material", "MeMadeMay", "Pattern", "Publisher", "memademay"}},
		{false, []string{"MeMadeMay"}},
	} {
		var got []string
		if err := pool.QueryRow(context.Background(), `SELECT craftsky_post_tag_spellings($1::jsonb, $2)`, record, tc.project).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tc.want) {
			t.Fatalf("project=%v: spellings=%v, want %v", tc.project, got, tc.want)
		}
	}
}

// UT-003, UT-006: source spelling preservation and DecodeFacets parity.
func TestHashtagSpellingsFacetParity(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	valid := `{"index":{"byteStart":0,"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}`
	for _, tc := range []struct {
		name, text, facets string
		want               []string
	}{
		{"valid", "#MeMadeMay", "[" + valid + "]", []string{"MeMadeMay"}},
		{"accepted field casing", "#MeMadeMay", `[{"Index":{"BYTESTART":0,"ByteEnd":10},"Features":[{"$TYPE":"app.bsky.richtext.facet#tag","Tag":"MeMadeMay"}]}]`, []string{"MeMadeMay"}},
		{"Unicode field folding", "#MeMadeMay", `[{"index":{"byteſtart":0,"byteEnd":10},"featureſ":[{"$type":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}]`, []string{"MeMadeMay"}},
		{"malformed case alias invalidates array", "#MeMadeMay", "[" + valid + `,{"Features":[{"$type":"app.bsky.richtext.facet#link","URI":12}]}]`, []string{}},
		{"duplicate", "#MeMadeMay", "[" + valid + "," + valid + "]", []string{"MeMadeMay"}},
		{"unknown feature", "#MeMadeMay", "[" + valid + `,{"features":[{"$type":"future#tag","tag":"Other"}]}]`, []string{"MeMadeMay"}},
		{"null facet", "#MeMadeMay", "[null," + valid + "]", []string{"MeMadeMay"}},
		{"bad range", "short", "[" + valid + "]", []string{}},
		{"negative range", "#MeMadeMay", `[{"index":{"byteStart":-1,"byteEnd":2},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Bad"}]}]`, []string{}},
		{"missing start defaults zero", "#MeMadeMay", `[{"index":{"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}]`, []string{"MeMadeMay"}},
		{"malformed array", "#MeMadeMay", `{"bad":true}`, []string{}},
		{"malformed sibling invalidates array", "#MeMadeMay", "[" + valid + `,{"index":{"byteStart":"0","byteEnd":2}}]`, []string{}},
		{"malformed recognized feature", "#MeMadeMay", "[" + valid + `,{"features":[{"$type":"app.bsky.richtext.facet#link","uri":12}]}]`, []string{}},
		{"overflow", "#MeMadeMay", "[" + valid + `,{"index":{"byteEnd":9223372036854775808}}]`, []string{}},
		{"fractional byte offset", "#MeMadeMay", "[" + valid + `,{"index":{"byteEnd":1.5}}]`, []string{}},
		{"unicode bytes and whitespace", "#Été", `[{"index":{"byteStart":0,"byteEnd":6},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"\u0085\u00a0Été\u3000"}]}]`, []string{"Été"}},
		{"uncased", "#編み物", `[{"index":{"byteStart":0,"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"編み物"}]}]`, []string{"編み物"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			// The public record extractor applies exact-spelling deduplication.
			record, err := json.Marshal(map[string]any{"text": tc.text, "facets": json.RawMessage(tc.facets)})
			if err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(context.Background(), `SELECT craftsky_post_tag_spellings($1::jsonb, false)`, record).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("spellings=%v, want %v", got, tc.want)
			}
			normalized := postutil.MergeTags(got)
			wantNormalized := postutil.ExtractTagsForText(tc.text, postutil.DecodeFacets(json.RawMessage(tc.facets)))
			slices.Sort(normalized)
			slices.Sort(wantNormalized)
			if !slices.Equal(normalized, wantNormalized) {
				t.Fatalf("SQL/Go parity: %v vs %v", normalized, wantNormalized)
			}
		})
	}
}

// IT-006 / FR-005, NFR-001 / AC-007: exact historical migration recovery.
func TestHashtagSpellingsMigrationRecovery(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	if err := testdb.ApplyMigrations(ctx, pool, "000090_hashtag_spellings.down.sql"); err != nil {
		t.Fatal(err)
	}
	source := `{"text":"#MeMadeMay","facets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}]}`
	// Minimal valid public post state; source spelling is absent from the normalized array.
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:recovery','cid');
 INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,tags,created_at) VALUES
 ('at://did:plc:recovery/social.craftsky.feed.post/legacy','did:plc:recovery','legacy','cid','#MeMadeMay','`+source+`'::jsonb,ARRAY['memademay'],'2026-10-08T12:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	var before, after string
	if err := pool.QueryRow(ctx, `SELECT row_to_json(p)::text FROM craftsky_posts p`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := testdb.ApplyMigrations(ctx, pool, "000090_hashtag_spellings.up.sql"); err != nil {
			t.Fatal(err)
		}
		var legacy, fresh []string
		if err := pool.QueryRow(ctx, `SELECT tag_spellings,craftsky_post_tag_spellings(record,is_project) FROM craftsky_posts`).Scan(&legacy, &fresh); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(legacy, []string{"MeMadeMay"}) || !slices.Equal(legacy, fresh) {
			t.Fatalf("recovered=%v fresh=%v", legacy, fresh)
		}
		if err := pool.QueryRow(ctx, `SELECT (to_jsonb(p)-'tag_spellings')::text FROM craftsky_posts p`).Scan(&after); err != nil {
			t.Fatal(err)
		}
		var a, b any
		if err := json.Unmarshal([]byte(before), &a); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(after), &b); err != nil {
			t.Fatal(err)
		}
		ba, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if strings.Compare(string(ba), string(bb)) != 0 {
			t.Fatalf("source projection changed: %s vs %s", ba, bb)
		}
		if err := testdb.ApplyMigrations(ctx, pool, "000090_hashtag_spellings.down.sql"); err != nil {
			t.Fatal(err)
		}
	}
}

// IT-006: recover every source and compare to the real fresh-ingestion path.
func TestHashtagSpellingsRecoveryMatchesFreshIndexing(t *testing.T) {
	legacy := testdb.WithMigratedSchema(t)
	fresh := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	if err := testdb.ApplyMigrations(ctx, legacy, "000090_hashtag_spellings.down.sql"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	idx := index.NewCraftskyPost(fresh, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, pool := range []*pgxpool.Pool{legacy, fresh} {
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:history','cid')`); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 12 {
		spelling := "memademay"
		if i >= 4 {
			spelling = "MeMadeMay"
		}
		source := map[string]any{"text": "#" + spelling, "createdAt": now.Add(-time.Hour).Format(time.RFC3339), "facets": []any{map[string]any{"index": map[string]int{"byteStart": 0, "byteEnd": 10}, "features": []any{map[string]string{"$type": "app.bsky.richtext.facet#tag", "tag": spelling}}}}}
		tags := []string{"memademay"}
		if i == 9 || i == 10 {
			if err := json.Unmarshal([]byte(hashtagAllSourcesRecord), &source); err != nil {
				t.Fatal(err)
			}
			source["createdAt"] = now.Add(-time.Hour).Format(time.RFC3339)
			source["text"] = "#Source"
			source["facets"] = []any{map[string]any{"index": map[string]int{"byteStart": 0, "byteEnd": 7}, "features": []any{map[string]string{"$type": "app.bsky.richtext.facet#tag", "tag": "Source"}}}}
			common := source["project"].(map[string]any)["common"].(map[string]any)
			common["craftType"] = "social.craftsky.feed.defs#knitting"
			common["tags"] = []string{"structured"}
			tags = []string{"source", "structured", "pattern", "designer", "publisher", "material"}
		}
		if i == 11 {
			source["text"] = "#꟎"
			source["facets"] = []any{map[string]any{"index": map[string]int{"byteStart": 0, "byteEnd": len("#꟎")}, "features": []any{map[string]string{"$type": "app.bsky.richtext.facet#tag", "tag": "꟎"}}}}
			tags = []string{"꟏"}
		}
		raw, err := json.Marshal(source)
		if err != nil {
			t.Fatal(err)
		}
		if i == 10 {
			// Exercise each retained source through accepted case aliases.
			for _, field := range []string{"project", "common", "tags", "pattern", "name", "nameFacets", "designer", "designerFacets", "publisher", "publisherFacets", "materials", "facets", "index", "byteStart", "byteEnd", "features", "tag"} {
				raw = []byte(strings.ReplaceAll(string(raw), `"`+field+`":`, `"`+strings.ToUpper(field)+`":`))
			}
		}
		key := fmt.Sprintf("p%d", i)
		uri := syntax.ATURI("at://did:plc:history/social.craftsky.feed.post/" + key)
		if _, err := legacy.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,tags,is_project,created_at) VALUES($1,'did:plc:history',$2,'cid',$3,$4::jsonb,$5,$6,$7)`, uri, key, source["text"], raw, tags, i == 9 || i == 10, now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := idx.Handle(ctx, tap.Event{URI: uri, DID: "did:plc:history", Rkey: syntax.RecordKey(key), Collection: "social.craftsky.feed.post", CID: "cid", Action: "create", Record: raw}); err != nil {
			t.Fatal(err)
		}
		for _, pool := range []*pgxpool.Pool{legacy, fresh} {
			if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES($1,'post','cid','clear')`, uri); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := testdb.ApplyMigrations(ctx, legacy, "000090_hashtag_spellings.up.sql"); err != nil {
		t.Fatal(err)
	}
	for i := range 12 {
		var recovered, ingested []string
		key := fmt.Sprintf("p%d", i)
		if err := legacy.QueryRow(ctx, `SELECT tag_spellings FROM craftsky_posts WHERE rkey=$1`, key).Scan(&recovered); err != nil {
			t.Fatal(err)
		}
		if err := fresh.QueryRow(ctx, `SELECT tag_spellings FROM craftsky_posts WHERE rkey=$1`, key).Scan(&ingested); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(recovered, ingested) || len(recovered) == 0 {
			t.Fatalf("%s recovered=%v fresh=%v", key, recovered, ingested)
		}
		if i == 10 && !slices.Equal(recovered, []string{"Designer", "Material", "Pattern", "Publisher", "Source", "structured"}) {
			t.Fatalf("case-alias recovery omitted a required source: %v", recovered)
		}
		if i == 11 && !slices.Equal(recovered, []string{"꟎"}) {
			t.Fatalf("Unicode source recovery=%v", recovered)
		}
	}
	for _, pool := range []*pgxpool.Pool{legacy, fresh} {
		rows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "mema", 10, now)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].Tag != "MeMadeMay" || rows[0].PostsLast28Days != 9 {
			t.Fatalf("suggestions=%v", rows)
		}
		unicodeRows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "꟎", 10, now)
		if err != nil {
			t.Fatal(err)
		}
		if len(unicodeRows) != 1 || unicodeRows[0].Tag != "꟎" || unicodeRows[0].PostsLast28Days != 1 {
			t.Fatalf("Unicode recovery suggestions=%v", unicodeRows)
		}
	}
}

// IT-009 / FR-006, FR-005 / AC-007, AC-012: block recovery, never guess.
func TestHashtagSpellingsMigrationRejectsAmbiguousSources(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	if err := testdb.ApplyMigrations(ctx, pool, "000090_hashtag_spellings.down.sql"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:ambiguous','cid')`); err != nil {
		t.Fatal(err)
	}
	const feature = `"$type":"app.bsky.richtext.facet#tag",`
	for _, fields := range []string{
		`"facets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{` + feature + `"tag":"OtherTag","Tag":"MeMadeMay"}]}]`,
		`"facets":[{"features":[{` + feature + `"Tag":"MeMadeMay","tag":"memademay"}]}]`,
		`"facets":[{"features":[{` + feature + `"tag":"MeMadeMay","Tag":"MeMadeMay"}]}]`,
		`"facets":[],"Facets":[]`,
		`"Text":"other"`,
		`"project":null,"Project":null`,
		`"project":{"common":{},"Common":{}}`,
		`"project":{"common":{"craftType":"knitting","CraftType":"sewing"}}`,
		`"project":{"common":{"pattern":{},"Pattern":{}}}`,
		`"project":{"common":{"pattern":{"designer":"one","Designer":"two"}}}`,
		`"project":{"common":{"pattern":{"publisher":"one","Publisher":"two"}}}`,
		`"project":{"common":{"pattern":{"nameFacets":[],"NameFacets":[]}}}`,
		`"project":{"common":{"pattern":{"designerFacets":[],"DesignerFacets":[]}}}`,
		`"project":{"common":{"pattern":{"publisherFacets":[],"PublisherFacets":[]}}}`,
		`"project":{"common":{"materials":[],"Materials":[]}}`,
		`"project":{"common":{"materials":[{"facets":[],"Facets":[]}]}}`,
		`"facets":[{"index":{},"Index":{}}]`,
		`"facets":[{"features":[],"Features":[]}]`,
		`"facets":[{"index":{"byteEnd":10,"ByteEnd":11}}]`,
		`"facets":[{"features":[{"$type":"app.bsky.richtext.facet#link","uri":"https://example.com","URI":"https://example.org"}]}]`,
		`"facets":[{"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:one","DID":"did:plc:two"}]}]`,
		`"project":{"common":{"tags":["one"],"Tags":["two"]}}`,
		`"project":{"common":{"pattern":{"name":"one","Name":"two"}}}`,
		`"project":{"common":{"materials":[{"text":"one","Text":"two"}]}}`,
		`"facets":[{"index":{"byteStart":0,"byteſtart":1}}]`,
		`"facets":[{"features":[{` + feature + `"$TYPE":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}]`,
	} {
		raw := `{"text":"#MeMadeMay",` + fields + `}`
		// Deliberately is_project=false: preflight validates stored source, not
		// just currently contributing candidates.
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,tags,created_at) VALUES('at://did:plc:ambiguous/social.craftsky.feed.post/legacy','did:plc:ambiguous','legacy','cid','#MeMadeMay',$1::jsonb,ARRAY['memademay'],'2026-10-09T11:00:00Z')`, raw); err != nil {
			t.Fatal(err)
		}
		var before string
		if err := pool.QueryRow(ctx, `SELECT to_jsonb(p)::text FROM craftsky_posts p`).Scan(&before); err != nil {
			t.Fatal(err)
		}
		err := testdb.ApplyMigrations(ctx, pool, "000090_hashtag_spellings.up.sql")
		if err == nil || !strings.Contains(err.Error(), "ambiguous hashtag source fields") {
			t.Fatalf("migration error=%v; want clear ambiguity failure for %s", err, fields)
		}
		var after string
		var hasColumn bool
		if err := pool.QueryRow(ctx, `SELECT to_jsonb(p)::text FROM craftsky_posts p`).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='craftsky_posts' AND column_name='tag_spellings')`).Scan(&hasColumn); err != nil {
			t.Fatal(err)
		}
		if before != after || hasColumn {
			t.Fatalf("failed migration changed source/schema: before=%s after=%s column=%v", before, after, hasColumn)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts`); err != nil {
			t.Fatal(err)
		}
	}
	// Resolve only the disposable fixture explicitly; migration never repairs it.
	const resolved = `{"text":"#MeMadeMay","facets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{"$type":"app.bsky.richtext.facet#tag","Tag":"MeMadeMay"}]}]}`
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,tags,created_at) VALUES('at://did:plc:ambiguous/social.craftsky.feed.post/legacy','did:plc:ambiguous','legacy','cid','#MeMadeMay',$1::jsonb,ARRAY['memademay'],'2026-10-09T11:00:00Z')`, resolved); err != nil {
		t.Fatal(err)
	}
	if err := testdb.ApplyMigrations(ctx, pool, "000090_hashtag_spellings.up.sql"); err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := pool.QueryRow(ctx, `SELECT tag_spellings FROM craftsky_posts`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"MeMadeMay"}) {
		t.Fatalf("resolved recovery=%v", got)
	}
}
