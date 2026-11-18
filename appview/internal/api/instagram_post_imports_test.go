package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

const instagramSourceKey = "abcdef0123456789abcdef0123456789"

type postImportAccounts struct{ account *instagram.AccountView }

func (a postImportAccounts) GetAccount(context.Context, syntax.DID) (*instagram.AccountView, error) {
	return a.account, nil
}

func verifiedPostImportAccount() *instagram.AccountView {
	return &instagram.AccountView{State: instagram.LinkActive, VerifiedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func instagramPostRequest(body, key string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(body))
	r.Header.Set("Idempotency-Key", key)
	ctx := middleware.WithDID(r.Context(), "did:plc:alice")
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	return r.WithContext(ctx)
}

func TestInstagramPostCreateRequiresActiveUncontestedVerification(t *testing.T) {
	for _, state := range []string{"missing", "inactive", "disputed", "conflict", "unverified"} {
		t.Run(state, func(t *testing.T) {
			account := verifiedPostImportAccount()
			switch state {
			case "missing":
				account = nil
			case "inactive":
				account.State = instagram.LinkMembershipInactive
			case "disputed":
				account.State = instagram.LinkDisputed
			case "conflict":
				account.ConflictPending = true
			case "unverified":
				account.VerifiedAt = time.Time{}
			}
			commands := &recordingAppendCommands{}
			handler := api.CreatePostHandler(&fakePostStore{}, nil, fakeResolver{}, api.DefaultMediaLimits(), nilLogger(), api.CreatePostHandlerOptions{Commands: commands, InstagramAccounts: postImportAccounts{account}})
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, instagramPostRequest(`{"text":"old caption","sponsored":false,"instagramImport":{"sourceKey":"`+instagramSourceKey+`","createdAt":"2020-06-15T12:00:00Z"}}`, uuid.NewString()))
			assertAPIError(t, rr, http.StatusForbidden, "instagram_verification_required")
			if commands.calls != 0 {
				t.Fatal("unverified import reached write commands")
			}
		})
	}
}

func TestInstagramPostCreatePreservesOriginalDateProvenanceAndSourceIdempotency(t *testing.T) {
	commands := &recordingAppendCommands{}
	commands.execute = func(request pdscommands.AppendCommandRequest) (pdscommands.CommandResult, error) {
		raw, err := request.BuildRecord(time.Now().UTC())
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		var record map[string]any
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		if record["createdAt"] != "2020-06-15T12:00:00Z" || record["externalImport"].(map[string]any)["source"] != "instagram" {
			t.Fatalf("historical record=%s", raw)
		}
		if strings.Contains(string(raw), instagramSourceKey) || strings.Contains(string(raw), "instagramImport") {
			t.Fatalf("private source leaked into record=%s", raw)
		}
		if request.FinalizeAccepted == nil {
			t.Fatal("missing atomic import finalizer")
		}
		terminal, err := request.Accepted(pdscommands.AuthoritativeRecord{URI: "at://did:plc:alice/social.craftsky.feed.post/3imported", CID: "bafy-imported", Record: raw})
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	handler := api.CreatePostHandler(&fakePostStore{}, nil, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger(), api.CreatePostHandlerOptions{Commands: commands, InstagramAccounts: postImportAccounts{verifiedPostImportAccount()}})
	var first syntax.RecordKey
	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, instagramPostRequest(`{"text":"old caption","sponsored":false,"instagramImport":{"sourceKey":"`+instagramSourceKey+`","createdAt":"2020-06-15T12:00:00Z"}}`, uuid.NewString()))
		if rr.Code != http.StatusCreated {
			t.Fatalf("response=%d %s", rr.Code, rr.Body.String())
		}
		var response api.PostResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.ExternalImport == nil || response.ExternalImport.Source != "instagram" || !response.CreatedAt.Equal(time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)) {
			t.Fatalf("response chronology/provenance=%s", rr.Body.String())
		}
		if i == 0 {
			first = commands.request.SelectedRkey
		} else if first != commands.request.SelectedRkey {
			t.Fatal("new client operation key could duplicate source")
		}
	}
}

func TestInstagramPostCreateValidatesImportMetadata(t *testing.T) {
	for _, metadata := range []string{
		`{"sourceKey":"not-a-source","createdAt":"2020-06-15T12:00:00Z"}`,
		`{"sourceKey":"` + instagramSourceKey + `","createdAt":"invalid"}`,
		`{"sourceKey":"` + instagramSourceKey + `","createdAt":"2099-01-01T00:00:00Z"}`,
		`{"sourceKey":"` + instagramSourceKey + `","createdAt":"2000-01-01T00:00:00Z"}`,
	} {
		req, err := api.DecodePostCreate(strings.NewReader(`{"text":"old caption","sponsored":false,"instagramImport":` + metadata + `}`))
		if err != nil {
			t.Fatal(err)
		}
		if api.ValidatePostCreate(req) == nil {
			t.Fatalf("accepted invalid metadata=%s", metadata)
		}
	}
	if _, err := api.DecodePostCreate(strings.NewReader(`{"text":"old caption","sponsored":false,"createdAt":"2020-06-15T12:00:00Z"}`)); err == nil {
		t.Fatal("ordinary create accepted createdAt")
	}
}

type recordingImportLookup struct {
	calls int
	owner syntax.DID
	keys  []string
}

