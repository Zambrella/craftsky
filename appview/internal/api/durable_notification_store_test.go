package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestNotificationStoreListsOnlyActiveDurableEventsWithStablePagination(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "avatar")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://did:plc:viewer/app.bsky.graph.follow/alice', 'did:plc:viewer',
		        'alice', 'follow-cid', 'did:plc:alice', '{}'::jsonb, now())
	`); err != nil {
		t.Fatal(err)
	}
	activity := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	for _, row := range []struct{ id, state string }{
		{"00000000-0000-0000-0000-000000000003", "active"},
		{"00000000-0000-0000-0000-000000000002", "active"},
		{"00000000-0000-0000-0000-000000000001", "retracted"},
	} {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO notification_events (
				id, recipient_did, actor_did, category, subject_key,
				source_uri, source_cid, source_rkey,
				eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
				state, first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
			) VALUES ($1::uuid, 'did:plc:viewer', 'did:plc:alice', 'follow', $1::uuid::text,
			          'at://did:plc:alice/app.bsky.graph.follow/' || $1::uuid::text, 'cid', $1::uuid::text,
			          'everyone', false, true, $2, $3, $3, $3, $3)
		`, row.id, row.state, activity); err != nil {
			t.Fatal(err)
		}
	}

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].ID != "00000000-0000-0000-0000-000000000003" || !first[0].ActorViewerIsFollowing || cursor == "" {
		t.Fatalf("first=%+v cursor=%q", first, cursor)
	}
	second, finalCursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 1, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].ID != "00000000-0000-0000-0000-000000000002" || !second[0].ActorViewerIsFollowing || finalCursor != "" {
		t.Fatalf("second=%+v cursor=%q", second, finalCursor)
	}
}

