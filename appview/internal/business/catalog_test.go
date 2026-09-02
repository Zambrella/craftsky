package business

import (
	"errors"
	"reflect"
	"testing"
)

func TestBusinessTypeCatalog(t *testing.T) {
	wantCatalog := []string{
		"dyer",
		"fiber-producer",
		"fiber-processor",
		"yarn-shop",
		"fabric-shop",
		"craft-supply-shop",
		"pattern-designer",
		"finished-goods-maker",
		"tool-maker",
		"teacher",
		"craft-studio",
		"repair-service",
		"technical-editor",
		"photographer",
		"publisher",
		"other-craft-business",
	}
	if got := BusinessTypeCatalog(); !reflect.DeepEqual(got, wantCatalog) {
		t.Fatalf("BusinessTypeCatalog() = %v, want %v", got, wantCatalog)
	}

	got, err := ValidateBusinessTypes([]string{"teacher", "dyer", "tool-maker"})
	if err != nil {
		t.Fatalf("ValidateBusinessTypes(valid): %v", err)
	}
	want := []string{"dyer", "tool-maker", "teacher"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical business types = %v, want %v", got, want)
	}

	for name, values := range map[string][]string{
		"unknown":    {"unknown"},
		"mixed case": {"Dyer"},
		"duplicate":  {"dyer", "dyer"},
		"six values": {"dyer", "fiber-producer", "fiber-processor", "yarn-shop", "fabric-shop", "teacher"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateBusinessTypes(values); !errors.Is(err, ErrInvalidBusinessTypes) {
				t.Fatalf("ValidateBusinessTypes(%v) error = %v, want ErrInvalidBusinessTypes", values, err)
			}
		})
	}
}

func TestOfferingCatalog(t *testing.T) {
	wantCatalog := []string{
		"yarn",
		"fiber",
		"fabric",
		"patterns",
		"kits",
		"notions",
		"tools",
		"finished-goods",
		"custom-work",
		"repairs",
		"classes",
		"studio-hire",
		"wholesale",
		"digital-products",
		"technical-editing",
		"photography-services",
		"fiber-processing",
	}
	if got := OfferingCatalog(); !reflect.DeepEqual(got, wantCatalog) {
		t.Fatalf("OfferingCatalog() = %v, want %v", got, wantCatalog)
	}

	got, err := ValidateOfferings([]string{"classes", "yarn", "repairs"})
	if err != nil {
		t.Fatalf("ValidateOfferings(valid): %v", err)
	}
	want := []string{"yarn", "repairs", "classes"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical offerings = %v, want %v", got, want)
	}

	for name, values := range map[string][]string{
		"unknown":       {"unknown"},
		"mixed case":    {"Yarn"},
		"duplicate":     {"yarn", "yarn"},
		"eleven values": {"yarn", "fiber", "fabric", "patterns", "kits", "notions", "tools", "finished-goods", "custom-work", "repairs", "classes"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateOfferings(values); !errors.Is(err, ErrInvalidOfferings) {
				t.Fatalf("ValidateOfferings(%v) error = %v, want ErrInvalidOfferings", values, err)
			}
		})
	}
}

func TestOpenCatalogValueClassification(t *testing.T) {
	knownValues := []string{"known-value", "another-known-value"}

	known := ClassifyOpenValue("known-value", knownValues)
	if known.Value != "known-value" || !known.Known {
		t.Fatalf("known classification = %+v, want preserved known value", known)
	}

	unknown := ClassifyOpenValue("future-independent-value", knownValues)
	if unknown.Value != "future-independent-value" || unknown.Known {
		t.Fatalf("unknown classification = %+v, want preserved unknown value", unknown)
	}

	if err := ValidateKnownValue("known-value", knownValues); err != nil {
		t.Fatalf("ValidateKnownValue(known): %v", err)
	}
	if err := ValidateKnownValue("future-independent-value", knownValues); !errors.Is(err, ErrUnknownCatalogValue) {
		t.Fatalf("ValidateKnownValue(unknown) error = %v, want ErrUnknownCatalogValue", err)
	}
}
