package business

import "errors"

var (
	ErrInvalidBusinessTypes = errors.New("business: invalid business types")
	ErrInvalidOfferings     = errors.New("business: invalid offerings")
	ErrUnknownCatalogValue  = errors.New("business: unknown catalog value")
)

type OpenValue struct {
	Value string `json:"value"`
	Known bool   `json:"known"`
}

var businessTypeCatalog = []string{
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

var offeringCatalog = []string{
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

func BusinessTypeCatalog() []string {
	return append([]string(nil), businessTypeCatalog...)
}

func ValidateBusinessTypes(values []string) ([]string, error) {
	if len(values) > 5 {
		return nil, ErrInvalidBusinessTypes
	}
	selected := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, duplicate := selected[value]; duplicate || !catalogContains(businessTypeCatalog, value) {
			return nil, ErrInvalidBusinessTypes
		}
		selected[value] = struct{}{}
	}
	canonical := make([]string, 0, len(values))
	for _, value := range businessTypeCatalog {
		if _, ok := selected[value]; ok {
			canonical = append(canonical, value)
		}
	}
	return canonical, nil
}

func OfferingCatalog() []string {
	return append([]string(nil), offeringCatalog...)
}

func ValidateOfferings(values []string) ([]string, error) {
	if len(values) > 10 {
		return nil, ErrInvalidOfferings
	}
	selected := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, duplicate := selected[value]; duplicate || !catalogContains(offeringCatalog, value) {
			return nil, ErrInvalidOfferings
		}
		selected[value] = struct{}{}
	}
	canonical := make([]string, 0, len(values))
	for _, value := range offeringCatalog {
		if _, ok := selected[value]; ok {
			canonical = append(canonical, value)
		}
	}
	return canonical, nil
}

func catalogContains(catalog []string, value string) bool {
	for _, candidate := range catalog {
		if candidate == value {
			return true
		}
	}
	return false
}

func ClassifyOpenValue(value string, knownValues []string) OpenValue {
	return OpenValue{Value: value, Known: catalogContains(knownValues, value)}
}

func ValidateKnownValue(value string, knownValues []string) error {
	if !catalogContains(knownValues, value) {
		return ErrUnknownCatalogValue
	}
	return nil
}
