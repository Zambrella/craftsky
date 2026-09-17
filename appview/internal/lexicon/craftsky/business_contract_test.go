package craftsky

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	indigolexicon "github.com/bluesky-social/indigo/atproto/lexicon"
)

func TestBusinessLexiconContract(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "lexicon", "social", "craftsky")
	defs := readLexicon(t, filepath.Join(root, "business", "defs.json"))
	profile := readLexicon(t, filepath.Join(root, "business", "profile.json"))
	event := readLexicon(t, filepath.Join(root, "business", "event.json"))

	assertLexiconID(t, defs, "social.craftsky.business.defs")
	assertLexiconID(t, profile, "social.craftsky.business.profile")
	assertLexiconID(t, event, "social.craftsky.business.event")

	profileMain := definition(t, profile, "main")
	if got := profileMain["key"]; got != "literal:self" {
		t.Fatalf("profile key = %v, want literal:self", got)
	}
	profileRecord := object(t, profileMain, "record")
	if required, present := profileRecord["required"]; present && len(array(t, required)) != 0 {
		t.Fatalf("profile required = %v, want no required fields", required)
	}
	profileProperties := object(t, profileRecord, "properties")
	for _, field := range []string{"businessTypes", "offerings"} {
		if got := integer(t, object(t, profileProperties, field)["maxLength"]); got != 20 {
			t.Errorf("profile %s maxLength = %d, want 20", field, got)
		}
	}
	products := object(t, profileProperties, "products")
	if got := integer(t, products["maxLength"]); got != 20 {
		t.Errorf("products maxLength = %d, want 20", got)
	}
	location := object(t, profileProperties, "location")
	if got := location["ref"]; got != "community.lexicon.location.address" {
		t.Errorf("location ref = %v", got)
	}

	eventMain := definition(t, event, "main")
	if got := eventMain["key"]; got != "tid" {
		t.Fatalf("event key = %v, want tid", got)
	}
	eventRecord := object(t, eventMain, "record")
	wantRequired := []string{"name", "startsAt", "endsAt", "roles", "createdAt"}
	if got := stringsFrom(t, eventRecord["required"]); !reflect.DeepEqual(got, wantRequired) {
		t.Fatalf("event required = %v, want %v", got, wantRequired)
	}
	eventProperties := object(t, eventRecord, "properties")
	roles := object(t, eventProperties, "roles")
	if integer(t, roles["minLength"]) != 1 || integer(t, roles["maxLength"]) != 10 {
		t.Errorf("roles bounds = %v/%v, want 1/10", roles["minLength"], roles["maxLength"])
	}
	if _, present := eventProperties["location"]; present {
		t.Error("structured event location must not be defined")
	}

	for _, forbidden := range []string{
		"availability", "checkout", "completed", "disclaimer", "guarantee", "inventory",
		"onlineUri", "shipping", "synchronization", "tax",
	} {
		if containsJSONKey(defs, forbidden) || containsJSONKey(profile, forbidden) || containsJSONKey(event, forbidden) {
			t.Errorf("forbidden field %q appears in business lexicons", forbidden)
		}
	}

	addressPath := filepath.Join("..", "..", "..", "cmd", "lexgen", "external",
		"community.lexicon.location.address.bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq.json")
	address := readLexicon(t, addressPath)
	if address["$type"] != "com.atproto.lexicon.schema" {
		t.Errorf("address $type = %v", address["$type"])
	}
	assertLexiconID(t, address, "community.lexicon.location.address")
	addressMain := definition(t, address, "main")
	if got := addressMain["type"]; got != "object" {
		t.Fatalf("address main type = %v, want object", got)
	}
	if got := stringsFrom(t, addressMain["required"]); !reflect.DeepEqual(got, []string{"country"}) {
		t.Fatalf("address required = %v, want [country]", got)
	}
	addressProperties := object(t, addressMain, "properties")
	if got := sortedKeys(addressProperties); !reflect.DeepEqual(got, []string{"country", "locality", "name", "postalCode", "region", "street"}) {
		t.Fatalf("address properties = %v", got)
	}
	country := object(t, addressProperties, "country")
	if country["type"] != "string" || integer(t, country["minLength"]) != 2 || integer(t, country["maxLength"]) != 10 {
		t.Fatalf("address country contract = %#v", country)
	}

	catalog := indigolexicon.NewBaseCatalog()
	if err := catalog.LoadDirectory(filepath.Join(root, "business")); err != nil {
		t.Fatalf("load business lexicons: %v", err)
	}
	products20 := make([]any, 20)
	for index := range products20 {
		products20[index] = map[string]any{
			"title": "Product",
			"uri":   fmt.Sprintf("https://example.com/products/%d", index),
		}
	}
	record := map[string]any{
		"$type":    "social.craftsky.business.profile",
		"products": products20,
	}
	if err := indigolexicon.ValidateRecord(catalog, record, "social.craftsky.business.profile", 0); err != nil {
		t.Fatalf("validate profile with 20 products: %v", err)
	}
	record["products"] = append(products20, map[string]any{"title": "Extra", "uri": "https://example.com/extra"})
	if err := indigolexicon.ValidateRecord(catalog, record, "social.craftsky.business.profile", 0); err == nil {
		t.Fatal("profile with 21 products passed lexicon validation")
	}

	actor := readLexicon(t, filepath.Join(root, "actor", "profile.json"))
	assertLexiconID(t, actor, "social.craftsky.actor.profile")
	actorMain := definition(t, actor, "main")
	if got := actorMain["key"]; got != "literal:self" {
		t.Fatalf("actor profile key = %v, want literal:self", got)
	}
	actorRecord := object(t, actorMain, "record")
	if _, required := actorRecord["required"]; required {
		t.Fatal("actor profile must keep every field optional")
	}
	actorProperties := object(t, actorRecord, "properties")
	if got := sortedKeys(actorProperties); !reflect.DeepEqual(got, []string{"crafts"}) {
		t.Fatalf("actor profile properties = %v, want [crafts]", got)
	}
	crafts := object(t, actorProperties, "crafts")
	craftItems := object(t, crafts, "items")
	if crafts["type"] != "array" || integer(t, crafts["maxLength"]) != 10 ||
		craftItems["type"] != "string" || integer(t, craftItems["maxLength"]) != 50 ||
		integer(t, craftItems["maxGraphemes"]) != 50 {
		t.Fatalf("actor crafts contract = %#v", crafts)
	}
}

