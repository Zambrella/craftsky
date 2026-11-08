package api_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
)

type publicReportPages struct{ publicPageStub }

func (s *publicReportPages) PublicPost(_ context.Context, did syntax.DID, key syntax.RecordKey) (*api.PublicPost, error) {
	s.calls++
	return &api.PublicPost{
		Author: api.PublicAuthor{DID: did}, Rkey: key, CID: "cid",
		URI: syntax.ATURI("at://" + did.String() + "/social.craftsky.feed.post/" + key.String()),
	}, s.err
}

func TestPublicReportAcceptsPublicTargetsWithoutReporterIdentity(t *testing.T) {
	for _, kind := range []string{"post", "profile"} {
		t.Run(kind, func(t *testing.T) {
			reports := &fakeReportCreator{}
			pages := &publicReportPages{}
			r := authedReportReq("POST", "/", `{"reasonType":"privacy","details":" private details ","contactEmail":" visitor@example.test "}`, "did:plc:spoof", "spoofed-device")
			r.SetPathValue("did", "did:plc:alice")
			r.SetPathValue("rkey", "post1")
			r.SetPathValue("handleOrDid", "did:plc:alice")
			w := httptest.NewRecorder()
			api.PublicReportHandler(pages, nil, reports, kind).ServeHTTP(w, r)
			if w.Code != 201 || pages.calls != 1 {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, pages.calls, w.Body.String())
			}
			input := reports.lastInput
			if input.ReporterDID != "" || input.DeviceID != nil || input.SubjectDID != "did:plc:alice" {
				t.Fatal("guest report trusted caller identity")
			}
			assertStringPtr(t, "contact", input.ContactEmail, ptrString("visitor@example.test"))
			assertStringPtr(t, "details", input.Details, ptrString("private details"))
			if kind == "post" {
				assertStringPtr(t, "subject", input.SubjectURI, ptrString("at://did:plc:alice/social.craftsky.feed.post/post1"))
			}
			if strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "visitor") || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("report acknowledgement leaked private fields")
			}
		})
	}
}

func TestPublicReportFailsClosedForUnavailableTargetsAndInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name, body, query string
		err               error
		status            int
	}{
		{name: "opt-out", body: `{"reasonType":"spam"}`, err: api.ErrPublicAuthenticationRequired, status: 403},
		{name: "missing", body: `{"reasonType":"spam"}`, err: api.ErrPostNotFound, status: 404},
		{name: "unavailable", body: `{"reasonType":"spam"}`, err: errors.New("PRIVATE_CANARY"), status: 503},
		{name: "email", body: `{"reasonType":"spam","contactEmail":"not an email"}`, status: 422},
		{name: "spoofed reporter", body: `{"reasonType":"spam","reporterDid":"did:plc:fake"}`, status: 400},
		{name: "query", body: `{"reasonType":"spam"}`, query: "?viewerDid=did:plc:fake", status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			pages := &publicReportPages{publicPageStub: publicPageStub{err: test.err}}
			reports := &fakeReportCreator{}
			r := httptest.NewRequest("POST", "/"+test.query, strings.NewReader(test.body))
			r.SetPathValue("did", "did:plc:alice")
			r.SetPathValue("rkey", "post1")
			w := httptest.NewRecorder()
			api.PublicReportHandler(pages, nil, reports, "post").ServeHTTP(w, r)
			if w.Code != test.status || reports.lastInput.SubjectDID != "" || strings.Contains(w.Body.String(), "PRIVATE_CANARY") {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
