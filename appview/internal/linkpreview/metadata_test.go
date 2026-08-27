package linkpreview

import (
	"bytes"
	"io"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// UT-004: metadata fields select their fallback families independently and
// page-declared destinations never replace the validated fetch URL.
func TestExtractMetadataFallbacks(t *testing.T) {
	t.Parallel()

	finalURL := mustParseURL(t, "https://final.example/pattern")
	tests := []struct {
		name            string
		html            string
		wantTitle       string
		wantDescription string
	}{
		{
			name: "independent families",
			html: `<head>
				<meta property="og:title" content="  Open   Graph Title ">
				<meta property="og:description" content="   ">
				<meta name="twitter:title" content="Twitter title">
				<meta name="twitter:description" content="Twitter description">
				<meta name="description" content="HTML description">
				<title>HTML title</title>
			</head>`,
			wantTitle:       "Open Graph Title",
			wantDescription: "Twitter description",
		},
		{
			name: "empty values skipped",
			html: `<head>
				<meta property="og:title" content="">
				<meta name="twitter:title" content="Twitter title">
				<meta property="og:description" content="OG description">
				<title>HTML title</title>
			</head>`,
			wantTitle:       "Twitter title",
			wantDescription: "OG description",
		},
		{
			name: "HTML fallbacks and destination metadata ignored",
			html: `<head>
				<link rel="canonical" href="https://attacker.example/canonical">
				<meta property="og:url" content="https://attacker.example/og">
				<meta name="description" content="HTML description">
				<title>HTML title</title>
			</head>`,
			wantTitle:       "HTML title",
			wantDescription: "HTML description",
		},
		{
			name:            "hostname and empty description fallback",
			html:            `<head></head>`,
			wantTitle:       "final.example",
			wantDescription: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ExtractMetadata([]byte(tt.html), finalURL)
			if err != nil {
				t.Fatalf("ExtractMetadata(): %v", err)
			}
			if got.Title != tt.wantTitle || got.Description != tt.wantDescription {
				t.Fatalf("metadata = %#v, want title %q description %q", got, tt.wantTitle, tt.wantDescription)
			}
			if got.URL.String() != finalURL.String() {
				t.Fatalf("metadata URL = %q, want validated %q", got.URL, finalURL)
			}
		})
	}
}

// UT-005: normalization and clamping satisfy both limits without splitting a
// Unicode grapheme or producing invalid UTF-8.
func TestClampMetadata(t *testing.T) {
	t.Parallel()

	family := "👨‍👩‍👧‍👦"
	tests := []struct {
		name         string
		input        string
		maxGraphemes int
		maxBytes     int
		want         string
	}{
		{name: "collapse whitespace", input: "  knit\n\t pattern  ", maxGraphemes: 200, maxBytes: 2000, want: "knit pattern"},
		{name: "combining grapheme", input: strings.Repeat("e\u0301", 3), maxGraphemes: 2, maxBytes: 100, want: strings.Repeat("e\u0301", 2)},
		{name: "emoji grapheme", input: family + family, maxGraphemes: 1, maxBytes: 100, want: family},
		{name: "byte limit", input: "ééé", maxGraphemes: 3, maxBytes: 5, want: "éé"},
		{name: "single grapheme exceeds byte limit", input: "e" + strings.Repeat("\u0301", 1000), maxGraphemes: 2, maxBytes: 100, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClampMetadata(tt.input, tt.maxGraphemes, tt.maxBytes)
			if got != tt.want {
				t.Fatalf("ClampMetadata() = %q, want %q", got, tt.want)
			}
			if !utf8.ValidString(got) || len(got) > tt.maxBytes || uniseg.GraphemeClusterCount(got) > tt.maxGraphemes {
				t.Fatalf("ClampMetadata() violated limits: bytes=%d graphemes=%d", len(got), uniseg.GraphemeClusterCount(got))
			}
		})
	}
}

