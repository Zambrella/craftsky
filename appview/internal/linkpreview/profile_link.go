package linkpreview

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"golang.org/x/net/html"
)

type ProfileLinkPreview struct {
	Reachable bool
	Icon      *Thumbnail
}

// FetchProfileLinkPreview uses the existing DNS/redirect/body limits. Failure
// remains advisory: an unreachable destination can still be saved by its owner.
func (s *Service) FetchProfileLinkPreview(ctx context.Context, raw string) (ProfileLinkPreview, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	response, finalURL, err := s.fetcher.FetchPage(ctx, raw)
	if err != nil {
		return ProfileLinkPreview{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return ProfileLinkPreview{}, nil
	}
	result := ProfileLinkPreview{Reachable: true}
	candidates := []string{}
	if isHTMLContentType(response.Header.Get("Content-Type")) {
		head, err := DecodeHTMLHead(response.Body, response.Header.Get("Content-Type"))
		if err == nil {
			tokenizer := html.NewTokenizer(bytes.NewReader(head))
			for tokenizer.Next() != html.ErrorToken {
				token := tokenizer.Token()
				if token.Data != "link" {
					continue
				}
				attrs := attributeMap(token.Attr)
				rel := " " + strings.ToLower(attrs["rel"]) + " "
				if !strings.Contains(rel, " icon ") && !strings.Contains(rel, " apple-touch-icon ") {
					continue
				}
				candidate, err := finalURL.Parse(attrs["href"])
				if err == nil {
					candidates = append(candidates, candidate.String())
				}
			}
		}
	}
	fallback := *finalURL
	fallback.Path = "/favicon.ico"
	fallback.RawPath = ""
	fallback.RawQuery = ""
	fallback.Fragment = ""
	candidates = append(candidates, fallback.String())
	for _, candidate := range candidates {
		thumbnail := s.fetchThumbnail(ctx, candidate)
		if thumbnail == nil {
			continue
		}
		result.Icon = s.profileIcon(ctx, thumbnail.Bytes)
		if result.Icon != nil {
			break
		}
	}
	return result, nil
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
