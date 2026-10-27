package index_test

import (
	"bytes"
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

// IT-002 / FR-002: the private owner form is based on one current version.
func TestPostEditSeedLoadsCurrentOwnerVersion(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	event.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	f.ingest(event)
	f.drain()
	handler := api.GetPostEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{})
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:actor/3aaaaaaaaaaa5/edit", nil)
	request.SetPathValue("did", event.DID.String())
	request.SetPathValue("rkey", event.Rkey.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Post                    api.PostResponse
		BaseCID, RecordRevision string
		Editable                struct {
			Text      string
			Sponsored bool
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Post.CID != event.CID.String() || result.BaseCID != event.CID.String() || result.RecordRevision != event.Rev.String() || result.Editable.Text != "hello recipient" {
		t.Fatalf("incomplete owner seed: %s", response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("private seed may be cached: %v", response.Header())
	}
}

// UT-001 / FR-002, FR-005: seed all supported project values without asking
// the client to round-trip unknown authored extensions.
func TestPostEditSeedIncludesSupportedProjectFields(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", `,"project":{"futureProject":9007199254740993,"common":{"craftType":"social.craftsky.feed.defs#sewing","title":"Linen shirt","pattern":{"selfDrafted":true,"name":"My pattern","futurePattern":"keep server-side"},"materials":[{"text":"Linen","futureMaterial":"keep server-side"}],"futureCommon":"keep server-side"},"details":{"$type":"social.craftsky.project.sewing#details","sizeMade":"M","futureDetail":"keep server-side"}}`)
	event.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	f.ingest(event)
	f.drain()
	handler := api.GetPostEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{})
	request := httptest.NewRequest(http.MethodGet, "/edit", nil)
	request.SetPathValue("did", event.DID.String())
	request.SetPathValue("rkey", event.Rkey.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Editable struct {
			Project struct {
				Common struct {
					Title   string
					Pattern struct {
						SelfDrafted bool
						Name        string
					}
					Materials []struct {
						ExistingIndex int
						Text          string
					}
				}
				Details struct{ SizeMade string }
			}
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	project := result.Editable.Project
	if project.Common.Title != "Linen shirt" || !project.Common.Pattern.SelfDrafted || project.Common.Pattern.Name != "My pattern" || project.Details.SizeMade != "M" || len(project.Common.Materials) != 1 || project.Common.Materials[0].Text != "Linen" {
		t.Fatalf("supported project values missing: %s", response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(fields["editable"], []byte("future")) {
		t.Fatalf("raw extensions leaked to editable form: %s", fields["editable"])
	}
}

// IT-002 / RULE-002: an invalidated source must not authorize an editor even
// when the last visible serving version still has the same CID.
func TestPostEditSeedWaitsForValidatedCurrentSource(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	f.ingest(event)
	f.drain()
	f.exec(`UPDATE tap_source_records SET semantic_validation_status='invalid' WHERE uri=$1`, event.URI)
	handler := api.GetPostEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{})
	request := httptest.NewRequest(http.MethodGet, "/edit", nil)
	request.SetPathValue("did", event.DID.String())
	request.SetPathValue("rkey", event.Rkey.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
	if response.Code != 503 || response.Header().Get("Retry-After") != "1" || bytes.Contains(response.Body.Bytes(), []byte("hello recipient")) {
		t.Fatalf("unvalidated source exposed a form: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestPostEditSeedWaitsForInitialProjection(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	f.ingest(event)
	handler := api.GetPostEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{})
	read := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/edit", nil)
		request.SetPathValue("did", event.DID.String())
		request.SetPathValue("rkey", event.Rkey.String())
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
		return response
	}
	if response := read(); response.Code != 503 || response.Header().Get("Retry-After") != "1" {
		t.Fatalf("unprojected source is not retryable: status=%d body=%s", response.Code, response.Body.String())
	}
	f.drain()
	if response := read(); response.Code != 200 {
		t.Fatalf("projected source did not become editable: %d %s", response.Code, response.Body.String())
	}
}

func TestPostEditSeedEligibilityAndVersionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, sql string
		viewer    syntax.DID
		status    int
	}{
		{"nonOwner", "", "did:plc:recipient", 403},
		{"anonymous", "", "", 403},
		{"sourceAhead", `UPDATE tap_source_records SET cid='new-source-cid' WHERE did='did:plc:actor'`, "did:plc:actor", 503},
		{"uncertain", `UPDATE tap_source_records SET ordering_status='uncertain' WHERE did='did:plc:actor'`, "did:plc:actor", 503},
		{"generationChanged", `UPDATE tap_source_records SET projection_generation=2 WHERE did='did:plc:actor'`, "did:plc:actor", 503},
		{"missingMembership", `DELETE FROM craftsky_profiles WHERE did='did:plc:actor'`, "did:plc:actor", 404},
		{"departed", `UPDATE owner_lifecycles SET state='departed',generation=2,transitioned_at=now(),updated_at=now() WHERE owner_did='did:plc:actor'`, "did:plc:actor", 404},
		{"deleted", `UPDATE tap_source_records SET action='delete',record=NULL,record_bytes=0 WHERE did='did:plc:actor'`, "did:plc:actor", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newNotificationPipeline(t)
			event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
			f.ingest(event)
			f.drain()
			if test.sql != "" {
				f.exec(test.sql)
			}
			handler := api.GetPostEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{})
			request := httptest.NewRequest(http.MethodGet, "/edit", nil)
			request.SetPathValue("did", event.DID.String())
			request.SetPathValue("rkey", event.Rkey.String())
			if test.viewer != "" {
				request = request.WithContext(middleware.WithDID(request.Context(), test.viewer))
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || bytes.Contains(response.Body.Bytes(), []byte("hello recipient")) {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.status, response.Body.String())
			}
		})
	}
}
