package business

import "errors"

const MaxImageBytes int64 = 15 * 1024 * 1024

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