func (s *recordingImportLookup) Lookup(_ context.Context, owner syntax.DID, keys []string) ([]instagram.PostImport, error) {
	s.calls++
	s.owner = owner
	s.keys = keys
	return []instagram.PostImport{{SourceKey: instagramSourceKey, URI: "at://did:plc:alice/social.craftsky.feed.post/3imported", Kind: "project"}}, nil
}

func TestInstagramPostLookupScopesOwnerAndBoundsInput(t *testing.T) {
	store := &recordingImportLookup{}
	handler := api.LookupInstagramPostImportsHandler(postImportAccounts{verifiedPostImportAccount()}, store)
	tooManyKeys := make([]string, 101)
	for i := range tooManyKeys {
		tooManyKeys[i] = instagramSourceKey
	}
	tooManyJSON, _ := json.Marshal(map[string]any{"sourceKeys": tooManyKeys})
	for _, body := range []string{`{}`, `{"sourceKeys":[]}`, `{"sourceKeys":["invalid"]}`, string(tooManyJSON)} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, instagramPostRequest(body, uuid.NewString()))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("invalid input status=%d %s", rr.Code, rr.Body.String())
		}
	}
	if store.calls != 0 {
		t.Fatal("invalid keys reached lookup store")
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, instagramPostRequest(`{"sourceKeys":["`+instagramSourceKey+`"]}`, uuid.NewString()))
	if rr.Code != http.StatusOK || store.calls != 1 || store.owner != "did:plc:alice" || !strings.Contains(rr.Body.String(), `"kind":"project"`) {
		t.Fatalf("lookup=%d %s owner=%s", rr.Code, rr.Body.String(), store.owner)
	}
}

type fixedImportHistory struct{ item instagram.PostImport }

func (s fixedImportHistory) Lookup(context.Context, syntax.DID, []string) ([]instagram.PostImport, error) {
	return []instagram.PostImport{s.item}, nil
}

func TestInstagramPostCreateReplaysAcceptedCommandWithPinnedEarlierDayIdentity(t *testing.T) {
	key := uuid.New()
	history := fixedImportHistory{instagram.PostImport{SourceKey: instagramSourceKey, URI: "at://did:plc:alice/social.craftsky.feed.post/3earlierday", Kind: "standard", OperationKey: key}}
	commands := &recordingAppendCommands{result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusCreated, ResponseBody: json.RawMessage(`{"uri":"at://did:plc:alice/social.craftsky.feed.post/3earlierday"}`)}}}
	handler := api.CreatePostHandler(&fakePostStore{}, nil, fakeResolver{}, api.DefaultMediaLimits(), nilLogger(), api.CreatePostHandlerOptions{Commands: commands, InstagramAccounts: postImportAccounts{verifiedPostImportAccount()}, InstagramImports: history})
	body := `{"text":"old caption","sponsored":false,"instagramImport":{"sourceKey":"` + instagramSourceKey + `","createdAt":"2020-06-16T12:00:00Z"}}`
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, instagramPostRequest(body, key.String()))
	if rr.Code != http.StatusCreated || commands.calls != 1 {
		t.Fatalf("exact accepted replay rejected because source key was pinned on earlier day: %d %s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, instagramPostRequest(body, uuid.NewString()))
	assertAPIError(t, rr, http.StatusConflict, "instagram_post_already_imported")
	if commands.calls != 1 {
		t.Fatal("new draft for accepted/deleted imported source reached write commands")
	}
}

func TestInstagramProjectVideoUsesExistingVerifiedVideoPublication(t *testing.T) {
	commands := &recordingAppendCommands{}
	commands.execute = func(request pdscommands.AppendCommandRequest) (pdscommands.CommandResult, error) {
		raw, err := request.BuildRecord(time.Now())
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		var record map[string]any
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		if record["project"] == nil || record["embed"].(map[string]any)["$type"] != "app.bsky.embed.video" || record["externalImport"] == nil {
			t.Fatalf("project/video/provenance missing: %s", raw)
		}
		if strings.Contains(string(raw), "private-job-id") || strings.Contains(string(raw), instagramSourceKey) {
			t.Fatalf("private import data entered PDS record: %s", raw)
		}
		terminal, err := request.Accepted(pdscommands.AuthoritativeRecord{URI: "at://did:plc:alice/social.craftsky.feed.post/3importedvideo", CID: "bafy-imported", Record: raw})
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	verifier := &fakeVideoCompletionVerifier{}
	handler := api.CreatePostHandler(&fakePostStore{}, nil, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger(), api.CreatePostHandlerOptions{Commands: commands, InstagramAccounts: postImportAccounts{verifiedPostImportAccount()}, VideoCompletionVerifier: verifier})
	body := `{"text":"old video caption","sponsored":false,"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}},"embed":{"video":{"jobId":"private-job-id","blob":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":123}}},"instagramImport":{"sourceKey":"` + instagramSourceKey + `","createdAt":"2020-06-15T12:00:00Z"}}`
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, instagramPostRequest(body, uuid.NewString()))
	if rr.Code != http.StatusCreated || commands.calls != 1 || verifier.calls != 1 {
		t.Fatalf("video import publication=%d %s commands=%d verification=%d", rr.Code, rr.Body.String(), commands.calls, verifier.calls)
	}
}
