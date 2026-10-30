package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

// IT-008: only a newly completed video can request bounded re-upload recovery.
func TestEditPostMissingReplacementVideoBlobSupportsRecovery(t *testing.T) {
	const blob = `{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"video/mp4","size":200}`
	for _, tc := range []struct {
		name                 string
		replacement, resumed bool
	}{
		{"replacement", true, false}, {"resumed replacement", true, true}, {"retained", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"`
			changes := `{"embed":{"$type":"app.bsky.embed.video","jobId":"completed-job","video":` + blob + `,"alt":"new video"}}`
			if !tc.replacement {
				original += `,"embed":{"$type":"app.bsky.embed.video","video":` + blob + `,"alt":"old video"}`
				changes = `{"embed":{"alt":"updated alt"}}`
			}
			original += `}`
			commands := postEditRejectingExecutor(func(ctx context.Context, request pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error) {
				if !tc.resumed {
					_, err := request.BuildPreparedRecord(pdscommands.AuthoritativeRecord{URI: request.URI, CID: request.ExpectedCID, Record: json.RawMessage(original)}, pdscommands.PreparedCommand{CreatedAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)})
					if err != nil {
						t.Fatalf("prepare: %v", err)
					}
				}
				return pdscommands.CommandResult{TerminalResult: request.Rejected(&atclient.APIError{StatusCode: 400, Name: "BlobNotFound"})}, nil
			})
			request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3existingpost", strings.NewReader(`{"changes":`+changes+`}`))
			request.SetPathValue("did", "did:plc:alice")
			request.SetPathValue("rkey", "3existingpost")
			request.Header.Set("If-Match", "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq")
			request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
			ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
			ctx = middleware.WithOwnerGeneration(ctx, 1)
			ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
			response := httptest.NewRecorder()
			api.CommandEditPostHandler(&fakePostStore{}, commands, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), api.EditPostHandlerOptions{VideoCompletionVerifier: &fakeVideoCompletionVerifier{}}).ServeHTTP(response, request.WithContext(ctx))
			if tc.replacement {
				assertAPIError(t, response, 502, "video_blob_missing")
			} else if strings.Contains(response.Body.String(), "video_blob_missing") {
				t.Fatalf("retained video must not request re-upload: %s", response.Body.String())
			}
		})
	}
}

type postEditRejectingExecutor func(context.Context, pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error)

func (f postEditRejectingExecutor) Put(ctx context.Context, request pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error) {
	return f(ctx, request)
}

// IT-001 first boundary: same addressed post and unchanged publication identity.
func TestEditPostUpdatesExistingRecord(t *testing.T) {
	const priorCID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	const uri = "at://did:plc:alice/social.craftsky.feed.post/3existingpost"
	commands := &recordingPostEditCommands{current: pdscommands.AuthoritativeRecord{
		URI: uri, CID: priorCID,
		Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T12:00:00.000Z","futureField":{"value":9007199254740993}}`),
	}}
	handler := api.CommandEditPostHandler(&fakePostStore{}, commands, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits())
	request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3existingpost", strings.NewReader(`{"changes":{"text":"after"}}`))
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3existingpost")
	request.Header.Set("If-Match", priorCID)
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if commands.calls != 1 || commands.request.URI != uri || commands.request.ExpectedCID != priorCID || commands.request.OperationKind != "post.edit" || !commands.request.StrictExpectedCID {
		t.Fatalf("wrong addressed edit: calls=%d request=%+v", commands.calls, commands.request)
	}
	var record map[string]json.RawMessage
	if err := json.Unmarshal(commands.desired, &record); err != nil {
		t.Fatal(err)
	}
	if string(record["text"]) != `"after"` || string(record["createdAt"]) != `"2026-01-01T12:00:00.000Z"` || string(record["futureField"]) != `{"value":9007199254740993}` {
		t.Fatalf("edit lost authored content: %s", commands.desired)
	}
	if string(record["editedAt"]) != `"2026-10-26T12:00:00Z"` {
		t.Fatalf("edit must use durable command time: %s", commands.desired)
	}
	var result struct {
		Post              *api.PostResponse `json:"post"`
		Changed           bool              `json:"changed"`
		AcceptedCID       string            `json:"acceptedCid"`
		EditedAt          *time.Time        `json:"editedAt"`
		EditTimeSource    string            `json:"editTimeSource"`
		ProjectionPending bool              `json:"projectionPending"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.AcceptedCID != "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m" || result.EditedAt == nil || !result.EditedAt.Equal(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("result=%s", response.Body.String())
	}
}

type recordingPostEditCommands struct {
	pool     *pgxpool.Pool
	calls    int
	current  pdscommands.AuthoritativeRecord
	desired  json.RawMessage
	request  pdscommands.AddressedPutCommandRequest
	terminal pdscommands.TerminalResult
}

// IT-001: an unchanged save is not a new content edit.
func TestEditPostNoOpPreservesEditTime(t *testing.T) {
	commands := &recordingPostEditCommands{current: pdscommands.AuthoritativeRecord{
		URI:    "at://did:plc:alice/social.craftsky.feed.post/3existingpost",
		CID:    "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq",
		Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T12:00:00.000Z","editedAt":"2026-10-25T11:00:00.000Z"}`),
	}}
	handler := api.CommandEditPostHandler(&fakePostStore{}, commands, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits())
	request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3existingpost", strings.NewReader(`{"changes":{"text":"before"}}`))
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3existingpost")
	request.Header.Set("If-Match", commands.current.CID.String())
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var value struct{ EditedAt string }
	if err := json.Unmarshal(commands.desired, &value); err != nil {
		t.Fatal(err)
	}
	if value.EditedAt != "2026-10-25T11:00:00.000Z" {
		t.Fatalf("no-op advanced edit time: %s", commands.desired)
	}
}

