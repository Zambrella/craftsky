package imagesafety_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/pdseffects"
)

type imageDirectory struct {
	identity *identity.Identity
}

func (directory imageDirectory) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	return directory.identity, nil
}

func (directory imageDirectory) LookupDID(context.Context, syntax.DID) (*identity.Identity, error) {
	return directory.identity, nil
}

func (directory imageDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return directory.identity, nil
}

func (imageDirectory) Purge(context.Context, syntax.AtIdentifier) error { return nil }

type imageOrigin struct {
	url *url.URL
}

func (origin imageOrigin) ValidateOrigin(context.Context, string) (*url.URL, error) {
	return origin.url, nil
}

func TestPDSBlobFetcherValidatesBytesMIMEAndCID(t *testing.T) {
	var encoded bytes.Buffer
	fixture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	fixture.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	if err := png.Encode(&encoded, fixture); err != nil {
		t.Fatal(err)
	}
	body := encoded.Bytes()
	blobCID, _, err := pdseffects.PredictBlobCID(body)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.sync.getBlob" ||
			r.URL.Query().Get("did") != "did:plc:image" || r.URL.Query().Get("cid") == "" {
			t.Fatalf("unexpected request %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	originURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	did := syntax.DID("did:plc:image")
	fetcher, err := imagesafety.NewPDSBlobFetcher(imageDirectory{identity: &identity.Identity{
		DID: did,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: server.URL},
		},
	}}, server.Client(), imageOrigin{url: originURL}, int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	source := imagesafety.BlobSource{
		DID: did, BlobCID: blobCID, DeclaredMIME: "image/png", DeclaredSize: int64(len(body)),
	}
	got, err := fetcher.Fetch(context.Background(), source)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("fetch bytes=%d err=%v", len(got), err)
	}

	source.DeclaredMIME = "image/jpeg"
	if _, err := fetcher.Fetch(context.Background(), source); err == nil {
		t.Fatal("MIME mismatch was accepted")
	}
	source.DeclaredMIME = "image/png"
	source.BlobCID = syntax.CID("bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a")
	if _, err := fetcher.Fetch(context.Background(), source); err == nil {
		t.Fatal("CID mismatch was accepted")
	}
}
