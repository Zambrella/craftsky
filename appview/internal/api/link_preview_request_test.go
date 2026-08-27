package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

// UT-012: preview and external-create requests decode strictly and reject
// oversized values or incompatible post shapes before side effects.
func TestDecodeAndValidateLinkPreviewAndExternalRequests(t *testing.T) {
	t.Parallel()

	t.Run("preview request", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name    string
			body    string
			wantURL string
			wantErr bool
		}{
			{name: "valid", body: `{"url":"https://public.example/pattern"}`, wantURL: "https://public.example/pattern"},
			{name: "missing", body: `{}`, wantErr: true},
			{name: "extra", body: `{"url":"https://public.example","extra":true}`, wantErr: true},
			{name: "wrong type", body: `{"url":42}`, wantErr: true},
			{name: "trailing JSON", body: `{"url":"https://public.example"}{}`, wantErr: true},
			{name: "over 8192 UTF-8 bytes", body: `{"url":"https://public.example/` + strings.Repeat("é", 4090) + `"}`, wantErr: true},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				request, err := api.DecodeLinkPreviewRequest(strings.NewReader(tt.body))
				if (err != nil) != tt.wantErr {
					t.Fatalf("DecodeLinkPreviewRequest() error = %v, wantErr %t", err, tt.wantErr)
				}
				if err == nil && request.URL != tt.wantURL {
					t.Fatalf("URL = %q, want %q", request.URL, tt.wantURL)
				}
			})
		}
	})

	validExternal := `{"uri":"https://final.example/pattern","title":"Pattern","description":"Description"}`
	validBlob := `{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"image/webp","size":1000000}`
	for _, tt := range []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "metadata only without matching facet", body: `{"text":"plain text","embed":{"external":` + validExternal + `}}`},
		{name: "thumbnail", body: `{"text":"link","embed":{"external":{"uri":"https://final.example","title":"Pattern","description":"","thumb":` + validBlob + `}}}`},
		{name: "quote and external", body: `{"text":"link","embed":{"quote":{"uri":"at://did:plc:abc/social.craftsky.feed.post/rk","cid":"bafyquote"},"external":` + validExternal + `}}`, wantErr: true},
		{name: "images and external", body: `{"text":"link","images":[{"image":{"ref":{"$link":"bafyimage"},"mimeType":"image/jpeg","size":1},"alt":""}],"embed":{"external":` + validExternal + `}}`, wantErr: true},
		{name: "project and external", body: `{"text":"link","project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}},"embed":{"external":` + validExternal + `}}`, wantErr: true},
		{name: "oversized title", body: `{"text":"link","embed":{"external":{"uri":"https://final.example","title":"` + strings.Repeat("a", 201) + `","description":""}}}`, wantErr: true},
		{name: "invalid URI", body: `{"text":"link","embed":{"external":{"uri":"file:///private","title":"Pattern","description":""}}}`, wantErr: true},
		{name: "oversized thumbnail", body: `{"text":"link","embed":{"external":{"uri":"https://final.example","title":"Pattern","description":"","thumb":{"ref":{"$link":"bafy"},"mimeType":"image/png","size":1000001}}}}`, wantErr: true},
		{name: "unsupported thumbnail", body: `{"text":"link","embed":{"external":{"uri":"https://final.example","title":"Pattern","description":"","thumb":{"ref":{"$link":"bafy"},"mimeType":"image/gif","size":10}}}}`, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			request, err := api.DecodePostCreate(strings.NewReader(tt.body))
			if err == nil {
				err = api.ValidatePostCreate(request)
			}
			if tt.wantErr {
				var fieldError *api.FieldError
				if !errors.As(err, &fieldError) {
					t.Fatalf("error = %v, want FieldError", err)
				}
			} else if err != nil {
				t.Fatalf("decode/validate external: %v", err)
			}
		})
	}
}
