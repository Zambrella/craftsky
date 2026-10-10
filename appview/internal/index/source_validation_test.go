package index

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

func TestSourceValidatorEnforcesCraftskyProfileFixedKeyForEveryAction(t *testing.T) {
	validProfile := json.RawMessage(`{"$type":"social.craftsky.actor.profile","crafts":["knitting"]}`)
	for _, test := range []struct {
		name   string
		action string
		rkey   string
		record json.RawMessage
		want   ValidationStatus
	}{
		{name: "create self", action: "create", rkey: "self", record: validProfile, want: ValidationValid},
		{name: "update self", action: "update", rkey: "self", record: validProfile, want: ValidationValid},
		{name: "delete self", action: "delete", rkey: "self", want: ValidationValid},
		{name: "create other", action: "create", rkey: "other", record: validProfile, want: ValidationInvalid},
		{name: "update other", action: "update", rkey: "other", record: validProfile, want: ValidationInvalid},
		{name: "delete other", action: "delete", rkey: "other", want: ValidationInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := validateSourceRecord(tap.Event{
				Collection: craftskyProfileNSID,
				Rkey:       syntax.RecordKey(test.rkey),
				Action:     test.action,
				Record:     test.record,
			})
			if result.StructuralStatus != test.want {
				t.Fatalf("structural status = %q, want %q (reason %q)", result.StructuralStatus, test.want, result.Reason)
			}
			if test.want == ValidationValid && result.SemanticStatus != ValidationValid {
				t.Fatalf("semantic status = %q, want valid (reason %q)", result.SemanticStatus, result.Reason)
			}
		})
	}
}

func TestSourceValidatorRejectsCraftskyProfileOutsideLexiconShape(t *testing.T) {
	for _, test := range []struct {
		name   string
		crafts []string
	}{
		{name: "too many crafts", crafts: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}},
		{name: "craft too long", crafts: []string{strings.Repeat("x", 51)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			record, err := json.Marshal(map[string]any{
				"$type":  craftskyProfileNSID,
				"crafts": test.crafts,
			})
			if err != nil {
				t.Fatal(err)
			}
			result := validateSourceRecord(tap.Event{
				Collection: craftskyProfileNSID,
				Rkey:       "self",
				Action:     "create",
				Record:     record,
			})
			if result.StructuralStatus != ValidationInvalid {
				t.Fatalf("structural status = %q, want invalid", result.StructuralStatus)
			}
		})
	}
}

func TestSourceValidatorEnforcesCollectionKeyPolicyForEveryAction(t *testing.T) {
	collections := []struct {
		collection  syntax.NSID
		validRkey   syntax.RecordKey
		invalidRkey syntax.RecordKey
	}{
		{collection: craftskyProfileNSID, validRkey: "self", invalidRkey: "other"},
		{collection: craftskyPostNSID, validRkey: "3aaaaaaaaaaa2", invalidRkey: "post"},
		{collection: craftskyLikeNSID, validRkey: "3aaaaaaaaaaa2", invalidRkey: "like"},
		{collection: craftskyRepostNSID, validRkey: "3aaaaaaaaaaa2", invalidRkey: "repost"},
		{collection: businessProfileCollection, validRkey: "self", invalidRkey: "other"},
		{collection: businessEventCollection, validRkey: "3aaaaaaaaaaa2", invalidRkey: "event"},
		{collection: blueskyProfileNSID, validRkey: "self", invalidRkey: "other"},
		{collection: blueskyFollowNSID, validRkey: "3aaaaaaaaaaa2", invalidRkey: "follow"},
		{collection: blueskyBlockNSID, validRkey: "3aaaaaaaaaaa2", invalidRkey: "block"},
	}
	for _, collection := range collections {
		for _, action := range []string{"create", "update", "delete"} {
			t.Run(collection.collection.String()+"/"+action, func(t *testing.T) {
				valid := validateSourceRecord(tap.Event{
					Collection: collection.collection,
					Rkey:       collection.validRkey,
					Action:     action,
					Record:     json.RawMessage(`{}`),
				})
				if valid.Reason == "invalid_record_key" {
					t.Fatalf("valid key rejected: %+v", valid)
				}

				invalid := validateSourceRecord(tap.Event{
					Collection: collection.collection,
					Rkey:       collection.invalidRkey,
					Action:     action,
					Record:     json.RawMessage(`{}`),
				})
				if invalid.StructuralStatus != ValidationInvalid || invalid.Reason != "invalid_record_key" {
					t.Fatalf("invalid key result = %+v, want structural invalid_record_key", invalid)
				}
			})
		}
	}
}

