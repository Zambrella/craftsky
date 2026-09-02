package craftsky

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	indigolexicon "github.com/bluesky-social/indigo/atproto/lexicon"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
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
	encoded, err := encodeCanonicalCBOR(address)
	if err != nil {
		t.Fatalf("encode address DAG-CBOR: %v", err)
	}
	hash, err := multihash.Sum(encoded, multihash.SHA2_256, -1)
	if err != nil {
		t.Fatalf("hash address DAG-CBOR: %v", err)
	}
	if got := cid.NewCidV1(cid.DagCBOR, hash).String(); got != "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq" {
		t.Fatalf("address CID = %s", got)
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

	actorPath := filepath.Join(root, "actor", "profile.json")
	actorBytes, err := os.ReadFile(actorPath)
	if err != nil {
		t.Fatalf("read actor profile: %v", err)
	}
	digest := sha256.Sum256(actorBytes)
	if got := hex.EncodeToString(digest[:]); got != "29b3167edab98bb360e2713a1f55861c75180ba551c2adf052fcf639c2589f4f" {
		t.Fatalf("actor profile digest changed: %s", got)
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

func encodeCanonicalCBOR(value any) ([]byte, error) {
	var encoded bytes.Buffer
	if err := appendCanonicalCBOR(&encoded, value); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func appendCanonicalCBOR(encoded *bytes.Buffer, value any) error {
	switch value := value.(type) {
	case nil:
		encoded.WriteByte(0xf6)
	case bool:
		if value {
			encoded.WriteByte(0xf5)
		} else {
			encoded.WriteByte(0xf4)
		}
	case float64:
		if value < 0 || value != float64(uint64(value)) {
			return fmt.Errorf("unsupported JSON number %v", value)
		}
		appendCBORHead(encoded, 0, uint64(value))
	case string:
		appendCBORHead(encoded, 3, uint64(len(value)))
		encoded.WriteString(value)
	case []any:
		appendCBORHead(encoded, 4, uint64(len(value)))
		for _, child := range value {
			if err := appendCanonicalCBOR(encoded, child); err != nil {
				return err
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if len(keys[i]) != len(keys[j]) {
				return len(keys[i]) < len(keys[j])
			}
			return keys[i] < keys[j]
		})
		appendCBORHead(encoded, 5, uint64(len(keys)))
		for _, key := range keys {
			if err := appendCanonicalCBOR(encoded, key); err != nil {
				return err
			}
			if err := appendCanonicalCBOR(encoded, value[key]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported JSON value %T", value)
	}
	return nil
}

func appendCBORHead(encoded *bytes.Buffer, major byte, value uint64) {
	major <<= 5
	switch {
	case value < 24:
		encoded.WriteByte(major | byte(value))
	case value <= 0xff:
		encoded.WriteByte(major | 24)
		encoded.WriteByte(byte(value))
	case value <= 0xffff:
		encoded.WriteByte(major | 25)
		var buffer [2]byte
		binary.BigEndian.PutUint16(buffer[:], uint16(value))
		encoded.Write(buffer[:])
	case value <= 0xffffffff:
		encoded.WriteByte(major | 26)
		var buffer [4]byte
		binary.BigEndian.PutUint32(buffer[:], uint32(value))
		encoded.Write(buffer[:])
	default:
		encoded.WriteByte(major | 27)
		var buffer [8]byte
		binary.BigEndian.PutUint64(buffer[:], value)
		encoded.Write(buffer[:])
	}
}
