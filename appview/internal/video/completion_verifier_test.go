package video_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"

	"social.craftsky/appview/internal/video"
)

const testVideoCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"

type fakeJobStatusClient struct {
	status *appbsky.VideoDefs_JobStatus
	err    error
}

func (f fakeJobStatusClient) GetJobStatus(context.Context, string) (*appbsky.VideoDefs_JobStatus, error) {
	return f.status, f.err
}

func TestCompletionVerifier_VerifiesOwnerJobAndExactBlob(t *testing.T) {
	t.Parallel()
	owner := syntax.DID("did:plc:alice")
	submitted := video.Blob{CID: syntax.CID(testVideoCID), MIMEType: "video/mp4", Size: 123}
	alreadyExists := "already_exists"
	upstreamSecret := "upstream detail for did:plc:bob job-foreign"

	tests := []struct {
		name    string
		status  *appbsky.VideoDefs_JobStatus
		callErr error
		wantOK  bool
	}{
		{name: "completed", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "did:plc:alice", lexBlob(testVideoCID, "video/mp4", 123)), wantOK: true},
		{name: "already exists with blob", status: jobStatusWithError("JOB_STATE_FAILED", "job-1", "did:plc:alice", lexBlob(testVideoCID, "video/mp4", 123), &alreadyExists), wantOK: true},
		{name: "incomplete", status: jobStatus("JOB_STATE_PROCESSING", "job-1", "did:plc:alice", nil)},
		{name: "failed", status: jobStatus("JOB_STATE_FAILED", "job-1", "did:plc:alice", nil)},
		{name: "already exists without blob", status: jobStatusWithError("JOB_STATE_FAILED", "job-1", "did:plc:alice", nil, &alreadyExists)},
		{name: "foreign owner", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "did:plc:bob", lexBlob(testVideoCID, "video/mp4", 123))},
		{name: "invalid owner", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "not-a-did", lexBlob(testVideoCID, "video/mp4", 123))},
		{name: "different job", status: jobStatus("JOB_STATE_COMPLETED", "job-foreign", "did:plc:alice", lexBlob(testVideoCID, "video/mp4", 123))},
		{name: "different CID", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "did:plc:alice", lexBlob("bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe", "video/mp4", 123))},
		{name: "different MIME", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "did:plc:alice", lexBlob(testVideoCID, "video/webm", 123))},
		{name: "different size", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "did:plc:alice", lexBlob(testVideoCID, "video/mp4", 124))},
		{name: "oversized service blob", status: jobStatus("JOB_STATE_COMPLETED", "job-1", "did:plc:alice", lexBlob(testVideoCID, "video/mp4", 300_000_001))},
		{name: "missing status"},
		{name: "service outage", callErr: errors.New(upstreamSecret)},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			verifier := video.NewCompletionVerifier(fakeJobStatusClient{status: test.status, err: test.callErr})
			verified, err := verifier.Verify(context.Background(), owner, "job-1", submitted)
			if test.wantOK {
				if err != nil {
					t.Fatalf("Verify: %v", err)
				}
				if verified != submitted {
					t.Fatalf("verified = %+v, want %+v", verified, submitted)
				}
				return
			}
			if err == nil {
				t.Fatal("Verify returned nil error")
			}
			message := err.Error()
			for _, secret := range []string{"did:plc:bob", "job-foreign", upstreamSecret, testVideoCID} {
				if strings.Contains(message, secret) {
					t.Fatalf("error disclosed %q: %q", secret, message)
				}
			}
		})
	}
}

func jobStatus(state, jobID, did string, blob *lexutil.LexBlob) *appbsky.VideoDefs_JobStatus {
	return jobStatusWithError(state, jobID, did, blob, nil)
}

func jobStatusWithError(state, jobID, did string, blob *lexutil.LexBlob, serviceError *string) *appbsky.VideoDefs_JobStatus {
	return &appbsky.VideoDefs_JobStatus{State: state, JobId: jobID, Did: did, Blob: blob, Error: serviceError}
}

func lexBlob(value, mimeType string, size int64) *lexutil.LexBlob {
	parsed := cid.MustParse(value)
	return &lexutil.LexBlob{Ref: lexutil.LexLink(parsed), MimeType: mimeType, Size: size}
}