// UT-002: false, empty and explicit removal must not fall back to old values.
func TestEditPostPresenceChanges(t *testing.T) {
	for _, tc := range []struct{ name, changes, field, want string }{
		{"sponsorship false", `{"sponsored":false}`, "sponsored", `false`},
		{"empty languages", `{"langs":[]}`, "langs", `[]`},
		{"remove languages", `{"langs":null}`, "langs", ``},
		{"remove facets", `{"facets":null}`, "facets", ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":true,"langs":["en"],"facets":[],"createdAt":"2026-01-01T12:00:00Z"}`, tc.changes)
			if response.Code != 200 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(commands.desired, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields[tc.field]) != tc.want {
				t.Fatalf("field %s = %s, want %s", tc.field, fields[tc.field], tc.want)
			}
		})
	}
}

// UT-004 / FR-003: retained media is copied from the exact base, including
// unknown fields, while ordering and supported metadata remain editable.
func TestEditPostPreservesUnknownAspectRatioFields(t *testing.T) {
	const blob = `{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":200}`
	for _, kind := range []string{"image", "video"} {
		t.Run(kind, func(t *testing.T) {
			media := `,"images":[{"image":` + blob + `,"alt":"old","aspectRatio":{"width":4,"height":3,"futureRatio":9007199254740993}}]`
			changes := `{"images":[{"existingIndex":0,"aspectRatio":{"width":8,"height":3}}]}`
			if kind == "video" {
				media = `,"embed":{"$type":"app.bsky.embed.video","video":` + strings.Replace(blob, "image/jpeg", "video/mp4", 1) + `,"alt":"old","aspectRatio":{"width":4,"height":3,"futureRatio":9007199254740993}}`
				changes = `{"embed":{"aspectRatio":{"width":8,"height":3}}}`
			}
			commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"`+media+`}`, changes)
			if response.Code != 200 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if !bytes.Contains(commands.desired, []byte(`"futureRatio":9007199254740993`)) || !bytes.Contains(commands.desired, []byte(`"width":8`)) {
				t.Fatalf("known ratio edit lost authored sibling: %s", commands.desired)
			}
		})
	}
}

func TestEditPostReordersRetainedImagesWithoutLosingAuthoredFields(t *testing.T) {
	const image = `{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":200}`
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","images":[{"image":`+image+`,"alt":"first","futureImage":9007199254740993},{"image":`+image+`,"alt":"second"}]}`, `{"images":[{"existingIndex":1,"alt":"updated second"},{"existingIndex":0}]}`)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var record struct{ Images []map[string]json.RawMessage }
	if err := json.Unmarshal(commands.desired, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Images) != 2 || string(record.Images[0]["alt"]) != `"updated second"` || string(record.Images[1]["futureImage"]) != `9007199254740993` || record.Images[0]["existingIndex"] != nil {
		t.Fatalf("retained image merge is not lossless: %s", commands.desired)
	}
}

