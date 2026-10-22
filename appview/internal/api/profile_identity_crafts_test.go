package api_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

type fakeIdentityDecorationReader struct {
	fakeProfileCustomisationBatchReader
	decorations map[syntax.DID]api.ProfileIdentityDecoration
	batchCalls  int
}

func (f *fakeIdentityDecorationReader) ReadIdentityBatch(_ context.Context, dids []syntax.DID) (map[syntax.DID]api.ProfileIdentityDecoration, error) {
	f.batchCalls++
	f.dids = dids
	return f.decorations, nil
}

func TestIdentityCraftsUseOneBatchAndPreserveShells(t *testing.T) {
	reader := &fakeIdentityDecorationReader{decorations: map[syntax.DID]api.ProfileIdentityDecoration{
		"did:plc:alice": {Customisation: api.DefaultProfileCustomisation, Crafts: []string{"knitting", "crochet"}},
	}}
	hydrator := api.NewIdentityCustomisationHydrator(reader, fakeCustomisationTiers{"did:plc:alice": subscriptions.TierFree})
	body, err := hydrator.HydrateJSON(context.Background(), []byte(`{"items":[
		{"author":{"did":"did:plc:alice","handle":"alice.test"}},
		{"quote":{"author":{"did":"did:plc:alice","handle":"alice.test"}}},
		{"did":"did:plc:alice","handle":"alice.test","crafts":[]},
		{"did":"did:plc:alice","handle":"alice.test","blocking":true},
		{"did":"did:plc:alice","handle":"alice.test","blockedBy":true},
		{"did":"did:plc:alice","handle":"alice.test","available":false},
		{"state":"blocked"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	if reader.batchCalls != 1 || reader.calls != 0 || len(reader.dids) != 1 {
		t.Fatalf("expected one deduplicated combined lookup: %+v", reader)
	}
	var result struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	for _, identity := range []map[string]any{
		result.Items[0]["author"].(map[string]any),
		result.Items[1]["quote"].(map[string]any)["author"].(map[string]any),
	} {
		if !reflect.DeepEqual(identity["crafts"], []any{"knitting", "crochet"}) {
			t.Fatalf("author crafts = %v", identity["crafts"])
		}
	}
	if !reflect.DeepEqual(result.Items[2]["crafts"], []any{}) {
		t.Fatal("explicit empty crafts were replaced")
	}
	for _, shell := range result.Items[3:] {
		if _, exists := shell["crafts"]; exists {
			t.Fatalf("crafts restored into a shell: %v", shell)
		}
	}
}

func TestProfileIdentityBatchReadsPublicCraftsWithCustomisation(t *testing.T) {
	pool := testdb.WithSchema(t, profileCustomisationStoreTestDDL+profileCustomisationSchemaMigrations(t))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE craftsky_profiles SET crafts = ARRAY['knitting','crochet'] WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatal(err)
	}
	got, err := api.NewProfileCustomisationStore(pool).ReadIdentityBatch(ctx, []syntax.DID{"did:plc:alice", "did:plc:bob", "did:plc:alice", "did:plc:outside"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !reflect.DeepEqual(got["did:plc:alice"].Crafts, []string{"knitting", "crochet"}) {
		t.Fatalf("combined identity batch = %+v", got)
	}
	if got["did:plc:alice"].Customisation != api.DefaultProfileCustomisation {
		t.Fatal("customisation defaults changed")
	}
}
