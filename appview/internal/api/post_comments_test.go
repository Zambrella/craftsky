package api_test

import (
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/relationships"
	"strconv"
	"strings"
	"testing"
	"time"
)

func replyItemsContainURI(items []api.ReplyItem, uri string) bool {
	for _, item := range items {
		if item.Post.URI == uri {
			return true
		}
	}
	return false
}

func TestListCommentRepliesShapesMutedBranchAndOmitsBlockedBranch(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:root/social.craftsky.feed.post/root"
	commentURI := "at://did:plc:commenter/social.craftsky.feed.post/comment"
	target := testReplyRow("did:plc:commenter", "comment", "comment", rootURI, rootURI, time.Now())
	muted := testReplyRow("did:plc:bob", "muted", "protected muted text", rootURI, commentURI, time.Now().Add(time.Second))
	child := testReplyRow("did:plc:carol", "child", "descendant text", rootURI, muted.URI, time.Now().Add(2*time.Second))

	for _, test := range []struct {
		name      string
		state     relationships.State
		wantItems int
		wantState string
	}{
		{name: "mute collapses full descendant branch", state: relationships.State{Muted: true}, wantItems: 1, wantState: "muted"},
		{name: "block omits full descendant branch", state: relationships.State{Blocking: true}, wantItems: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &fakePostStore{
				one:       target,
				replyRows: []*api.PostRow{muted, child},
				relationshipStates: map[syntax.DID]relationships.State{
					"did:plc:bob": test.state,
				},
			}
			h := hydrateProductionSummaries(api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
				"did:plc:bob": "bob.example", "did:plc:carol": "carol.example",
			}}, nilLogger()), map[syntax.DID]business.AccountType{
				"did:plc:bob": business.AccountTypeBusiness, "did:plc:carol": business.AccountTypeBusiness,
			})
			req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:commenter/comment/replies", "", "did:plc:alice")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			if test.state.Blocking && (strings.Contains(rr.Body.String(), "accountType") || strings.Contains(rr.Body.String(), "business")) {
				t.Fatalf("blocked production reply branch exposed business fields: %s", rr.Body.String())
			}
			var body map[string]any
			_ = json.Unmarshal(rr.Body.Bytes(), &body)
			items := body["items"].([]any)
			if len(items) != test.wantItems || strings.Contains(rr.Body.String(), "protected muted text") || strings.Contains(rr.Body.String(), "descendant text") {
				t.Fatalf("body = %s", rr.Body.String())
			}
			if test.wantState != "" {
				post := items[0].(map[string]any)["post"].(map[string]any)
				if post["availability"] != test.wantState {
					t.Fatalf("post = %+v", post)
				}
			}
		})
	}
}