func TestEditPostRetainsVideoWithoutOriginalJob(t *testing.T) {
	const embed = `{"$type":"app.bsky.embed.video","video":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"video/mp4","size":200},"alt":"original video","futureVideo":9007199254740993}`
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","embed":`+embed+`}`, `{"text":"after"}`)
	if response.Code != 200 {
		t.Fatalf("retained video needs no old job: %d %s", response.Code, response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(commands.desired, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["embed"]) != embed {
		t.Fatalf("retained video changed: %s", commands.desired)
	}
}

func TestEditPostUpdatesExternalCardWithoutLosingThumb(t *testing.T) {
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","embed":{"$type":"app.bsky.embed.external","futureEmbed":9007199254740993,"external":{"uri":"https://example.com/pattern","title":"Old pattern","description":"Details","futureExternal":"keep","thumb":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":200}}}}`, `{"embed":{"external":{"title":"New pattern"}}}`)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var record struct {
		Embed struct {
			FutureEmbed json.Number
			External    struct {
				Title, Description, FutureExternal string
				Thumb                              json.RawMessage
			}
		}
	}
	if err := json.Unmarshal(commands.desired, &record); err != nil {
		t.Fatal(err)
	}
	if record.Embed.FutureEmbed != "9007199254740993" || record.Embed.External.Title != "New pattern" || record.Embed.External.Description != "Details" || record.Embed.External.FutureExternal != "keep" || len(record.Embed.External.Thumb) == 0 {
		t.Fatalf("external card lost retained values: %s", commands.desired)
	}
}

// UT-004 / FR-010: a newly attached video must use the existing verified
// completion flow; retained-video permission cannot authorize a new blob.
func TestEditPostCannotAttachUnverifiedVideo(t *testing.T) {
	_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"embed":{"$type":"app.bsky.embed.video","video":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6y6sqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"video/mp4","size":200}}}`)
	if response.Code != 422 {
		t.Fatalf("unverified new video accepted: %d %s", response.Code, response.Body.String())
	}
}

func servePostEdit(t *testing.T, current, changes string, options ...api.EditPostHandlerOptions) (*recordingPostEditCommands, *httptest.ResponseRecorder) {
	return servePostEditUsingStore(t, &fakePostStore{}, current, changes, options...)
}

func servePostEditUsingStore(t *testing.T, store interface {
	ReadAuthor(context.Context, string) (*api.PostAuthorRow, error)
}, current, changes string, options ...api.EditPostHandlerOptions) (*recordingPostEditCommands, *httptest.ResponseRecorder) {
	t.Helper()
	commands := &recordingPostEditCommands{current: pdscommands.AuthoritativeRecord{
		URI: "at://did:plc:alice/social.craftsky.feed.post/3existingpost",
		CID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", Record: json.RawMessage(current),
	}}
	request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3existingpost", strings.NewReader(`{"changes":`+changes+`}`))
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3existingpost")
	request.Header.Set("If-Match", commands.current.CID.String())
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	response := httptest.NewRecorder()
	api.CommandEditPostHandler(store, commands, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), options...).ServeHTTP(response, request.WithContext(ctx))
	return commands, response
}

func TestEditPostVerifiesReplacementVideo(t *testing.T) {
	verifier := &fakeVideoCompletionVerifier{}
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"embed":{"$type":"app.bsky.embed.video","jobId":"completed-job","video":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"video/mp4","size":200},"alt":"new video"}}`, api.EditPostHandlerOptions{VideoCompletionVerifier: verifier})
	if response.Code != 200 || verifier.calls != 1 || verifier.owner != "did:plc:alice" || verifier.jobID != "completed-job" {
		t.Fatalf("new video was not verified for this owner: status=%d calls=%d body=%s", response.Code, verifier.calls, response.Body.String())
	}
	if bytes.Contains(commands.desired, []byte("jobId")) {
		t.Fatalf("private completion handle published: %s", commands.desired)
	}
}

// UT-002: explicit null may clear an optional field, never a required boolean.
func TestEditPostRejectsNullSponsorship(t *testing.T) {
	_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":true,"createdAt":"2026-01-01T00:00:00Z"}`, `{"sponsored":null}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

// UT-002: offsets for the previous text cannot survive a changed text value.
func TestEditPostChangedTextDoesNotRetainOldFacets(t *testing.T) {
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"#before","sponsored":false,"facets":[{"index":{"byteStart":0,"byteEnd":7},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"before"}]}],"createdAt":"2026-01-01T00:00:00Z"}`, `{"text":"after"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(commands.desired, &fields); err != nil {
		t.Fatal(err)
	}
	if _, present := fields["facets"]; present {
		t.Fatalf("stale facets retained: %s", commands.desired)
	}
}

