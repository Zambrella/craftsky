package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/lexicon/craftsky"
)

var forbiddenCommerceTerms = []string{
	"disclaimer",
	"inventory",
	"availability",
	"synchronization",
	"shipping",
	"tax",
	"checkout",
	"accuracy",
	"guarantee",
}

func TestBusinessProductPriceWireAndSchemaContract(t *testing.T) {
	assertPriceFields(t, reflect.TypeOf(business.Price{}), "AppView price")
	assertPriceFields(t, reflect.TypeOf(craftsky.BusinessDefs_Price{}), "generated PDS price")

	response := api.ProfileResponse{
		DID:               syntax.DID("did:plc:seller"),
		Handle:            syntax.Handle("seller.test"),
		IsCraftskyProfile: true,
		Crafts:            []string{},
		Business: &business.ProfileView{Products: []business.ProductView{{
			Title: "Seller product",
			Price: &business.Price{Amount: "12.34", Currency: "USD"},
		}}},
	}
	wire, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal profile response: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(wire, &body); err != nil {
		t.Fatalf("decode profile response: %v", err)
	}
	price := body["business"].(map[string]any)["products"].([]any)[0].(map[string]any)["price"].(map[string]any)
	if got := sortedMapKeys(price); !slices.Equal(got, []string{"amount", "currency"}) {
		t.Fatalf("AppView price fields = %v, want amount/currency only", got)
	}
	if price["amount"] != "12.34" || price["currency"] != "USD" {
		t.Fatalf("AppView price = %v, want exact seller-authored values", price)
	}

	priceSchema, productSchema := readProductContractSchemas(t)
	properties := priceSchema["properties"].(map[string]any)
	if got := sortedMapKeys(properties); !slices.Equal(got, []string{"amount", "currency"}) {
		t.Fatalf("lexicon price properties = %v, want amount/currency only", got)
	}
	required := stringsFromAny(t, priceSchema["required"])
	slices.Sort(required)
	if !slices.Equal(required, []string{"amount", "currency"}) {
		t.Fatalf("lexicon price required = %v, want amount/currency", required)
	}
	if description, _ := priceSchema["description"].(string); !strings.Contains(strings.ToLower(description), "seller-authored") {
		t.Fatalf("lexicon price description = %q, want seller-authored contract", description)
	}
	assertNoForbiddenCommerceSemantics(t, body, priceSchema, productSchema)
}

func TestBusinessProductCommerceSemanticsRegression(t *testing.T) {
	assertPriceFields(t, reflect.TypeOf(business.Price{}), "AppView price")
	assertPriceFields(t, reflect.TypeOf(craftsky.BusinessDefs_Price{}), "generated PDS price")
	priceSchema, productSchema := readProductContractSchemas(t)
	wire := business.ProfileView{Products: []business.ProductView{{
		Title: "Seller product",
		Price: &business.Price{Amount: "12.34", Currency: "USD"},
	}}}
	assertNoForbiddenCommerceSemantics(t, wire, priceSchema, productSchema)
}

func assertPriceFields(t *testing.T, typ reflect.Type, contract string) {
	t.Helper()
	fields := make([]string, 0, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		name := strings.Split(typ.Field(index).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields = append(fields, name)
		}
	}
	slices.Sort(fields)
	if !slices.Equal(fields, []string{"amount", "currency"}) {
		t.Fatalf("%s reflected JSON fields = %v, want amount/currency only", contract, fields)
	}
}

func readProductContractSchemas(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "lexicon", "social", "craftsky", "business", "defs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read business defs: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode business defs: %v", err)
	}
	defs := schema["defs"].(map[string]any)
	return defs["price"].(map[string]any), defs["product"].(map[string]any)
}

func assertNoForbiddenCommerceSemantics(t *testing.T, values ...any) {
	t.Helper()
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal commerce contract: %v", err)
		}
		lower := strings.ToLower(string(encoded))
		for _, forbidden := range forbiddenCommerceTerms {
			if strings.Contains(lower, forbidden) {
				t.Errorf("forbidden commerce semantic %q appears in contract: %s", forbidden, encoded)
			}
		}
	}
}

func sortedMapKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func stringsFromAny(t *testing.T, value any) []string {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %T, want array", value)
	}
	result := make([]string, len(values))
	for index, value := range values {
		result[index], ok = value.(string)
		if !ok {
			t.Fatalf("array item = %T, want string", value)
		}
	}
	return result
}