func readLexicon(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func assertLexiconID(t *testing.T, lexicon map[string]any, want string) {
	t.Helper()
	if lexicon["lexicon"] != float64(1) || lexicon["id"] != want {
		t.Fatalf("lexicon identity = %v/%v, want 1/%s", lexicon["lexicon"], lexicon["id"], want)
	}
}

func definition(t *testing.T, lexicon map[string]any, name string) map[string]any {
	t.Helper()
	return object(t, object(t, lexicon, "defs"), name)
}

func object(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %T, want object", key, object[key])
	}
	return value
}

func array(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %T, want array", value)
	}
	return result
}

func stringsFrom(t *testing.T, value any) []string {
	t.Helper()
	values := array(t, value)
	result := make([]string, len(values))
	for index, value := range values {
		var ok bool
		result[index], ok = value.(string)
		if !ok {
			t.Fatalf("array value = %T, want string", value)
		}
	}
	return result
}

func integer(t *testing.T, value any) int {
	t.Helper()
	number, ok := value.(float64)
	if !ok {
		t.Fatalf("value = %T, want number", value)
	}
	return int(number)
}

func containsJSONKey(value any, key string) bool {
	switch value := value.(type) {
	case map[string]any:
		for childKey, child := range value {
			if childKey == key || containsJSONKey(child, key) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if containsJSONKey(child, key) {
				return true
			}
		}
	}
	return false
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