func TestListCommentReplies_HappyPath_PaginatesEngagementAndAuthorHandles(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	parentURI := "at://did:plc:alice/social.craftsky.feed.post/comment"
	comment := testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())
	replies := []*api.PostRow{
		{URI: "at://did:plc:bob/social.craftsky.feed.post/reply1", DID: "did:plc:bob", Rkey: "reply1", CID: "bafy1", Text: "first"},
		{URI: "at://did:plc:carol/social.craftsky.feed.post/reply2", DID: "did:plc:carol", Rkey: "reply2", CID: "bafy2", Text: "second"},
	}
	store := &fakePostStore{
		one:         comment,
		replyRows:   replies,
		replyCursor: "next-replies",
		engagement: map[string]api.EngagementSummary{
			replies[0].URI: {LikeCount: 2, RepostCount: 1, ReplyCount: 4, ViewerHasLiked: true, ViewerHasReplied: true},
			replies[1].URI: {LikeCount: 1, RepostCount: 0, ReplyCount: 0, ViewerHasReposted: true},
		},
	}
	h := hydrateProductionSummaries(api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger()), map[syntax.DID]business.AccountType{"did:plc:bob": business.AccountTypeBusiness})
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?limit=2&cursor=opaque", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var rawBody struct {
		Items []struct {
			Post struct {
				Author struct {
					AccountType string `json:"accountType"`
				} `json:"author"`
			} `json:"post"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &rawBody); err != nil || len(rawBody.Items) != 2 || rawBody.Items[0].Post.Author.AccountType != "business" || rawBody.Items[1].Post.Author.AccountType != "regular" {
		t.Fatalf("reply author account types = %+v, error %v", rawBody, err)
	}
	var resp api.ReplyPage
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if store.lastDID != "did:plc:alice" || store.lastRkey != "comment" {
		t.Fatalf("target lookup = %s/%s", store.lastDID, store.lastRkey)
	}
	if store.lastReplyParentURI != parentURI || store.lastReplyLimit != 2 || store.lastReplyCursor != "opaque" {
		t.Fatalf("reply lookup = uri:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyLimit, store.lastReplyCursor)
	}
	if !resp.Loaded {
		t.Fatal("loaded = false, want true")
	}
	if len(resp.Items) != 2 || resp.Items[0].Post.Author.Handle != "bob.example" || resp.Items[1].Post.Author.Handle != "carol.example" {
		t.Fatalf("items = %+v", resp.Items)
	}
	if resp.Items[0].Flattened || resp.Items[0].ReplyingTo != nil {
		t.Fatalf("direct reply should not be flattened: %+v", resp.Items[0])
	}
	if resp.Items[0].Post.LikeCount != 2 || resp.Items[0].Post.RepostCount != 1 || resp.Items[0].Post.ReplyCount != 4 || !resp.Items[0].Post.ViewerHasLiked || !resp.Items[0].Post.ViewerHasReplied {
		t.Errorf("item0 engagement = %+v", resp.Items[0])
	}
	if !resp.Items[1].Post.ViewerHasReposted {
		t.Errorf("item1 engagement = %+v", resp.Items[1])
	}
	if store.engagementCalls != 1 || store.lastEngagementViewer != "did:plc:viewer" || len(store.lastEngagementURIs) != 2 {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
	}
	if resp.Cursor != "next-replies" {
		t.Errorf("cursor = %q", resp.Cursor)
	}
}

func TestListCommentReplies_NestedBranchReplyIncludesFlattenedMetadata(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	parentReply := testReplyRow("did:plc:carol", "reply", "reply", root.URI, comment.URI, base.Add(2*time.Minute))
	nestedReply := testReplyRow("did:plc:dave", "nested", "nested", root.URI, parentReply.URI, base.Add(3*time.Minute))
	displayName := "Carol"
	parentReply.AuthorDisplayName = &displayName
	store := &fakePostStore{
		one:       comment,
		replyRows: []*api.PostRow{parentReply, nestedReply},
		postsByURI: map[string]*api.PostRow{
			parentReply.URI: parentReply,
		},
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:bob/comment/replies", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ReplyPage
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %+v", resp.Items)
	}
	if resp.Items[0].Flattened {
		t.Fatalf("direct reply flattened = true: %+v", resp.Items[0])
	}
	nested := resp.Items[1]
	if !nested.Flattened || nested.ReplyingTo == nil {
		t.Fatalf("nested reply missing flattened metadata: %+v", nested)
	}
	if nested.ReplyingTo.URI != parentReply.URI || nested.ReplyingTo.DID != parentReply.DID || nested.ReplyingTo.Handle != "carol.example" {
		t.Fatalf("replyingTo = %+v", nested.ReplyingTo)
	}
	if nested.ReplyingTo.DisplayName == nil || *nested.ReplyingTo.DisplayName != "Carol" {
		t.Fatalf("displayName = %+v", nested.ReplyingTo.DisplayName)
	}
}

func TestListCommentReplies_OmitsBlockedParentOutsidePage(t *testing.T) {
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:root", "root", "root", base)
	comment := testReplyRow("did:plc:commenter", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	parent := testReplyRow("did:plc:parent", "parent", "parent", root.URI, comment.URI, base.Add(2*time.Minute))
	child := testReplyRow("did:plc:child", "child", "child", root.URI, parent.URI, base.Add(3*time.Minute))

	for _, test := range []struct {
		name  string
		state relationships.State
	}{
		{name: "viewer blocks parent", state: relationships.State{Blocking: true}},
		{name: "parent blocks viewer", state: relationships.State{BlockedBy: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &fakePostStore{
				one:                comment,
				replyRows:          []*api.PostRow{child},
				postsByURI:         map[string]*api.PostRow{parent.URI: parent},
				relationshipStates: map[syntax.DID]relationships.State{"did:plc:parent": test.state},
			}
			handler := hydrateProductionSummaries(
				api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
					"did:plc:child": "child.example", "did:plc:parent": "parent.example",
				}}, nilLogger()),
				map[syntax.DID]business.AccountType{
					"did:plc:child": business.AccountTypeBusiness, "did:plc:parent": business.AccountTypeBusiness,
				},
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:commenter/comment/replies", "", "did:plc:viewer"))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var page struct {
				Items []struct {
					Post struct {
						Author struct {
							AccountType string `json:"accountType"`
						} `json:"author"`
					} `json:"post"`
					ReplyingTo map[string]any `json:"replyingTo"`
				} `json:"items"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || len(page.Items) != 1 {
				t.Fatalf("decode page = %+v, error %v; body=%s", page, err, response.Body.String())
			}
			if page.Items[0].Post.Author.AccountType != "business" {
				t.Fatalf("visible child accountType = %q", page.Items[0].Post.Author.AccountType)
			}
			if page.Items[0].ReplyingTo != nil {
				t.Fatalf("blocked parent leaked through replyingTo: %+v", page.Items[0].ReplyingTo)
			}
		})
	}
}