// UT-002 / IT-009: edit supported leaves without replacing authored siblings.
func TestEditPostProjectLeafChangesPreserveUnknownSiblings(t *testing.T) {
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00.000Z","project":{"futureProject":true,"common":{"craftType":"social.craftsky.feed.defs#sewing","title":"Old title","futureCommon":{"integer":9007199254740993},"pattern":{"name":"Keep name","selfDrafted":true,"futurePattern":"keep"}}}}`, `{"project":{"common":{"title":null,"pattern":{"selfDrafted":false}}}}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var fields struct {
		Project struct {
			FutureProject bool
			Common        struct {
				Title        *string
				FutureCommon struct{ Integer json.Number }
				Pattern      struct {
					Name, FuturePattern string
					SelfDrafted         bool
				}
			}
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(commands.desired)))
	decoder.UseNumber()
	if err := decoder.Decode(&fields); err != nil {
		t.Fatal(err)
	}
	common := fields.Project.Common
	if !fields.Project.FutureProject || common.Title != nil || common.FutureCommon.Integer != "9007199254740993" || common.Pattern.SelfDrafted || common.Pattern.Name != "Keep name" || common.Pattern.FuturePattern != "keep" {
		t.Fatalf("project merge lost data: %s", commands.desired)
	}
}

// UT-002 / IT-009: an explicit list can reorder retained elements without
// losing unfamiliar fields or publishing the editor's source indices.
func TestEditPostReordersMaterialsWithUnknownSiblings(t *testing.T) {
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing","materials":[{"text":"Cotton","future":"first"},{"text":"Thread","future":"second"}]}}}`, `{"project":{"common":{"materials":[{"existingIndex":1,"text":"Silk thread"},{"existingIndex":0}]}}}`)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var record struct {
		Project struct {
			Common struct {
				Materials []struct {
					Text, Future  string
					ExistingIndex *int
				}
			}
		}
	}
	if err := json.Unmarshal(commands.desired, &record); err != nil {
		t.Fatal(err)
	}
	entries := record.Project.Common.Materials
	if len(entries) != 2 || entries[0].Text != "Silk thread" || entries[0].Future != "second" || entries[1].Text != "Cotton" || entries[1].Future != "first" || entries[0].ExistingIndex != nil || entries[1].ExistingIndex != nil {
		t.Fatalf("retained material data lost: %s", commands.desired)
	}
}

// UT-002 / IT-009: specialized project fields are leaves of the retained variant.
func TestEditPostDetailsLeafPreservesUnknownSiblings(t *testing.T) {
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing"},"details":{"$type":"social.craftsky.project.sewing#details","projectType":"social.craftsky.project.defs#garment","sizeMade":"12","fitNotes":"old","futureDetail":9007199254740993}}}`, `{"project":{"details":{"fitNotes":"new"}}}`)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var record struct {
		Project struct {
			Details struct {
				Type                            string `json:"$type"`
				ProjectType, SizeMade, FitNotes string
				FutureDetail                    json.Number
			}
		}
	}
	if err := json.Unmarshal(commands.desired, &record); err != nil {
		t.Fatal(err)
	}
	d := record.Project.Details
	if d.Type != "social.craftsky.project.sewing#details" || d.ProjectType != "social.craftsky.project.defs#garment" || d.SizeMade != "12" || d.FitNotes != "new" || d.FutureDetail != "9007199254740993" {
		t.Fatalf("details lost content: %s", commands.desired)
	}
}

