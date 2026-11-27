package linkpreview

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"golang.org/x/net/html"
)

const maxProfileIconCandidates = 8

type ProfileLinkPreview struct {
	Reachable bool
	Icon      *Thumbnail
}

// FetchProfileLinkPreview uses the existing DNS/redirect/body limits. Failure
// remains advisory: an unreachable destination can still be saved by its owner.
func (s *Service) FetchProfileLinkPreview(ctx context.Context, raw string) (ProfileLinkPreview, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	pageCtx, cancelPage := context.WithTimeout(ctx, pagePhaseTimeout)
	defer cancelPage()
	response, finalURL, err := s.fetcher.FetchPage(pageCtx, raw)
	if err != nil {
		return ProfileLinkPreview{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return ProfileLinkPreview{}, nil
	}
	return ProfileLinkPreview{Reachable: true, Icon: s.findProfileIcon(ctx, response, finalURL)}, nil
}

func (s *Service) findProfileIcon(ctx context.Context, response *http.Response, finalURL *url.URL) *Thumbnail {
	seen := make(map[string]bool)
	contentType := response.Header.Get("Content-Type")
	if isHTMLContentType(contentType) {
		if tokenizer, _, err := newBoundedHTMLTokenizer(response.Body, contentType); err == nil {
			// Some sites stream icon links into the body after closing the head.
			for ctx.Err() == nil && len(seen) < maxProfileIconCandidates {
				tokenType := tokenizer.Next()
				if tokenType == html.ErrorToken {
					break
				}
				if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
					continue
				}
				token := tokenizer.Token()
				if token.Data != "link" {
					continue
				}
				attrs := attributeMap(token.Attr)
				rel := strings.Fields(strings.ToLower(attrs["rel"]))
				href := strings.TrimSpace(attrs["href"])
				if href == "" || !slices.Contains(rel, "icon") && !slices.Contains(rel, "apple-touch-icon") {
					continue
				}
				candidate, err := finalURL.Parse(href)
				if err != nil {
					continue
				}
				target := candidate.String()
				if seen[target] {
					continue
				}
				seen[target] = true
				if icon := s.fetchProfileIcon(ctx, target); icon != nil {
					return icon
				}
			}
		}
	}
	fallback := finalURL.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()
	if seen[fallback] {
		return nil
	}
	return s.fetchProfileIcon(ctx, fallback)
}

func (s *Service) fetchProfileIcon(ctx context.Context, target string) *Thumbnail {
	thumbnail := s.fetchThumbnail(ctx, target)
	if thumbnail == nil {
		return nil
	}
	return s.profileIcon(ctx, thumbnail.Bytes)
}

func (s *Service) profileIcon(ctx context.Context, input []byte) *Thumbnail {
	select {
	case s.decodeSlots <- struct{}{}:
	case <-ctx.Done():
		return nil
	}
	defer func() { <-s.decodeSlots }()
	decoded, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil
	}
	icon := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.CatmullRom.Scale(icon, icon.Bounds(), decoded, decoded.Bounds(), draw.Over, nil)
	var encoded bytes.Buffer
	if png.Encode(&encoded, icon) != nil || encoded.Len() > 100_000 {
		return nil
	}
	return &Thumbnail{Bytes: encoded.Bytes(), MIMEType: "image/png", Width: 64, Height: 64}
}