func TestListCommentReplies_CapsPageSizeAtTen(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{
		one:       testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now()),
		replyRows: []*api.PostRow{},
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?limit=20", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 10 {
		t.Fatalf("reply limit = %d, want capped 10", store.lastReplyLimit)
	}
}

func TestListCommentReplies_IR002UsesAuthoritativeContentLanguagesForCounts(t *testing.T) {
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	comment := testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())
	reply := testReplyRow("did:plc:bob", "reply", "reply", rootURI, comment.URI, time.Now())
	store := &fakePostStore{one: comment, replyRows: []*api.PostRow{reply}}
	preferences := &fakeLanguagePreferenceReader{preferences: languages.Preferences{
		PrimaryLanguage:  "fr",
		ContentLanguages: []string{"en", "de"},
	}}
	h := api.ListCommentRepliesHandler(
		store,
		fakeResolver{handlesByDID: map[string]syntax.Handle{
			"did:plc:alice": "alice.example",
			"did:plc:bob":   "bob.example",
		}},
		nilLogger(),
		preferences,
	)
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if got, want := store.lastEngagementLanguages, []string{"en", "de"}; !slices.Equal(got, want) {
		t.Fatalf("content languages = %v, want %v", got, want)
	}
}

func TestGetPostComments_ReturnsRootAndCommentsOnly(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	reply := testReplyRow("did:plc:carol", "reply1", "reply", root.URI, comment.URI, base.Add(2*time.Minute))
	root.Langs = []string{"fr-CA"}
	comment.Langs = []string{"cy"}
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{comment},
		engagement: map[string]api.EngagementSummary{
			root.URI:    {ReplyCount: 1},
			comment.URI: {ReplyCount: 1},
			reply.URI:   {ReplyCount: 0},
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "root" || resp.Post.Author.Handle != "alice.example" {
		t.Fatalf("root post = %+v", resp.Post)
	}
	if !slices.Equal(resp.Post.Langs, []string{"fr-CA"}) {
		t.Fatalf("IT-020 thread root langs = %v", resp.Post.Langs)
	}
	if resp.Sort != "oldest" {
		t.Fatalf("sort = %q, want oldest", resp.Sort)
	}
	if len(resp.Comments.Items) != 1 {
		t.Fatalf("comments len = %d, want 1: %+v", len(resp.Comments.Items), resp.Comments.Items)
	}
	if resp.Comments.Items[0].Post.Rkey != "comment1" {
		t.Fatalf("comment item = %+v", resp.Comments.Items[0])
	}
	if !slices.Equal(resp.Comments.Items[0].Post.Langs, []string{"cy"}) {
		t.Fatalf("IT-020 thread comment langs = %v", resp.Comments.Items[0].Post.Langs)
	}
	if resp.Comments.Items[0].Post.Rkey == reply.Rkey {
		t.Fatalf("nested reply was returned as top-level comment: %+v", resp.Comments.Items[0])
	}
	if len(resp.Comments.Items[0].Replies.Items) != 0 || resp.Comments.Items[0].Replies.Loaded {
		t.Fatalf("replies should not be expanded by default: %+v", resp.Comments.Items[0].Replies)
	}
	if store.lastCommentRootURI != root.URI || store.lastCommentLimit != 10 || store.lastCommentCursor != "" || store.lastCommentSort != "oldest" || store.lastCommentViewerDID != "did:plc:viewer" {
		t.Fatalf("comment lookup = root:%q limit:%d cursor:%q sort:%q viewer:%q", store.lastCommentRootURI, store.lastCommentLimit, store.lastCommentCursor, store.lastCommentSort, store.lastCommentViewerDID)
	}
	if store.lastViewerDID != "did:plc:viewer" {
		t.Fatalf("root viewer DID = %q, want did:plc:viewer", store.lastViewerDID)
	}
}

func TestGetPostComments_IR002UsesAuthoritativeContentLanguagesForCounts(t *testing.T) {
	root := testPostRow("did:plc:alice", "root", "root", time.Now())
	comment := testReplyRow("did:plc:bob", "comment", "comment", root.URI, root.URI, time.Now())
	store := &fakePostStore{one: root, commentRows: []*api.PostRow{comment}}
	preferences := &fakeLanguagePreferenceReader{preferences: languages.Preferences{
		PrimaryLanguage:  "fr",
		ContentLanguages: []string{"en", "de"},
	}}
	h := api.GetPostCommentsHandler(
		store,
		fakeResolver{handlesByDID: map[string]syntax.Handle{
			"did:plc:alice": "alice.example",
			"did:plc:bob":   "bob.example",
		}},
		nilLogger(),
		preferences,
	)
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if got, want := store.lastEngagementLanguages, []string{"en", "de"}; !slices.Equal(got, want) {
		t.Fatalf("content languages = %v, want %v", got, want)
	}
}

func TestGetPostComments_AttachesQuoteViewsToPostShapedResponses(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	rootQuoteURI := "at://did:plc:carol/social.craftsky.feed.post/root-target"
	rootQuoteCID := "bafyRootTarget"
	commentQuoteURI := "at://did:plc:dave/social.craftsky.feed.post/hidden-target"
	commentQuoteCID := "bafyHiddenTarget"
	root := testPostRow("did:plc:alice", "root", "root quote commentary", base)
	root.QuoteURI = &rootQuoteURI
	root.QuoteCID = &rootQuoteCID
	comment := testReplyRow("did:plc:bob", "comment1", "comment quote commentary", root.URI, root.URI, base.Add(time.Minute))
	comment.QuoteURI = &commentQuoteURI
	comment.QuoteCID = &commentQuoteCID
	quotedRoot := testPostRow("did:plc:carol", "root-target", "quoted root target", base.Add(-time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{comment},
		quoteViews: map[string]*api.QuoteViewRow{
			rootQuoteURI:    {State: "visible", Post: quotedRoot},
			commentQuoteURI: {State: "hidden"},
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(store.lastQuoteViewRefs) != 2 {
		t.Fatalf("quote view refs = %+v, want 2 refs", store.lastQuoteViewRefs)
	}
	if resp.Post.QuoteView == nil || resp.Post.QuoteView.State != "visible" || resp.Post.QuoteView.Post == nil {
		t.Fatalf("root quoteView = %+v, want visible preview", resp.Post.QuoteView)
	}
	if resp.Post.QuoteView.Post.Author.Handle != "carol.example" {
		t.Fatalf("root quote preview author = %+v", resp.Post.QuoteView.Post.Author)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Post.QuoteView == nil || resp.Comments.Items[0].Post.QuoteView.State != "hidden" {
		t.Fatalf("comment item = %+v, want hidden quoteView", resp.Comments.Items)
	}
}

func TestGetPostComments_CommentItemsIncludePlacement(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	store := &fakePostStore{one: root, commentRows: []*api.PostRow{comment}}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Comments.Items) != 1 {
		t.Fatalf("items = %+v", resp.Comments.Items)
	}
	if resp.Comments.Items[0].Placement != "normal" {
		t.Fatalf("placement = %q, want normal", resp.Comments.Items[0].Placement)
	}
}

func TestGetPostComments_CommentItemsAlwaysIncludeRepliesObject(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	store := &fakePostStore{one: root, commentRows: []*api.PostRow{comment}}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var raw struct {
		Comments struct {
			Items []struct {
				Replies *struct {
					Loaded bool            `json:"loaded"`
					Items  json.RawMessage `json:"items"`
					Cursor *string         `json:"cursor,omitempty"`
				} `json:"replies"`
			} `json:"items"`
		} `json:"comments"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw.Comments.Items) != 1 {
		t.Fatalf("items len = %d", len(raw.Comments.Items))
	}
	if raw.Comments.Items[0].Replies == nil {
		t.Fatal("replies object is missing")
	}
	if raw.Comments.Items[0].Replies.Loaded {
		t.Fatal("replies.loaded = true, want false before expansion")
	}
	if string(raw.Comments.Items[0].Replies.Items) != "[]" {
		t.Fatalf("replies.items = %s, want []", raw.Comments.Items[0].Replies.Items)
	}
	if raw.Comments.Items[0].Replies.Cursor != nil {
		t.Fatalf("replies.cursor should be omitted when not loaded, got %q", *raw.Comments.Items[0].Replies.Cursor)
	}
}

func TestGetPostComments_FocusQueryIdentifiesIncludedComment(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{comment},
		postByURI:   comment,
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(comment.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil {
		t.Fatal("focus metadata missing")
	}
	if resp.Focus.Status != "included" || resp.Focus.URI != comment.URI || resp.Focus.Kind != "comment" {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Placement != "focused" || resp.Comments.Items[0].Post.URI != comment.URI {
		t.Fatalf("focused comment item = %+v", resp.Comments.Items)
	}
}

func TestGetPostComments_FocusedCommentOutsidePageIsIncludedFirst(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	pageComment := testReplyRow("did:plc:bob", "page-comment", "page", root.URI, root.URI, base.Add(time.Minute))
	focusedComment := testReplyRow("did:plc:carol", "focused-comment", "focused", root.URI, root.URI, base.Add(20*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{pageComment},
		postByURI:   focusedComment,
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedComment.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "comment" {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if got := len(resp.Comments.Items); got != 2 {
		t.Fatalf("comments len = %d, want focused extra plus page item: %+v", got, resp.Comments.Items)
	}
	if resp.Comments.Items[0].Post.URI != focusedComment.URI || resp.Comments.Items[0].Placement != "focused" {
		t.Fatalf("first item = %+v", resp.Comments.Items[0])
	}
	if resp.Comments.Items[1].Post.URI != pageComment.URI || resp.Comments.Items[1].Placement != "normal" {
		t.Fatalf("second item = %+v", resp.Comments.Items[1])
	}
}

func TestGetPostComments_FocusedReplyExpandsCommentBranch(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	pageComment := testReplyRow("did:plc:bob", "page-comment", "page", root.URI, root.URI, base.Add(time.Minute))
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(20*time.Minute))
	firstReply := testReplyRow("did:plc:erin", "first-reply", "first", root.URI, comment.URI, base.Add(20*time.Minute+30*time.Second))
	focusedReply := testReplyRow("did:plc:dave", "focused-reply", "reply", root.URI, comment.URI, base.Add(21*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{pageComment},
		replyRows:   []*api.PostRow{firstReply, focusedReply},
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if got := len(resp.Comments.Items); got != 2 {
		t.Fatalf("comments len = %d, want focused branch plus page item: %+v", got, resp.Comments.Items)
	}
	focusedBranch := resp.Comments.Items[0]
	if focusedBranch.Post.URI != comment.URI || focusedBranch.Placement != "focused" {
		t.Fatalf("focused branch = %+v", focusedBranch)
	}
	if !focusedBranch.Replies.Loaded || len(focusedBranch.Replies.Items) != 2 {
		t.Fatalf("focused branch replies = %+v", focusedBranch.Replies)
	}
	if focusedBranch.Replies.Items[0].Post.URI != firstReply.URI || focusedBranch.Replies.Items[0].Flattened {
		t.Fatalf("first reply item = %+v", focusedBranch.Replies.Items[0])
	}
	if focusedBranch.Replies.Items[1].Post.URI != focusedReply.URI || focusedBranch.Replies.Items[1].Flattened {
		t.Fatalf("focused reply item = %+v", focusedBranch.Replies.Items[1])
	}
	if store.lastReplyParentURI != comment.URI || store.lastReplyRootURI != root.URI || store.lastReplyLimit != 10 || store.lastReplyCursor != "" {
		t.Fatalf("branch reply lookup = parent:%q root:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyRootURI, store.lastReplyLimit, store.lastReplyCursor)
	}
}

func TestGetPostComments_FocusedReplyBranchUsesBoundedPage(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	focusedReply := testReplyRow("did:plc:dave", "focused-reply", "reply", root.URI, comment.URI, base.Add(30*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		replyRows:   []*api.PostRow{focusedReply},
		replyCursor: "next-replies",
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Comments.Items) != 1 {
		t.Fatalf("comments = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies
	if !replies.Loaded {
		t.Fatalf("replies not loaded: %+v", replies)
	}
	if len(replies.Items) != 1 {
		t.Fatalf("focused reply branch len = %d, want bounded branch page", len(replies.Items))
	}
	if replies.Items[0].Post.URI != focusedReply.URI {
		t.Fatalf("focused reply item = %+v", replies.Items[0])
	}
	if replies.Cursor != "next-replies" {
		t.Fatalf("reply cursor = %q", replies.Cursor)
	}
	if store.lastReplyParentURI != comment.URI || store.lastReplyRootURI != root.URI || store.lastReplyLimit != 10 || store.lastReplyCursor != "" {
		t.Fatalf("branch reply lookup = parent:%q root:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyRootURI, store.lastReplyLimit, store.lastReplyCursor)
	}
}

func TestGetPostComments_FocusedReplyAfterFirstBranchPageIsIncluded(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	firstPage := make([]*api.PostRow, 0, 10)
	for i := 0; i < 10; i++ {
		firstPage = append(firstPage, testReplyRow("did:plc:dave", "reply-before-"+strconv.Itoa(i), "before", root.URI, comment.URI, base.Add(time.Duration(i+2)*time.Minute)))
	}
	focusedReply := testReplyRow("did:plc:erin", "focused-reply", "target", root.URI, comment.URI, base.Add(20*time.Minute))
	store := &fakePostStore{
		one:               root,
		commentRows:       []*api.PostRow{},
		replyRows:         firstPage,
		replyCursor:       "first-page-cursor",
		aroundReplyRows:   append(firstPage[1:], focusedReply),
		aroundReplyCursor: "after-focus-cursor",
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
		"did:plc:erin":  "erin.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || !resp.Comments.Items[0].Replies.Loaded {
		t.Fatalf("focused branch = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies.Items
	if len(replies) > 10 {
		t.Fatalf("focused branch loaded %d replies, want bounded page", len(replies))
	}
	if !replyItemsContainURI(replies, focusedReply.URI) {
		t.Fatalf("focused reply missing from bounded branch page: %+v", replies)
	}
}

func TestGetPostComments_FocusStatusContract(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	otherRoot := testPostRow("did:plc:alice", "other", "other", base)
	mismatched := testReplyRow("did:plc:bob", "mismatched", "mismatch", otherRoot.URI, otherRoot.URI, base.Add(time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		postsByURI: map[string]*api.PostRow{
			mismatched.URI: mismatched,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())

	t.Run("malformed", func(t *testing.T) {
		req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus=not-an-at-uri", "", "did:plc:viewer")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var body envelope.Error
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if body.Error != "invalid_focus" {
			t.Fatalf("error = %q", body.Error)
		}
	})

	t.Run("not found", func(t *testing.T) {
		missingURI := "at://did:plc:missing/social.craftsky.feed.post/missing"
		req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(missingURI), "", "did:plc:viewer")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var resp api.CommentSectionResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Focus == nil || resp.Focus.URI != missingURI || resp.Focus.Status != "notFound" || resp.Focus.Kind != "" {
			t.Fatalf("focus = %+v", resp.Focus)
		}
	})

	t.Run("mismatched root", func(t *testing.T) {
		req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(mismatched.URI), "", "did:plc:viewer")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var resp api.CommentSectionResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Focus == nil || resp.Focus.URI != mismatched.URI || resp.Focus.Status != "mismatchedRoot" || resp.Focus.Kind != "" {
			t.Fatalf("focus = %+v", resp.Focus)
		}
	})
}

func TestGetPostComments_InvalidCursorUsesStandardEnvelope(t *testing.T) {
	t.Parallel()
	root := testPostRow("did:plc:alice", "root", "root", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	store := &fakePostStore{one: root, commentErr: envelope.ErrInvalidCursor}
	h := api.GetPostCommentsHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?cursor=bad", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "invalid_cursor" || body.Message == "" {
		t.Fatalf("error envelope = %+v", body)
	}
}

func TestGetPostComments_RejectsNonRootPost(t *testing.T) {
	t.Parallel()
	root := testPostRow("did:plc:alice", "root", "root", time.Now())
	comment := testReplyRow("did:plc:alice", "comment", "comment", root.URI, root.URI, time.Now())
	store := &fakePostStore{one: comment}
	h := api.GetPostCommentsHandler(store, fakeResolver{}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "invalid_post_role" {
		t.Fatalf("error = %q", body.Error)
	}
	if store.lastCommentRootURI != "" {
		t.Fatalf("ListRootComments should not run for non-root target")
	}
}

func TestGetPostComments_DeeperFocusedReplyIncludesFlattenedMetadata(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	parentReply := testReplyRow("did:plc:dave", "parent-reply", "parent", root.URI, comment.URI, base.Add(2*time.Minute))
	deeperReply := testReplyRow("did:plc:erin", "deeper-reply", "deep", root.URI, parentReply.URI, base.Add(3*time.Minute))
	displayName := "Dave"
	parentReply.AuthorDisplayName = &displayName
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		replyRows:   []*api.PostRow{parentReply, deeperReply},
		postsByURI: map[string]*api.PostRow{
			deeperReply.URI: deeperReply,
			parentReply.URI: parentReply,
			comment.URI:     comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
		"did:plc:erin":  "erin.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(deeperReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Post.URI != comment.URI {
		t.Fatalf("comments = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies.Items
	if len(replies) != 2 {
		t.Fatalf("replies = %+v", replies)
	}
	if replies[0].Post.URI != parentReply.URI || replies[0].Flattened {
		t.Fatalf("parent reply = %+v", replies[0])
	}
	got := replies[1]
	if got.Post.URI != deeperReply.URI || !got.Flattened {
		t.Fatalf("flattened reply = %+v", got)
	}
	if got.ReplyingTo == nil || got.ReplyingTo.URI != parentReply.URI || got.ReplyingTo.DID != parentReply.DID || got.ReplyingTo.Handle != "dave.example" {
		t.Fatalf("replyingTo = %+v", got.ReplyingTo)
	}
	if got.ReplyingTo.DisplayName == nil || *got.ReplyingTo.DisplayName != "Dave" {
		t.Fatalf("replyingTo displayName = %+v", got.ReplyingTo.DisplayName)
	}
}

func TestGetPostComments_OmitsBlockedParentOutsideFocusedPage(t *testing.T) {
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:root", "root", "root", base)
	comment := testReplyRow("did:plc:commenter", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	parent := testReplyRow("did:plc:parent", "parent", "parent", root.URI, comment.URI, base.Add(2*time.Minute))
	child := testReplyRow("did:plc:child", "child", "child", root.URI, parent.URI, base.Add(3*time.Minute))

	for _, test := range []struct {
		name  string
		state relationships.State
	}{
		{name: "viewer blocks parent", state: relationships.State{Blocking: true}},
		{name: "parent blocks viewer", state: relationships.State{BlockedBy: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &fakePostStore{
				one:                root,
				commentRows:        []*api.PostRow{},
				replyRows:          []*api.PostRow{},
				aroundReplyRows:    []*api.PostRow{child},
				postsByURI:         map[string]*api.PostRow{child.URI: child, parent.URI: parent, comment.URI: comment},
				relationshipStates: map[syntax.DID]relationships.State{"did:plc:parent": test.state},
			}
			handler := hydrateProductionSummaries(
				api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
					"did:plc:root": "root.example", "did:plc:commenter": "commenter.example",
					"did:plc:child": "child.example", "did:plc:parent": "parent.example",
				}}, nilLogger()),
				map[syntax.DID]business.AccountType{
					"did:plc:root": business.AccountTypeRegular, "did:plc:commenter": business.AccountTypeRegular,
					"did:plc:child": business.AccountTypeBusiness, "did:plc:parent": business.AccountTypeBusiness,
				},
			)
			path := "/v1/posts/did:plc:root/root/comments?focus=" + url.QueryEscape(child.URI)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, authedPostPathReq(http.MethodGet, path, "", "did:plc:viewer"))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var page struct {
				Comments struct {
					Items []struct {
						Replies struct {
							Items []struct {
								Post struct {
									Author struct {
										AccountType string `json:"accountType"`
									} `json:"author"`
								} `json:"post"`
								ReplyingTo map[string]any `json:"replyingTo"`
							} `json:"items"`
						} `json:"replies"`
					} `json:"items"`
				} `json:"comments"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || len(page.Comments.Items) != 1 || len(page.Comments.Items[0].Replies.Items) != 1 {
				t.Fatalf("decode focused page = %+v, error %v; body=%s", page, err, response.Body.String())
			}
			item := page.Comments.Items[0].Replies.Items[0]
			if item.Post.Author.AccountType != "business" {
				t.Fatalf("visible focused child accountType = %q", item.Post.Author.AccountType)
			}
			if item.ReplyingTo != nil {
				t.Fatalf("blocked focused parent leaked through replyingTo: %+v", item.ReplyingTo)
			}
		})
	}
}

func TestGetPostComments_DeepFocusedReplyResolvesCommentAncestor(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	reply1 := testReplyRow("did:plc:dave", "reply-1", "one", root.URI, comment.URI, base.Add(2*time.Minute))
	reply2 := testReplyRow("did:plc:erin", "reply-2", "two", root.URI, reply1.URI, base.Add(3*time.Minute))
	focusedReply := testReplyRow("did:plc:frank", "reply-3", "three", root.URI, reply2.URI, base.Add(4*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		replyRows:   []*api.PostRow{reply1, reply2, focusedReply},
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			reply2.URI:       reply2,
			reply1.URI:       reply1,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
		"did:plc:erin":  "erin.example",
		"did:plc:frank": "frank.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Post.URI != comment.URI || resp.Comments.Items[0].Placement != "focused" {
		t.Fatalf("comments = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies.Items
	if len(replies) != 3 {
		t.Fatalf("replies = %+v", replies)
	}
	got := replies[2]
	if got.Post.URI != focusedReply.URI || !got.Flattened || got.ReplyingTo == nil || got.ReplyingTo.URI != reply2.URI {
		t.Fatalf("focused reply = %+v", got)
	}
}

func TestListCommentReplies_DefaultLimit(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{one: testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 10 {
		t.Fatalf("default limit = %d, want 10", store.lastReplyLimit)
	}
}

func TestListCommentReplies_LimitCapsAt10(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{one: testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?limit=500", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 10 {
		t.Fatalf("capped limit = %d, want 10", store.lastReplyLimit)
	}
}

func TestListCommentReplies_MissingTarget_404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{oneErr: api.ErrPostNotFound}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/missing/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "missing")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 0 {
		t.Fatalf("ListCommentBranchReplies should not run for missing target")
	}
}

func TestListCommentReplies_RejectsNonCommentTarget(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	reply := testReplyRow("did:plc:carol", "reply", "reply", root.URI, comment.URI, base.Add(2*time.Minute))

	tests := []struct {
		name string
		row  *api.PostRow
		path string
	}{
		{name: "root", row: root, path: "/v1/posts/did:plc:alice/root/replies"},
		{name: "nested reply", row: reply, path: "/v1/posts/did:plc:carol/reply/replies"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := &fakePostStore{one: tc.row}
			h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
			req := authedPostPathReq(http.MethodGet, tc.path, "", "did:plc:viewer")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			var body envelope.Error
			_ = json.NewDecoder(rr.Body).Decode(&body)
			if body.Error != "invalid_post_role" {
				t.Fatalf("error = %q", body.Error)
			}
			if store.lastReplyParentURI != "" {
				t.Fatalf("ListCommentBranchReplies should not run for non-comment target")
			}
		})
	}
}

func TestListCommentReplies_BadDID_400(t *testing.T) {
	t.Parallel()
	h := api.ListCommentRepliesHandler(&fakePostStore{}, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/not-a-did/root/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "root")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListCommentReplies_InvalidCursor_400(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{
		one:      testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now()),
		replyErr: envelope.ErrInvalidCursor,
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?cursor=bad", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "invalid_cursor" {
		t.Fatalf("error = %q", body.Error)
	}
}