// UT-002: all remaining supported detail variants use presence-aware leaves.
func TestEditPostCraftDetailsChanges(t *testing.T) {
	for _, tc := range []struct{ craft, base, changes, field, want string }{
		{"knitting", `"needleSizeMm":"3","gauge":{"stitches":20,"rows":30,"measurement":10,"unit":"cm","futureGauge":"keep"}`, `"needleSizeMm":"4","gauge":{"stitches":22}`, "needleSizeMm", `"4"`},
		{"crochet", `"hookSizeMm":"3","gauge":{"stitches":20,"rows":30,"measurement":10,"unit":"cm","futureGauge":"keep"}`, `"hookSizeMm":"4","gauge":{"stitches":22}`, "hookSizeMm", `"4"`},
		{"quilting", `"size":"small","piecingTechnique":"hand"`, `"size":"large"`, "size", `"large"`},
	} {
		t.Run(tc.craft, func(t *testing.T) {
			current := `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#` + tc.craft + `"},"details":{"$type":"social.craftsky.project.` + tc.craft + `#details","futureDetail":9007199254740993,` + tc.base + `}}}`
			commands, response := servePostEdit(t, current, `{"project":{"details":{`+tc.changes+`}}}`)
			if response.Code != 200 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var record struct {
				Project struct{ Details map[string]json.RawMessage }
			}
			if err := json.Unmarshal(commands.desired, &record); err != nil {
				t.Fatal(err)
			}
			d := record.Project.Details
			if string(d[tc.field]) != tc.want || string(d["futureDetail"]) != "9007199254740993" {
				t.Fatalf("details lost content: %s", commands.desired)
			}
			if tc.craft != "quilting" {
				var gauge struct {
					Stitches, Rows, Measurement int
					Unit, FutureGauge           string
				}
				if err := json.Unmarshal(d["gauge"], &gauge); err != nil {
					t.Fatal(err)
				}
				if gauge.Stitches != 22 || gauge.Rows != 30 || gauge.Measurement != 10 || gauge.Unit != "cm" || gauge.FutureGauge != "keep" {
					t.Fatalf("gauge lost content: %s", d["gauge"])
				}
			}
		})
	}
}

// UT-002: annotations belong to the specific authored string they describe.
func TestEditPostChangedPatternTextDoesNotRetainOldFacets(t *testing.T) {
	commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing","pattern":{"name":"#old","nameFacets":[{"index":{"byteStart":0,"byteEnd":4},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"old"}]}],"future":"keep"}}}}`, `{"project":{"common":{"pattern":{"name":"new"}}}}`)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var record struct {
		Project struct {
			Common struct{ Pattern map[string]json.RawMessage }
		}
	}
	if err := json.Unmarshal(commands.desired, &record); err != nil {
		t.Fatal(err)
	}
	pattern := record.Project.Common.Pattern
	if _, present := pattern["nameFacets"]; present {
		t.Fatalf("stale pattern facets retained: %s", commands.desired)
	}
	if string(pattern["name"]) != `"new"` || string(pattern["future"]) != `"keep"` {
		t.Fatalf("pattern data lost: %s", commands.desired)
	}
}

// UT-004: rich-text offsets are UTF-8 bytes, not characters or partial runes.
func TestEditPostValidatesUTF8FacetRanges(t *testing.T) {
	for _, tc := range []struct {
		name, indices string
		status        int
	}{
		{"valid multibyte text", `"byteStart":5,"byteEnd":10`, 200},
		{"inside emoji", `"byteStart":1,"byteEnd":4`, 422},
		{"outside text", `"byteStart":5,"byteEnd":99`, 422},
		{"empty range", `"byteStart":5,"byteEnd":5`, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"🧶 café","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"facets":[{"index":{`+tc.indices+`},"features":[{"$type":"app.bsky.richtext.facet#link","uri":"https://example.com"}]}]}`)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
			}
		})
	}
}

func TestEditPostValidatesNestedUTF8FacetRanges(t *testing.T) {
	_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing","pattern":{"name":"🧶 pattern"}}}}`, `{"project":{"common":{"pattern":{"nameFacets":[{"index":{"byteStart":1,"byteEnd":4},"features":[{"$type":"app.bsky.richtext.facet#link","uri":"https://example.com"}]}]}}}}`)
	var result struct{ Fields map[string]string }
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 422 || result.Fields["project.common.pattern.nameFacets"] == "" {
		t.Fatalf("invalid nested facet accepted: %d %s", response.Code, response.Body.String())
	}
}

