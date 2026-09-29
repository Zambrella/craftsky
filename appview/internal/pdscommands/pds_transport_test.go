package pdscommands

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestTranslateProtocolErrorClassifiesDefiniteClientRejection(t *testing.T) {
	upstream := &atclient.APIError{StatusCode: http.StatusBadRequest, Name: "InvalidRequest"}
	err := translateProtocolError(upstream)
	if !errors.Is(err, ErrDispatchRejected) || !errors.Is(err, upstream) {
		t.Fatalf("translated error = %v", err)
	}
}

func TestTranslateProtocolErrorLeavesServerAndTransportFailuresAmbiguous(t *testing.T) {
	for _, upstream := range []error{
		&atclient.APIError{StatusCode: http.StatusInternalServerError, Name: "InternalError"},
		errors.New("connection reset"),
	} {
		if err := translateProtocolError(upstream); !errors.Is(err, upstream) || errors.Is(err, ErrDispatchRejected) {
			t.Fatalf("translated error = %v", err)
		}
	}
}
