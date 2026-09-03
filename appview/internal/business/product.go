package business

import (
	"encoding/json"
	"errors"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const MaxImageBytes int64 = 2_000_000

var (
	ErrInvalidImage         = errors.New("business: invalid image")
	ErrProductImageRequired = errors.New("business: product image required")
	ErrInvalidProducts      = errors.New("business: invalid products")
)

type AspectRatio struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type Image struct {
	MIMEType    string       `json:"mimeType"`
	Size        int64        `json:"size"`
	Alt         string       `json:"alt,omitempty"`
	AspectRatio *AspectRatio `json:"aspectRatio,omitempty"`
}

type HydratedImage struct {
	CID         syntax.CID   `json:"cid"`
	MIME        string       `json:"mime"`
	Size        int64        `json:"size"`
	Alt         string       `json:"alt"`
	AspectRatio *AspectRatio `json:"aspectRatio,omitempty"`
}

type Product struct {
	Title string `json:"title"`
	URI   string `json:"uri"`
	Image *Image `json:"image,omitempty"`
}

func ValidateProductImage(image *Image, firstParty bool) error {
	if image == nil {
		if firstParty {
			return ErrProductImageRequired
		}
		return nil
	}
	if image.Size < 0 || image.Size > MaxImageBytes {
		return ErrInvalidImage
	}
	switch image.MIMEType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return ErrInvalidImage
	}
	if err := ValidateText(TextFieldImageAlt, image.Alt); err != nil {
		return ErrInvalidImage
	}
	if image.AspectRatio != nil && (image.AspectRatio.Width <= 0 || image.AspectRatio.Height <= 0) {
		return ErrInvalidImage
	}
	return nil
}

func ValidateProductCollection(products []Product) ([]Product, error) {
	if len(products) > 4 {
		return nil, ErrInvalidProducts
	}
	seen := make(map[string]struct{}, len(products))
	for _, product := range products {
		if _, duplicate := seen[product.URI]; duplicate {
			return nil, ErrInvalidProducts
		}
		seen[product.URI] = struct{}{}
	}
	return append([]Product(nil), products...), nil
}

func hydrateIndependentImage(raw json.RawMessage) *HydratedImage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	decoded, err := atdata.UnmarshalJSON(raw)
	if err != nil {
		return nil
	}
	blob, ok := decoded["image"].(atdata.Blob)
	if !ok || !blob.Ref.IsDefined() {
		return nil
	}
	alt := ""
	if rawAlt, exists := decoded["alt"]; exists {
		var ok bool
		alt, ok = rawAlt.(string)
		if !ok {
			return nil
		}
	}
	var aspectRatio *AspectRatio
	if rawAspect, exists := decoded["aspectRatio"]; exists {
		aspect, ok := rawAspect.(map[string]any)
		width, widthOK := aspect["width"].(int64)
		height, heightOK := aspect["height"].(int64)
		if !ok || !widthOK || !heightOK {
			return nil
		}
		aspectRatio = &AspectRatio{Width: width, Height: height}
	}
	if ValidateProductImage(&Image{
		MIMEType: blob.MimeType, Size: blob.Size,
		Alt: alt, AspectRatio: aspectRatio,
	}, false) != nil {
		return nil
	}
	return &HydratedImage{
		CID: syntax.CID(blob.Ref.String()), MIME: blob.MimeType, Size: blob.Size,
		Alt: alt, AspectRatio: aspectRatio,
	}
}