// UT-004: field feedback survives the durable command's rejected result.
func TestEditPostValidationReturnsActionableFields(t *testing.T) {
	_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"text":""}`)
	var result struct {
		Error  string
		Fields map[string]string
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 422 || result.Error != "validation_failed" || result.Fields["text"] == "" {
		t.Fatalf("validation lost field feedback: status=%d body=%s", response.Code, response.Body.String())
	}
}

// UT-004: edits retain creation's grapheme and UTF-8 byte limits.
func TestEditPostTextLimits(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		status     int
	}{
		{"grapheme limit", strings.Repeat("🧶", 2000), 200},
		{"over grapheme limit", strings.Repeat("🧶", 2001), 422},
		{"byte limit", "a" + strings.Repeat("\u0301", 9999) + "b", 200},
		{"over byte limit", "a" + strings.Repeat("\u0301", 10000) + "b", 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes, _ := json.Marshal(map[string]string{"text": tc.text})
			_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, string(changes))
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
			}
			if tc.status == 422 {
				var result struct {
					Error  string
					Fields map[string]string
				}
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Error != "validation_failed" || result.Fields["text"] == "" {
					t.Fatalf("limit error lost field: %s", response.Body.String())
				}
			}
		})
	}
}

// IT-009 / FR-005: an unfamiliar detail variant remains opaque to this editor.
func TestEditPostCannotRemoveUnsupportedProjectDetails(t *testing.T) {
	const record = `{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing"},"details":{"$type":"social.craftsky.project.future#details","futureInteger":9007199254740993}}}`
	for _, changes := range []string{`{"project":{"details":null}}`, `{"project":{"details":{"$type":"social.craftsky.project.sewing#details","sizeMade":"12"}}}`} {
		_, response := servePostEdit(t, record, changes)
		if response.Code != 400 {
			t.Fatalf("unsupported details were replaced: status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

// IT-009: preserving an opaque variant must still allow supported content edits.
func TestEditPostPreservesUnsupportedVariants(t *testing.T) {
	for _, tc := range []struct{ name, extra, changes, field string }{
		{"project details", `,"project":{"common":{"craftType":"social.craftsky.feed.defs#sewing","title":"before"},"details":{"$type":"social.craftsky.project.future#details","futureInteger":9007199254740993}}`, `{"project":{"common":{"title":"after"}}}`, "project"},
		{"embed", `,"embed":{"$type":"social.craftsky.embed.future","futureInteger":9007199254740993}`, `{"text":"after"}`, "embed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commands, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"`+tc.extra+`}`, tc.changes)
			if response.Code != 200 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(commands.desired, &fields); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(fields[tc.field]), `"futureInteger":9007199254740993`) {
				t.Fatalf("opaque data was lost: %s", commands.desired)
			}
		})
	}
}

// IT-009: open-enum values from an existing valid record survive unrelated
// supported edits; this does not allow introducing a new unsupported craft.
func TestEditPostPreservesExistingUnknownCraftValues(t *testing.T) {
	for _, tc := range []struct {
		name, common, changes string
		status                int
	}{
		{"retained primary", `{"craftType":"social.craftsky.feed.defs#futureCraft"}`, `{"text":"after"}`, 200},
		{"retained additional", `{"craftType":"social.craftsky.feed.defs#sewing","additionalCrafts":[{"craftType":"social.craftsky.feed.defs#futureCraft"}]}`, `{"project":{"common":{"title":"after"}}}`, 200},
		{"new primary", `{"craftType":"social.craftsky.feed.defs#sewing"}`, `{"project":{"common":{"craftType":"social.craftsky.feed.defs#futureCraft"}}}`, 422},
		{"new additional", `{"craftType":"social.craftsky.feed.defs#sewing"}`, `{"project":{"common":{"additionalCrafts":[{"craftType":"social.craftsky.feed.defs#futureCraft"}]}}}`, 422},
		{"blank retained additional", `{"craftType":"social.craftsky.feed.defs#sewing","additionalCrafts":[{"craftType":""}]}`, `{"text":"after"}`, 422},
		{"duplicate retained", `{"craftType":"social.craftsky.feed.defs#sewing","additionalCrafts":[{"craftType":"social.craftsky.feed.defs#futureCraft"}]}`, `{"project":{"common":{"additionalCrafts":[{"existingIndex":0},{"craftType":"social.craftsky.feed.defs#futureCraft"}]}}}`, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, response := servePostEdit(t, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":`+tc.common+`}}`, tc.changes)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
			}
		})
	}
}

