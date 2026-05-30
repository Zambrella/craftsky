// appview/internal/api/report_test.go
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

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

type fakeReportPostTargets struct {
	target *api.PostReportTarget
}

func (f fakeReportPostTargets) ResolvePostReportTarget(context.Context, syntax.DID, syntax.RecordKey) (*api.PostReportTarget, error) {
	return f.target, nil
}

type fakeReportAccountTargets struct {
	target *api.AccountReportTarget
}

func (f fakeReportAccountTargets) ResolveAccountReportTarget(context.Context, string) (*api.AccountReportTarget, error) {
	return f.target, nil
}

type fakeReportCreator struct {
	lastInput api.CreateReportInput
}

func (f *fakeReportCreator) CreateReport(_ context.Context, input api.CreateReportInput) (*api.ReportRow, error) {
	f.lastInput = input
	return &api.ReportRow{ID: "report-1", CreatedAt: time.Now()}, nil
}

type fakeReportForwarder struct {
	lastInput api.ReportForwardingInput
}

func (f *fakeReportForwarder) Prepare(_ context.Context, input api.ReportForwardingInput) (api.ForwardingMetadata, error) {
	f.lastInput = input
	schema := "atproto-create-report-v0"
	return api.ForwardingMetadata{Status: "prepared_not_submitted", SchemaVersion: &schema, PreparedAt: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)}, nil
}

func TestReportPostHandler_AcceptsValidRequest(t *testing.T) {
	t.Parallel()
	reports := &fakeReportCreator{}
	forwarder := &fakeReportForwarder{}
	h := api.ReportPostHandler(fakeReportPostTargets{target: &api.PostReportTarget{
		DID:         "did:plc:bob",
		Rkey:        "3lf2abc",
		URI:         "at://did:plc:bob/social.craftsky.feed.post/3lf2abc",
		CIDSnapshot: "bafy-post-v1",
	}}, reports, forwarder, nilLogger())
	req := authedReportReq(http.MethodPost, "/v1/posts/did:plc:bob/3lf2abc/reports", `{"reasonType":"spam","details":" private details "}`, "did:plc:alice", "device-1")
	req.SetPathValue("did", "did:plc:bob")
	req.SetPathValue("rkey", "3lf2abc")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 2 || body["reportId"] != "report-1" || body["status"] != "accepted" {
		t.Fatalf("body = %v, want minimal accepted response", body)
	}
	if reports.lastInput.ReporterDID != "did:plc:alice" || reports.lastInput.SubjectType != api.ReportSubjectPost {
		t.Fatalf("report input = %+v", reports.lastInput)
	}
	assertStringPtr(t, "details", reports.lastInput.Details, ptrString("private details"))
	assertStringPtr(t, "deviceID", reports.lastInput.DeviceID, ptrString("device-1"))
	if forwarder.lastInput.Details == nil || *forwarder.lastInput.Details != "private details" {
		t.Fatalf("forwarder input details = %v", forwarder.lastInput.Details)
	}
	if strings.Contains(rr.Body.String(), "private details") || strings.Contains(rr.Body.String(), "spam") {
		t.Fatalf("response leaked private report data: %s", rr.Body.String())
	}
}

func TestReportProfileHandler_AcceptsValidRequest(t *testing.T) {
	t.Parallel()
	reports := &fakeReportCreator{}
	forwarder := &fakeReportForwarder{}
	h := api.ReportProfileHandler(fakeReportAccountTargets{target: &api.AccountReportTarget{
		DID:                     "did:plc:bob",
		SubmittedHandleSnapshot: "bob.craftsky.social",
	}}, reports, forwarder, nilLogger())
	req := authedReportReq(http.MethodPost, "/v1/profiles/bob.craftsky.social/reports", `{"reasonType":"impersonation"}`, "did:plc:alice", "device-2")
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 2 || body["reportId"] != "report-1" || body["status"] != "accepted" {
		t.Fatalf("body = %v, want minimal accepted response", body)
	}
	if reports.lastInput.SubjectType != api.ReportSubjectAccount || reports.lastInput.SubjectDID != "did:plc:bob" {
		t.Fatalf("report input = %+v", reports.lastInput)
	}
	assertStringPtr(t, "handle snapshot", reports.lastInput.SubmittedHandleSnapshot, ptrString("bob.craftsky.social"))
}

func TestReportPostHandler_RejectsSelfReport(t *testing.T) {
	t.Parallel()
	reports := &fakeReportCreator{}
	forwarder := &fakeReportForwarder{}
	h := api.ReportPostHandler(fakeReportPostTargets{target: &api.PostReportTarget{
		DID:         "did:plc:alice",
		Rkey:        "3lf2abc",
		URI:         "at://did:plc:alice/social.craftsky.feed.post/3lf2abc",
		CIDSnapshot: "bafy-post-v1",
	}}, reports, forwarder, nilLogger())
	req := authedReportReq(http.MethodPost, "/v1/posts/did:plc:alice/3lf2abc/reports", `{"reasonType":"spam"}`, "did:plc:alice", "device-1")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "3lf2abc")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid_report_target") {
		t.Fatalf("body = %s, want invalid_report_target", rr.Body.String())
	}
	if reports.lastInput.ReporterDID != "" {
		t.Fatalf("report was persisted: %+v", reports.lastInput)
	}
}

func TestReportProfileHandler_RejectsSelfReport(t *testing.T) {
	t.Parallel()
	reports := &fakeReportCreator{}
	forwarder := &fakeReportForwarder{}
	h := api.ReportProfileHandler(fakeReportAccountTargets{target: &api.AccountReportTarget{
		DID:                     "did:plc:alice",
		SubmittedHandleSnapshot: "alice.craftsky.social",
	}}, reports, forwarder, nilLogger())
	req := authedReportReq(http.MethodPost, "/v1/profiles/alice.craftsky.social/reports", `{"reasonType":"impersonation"}`, "did:plc:alice", "device-2")
	req.SetPathValue("handleOrDid", "alice.craftsky.social")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid_report_target") {
		t.Fatalf("body = %s, want invalid_report_target", rr.Body.String())
	}
	if reports.lastInput.ReporterDID != "" {
		t.Fatalf("report was persisted: %+v", reports.lastInput)
	}
}

func authedReportReq(method, path, body, did, deviceID string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	ctx := middleware.WithDID(req.Context(), syntax.DID(did))
	ctx = middleware.WithDeviceID(ctx, deviceID)
	return req.WithContext(ctx)
}
