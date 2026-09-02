package business

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestProductImageValidation(t *testing.T) {
	if err := ValidateProductImage(nil, true); !errors.Is(err, ErrProductImageRequired) {
		t.Fatalf("first-party missing image error = %v, want ErrProductImageRequired", err)
	}
	if err := ValidateProductImage(nil, false); err != nil {
		t.Fatalf("independent missing image: %v", err)
	}

	for _, mimeType := range []string{"image/jpeg", "image/png", "image/webp"} {
		image := &Image{MIMEType: mimeType, Size: MaxImageBytes, Alt: strings.Repeat("a", 1000)}
		if err := ValidateProductImage(image, true); err != nil {
			t.Errorf("valid %s image: %v", mimeType, err)
		}
	}

	invalid := []*Image{
		{MIMEType: "image/gif", Size: 1},
		{MIMEType: "image/jpeg", Size: MaxImageBytes + 1},
		{MIMEType: "image/jpeg", Size: 1, Alt: strings.Repeat("a", 1001)},
		{MIMEType: "image/jpeg", Size: 1, Alt: strings.Repeat("👨‍👩‍👧‍👦", 41)},
	}
	for _, image := range invalid {
		if err := ValidateProductImage(image, true); !errors.Is(err, ErrInvalidImage) {
			t.Errorf("ValidateProductImage(%+v) error = %v, want ErrInvalidImage", image, err)
		}
	}
}

func TestProductCollectionValidation(t *testing.T) {
	four := []Product{
		{Title: "One", URI: "https://example.com/product/one"},
		{Title: "Two", URI: "https://example.com/product/two"},
		{Title: "Three", URI: "https://example.com/product/three"},
		{Title: "Four", URI: "https://example.com/product/four"},
	}
	got, err := ValidateProductCollection(four)
	if err != nil {
		t.Fatalf("ValidateProductCollection(four): %v", err)
	}
	if !reflect.DeepEqual(got, four) {
		t.Fatalf("validated products = %v, want authored order %v", got, four)
	}

	five := append(append([]Product(nil), four...), Product{Title: "Five", URI: "https://example.com/product/five"})
	if _, err := ValidateProductCollection(five); !errors.Is(err, ErrInvalidProducts) {
		t.Fatalf("five products error = %v, want ErrInvalidProducts", err)
	}

	duplicates := []Product{
		{Title: "One", URI: "https://example.com/product"},
		{Title: "Duplicate", URI: "https://example.com/product"},
	}
	if _, err := ValidateProductCollection(duplicates); !errors.Is(err, ErrInvalidProducts) {
		t.Fatalf("duplicate products error = %v, want ErrInvalidProducts", err)
	}

	distinctStrings := []Product{
		{Title: "Case", URI: "https://example.com/Product"},
		{Title: "Path", URI: "https://example.com/product"},
		{Title: "Query", URI: "https://example.com/product?variant=one"},
		{Title: "Other query", URI: "https://example.com/product?variant=two"},
	}
	if _, err := ValidateProductCollection(distinctStrings); err != nil {
		t.Fatalf("distinct URI strings rejected: %v", err)
	}
}
