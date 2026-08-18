package federatedhttp

import (
	"io"
	"net/http"
)

type boundaryTransport struct {
	base          http.RoundTripper
	policy        *Policy
	purpose       Purpose
	responseLimit int64
}

func (t *boundaryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, &Error{Kind: KindDestinationRejected, Purpose: t.purpose}
	}
	if _, err := t.policy.ValidateURL(request.Context(), request.URL.String()); err != nil {
		return nil, &Error{Kind: Classify(err), Purpose: t.purpose, Cause: err}
	}
	response, err := t.base.RoundTrip(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, failure(t.purpose, err)
	}
	if response == nil || response.Body == nil {
		return nil, &Error{Kind: KindUpstreamFailure, Purpose: t.purpose}
	}
	if response.ContentLength > t.responseLimit {
		_ = response.Body.Close()
		return nil, &Error{Kind: KindResponseTooLarge, Purpose: t.purpose}
	}
	response.Body = &limitedReadCloser{
		body:      response.Body,
		remaining: t.responseLimit,
		purpose:   t.purpose,
	}
	return response, nil
}

type limitedReadCloser struct {
	body      io.ReadCloser
	remaining int64
	purpose   Purpose
	exceeded  bool
}

func (r *limitedReadCloser) Read(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	if r.exceeded {
		return 0, &Error{Kind: KindResponseTooLarge, Purpose: r.purpose}
	}
	if r.remaining > 0 {
		if int64(len(buffer)) > r.remaining {
			buffer = buffer[:r.remaining]
		}
		n, err := r.body.Read(buffer)
		r.remaining -= int64(n)
		return n, err
	}

	var probe [1]byte
	n, err := r.body.Read(probe[:])
	if n == 0 {
		return 0, err
	}
	r.exceeded = true
	_ = r.body.Close()
	return 0, &Error{Kind: KindResponseTooLarge, Purpose: r.purpose}
}

func (r *limitedReadCloser) Close() error {
	return r.body.Close()
}