// UT-006: HTML head decoding honors declared charsets, stops at a completed
// head, and never reads beyond the raw page limit.
func TestDecodeHTMLHead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		input       []byte
		want        string
		wantErr     bool
	}{
		{
			name:        "UTF-8 stops at completed head",
			contentType: "text/html; charset=utf-8",
			input:       append([]byte(`<html><head><title>Pattern</title></head>`), bytes.Repeat([]byte("x"), maxPageBytes)...),
			want:        "Pattern",
		},
		{
			name:        "meta-declared Windows-1252",
			contentType: "text/html",
			input:       []byte("<head><meta charset=windows-1252><title>caf\xe9</title></head>"),
			want:        "café",
		},
		{
			name:        "XHTML response charset",
			contentType: "application/xhtml+xml; charset=iso-8859-15",
			input:       []byte("<html><head><title>20 \xa4</title></head></html>"),
			want:        "20 €",
		},
		{
			name:        "malformed declaration falls back safely",
			contentType: "text/html",
			input:       []byte(`<head><meta charset="not a charset"><title>Pattern</title></head>`),
			want:        "Pattern",
		},
		{
			name:        "EOF without closing head",
			contentType: "text/html; charset=utf-8",
			input:       []byte(`<html><head><title>Pattern</title>`),
			want:        "Pattern",
		},
		{
			name:        "unsupported response charset",
			contentType: "text/html; charset=x-craftsky-unknown",
			input:       []byte(`<head><title>Pattern</title></head>`),
			wantErr:     true,
		},
		{
			name:        "unclosed head over raw limit",
			contentType: "text/html; charset=utf-8",
			input:       append([]byte(`<head><title>Pattern</title>`), bytes.Repeat([]byte("x"), maxPageBytes)...),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reader := &countingReader{reader: bytes.NewReader(tt.input)}
			head, err := DecodeHTMLHead(reader, tt.contentType)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DecodeHTMLHead() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Fatalf("DecodeHTMLHead(): %v", err)
				}
				metadata, err := ExtractMetadata(head, mustParseURL(t, "https://final.example/pattern"))
				if err != nil {
					t.Fatalf("ExtractMetadata(): %v", err)
				}
				if metadata.Title != tt.want {
					t.Fatalf("title = %q, want %q", metadata.Title, tt.want)
				}
			}
			if reader.read > maxPageBytes {
				t.Fatalf("raw bytes read = %d, want <= %d", reader.read, maxPageBytes)
			}
		})
	}
}

// UT-007: thumbnail candidates honor the first valid base URL, preserve
// priority/document order, deduplicate, and stop at three.
func TestExtractThumbnailCandidates(t *testing.T) {
	t.Parallel()

	metadata, err := ExtractMetadata([]byte(`<head>
		<base href=":not-a-url">
		<base href="https://assets.example/patterns/">
		<base href="https://ignored.example/">
		<meta name="twitter:image" content="twitter.webp?sig=a%2Bb">
		<meta property="og:image" content="cover.jpg?width=1200&amp;token=x">
		<meta property="og:image:secure_url" content="https://cdn.example/second.png?q=1">
		<meta property="og:image" content="cover.jpg?width=1200&amp;token=x">
		<meta property="og:image" content="https://cdn.example/third.jpg">
		<meta property="og:image" content="https://cdn.example/fourth.jpg">
	</head>`), mustParseURL(t, "https://final.example/posts/one"))
	if err != nil {
		t.Fatalf("ExtractMetadata(): %v", err)
	}

	want := []string{
		"https://assets.example/patterns/cover.jpg?width=1200&token=x",
		"https://cdn.example/second.png?q=1",
		"https://cdn.example/third.jpg",
	}
	if len(metadata.ThumbnailCandidates) != len(want) {
		t.Fatalf("candidate count = %d, want %d: %v", len(metadata.ThumbnailCandidates), len(want), metadata.ThumbnailCandidates)
	}
	for index, candidate := range metadata.ThumbnailCandidates {
		if candidate.String() != want[index] {
			t.Fatalf("candidate %d = %q, want %q", index, candidate, want[index])
		}
	}
}

type countingReader struct {
	reader io.Reader
	read   int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += n
	return n, err
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return parsed
}