func TestNotificationStoreKeepsDurableRowButWithholdsTakenDownContent(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:actor")
	post := seedPost(t, pool, "did:plc:viewer", "subject", "secret content", time.Now())
	seedModerationOutput(t, pool, "post", "", post, "takedown", time.Now())
	if _, err := pool.Exec(context.Background(), `INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,subject_uri,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at)VALUES('00000000-0000-0000-0000-000000000001','did:plc:viewer','did:plc:actor','like',$1,'source','cid','r',$1,'everyone',false,true,'active',now(),now(),now(),now())`, post); err != nil {
		t.Fatal(err)
	}
	rows, _, err := api.NewPostStore(pool).ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].SubjectPost != nil {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestNotificationListWithholdsModeratedReplySourceAndQuoteTarget(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `CREATE TABLE atproto_identity_cache(did TEXT PRIMARY KEY,handle TEXT NOT NULL,handle_lower TEXT NOT NULL UNIQUE,resolved_at TIMESTAMPTZ NOT NULL,updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:actor")
	seedBskyProfile(t, pool, "did:plc:actor", "Actor", "avatar")
	if _, err := pool.Exec(context.Background(), `INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES('did:plc:actor','actor.example','actor.example',now()),('did:plc:viewer','viewer.example','viewer.example',now())`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	parent := seedPost(t, pool, "did:plc:viewer", "parent", "visible parent", now)
	hiddenReply := seedReplyPost(t, pool, "did:plc:actor", "hidden-reply", "secret reply", parent, parent, now.Add(time.Second))
	quoteTarget := seedPost(t, pool, "did:plc:viewer", "quote-target", "secret target", now.Add(2*time.Second))
	quoteSource := seedQuotePost(t, pool, "did:plc:actor", "quote-source", "visible quote", quoteTarget, "bafycid-quote-target", now.Add(3*time.Second))
	seedModerationOutput(t, pool, "post", "did:plc:actor", hiddenReply, "hide", now.Add(4*time.Second))
	seedModerationOutput(t, pool, "post", "did:plc:viewer", quoteTarget, "takedown", now.Add(5*time.Second))
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,subject_uri,subject_cid,parent_uri,parent_cid,root_uri,root_cid,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at)
		VALUES('00000000-0000-0000-0000-000000000001','did:plc:viewer','did:plc:actor','reply','reply',$1,'bafycid-hidden-reply','hidden-reply',$2,'bafycid-parent',$2,'parentcid',$2,'rootcid','everyone',false,true,'active',now(),now(),now(),now())
	`, hiddenReply, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,subject_uri,subject_cid,quoted_uri,quoted_cid,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at)
		VALUES('00000000-0000-0000-0000-000000000002','did:plc:viewer','did:plc:actor','quote','quote',$1,'bafycid-quote-source','quote-source',$1,'bafycid-quote-source',$2,'bafycid-quote-target','everyone',false,true,'active',now(),now()+interval '1 second',now(),now())
	`, quoteSource, quoteTarget); err != nil {
		t.Fatal(err)
	}

	handler := api.ListNotificationsHandler(api.NewPostStore(pool), fakeResolver{}, nilLogger())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, forbidden := range []string{hiddenReply, "bafycid-hidden-reply", "hidden-reply", "secret reply", quoteTarget, "bafycid-quote-target", "secret target"} {
		if strings.Contains(recorder.Body.String(), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, recorder.Body.String())
		}
	}
	var page api.NotificationPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items=%d", len(page.Items))
	}
	byType := map[api.NotificationType]*api.NotificationItem{}
	for _, item := range page.Items {
		byType[item.Type] = item
	}
	reply := byType[api.NotificationTypeReply]
	if reply == nil || reply.URI != "" || reply.CID != "" || reply.Rkey != "" || reply.Reply == nil || reply.Reply.Available || reply.ContentAvailable == nil || *reply.ContentAvailable || reply.SubjectPost == nil {
		t.Fatalf("reply=%+v", reply)
	}
	quote := byType[api.NotificationTypeQuote]
	if quote == nil || quote.URI == "" || quote.SubjectPost == nil || quote.SubjectPost.Quote != nil || quote.References.Quoted == nil || quote.References.Quoted.Available {
		t.Fatalf("quote=%+v", quote)
	}
}

func TestNotificationStoreModeratesEveryReferenceRoleAcrossStates(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:actor")
	seedBskyProfile(t, pool, "did:plc:actor", "Actor", "avatar")

	type roleCase struct {
		category api.NotificationType
		role     string
	}
	roles := []roleCase{
		{api.NotificationTypeLike, "subject"},
		{api.NotificationTypeRepost, "subject"},
		{api.NotificationTypeReply, "source"},
		{api.NotificationTypeReply, "subject_parent"},
		{api.NotificationTypeReply, "root"},
		{api.NotificationTypeMention, "source_subject"},
		{api.NotificationTypeQuote, "source_subject"},
		{api.NotificationTypeQuote, "quoted"},
	}
	states := []string{"available", "missing", "hidden", "takedown"}
	type expectation struct {
		role, state string
	}
	expected := map[string]expectation{}
	sequence := 1
	for _, role := range roles {
		for _, state := range states {
			prefix := fmt.Sprintf("matrix-%02d", sequence)
			id := fmt.Sprintf("00000000-0000-0000-1000-%012d", sequence)
			sequence++
			baseline := func(name, did string) (string, string) {
				rkey := prefix + "-" + name
				uri := seedPost(t, pool, did, rkey, "visible "+name, time.Now())
				return uri, "bafycid-" + rkey
			}
			target := func(name, did string) (string, string) {
				rkey := prefix + "-" + name + "-" + state
				uri := "at://" + did + "/social.craftsky.feed.post/" + rkey
				cid := "bafycid-" + rkey
				if state != "missing" {
					seedPost(t, pool, did, rkey, "private "+state+" "+name, time.Now())
				}
				if state == "hidden" || state == "takedown" {
					value := "hide"
					if state == "takedown" {
						value = "takedown"
					}
					seedModerationOutput(t, pool, "post", did, uri, value, time.Now())
				}
				return uri, cid
			}

			sourceURI := "at://did:plc:actor/event/" + prefix
			sourceCID := "event-cid-" + prefix
			sourceRkey := prefix
			var subjectURI, subjectCID, parentURI, parentCID, rootURI, rootCID, quotedURI, quotedCID any
			switch role.category {
			case api.NotificationTypeLike, api.NotificationTypeRepost:
				subjectURI, subjectCID = target("subject", "did:plc:viewer")
			case api.NotificationTypeReply:
				sourceURI, sourceCID = baseline("source", "did:plc:actor")
				sourceRkey = prefix + "-source"
				subjectURI, subjectCID = baseline("parent", "did:plc:viewer")
				parentURI, parentCID = subjectURI, subjectCID
				rootURI, rootCID = baseline("root", "did:plc:viewer")
				switch role.role {
				case "source":
					sourceURI, sourceCID = target("source", "did:plc:actor")
					sourceRkey = prefix + "-source-" + state
				case "subject_parent":
					subjectURI, subjectCID = target("parent", "did:plc:viewer")
					parentURI, parentCID = subjectURI, subjectCID
				case "root":
					rootURI, rootCID = target("root", "did:plc:viewer")
				}
			case api.NotificationTypeMention:
				sourceURI, sourceCID = target("mention", "did:plc:actor")
				sourceRkey = prefix + "-mention-" + state
				subjectURI, subjectCID = sourceURI, sourceCID
			case api.NotificationTypeQuote:
				sourceURI, sourceCID = baseline("quote-source", "did:plc:actor")
				sourceRkey = prefix + "-quote-source"
				subjectURI, subjectCID = sourceURI, sourceCID
				quotedURI, quotedCID = baseline("quoted", "did:plc:viewer")
				if role.role == "source_subject" {
					sourceURI, sourceCID = target("quote-source", "did:plc:actor")
					sourceRkey = prefix + "-quote-source-" + state
					subjectURI, subjectCID = sourceURI, sourceCID
				} else {
					quotedURI, quotedCID = target("quoted", "did:plc:viewer")
				}
			}
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,subject_uri,subject_cid,parent_uri,parent_cid,root_uri,root_cid,quoted_uri,quoted_cid,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at)
				VALUES($1::uuid,'did:plc:viewer','did:plc:actor',$2,$1,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'everyone',false,true,'active',now(),now(),now(),now())
			`, id, role.category, sourceURI, sourceCID, sourceRkey, subjectURI, subjectCID, parentURI, parentCID, rootURI, rootCID, quotedURI, quotedCID); err != nil {
				t.Fatal(err)
			}
			expected[id] = expectation{role: role.role, state: state}
		}
	}

	rows, _, err := api.NewPostStore(pool).ListNotifications(context.Background(), "did:plc:viewer", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(expected) {
		t.Fatalf("rows=%d want=%d", len(rows), len(expected))
	}
	check := func(t *testing.T, ref *api.NotificationReference, wantAvailable bool) {
		t.Helper()
		if ref == nil || ref.Available != wantAvailable {
			t.Fatalf("ref=%+v available=%v", ref, wantAvailable)
		}
		if wantAvailable && (ref.URI == "" || ref.CID == "") {
			t.Fatalf("available ref missing identity: %+v", ref)
		}
		if !wantAvailable && (ref.URI != "" || ref.CID != "" || ref.Rkey != "") {
			t.Fatalf("unavailable ref leaked identity: %+v", ref)
		}
	}
	for _, row := range rows {
		exp := expected[row.ID]
		wantAvailable := exp.state == "available"
		switch exp.role {
		case "subject":
			check(t, row.References.Subject, wantAvailable)
			if (row.SubjectPost != nil) != wantAvailable {
				t.Fatalf("%s/%s subject post=%+v", row.Type, exp.state, row.SubjectPost)
			}
		case "source":
			check(t, &row.References.Source, wantAvailable)
		case "subject_parent":
			check(t, row.References.Subject, wantAvailable)
			check(t, row.References.Parent, wantAvailable)
			if (row.SubjectPost != nil) != wantAvailable {
				t.Fatalf("reply/%s subject post=%+v", exp.state, row.SubjectPost)
			}
		case "root":
			check(t, row.References.Root, wantAvailable)
		case "source_subject":
			check(t, &row.References.Source, wantAvailable)
			check(t, row.References.Subject, wantAvailable)
			if (row.SubjectPost != nil) != wantAvailable {
				t.Fatalf("%s/%s subject post=%+v", row.Type, exp.state, row.SubjectPost)
			}
		case "quoted":
			check(t, row.References.Quoted, wantAvailable)
		}
	}
}
