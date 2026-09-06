package video

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
)

const maxJobStatusResponseBytes = 64 * 1024

var ErrJobStatusUnavailable = errors.New("video job status unavailable")

type PublicJobStatusClient struct {
	endpoint string
	client   *http.Client
}

func NewPublicJobStatusClient(endpoint string, client *http.Client) (*PublicJobStatusClient, error) {
	origin, err := serviceOrigin(endpoint)
	if err != nil || client == nil {
		return nil, ErrJobStatusUnavailable
	}
	return &PublicJobStatusClient{endpoint: origin, client: client}, nil
}

func (client *PublicJobStatusClient) GetJobStatus(ctx context.Context, jobID string) (*appbsky.VideoDefs_JobStatus, error) {
	if client == nil || client.client == nil || strings.TrimSpace(jobID) == "" {
		return nil, ErrJobStatusUnavailable
	}
	endpoint, err := url.Parse(client.endpoint + "/xrpc/app.bsky.video.getJobStatus")
	if err != nil {
		return nil, ErrJobStatusUnavailable
	}
	query := endpoint.Query()
	query.Set("jobId", jobID)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, ErrJobStatusUnavailable
	}
	noRedirectClient := *client.client
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := noRedirectClient.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrJobStatusUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, ErrJobStatusUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxJobStatusResponseBytes+1))
	if err != nil || len(body) > maxJobStatusResponseBytes {
		return nil, ErrJobStatusUnavailable
	}
	var output appbsky.VideoGetJobStatus_Output
	if err := json.Unmarshal(body, &output); err != nil || output.JobStatus == nil {
		return nil, ErrJobStatusUnavailable
	}
	return output.JobStatus, nil
}