func (c *recordingPostEditCommands) Put(ctx context.Context, request pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error) {
	c.calls++
	c.request = request
	var record json.RawMessage
	var err error
	if request.BuildPreparedRecord != nil {
		record, err = request.BuildPreparedRecord(c.current, pdscommands.PreparedCommand{CreatedAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)})
	} else {
		record, err = request.BuildRecord(c.current)
	}
	if err != nil {
		return pdscommands.CommandResult{TerminalResult: request.Rejected(errors.Join(pdscommands.ErrMalformedCommand, err))}, nil
	}
	c.desired = record
	result, err := request.Accepted(pdscommands.AuthoritativeRecord{URI: request.URI, CID: "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m", Record: record})
	c.terminal = result
	if err == nil && c.pool != nil && request.FinalizeAccepted != nil && !bytes.Equal(c.current.Record, record) {
		err = pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
			return request.FinalizeAccepted(ctx, tx, pdscommands.PreparedCommand{CreatedAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)}, pdscommands.AuthoritativeRecord{URI: request.URI, CID: "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m", Record: record})
		})
	}
	return pdscommands.CommandResult{TerminalResult: result}, err
}

type seededPostEditStore struct {
	*fakePostStore
	snapshot       *api.PostEditSnapshot
	err            error
	eligibilityErr error
}

func (s *seededPostEditStore) CheckPostEditAllowed(context.Context, syntax.DID, syntax.RecordKey) error {
	return s.eligibilityErr
}

func (s *seededPostEditStore) ReadEditSnapshot(context.Context, syntax.DID, syntax.RecordKey) (*api.PostEditSnapshot, error) {
	return s.snapshot, s.err
}

func TestEditPostRejectsCurrentlyIneligiblePost(t *testing.T) {
	store := &seededPostEditStore{fakePostStore: &fakePostStore{}, eligibilityErr: api.ErrPostNotFound}
	commands, response := servePostEditUsingStore(t, store, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"text":"after"}`)
	if response.Code != 404 || len(commands.desired) != 0 {
		t.Fatalf("ineligible post accepted: status=%d desired=%s body=%s", response.Code, commands.desired, response.Body.String())
	}
}

func TestEditPostCannotAddMentionAcrossBlock(t *testing.T) {
	store := &fakePostStore{authorizationErr: api.ErrInteractionBlocked}
	commands, response := servePostEditUsingStore(t, store, `{"$type":"social.craftsky.feed.post","text":"hello recipient","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]}`)
	if response.Code != 403 || len(commands.desired) != 0 {
		t.Fatalf("blocked mention accepted: %d %s", response.Code, response.Body.String())
	}
}

// UT-007 / FR-011: acceptance does not depend on public projection hydration.
type confirmationOnlyPostStore struct {
	fakePostStore
	reads int
}

func (s *confirmationOnlyPostStore) ReadEditSnapshot(context.Context, syntax.DID, syntax.RecordKey) (*api.PostEditSnapshot, error) {
	s.reads++
	return nil, api.ErrPostNotFound
}
func TestEditPostAcceptanceSkipsPublicHydration(t *testing.T) {
	store := &confirmationOnlyPostStore{}
	_, response := servePostEditUsingStore(t, store, `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`, `{"text":"after"}`)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if store.reads != 0 {
		t.Fatalf("accepted write performed %d public projection reads", store.reads)
	}
}

// IT-008 / IR-010: retained source values are not new preview fetch requests.
func TestEditPostRetainsExternalURIAcrossCaptionChange(t *testing.T) {
	const raw = `{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","embed":{"$type":"app.bsky.embed.external","external":{"uri":"at://did:plc:other/social.craftsky.feed.post/target","title":"A link","description":"Existing card","futureField":"preserve"}}}`
	commands, response := servePostEdit(t, raw, `{"text":"after"}`)
	if response.Code != 200 {
		t.Fatalf("unchanged valid URI was rejected: %d %s", response.Code, response.Body.String())
	}
	var post struct {
		Embed json.RawMessage `json:"embed"`
	}
	if err := json.Unmarshal(commands.desired, &post); err != nil {
		t.Fatal(err)
	}
	var original struct {
		Embed json.RawMessage `json:"embed"`
	}
	_ = json.Unmarshal([]byte(raw), &original)
	var before, after any
	_ = json.Unmarshal(original.Embed, &before)
	_ = json.Unmarshal(post.Embed, &after)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("retained embed changed: %s", post.Embed)
	}
	_, rejected := servePostEdit(t, raw, `{"embed":{"external":{"uri":"at://did:plc:other/social.craftsky.feed.post/new"}}}`)
	if rejected.Code != 422 {
		t.Fatalf("new unsupported preview URI accepted: %d %s", rejected.Code, rejected.Body.String())
	}
}
