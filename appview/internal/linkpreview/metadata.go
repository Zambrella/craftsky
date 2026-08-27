package linkpreview

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/url"
	"strings"

	"github.com/rivo/uniseg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	maxPageBytes            = 2_000_000
	maxTitleGraphemes       = 200
	maxTitleBytes           = 2000
	maxDescriptionGraphemes = 300
	maxDescriptionBytes     = 3000
)

type Metadata struct {
	URL                 *url.URL
	Title               string
	Description         string
	ThumbnailCandidates []*url.URL
}

// DecodeHTMLHead converts a bounded HTML stream to UTF-8 and stops after a
// completed head. Exhausting the limit before then is treated as oversized.
func DecodeHTMLHead(input io.Reader, contentType string) ([]byte, error) {
	if _, parameters, err := mime.ParseMediaType(contentType); err != nil {
		return nil, err
	} else if label := parameters["charset"]; label != "" {
		if encoding, _ := charset.Lookup(label); encoding == nil {
			return nil, errors.New("unsupported HTML charset")
		}
	}
	limited := &io.LimitedReader{R: input, N: maxPageBytes}
	decoded, err := charset.NewReader(limited, contentType)
	if err != nil {
		return nil, err
	}

	var head bytes.Buffer
	tokenizer := html.NewTokenizer(decoded)
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			if limited.N == 0 {
				return nil, errors.New("HTML head exceeds raw byte limit")
			}
			return head.Bytes(), nil
		}
		head.Write(tokenizer.Raw())
		if tokenType == html.EndTagToken && strings.EqualFold(tokenizer.Token().Data, "head") {
			return head.Bytes(), nil
		}
	}
}

// ExtractMetadata parses an already bounded, decoded HTML head. Destination
// metadata is intentionally ignored; finalURL remains authoritative.
func ExtractMetadata(input []byte, finalURL *url.URL) (Metadata, error) {
	if finalURL == nil || finalURL.Hostname() == "" {
		return Metadata{}, errors.New("validated final URL is required")
	}
	var ogTitle, twitterTitle, htmlTitle string
	var ogDescription, twitterDescription, htmlDescription string
	var ogImages, twitterImages []string
	var baseURL *url.URL
	inTitle := false
	tokenizer := html.NewTokenizer(bytes.NewReader(input))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return Metadata{}, err
			}
			return metadataResult(finalURL, baseURL, ogTitle, twitterTitle, htmlTitle, ogDescription, twitterDescription, htmlDescription, ogImages, twitterImages), nil
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			switch strings.ToLower(token.Data) {
			case "title":
				inTitle = true
			case "base":
				if baseURL == nil {
					baseURL = validBaseURL(finalURL, attributeMap(token.Attr)["href"])
				}
			case "meta":
				attributes := attributeMap(token.Attr)
				content := normalizeMetadata(attributes["content"])
				switch strings.ToLower(attributes["property"]) {
				case "og:title":
					ogTitle = firstNonEmpty(ogTitle, content)
				case "og:description":
					ogDescription = firstNonEmpty(ogDescription, content)
				case "og:image", "og:image:url", "og:image:secure_url":
					if content != "" {
						ogImages = append(ogImages, content)
					}
				}
				switch strings.ToLower(attributes["name"]) {
				case "twitter:title":
					twitterTitle = firstNonEmpty(twitterTitle, content)
				case "twitter:description":
					twitterDescription = firstNonEmpty(twitterDescription, content)
				case "twitter:image", "twitter:image:src":
					if content != "" {
						twitterImages = append(twitterImages, content)
					}
				case "description":
					htmlDescription = firstNonEmpty(htmlDescription, content)
				}
			}
		case html.TextToken:
			if inTitle {
				htmlTitle = firstNonEmpty(htmlTitle, normalizeMetadata(string(tokenizer.Text())))
			}
		case html.EndTagToken:
			if strings.EqualFold(tokenizer.Token().Data, "title") {
				inTitle = false
			}
		}
	}
}

func metadataResult(finalURL, baseURL *url.URL, ogTitle, twitterTitle, htmlTitle, ogDescription, twitterDescription, htmlDescription string, imageGroups ...[]string) Metadata {
	copyURL := *finalURL
	return Metadata{
		URL:                 &copyURL,
		ThumbnailCandidates: resolveThumbnailCandidates(finalURL, baseURL, imageGroups...),
		Title: ClampMetadata(
			firstNonEmpty(ogTitle, twitterTitle, htmlTitle, finalURL.Hostname()),
			maxTitleGraphemes,
			maxTitleBytes,
		),
		Description: ClampMetadata(
			firstNonEmpty(ogDescription, twitterDescription, htmlDescription),
			maxDescriptionGraphemes,
			maxDescriptionBytes,
		),
	}
}

func validBaseURL(finalURL *url.URL, raw string) *url.URL {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || raw == "" {
		return nil
	}
	resolved := finalURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" || resolved.Hostname() == "" || resolved.User != nil {
		return nil
	}
	return resolved
}

func resolveThumbnailCandidates(finalURL, baseURL *url.URL, imageGroups ...[]string) []*url.URL {
	if baseURL == nil {
		baseURL = finalURL
	}
	seen := make(map[string]struct{}, 3)
	result := make([]*url.URL, 0, 3)
	for _, group := range imageGroups {
		for _, raw := range group {
			candidate, err := url.Parse(strings.TrimSpace(raw))
			if err != nil {
				continue
			}
			candidate = baseURL.ResolveReference(candidate)
			if candidate.Scheme != "http" && candidate.Scheme != "https" || candidate.Hostname() == "" || candidate.User != nil {
				continue
			}
			key := candidate.String()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, candidate)
			if len(result) == 3 {
				return result
			}
		}
	}
	return result
}

func attributeMap(attributes []html.Attribute) map[string]string {
	result := make(map[string]string, len(attributes))
	for _, attribute := range attributes {
		key := strings.ToLower(attribute.Key)
		if _, exists := result[key]; !exists {
			result[key] = attribute.Val
		}
	}
	return result
}

func normalizeMetadata(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func ClampMetadata(value string, maxGraphemes, maxBytes int) string {
	value = normalizeMetadata(value)
	if value == "" || maxGraphemes <= 0 || maxBytes <= 0 {
		return ""
	}
	var result strings.Builder
	graphemes := uniseg.NewGraphemes(value)
	for count := 0; count < maxGraphemes && graphemes.Next(); count++ {
		cluster := graphemes.Str()
		if result.Len()+len(cluster) > maxBytes {
			break
		}
		result.WriteString(cluster)
	}
	return result.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
