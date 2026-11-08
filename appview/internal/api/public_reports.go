package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

// PublicReportHandler uses the same private intake and case attachment as
// member reports. Caller credentials never establish a reporter identity here.
func PublicReportHandler(pages PublicPageReader, resolver HandleResolver, reports ReportCreator, resource string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		runID := middleware.GetRunID(ctx)
		fail := func(status int, code string) {
			envelope.WriteError(w, status, code, "Report could not be submitted.", runID, nil)
		}
		if r.URL.RawQuery != "" {
			fail(400, "invalid_request")
			return
		}
		req, details, ok := decodeValidateReport(w, r, runID)
		if !ok {
			return
		}
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		if resource == "post" {
			raw = r.PathValue("did")
		}
		if raw == "me" || raw == "recent" {
			fail(404, "not_found")
			return
		}
		did, err := resolveToDID(ctx, raw, resolver)
		if err != nil {
			if errors.Is(err, errInvalidIdentifier) {
				fail(400, "invalid_identifier")
			} else {
				fail(503, "identity_unavailable")
			}
			return
		}
		contact, _ := NormalizeReportContactEmail(req.ContactEmail)
		input := CreateReportInput{
			SubjectDID: did.String(), ReasonType: req.ReasonType, Details: details,
			ContactEmail: contact, ForwardingStatus: "prepared_not_submitted", ForwardingPreparedAt: time.Now().UTC(),
		}
		if resource == "post" {
			key, err := syntax.ParseRecordKey(r.PathValue("rkey"))
			if err != nil {
				fail(400, "invalid_identifier")
				return
			}
			post, err := pages.PublicPost(ctx, did, key)
			if err != nil {
				writePublicReadError(w, r, err)
				return
			}
			collection, uri, cid, rkey := craftskyPostNSID, post.URI.String(), post.CID.String(), post.Rkey.String()
			input.SubjectType = ReportSubjectPost
			input.SubjectCollection, input.SubjectURI, input.SubjectCIDSnapshot, input.SubjectRkey = &collection, &uri, &cid, &rkey
		} else {
			profile, err := pages.PublicProfile(ctx, did)
			if err != nil {
				writePublicReadError(w, r, err)
				return
			}
			input.SubjectType = ReportSubjectAccount
			input.SubmittedHandleSnapshot = optionalString(profile.Handle.String())
		}
		row, err := reports.CreateReport(ctx, input)
		if err != nil {
			observability.ReportRequestFailure(ctx, err, "api.PublicReportHandler", "store")
			fail(503, "unavailable")
			return
		}
		writeAcceptedReport(w, row.ID)
	})
}
