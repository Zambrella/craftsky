package observability

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"social.craftsky/appview/internal/auth"
)

func TestClassifyPDSErrorUsesBoundedCategoriesAndStages(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want PDSCategory
	}{
		{name: "nil success", err: nil, want: PDSCategoryNone},
		{name: "timeout", err: context.DeadlineExceeded, want: PDSCategoryTimeout},
		{name: "network", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, want: PDSCategoryNetwork},
		{name: "auth", err: auth.ErrPDSSessionExpired, want: PDSCategoryAuth},
		{name: "validation", err: &atclient.APIError{StatusCode: http.StatusBadRequest}, want: PDSCategoryValidation},
		{name: "forbidden", err: &atclient.APIError{StatusCode: http.StatusForbidden}, want: PDSCategoryForbidden},
		{name: "not found", err: auth.ErrRecordNotFound, want: PDSCategoryNotFound},
		{name: "rate limited", err: &atclient.APIError{StatusCode: http.StatusTooManyRequests}, want: PDSCategoryRateLimited},
		{name: "server", err: &atclient.APIError{StatusCode: http.StatusBadGateway}, want: PDSCategoryServer},
		{name: "unexpected", err: errors.New("strange failure"), want: PDSCategoryUnexpected},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyPDSError(tc.err); got != tc.want {
				t.Fatalf("ClassifyPDSError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}

	for _, stage := range []PDSStage{
		PDSStageSessionResume,
		PDSStageRequestBuild,
		PDSStagePDSRequest,
		PDSStagePDSResponse,
		PDSStagePostWriteIndexingWait,
	} {
		if got := NormalizePDSStage(string(stage)); got != stage {
			t.Fatalf("NormalizePDSStage(%q) = %q, want same", stage, got)
		}
	}
	if got := NormalizePDSStage("did:plc:raw"); got != PDSStageUnexpected {
		t.Fatalf("NormalizePDSStage(raw) = %q, want %q", got, PDSStageUnexpected)
	}
}

func TestPDSWriteOperationRegistryCoversCurrentWritePaths(t *testing.T) {
	for _, op := range []PDSOperation{
		PDSOperationOAuthSessionResume,
		PDSOperationProfilePutBsky,
		PDSOperationProfilePutCraftsky,
		PDSOperationPostCreate,
		PDSOperationPostDelete,
		PDSOperationBlobUpload,
		PDSOperationFollowCreate,
		PDSOperationFollowDelete,
		PDSOperationLikeCreate,
		PDSOperationLikeDelete,
		PDSOperationRepostCreate,
		PDSOperationRepostDelete,
	} {
		if !KnownPDSOperation(op) {
			t.Fatalf("KnownPDSOperation(%q) = false, want true", op)
		}
	}
	if KnownPDSOperation(PDSOperation("did:plc:raw")) {
		t.Fatal("raw/unbounded PDS operation was accepted")
	}
}
