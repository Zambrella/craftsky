package api

import (
	"net/http"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestBoundedPDSAPIErrorName(t *testing.T) {
	for _, test := range []struct {
		name string
		want string
	}{
		{name: "BlobNotFound", want: "BlobNotFound"},
		{name: "InvalidMimeType", want: "InvalidMimeType"},
		{name: "InvalidSize", want: "InvalidSize"},
		{name: "InvalidRecord", want: "InvalidRecord"},
		{name: "provider-authored-detail", want: "other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := &atclient.APIError{StatusCode: http.StatusBadRequest, Name: test.name}
			if got := boundedPDSAPIErrorName(err); got != test.want {
				t.Fatalf("boundedPDSAPIErrorName() = %q, want %q", got, test.want)
			}
		})
	}
}