func TestSourceValidatorAcceptsSupportedCreateAndUpdateRecords(t *testing.T) {
	const subject = `{"uri":"at://did:plc:subject/social.craftsky.feed.post/3aaaaaaaaaaa2","cid":"bafysubject"}`
	records := []struct {
		collection syntax.NSID
		rkey       syntax.RecordKey
		record     json.RawMessage
	}{
		{collection: craftskyProfileNSID, rkey: "self", record: json.RawMessage(`{"crafts":["knitting"]}`)},
		{collection: craftskyPostNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"text":"hello","sponsored":false,"createdAt":"2026-09-03T12:00:00Z"}`)},
		{collection: craftskyLikeNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":` + subject + `,"createdAt":"2026-09-03T12:00:00Z"}`)},
		{collection: craftskyRepostNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":` + subject + `,"createdAt":"2026-09-03T12:00:00Z"}`)},
		{collection: businessProfileCollection, rkey: "self", record: json.RawMessage(`{"tagline":"Independent"}`)},
		{collection: businessEventCollection, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"name":"Market","startsAt":"2026-09-10T10:00:00Z","endsAt":"2026-09-10T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T00:00:00Z"}`)},
		{collection: blueskyProfileNSID, rkey: "self", record: json.RawMessage(`{"displayName":"Actor"}`)},
		{collection: blueskyFollowNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":"did:plc:subject","createdAt":"2026-09-03T12:00:00Z"}`)},
		{collection: blueskyBlockNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":"did:plc:subject","createdAt":"2026-09-03T12:00:00Z"}`)},
	}
	for _, record := range records {
		for _, action := range []string{"create", "update"} {
			t.Run(record.collection.String()+"/"+action, func(t *testing.T) {
				result := validateSourceRecord(tap.Event{
					Collection: record.collection,
					Rkey:       record.rkey,
					Action:     action,
					Record:     record.record,
				})
				if result.StructuralStatus != ValidationValid || result.SemanticStatus != ValidationValid {
					t.Fatalf("validation result = %+v, want valid", result)
				}
			})
		}
	}
}

func TestSourceValidatorRejectsInvalidShapeIdentifiersAndSemantics(t *testing.T) {
	tests := []struct {
		name       string
		collection syntax.NSID
		rkey       syntax.RecordKey
		record     json.RawMessage
		status     ValidationStatus
		reason     string
	}{
		{name: "post missing required sponsored", collection: craftskyPostNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"text":"hello","createdAt":"2026-09-03T12:00:00Z"}`), status: ValidationInvalid, reason: "invalid_lexicon"},
		{name: "post invalid language", collection: craftskyPostNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"text":"hello","sponsored":false,"createdAt":"2026-09-03T12:00:00Z","langs":["not_a_language"]}`), status: ValidationInvalid, reason: "invalid_language"},
		{name: "like missing subject", collection: craftskyLikeNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"createdAt":"2026-09-03T12:00:00Z"}`), status: ValidationInvalid, reason: "missing_subject"},
		{name: "repost invalid subject URI", collection: craftskyRepostNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":{"uri":"not-an-at-uri","cid":"bafysubject"},"createdAt":"2026-09-03T12:00:00Z"}`), status: ValidationInvalid, reason: "invalid_subject"},
		{name: "business event invalid timestamp", collection: businessEventCollection, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"name":"Market","startsAt":"nope","endsAt":"2026-09-10T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T00:00:00Z"}`), status: ValidationInvalid, reason: "invalid_lexicon"},
		{name: "Bluesky profile invalid timestamp", collection: blueskyProfileNSID, rkey: "self", record: json.RawMessage(`{"createdAt":"nope"}`), status: ValidationInvalid, reason: "invalid_timestamp"},
		{name: "follow invalid DID", collection: blueskyFollowNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":"not-a-did","createdAt":"2026-09-03T12:00:00Z"}`), status: ValidationInvalid, reason: "invalid_subject"},
		{name: "block invalid timestamp", collection: blueskyBlockNSID, rkey: "3aaaaaaaaaaa2", record: json.RawMessage(`{"subject":"did:plc:subject","createdAt":"nope"}`), status: ValidationInvalid, reason: "invalid_timestamp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validateSourceRecord(tap.Event{
				Collection: test.collection,
				Rkey:       test.rkey,
				Action:     "create",
				Record:     test.record,
			})
			if result.SemanticStatus != test.status || result.Reason != test.reason {
				t.Fatalf("validation result = %+v, want semantic %s reason %s", result, test.status, test.reason)
			}
		})
	}
}

func TestSourceValidatorEnforcesAuthoritativeStringLimits(t *testing.T) {
	tests := []struct {
		name       string
		collection syntax.NSID
		rkey       syntax.RecordKey
		record     map[string]any
	}{
		{
			name: "Craftsky post text graphemes", collection: craftskyPostNSID, rkey: "3aaaaaaaaaaa2",
			record: map[string]any{"text": strings.Repeat("x", 2001), "sponsored": false, "createdAt": "2026-09-03T12:00:00Z"},
		},
		{
			name: "Bluesky display name graphemes", collection: blueskyProfileNSID, rkey: "self",
			record: map[string]any{"displayName": strings.Repeat("x", 65)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record, err := json.Marshal(test.record)
			if err != nil {
				t.Fatal(err)
			}
			result := validateSourceRecord(tap.Event{
				Collection: test.collection,
				Rkey:       test.rkey,
				Action:     "create",
				Record:     record,
			})
			if result.StructuralStatus != ValidationInvalid || result.Reason != "invalid_lexicon" {
				t.Fatalf("validation result = %+v, want structural invalid_lexicon", result)
			}
		})
	}
}

func TestSourceValidatorRejectsMismatchedExplicitType(t *testing.T) {
	result := validateSourceRecord(tap.Event{
		Collection: blueskyFollowNSID,
		Rkey:       "3aaaaaaaaaaa2",
		Action:     "create",
		Record:     json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"did:plc:subject","createdAt":"2026-09-03T12:00:00Z"}`),
	})
	if result.StructuralStatus != ValidationInvalid || result.Reason != "invalid_lexicon" {
		t.Fatalf("validation result = %+v, want structural invalid_lexicon", result)
	}
}

// UT-007 / FR-006, FR-004 / AC-012: preserve raw field order and duplicates.
func TestSourceValidatorRejectsAmbiguousHashtagFields(t *testing.T) {
	const feature = `"$type":"app.bsky.richtext.facet#tag",`
	const prefix = `{"text":"#MeMadeMay","sponsored":false,"createdAt":"2026-10-09T11:00:00Z",`
	facet := func(fields string) string {
		return `"facets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{` + feature + fields + `}]}]`
	}
	cases := []struct{ name, fields string }{
		{"different identities", facet(`"tag":"OtherTag","Tag":"MeMadeMay"`)},
		{"reverse order", facet(`"Tag":"MeMadeMay","tag":"OtherTag"`)},
		{"same identity", facet(`"tag":"memademay","Tag":"MeMadeMay"`)},
		{"same values", facet(`"tag":"MeMadeMay","Tag":"MeMadeMay"`)},
		{"exact duplicate", facet(`"tag":"OtherTag","tag":"MeMadeMay"`)},
		{"escaped alias", facet(`"tag":"OtherTag","\u0054ag":"MeMadeMay"`)},
		{"top text", `"Text":"other",` + facet(`"tag":"MeMadeMay"`)},
		{"top facets", facet(`"tag":"MeMadeMay"`) + `,"Facets":[]`},
		{"project", `"project":null,"Project":null`},
		{"common", `"project":{"common":{},"Common":{}}`},
		{"craft type", `"project":{"common":{"craftType":"knitting","CraftType":"sewing"}}`},
		{"structured tags", `"project":{"common":{"tags":["memademay"],"Tags":["MeMadeMay"]}}`},
		{"pattern", `"project":{"common":{"pattern":{},"Pattern":{}}}`},
		{"pattern name", `"project":{"common":{"pattern":{"name":"one","Name":"two"}}}`},
		{"pattern facets", `"project":{"common":{"pattern":{"designerFacets":[],"DesignerFacets":[]}}}`},
		{"materials", `"project":{"common":{"materials":[],"Materials":[]}}`},
		{"material text", `"project":{"common":{"materials":[{"text":"one","Text":"two"}]}}`},
		{"material facets", `"project":{"common":{"materials":[{"facets":[],"Facets":[]}]}}`},
		{"facet index", `"facets":[{"index":{},"Index":{}}]`},
		{"facet features", `"facets":[{"features":[],"Features":[]}]`},
		{"byte index Unicode fold", `"facets":[{"index":{"byteStart":0,"byteſtart":0}}]`},
		{"feature type", `"facets":[{"features":[{` + feature + `"$TYPE":"app.bsky.richtext.facet#tag","tag":"MeMadeMay"}]}]`},
		{"recognized link", `"facets":[{"features":[{"$type":"app.bsky.richtext.facet#link","uri":"https://example.com","URI":"https://example.org"}]}]`},
		{"recognized mention", `"facets":[{"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:one","DID":"did:plc:two"}]}]`},
	}
	for _, tc := range cases {
		for _, action := range []string{"create", "update"} {
			t.Run(tc.name+"/"+action, func(t *testing.T) {
				result := validateSourceRecord(tap.Event{Collection: craftskyPostNSID, Rkey: "3aaaaaaaaaaa2", Action: action, Record: json.RawMessage(prefix + tc.fields + `}`)})
				if result.StructuralStatus != ValidationInvalid || result.SemanticStatus != ValidationInvalid || result.Reason != "ambiguous_tag_fields" {
					t.Fatalf("result=%+v; want structural/semantic invalid ambiguous_tag_fields", result)
				}
			})
		}
	}
	for _, fields := range []string{
		facet(`"Tag":"MeMadeMay"`),
		`"facets":[{"Index":{"byteſtart":0,"ByteEnd":10},"Features":[{` + feature + `"Tag":"MeMadeMay"}]}]`,
		`"facets":[{"index":{"byteStart":0,"byteEnd":10},"features":[{` + feature + `"tag":"MeMadeMay"},{` + feature + `"tag":"memademay"}]}]`,
		`"future":{"tag":"one","Tag":"two"}`, // Unknown payload is not a tag source.
		`"facets":[{"features":[{"$type":"future#tag","tag":"one","Tag":"two"}]}]`,
		facet(`"tag":"MeMadeMay","future":1,"Future":2`),
	} {
		result := validateSourceRecord(tap.Event{Collection: craftskyPostNSID, Rkey: "3aaaaaaaaaaa2", Action: "create", Record: json.RawMessage(prefix + fields + `}`)})
		if result.StructuralStatus != ValidationValid || result.SemanticStatus != ValidationValid {
			t.Fatalf("unambiguous/unknown payload rejected: %s: %+v", fields, result)
		}
	}
	if result := validateSourceRecord(tap.Event{Collection: craftskyPostNSID, Rkey: "3aaaaaaaaaaa2", Action: "delete"}); result.StructuralStatus != ValidationValid {
		t.Fatalf("delete rejected: %+v", result)
	}
}
