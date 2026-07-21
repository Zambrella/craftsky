package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
)

type instagramWireCorpus struct {
	SchemaVersion         int                           `json:"schemaVersion"`
	FixturePolicy         instagramFixturePolicy        `json:"fixturePolicy"`
	Limits                instagramWireLimits           `json:"limits"`
	Requests              []instagramWireRequest        `json:"requests"`
	VerificationResponses []instagramWireResponse       `json:"verificationResponses"`
	ConfirmationResponses []instagramWireResponse       `json:"confirmationResponses"`
	AccountResponses      []instagramWireResponse       `json:"accountResponses"`
	ImportCreateResponses []instagramWireResponse       `json:"importCreateResponses"`
	ImportResponses       []instagramWireResponse       `json:"importResponses"`
	SuggestionResponses   []instagramWireResponse       `json:"suggestionResponses"`
	SuggestionActions     []instagramWireResponse       `json:"suggestionActionResponses"`
	PageContracts         []instagramPageContract       `json:"pageContracts"`
	ErrorContracts        []instagramWireResponse       `json:"errorContracts"`
	DeleteContracts       []instagramDeleteContract     `json:"deleteContracts"`
	CallbackContracts     []map[string]any              `json:"callbackContracts"`
	NotificationContract  instagramNotificationContract `json:"notificationContract"`
	ClientSafety          map[string]json.RawMessage    `json:"clientSafety"`
}

type instagramFixturePolicy struct {
	Classification          string `json:"classification"`
	ContainsUserDerivedData bool   `json:"containsUserDerivedData"`
	Description             string `json:"description"`
}

type instagramWireLimits struct {
	DefaultPageSize int `json:"defaultPageSize"`
	MaxPageSize     int `json:"maxPageSize"`
}

type instagramWireRequest struct {
	ID          string          `json:"id"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Body        json.RawMessage `json:"body"`
	BodyPresent *bool           `json:"bodyPresent"`
}

type instagramWireResponse struct {
	ID          string          `json:"id"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Status      int             `json:"status"`
	Shape       string          `json:"shape"`
	ReplayGroup string          `json:"replayGroup"`
	Body        json.RawMessage `json:"body"`
}

type instagramPageContract struct {
	ID             string          `json:"id"`
	Resource       string          `json:"resource"`
	Method         string          `json:"method"`
	Path           string          `json:"path"`
	RequestedLimit *int            `json:"requestedLimit"`
	EffectiveLimit int             `json:"effectiveLimit"`
	RequestCursor  *string         `json:"requestCursor"`
	Status         int             `json:"status"`
	Body           json.RawMessage `json:"body"`
}

type instagramDeleteContract struct {
	ID               string   `json:"id"`
	Resource         string   `json:"resource"`
	Method           string   `json:"method"`
	PathTemplate     string   `json:"pathTemplate"`
	Variants         []string `json:"variants"`
	Status           int      `json:"status"`
	BodyPresent      bool     `json:"bodyPresent"`
	MutatesOwnedOnly bool     `json:"mutatesOwnedOnly"`
}

type instagramNotificationContract struct {
	ID                 string                    `json:"id"`
	Method             string                    `json:"method"`
	Path               string                    `json:"path"`
	Status             int                       `json:"status"`
	Body               json.RawMessage           `json:"body"`
	UnknownClientCases []instagramUnknownFixture `json:"unknownClientCases"`
}

type instagramUnknownFixture struct {
	ID                    string          `json:"id"`
	ExpectedClientVariant string          `json:"expectedClientVariant"`
	Body                  json.RawMessage `json:"body"`
}

func TestInstagramWireCorpusUsesExactPublicResponseShapes(t *testing.T) {
	corpus := loadInstagramWireCorpus(t)
	if corpus.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d", corpus.SchemaVersion)
	}
	if corpus.FixturePolicy.Classification != "whollySynthetic" || corpus.FixturePolicy.ContainsUserDerivedData {
		t.Fatalf("unsafe fixture policy: %+v", corpus.FixturePolicy)
	}

	verificationStates := map[string]struct{}{}
	for _, fixture := range corpus.VerificationResponses {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertSuccessMetadata(t, fixture, map[int]bool{200: true, 201: true})
			body := decodeWireMap(t, fixture.Body)
			state, _ := body["state"].(string)
			verificationStates[state] = struct{}{}
			if fixture.Shape == "creation" {
				assertWireRoundTrip[instagramVerificationCreateResponse](t, fixture.Body)
				return
			}
			assertWireRoundTrip[instagramVerificationResponse](t, fixture.Body)
		})
	}
	assertStringSet(t, "verification states", verificationStates, []string{
		"pendingDm", "processing", "pendingConfirmation", "confirmed", "expired",
		"cancelled", "superseded", "rejected", "conflicted",
	})

	for _, fixture := range corpus.ConfirmationResponses {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertSuccessMetadata(t, fixture, map[int]bool{200: true})
			assertWireRoundTrip[instagramConfirmationResponse](t, fixture.Body)
		})
	}

	linkStates := map[string]struct{}{}
	for _, fixture := range corpus.AccountResponses {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertSuccessMetadata(t, fixture, map[int]bool{200: true})
			assertWireRoundTrip[instagramAccountStatusResponse](t, fixture.Body)
			body := decodeWireMap(t, fixture.Body)
			if account, ok := body["account"].(map[string]any); ok {
				linkStates[account["state"].(string)] = struct{}{}
			}
		})
	}
	assertStringSet(t, "account link states", linkStates, []string{
		"active", "membershipInactive", "revoked", "superseded", "disputed",
	})

	for _, fixture := range corpus.ImportCreateResponses {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertSuccessMetadata(t, fixture, map[int]bool{201: true})
			assertWireRoundTrip[instagramImportCreateResponse](t, fixture.Body)
		})
	}

	importStates := map[string]struct{}{}
	for _, fixture := range corpus.ImportResponses {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertSuccessMetadata(t, fixture, map[int]bool{200: true})
			assertWireRoundTrip[instagramImportResponse](t, fixture.Body)
			importStates[decodeWireMap(t, fixture.Body)["state"].(string)] = struct{}{}
		})
	}
	assertStringSet(t, "import states", importStates, []string{"active", "membershipInactive", "expired"})

	suggestionStates := map[string]struct{}{}
	for _, fixture := range corpus.SuggestionResponses {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertWireRoundTrip[instagramSuggestionResponse](t, fixture.Body)
			suggestionStates[decodeWireMap(t, fixture.Body)["state"].(string)] = struct{}{}
		})
	}
	assertStringSet(t, "suggestion states", suggestionStates, []string{
		"pending", "accepting", "accepted", "alreadyFollowing", "dismissed", "invalidated",
	})

	for _, fixture := range corpus.SuggestionActions {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			assertSuccessMetadata(t, fixture, map[int]bool{200: true})
			assertWireRoundTrip[instagramSuggestionActionResponse](t, fixture.Body)
		})
	}

	assertReplayGroupsAreIdentical(t, append(
		append([]instagramWireResponse(nil), corpus.ConfirmationResponses...),
		corpus.SuggestionActions...,
	))
}

func TestInstagramWireCorpusRequestsAreStrictAndCamelCase(t *testing.T) {
	corpus := loadInstagramWireCorpus(t)
	seen := map[string]bool{}
	for _, fixture := range corpus.Requests {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			if fixture.ID == "suggestion.accept" {
				if fixture.BodyPresent == nil || *fixture.BodyPresent {
					t.Fatal("bodyless accept request is not declared bodyless")
				}
				return
			}
			body := decodeWireMap(t, fixture.Body)
			assertCamelCaseObject(t, body, fixture.ID)
			req := httptest.NewRequest(fixture.Method, fixture.Path, bytes.NewReader(fixture.Body))
			switch fixture.ID {
			case "verification.create":
				var value struct{}
				mustStrictDecodeAndRoundTrip(t, req, &value, fixture.Body)
			case "verification.confirm":
				var value struct {
					Discoverable *bool `json:"discoverable"`
				}
				mustStrictDecodeAndRoundTrip(t, req, &value, fixture.Body)
				if value.Discoverable == nil {
					t.Fatal("discoverable omitted")
				}
			case "settings.discovery", "settings.reactivate":
				var value struct {
					Discoverable *bool `json:"discoverable,omitempty"`
					Reactivate   *bool `json:"reactivate,omitempty"`
				}
				mustStrictDecodeAndRoundTrip(t, req, &value, fixture.Body)
				if !validInstagramSettingsRequest(value.Discoverable, value.Reactivate) {
					t.Fatal("fixture is not a valid settings request")
				}
			case "import.create.manual", "import.create.instagramJson":
				var value struct {
					SourceType      instagram.ImportSourceType `json:"sourceType"`
					RetainUnmatched *bool                      `json:"retainUnmatched"`
					Entries         []instagram.ImportEntry    `json:"entries"`
				}
				mustStrictDecodeAndRoundTrip(t, req, &value, fixture.Body)
				if !value.SourceType.Valid() || value.RetainUnmatched == nil || len(value.Entries) == 0 {
					t.Fatal("fixture is not a valid import request")
				}
				for _, entry := range value.Entries {
					if !entry.Direction.Valid() {
						t.Fatalf("invalid direction %q", entry.Direction)
					}
				}
			case "import.disableRetention", "import.reactivate":
				var value struct {
					RetainUnmatched *bool `json:"retainUnmatched,omitempty"`
					Reactivate      *bool `json:"reactivate,omitempty"`
				}
				mustStrictDecodeAndRoundTrip(t, req, &value, fixture.Body)
				if value.RetainUnmatched == nil && value.Reactivate == nil {
					t.Fatal("empty import patch")
				}
			default:
				t.Fatalf("unvalidated request fixture %q", fixture.ID)
			}
			seen[fixture.ID] = true
		})
	}
	if len(seen) != len(corpus.Requests)-1 {
		t.Fatalf("validated %d body requests, corpus has %d", len(seen), len(corpus.Requests))
	}

	fixture := findWireRequest(t, corpus.Requests, "import.create.instagramJson")
	invalid := decodeWireMap(t, fixture.Body)
	invalid["rawArchive"] = "synthetic-private-archive"
	raw, err := json.Marshal(invalid)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(fixture.Method, fixture.Path, bytes.NewReader(raw))
	var destination struct {
		SourceType      instagram.ImportSourceType `json:"sourceType"`
		RetainUnmatched *bool                      `json:"retainUnmatched"`
		Entries         []instagram.ImportEntry    `json:"entries"`
	}
	if err := decodeStrictJSONObject(request, &destination); err == nil {
		t.Fatal("strict import boundary accepted a raw archive field")
	}
}

func TestInstagramWireCorpusPagesLockDefaultsMaximumAndCursors(t *testing.T) {
	corpus := loadInstagramWireCorpus(t)
	if corpus.Limits != (instagramWireLimits{DefaultPageSize: 20, MaxPageSize: 50}) {
		t.Fatalf("limits = %+v", corpus.Limits)
	}

	resources := map[string]map[string]bool{}
	for _, fixture := range corpus.PageContracts {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			if fixture.Method != "GET" || fixture.Status != 200 {
				t.Fatalf("metadata = %s %d", fixture.Method, fixture.Status)
			}
			kind := "default"
			if fixture.RequestedLimit != nil {
				kind = "max"
				if *fixture.RequestedLimit != corpus.Limits.MaxPageSize || fixture.EffectiveLimit != corpus.Limits.MaxPageSize {
					t.Fatalf("max limit metadata = requested %v effective %d", fixture.RequestedLimit, fixture.EffectiveLimit)
				}
				if fixture.RequestCursor == nil {
					t.Fatal("max-page fixture omits input cursor")
				}
			} else if fixture.EffectiveLimit != corpus.Limits.DefaultPageSize || fixture.RequestCursor != nil {
				t.Fatalf("default limit metadata = effective %d cursor %v", fixture.EffectiveLimit, fixture.RequestCursor)
			}
			if resources[fixture.Resource] == nil {
				resources[fixture.Resource] = map[string]bool{}
			}
			resources[fixture.Resource][kind] = true

			body := decodeWireMap(t, fixture.Body)
			_, cursorPresent := body["cursor"]
			if kind == "default" && cursorPresent {
				t.Fatal("default fixture must prove cursor omission")
			}
			if kind == "max" && !cursorPresent {
				t.Fatal("max fixture must prove cursor presence")
			}

			switch fixture.Resource {
			case "imports":
				assertWireRoundTrip[instagramImportPageResponse](t, fixture.Body)
				if fixture.RequestCursor != nil {
					if _, err := decodeInstagramImportCursor(*fixture.RequestCursor); err != nil {
						t.Fatalf("input cursor is not an opaque AppView cursor: %v", err)
					}
					if _, err := decodeInstagramImportCursor(body["cursor"].(string)); err != nil {
						t.Fatalf("output cursor is not an opaque AppView cursor: %v", err)
					}
				}
			case "suggestions":
				assertWireRoundTrip[instagramSuggestionPageResponse](t, fixture.Body)
				if fixture.RequestCursor != nil {
					if _, err := decodeInstagramSuggestionCursor(*fixture.RequestCursor); err != nil {
						t.Fatalf("input cursor is not an opaque AppView cursor: %v", err)
					}
					if _, err := decodeInstagramSuggestionCursor(body["cursor"].(string)); err != nil {
						t.Fatalf("output cursor is not an opaque AppView cursor: %v", err)
					}
				}
			default:
				t.Fatalf("unknown page resource %q", fixture.Resource)
			}
		})
	}
	for _, resource := range []string{"imports", "suggestions"} {
		if !resources[resource]["default"] || !resources[resource]["max"] {
			t.Fatalf("%s does not cover default and max pages: %v", resource, resources[resource])
		}
	}

	owner := syntax.DID("did:plc:syntheticowner")
	imports := &wireImportRepository{}
	importService, err := instagram.NewImportService(instagram.ImportServiceOptions{Repository: imports})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := importService.ListImports(context.Background(), owner, 0, nil); err != nil || imports.limit != 20 {
		t.Fatalf("import default limit = %d, err %v", imports.limit, err)
	}
	if _, _, err := importService.ListImports(context.Background(), owner, 500, nil); err != nil || imports.limit != 50 {
		t.Fatalf("import max limit = %d, err %v", imports.limit, err)
	}

	suggestions := &wireSuggestionRepository{}
	suggestionService, err := instagram.NewSuggestionService(instagram.SuggestionServiceOptions{
		Repository: suggestions,
		Policy:     wireEligibilityPolicy{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := suggestionService.ListSuggestions(context.Background(), owner, 0, nil); err != nil || suggestions.limit != 20 {
		t.Fatalf("suggestion default limit = %d, err %v", suggestions.limit, err)
	}
	if _, _, err := suggestionService.ListSuggestions(context.Background(), owner, 500, nil); err != nil || suggestions.limit != 50 {
		t.Fatalf("suggestion max limit = %d, err %v", suggestions.limit, err)
	}
}

func TestInstagramWireCorpusErrorsDeletesAndWebhookRetryMetadata(t *testing.T) {
	corpus := loadInstagramWireCorpus(t)
	errorCodes := map[string]bool{}
	for _, fixture := range corpus.ErrorContracts {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			if fixture.Status < 400 || fixture.Status > 599 {
				t.Fatalf("status = %d", fixture.Status)
			}
			assertWireRoundTrip[envelope.Error](t, fixture.Body)
			body := decodeWireMap(t, fixture.Body)
			if !reflect.DeepEqual(sortedKeys(body), []string{"error", "message", "requestId"}) {
				t.Fatalf("error keys = %v", sortedKeys(body))
			}
			errorCodes[body["error"].(string)] = true
		})
	}
	for _, code := range []string{
		"instagram_verification_unavailable", "rate_limited", "instagram_verification_not_found",
		"instagram_verification_state_conflict", "instagram_link_conflict", "profile_not_found",
		"invalid_request", "instagram_link_not_found", "instagram_reactivation_required",
		"request_too_large", "invalid_instagram_import", "invalid_cursor",
		"instagram_import_not_found", "instagram_import_inactive", "instagram_import_expired",
		"unmatched_data_unavailable", "instagram_suggestion_not_found",
		"instagram_suggestion_ineligible", "follow_write_unavailable",
	} {
		if !errorCodes[code] {
			t.Errorf("error contract %q is absent", code)
		}
	}

	wantDeleteVariants := map[string][]string{
		"verification": {"absent", "expiredTombstone", "foreign", "owned", "purged"},
		"account":      {"absent", "identicalReplay", "owned", "purged"},
		"import":       {"absent", "foreign", "identicalReplay", "owned", "purged"},
		"suggestion":   {"absent", "foreign", "identicalReplay", "ownedPending", "purged", "terminal"},
	}
	for _, contract := range corpus.DeleteContracts {
		if contract.Method != "DELETE" || contract.Status != 204 || contract.BodyPresent || !contract.MutatesOwnedOnly {
			t.Errorf("unsafe DELETE contract %+v", contract)
			continue
		}
		variants := append([]string(nil), contract.Variants...)
		sort.Strings(variants)
		if !reflect.DeepEqual(variants, wantDeleteVariants[contract.Resource]) {
			t.Errorf("%s DELETE variants = %v", contract.Resource, variants)
		}
		delete(wantDeleteVariants, contract.Resource)
	}
	if len(wantDeleteVariants) != 0 {
		t.Fatalf("missing DELETE contracts: %v", wantDeleteVariants)
	}

	callbacks := map[string]map[string]any{}
	for _, contract := range corpus.CallbackContracts {
		id, _ := contract["id"].(string)
		if id == "" {
			t.Fatal("callback contract without id")
		}
		callbacks[id] = contract
	}
	verifySuccess := callbacks["callback.verify.success"]
	if int(verifySuccess["status"].(float64)) != 200 || verifySuccess["responseBody"] != "synthetic-callback-challenge" || verifySuccess["reflectsChallenge"] != true {
		t.Fatalf("invalid successful callback metadata: %v", verifySuccess)
	}
	verifyFailure := callbacks["callback.verify.forbidden"]
	query := verifyFailure["query"].(map[string]any)
	if verifyFailure["reflectsChallenge"] != false || strings.Contains(verifyFailure["responseBody"].(string), query["hub.challenge"].(string)) {
		t.Fatalf("forbidden callback reflects private challenge: %v", verifyFailure)
	}
	for _, id := range []string{"callback.delivery.sourceIpLimited", "callback.delivery.globalLimited"} {
		contract := callbacks[id]
		if int(contract["status"].(float64)) != 429 || contract["persistPartial"] != false {
			t.Fatalf("invalid rate-limit metadata for %s: %v", id, contract)
		}
		retry, err := strconv.Atoi(contract["headers"].(map[string]any)["Retry-After"].(string))
		if err != nil || retry < 1 || retry > 60 {
			t.Fatalf("invalid Retry-After for %s: %v", id, contract)
		}
	}
	limited := callbacks["callback.delivery.invalidRedemptionLimited"]
	if int(limited["status"].(float64)) != 200 || limited["terminalDeduplicatedIgnored"] != true || limited["sensitiveFieldsCleared"] != true || limited["lookupAllowed"] != false {
		t.Fatalf("invalid per-sender limit metadata: %v", limited)
	}
	for _, id := range []string{"callback.delivery.supported", "callback.delivery.duplicate"} {
		contract := callbacks[id]
		if int(contract["status"].(float64)) != 200 || contract["bodyPresent"] != false || contract["durableBeforeAck"] != true {
			t.Fatalf("invalid durable acknowledgement metadata for %s: %v", id, contract)
		}
	}
}

func TestInstagramWireCorpusNotificationUnionAndPrivateFieldAbsence(t *testing.T) {
	corpus := loadInstagramWireCorpus(t)
	assertWireRoundTrip[NotificationPage](t, corpus.NotificationContract.Body)
	page := decodeWireMap(t, corpus.NotificationContract.Body)
	items := page["items"].([]any)
	if len(items) != 8 {
		t.Fatalf("notification item count = %d", len(items))
	}
	socialTypes := map[string]bool{}
	var system map[string]any
	for _, raw := range items {
		item := raw.(map[string]any)
		switch item["kind"] {
		case "social":
			socialTypes[item["type"].(string)] = true
			for _, required := range []string{"uri", "cid", "rkey", "actor", "references"} {
				if _, ok := item[required]; !ok {
					t.Errorf("social %s omits %s", item["type"], required)
				}
			}
			if _, ok := item["system"]; ok {
				t.Errorf("social %s contains system payload", item["type"])
			}
		case "system":
			system = item
		default:
			t.Fatalf("server fixture contains unknown kind %v", item["kind"])
		}
	}
	for _, value := range []string{"follow", "like", "repost", "reply", "mention", "quote", "everythingElse"} {
		if !socialTypes[value] {
			t.Errorf("social notification type %q is absent", value)
		}
	}
	if system == nil || system["type"] != "instagramMatch" {
		t.Fatal("Instagram system notification is absent")
	}
	for _, forbidden := range []string{"uri", "cid", "rkey", "actor", "references", "subjectPost", "reply", "contentAvailable"} {
		if _, ok := system[forbidden]; ok {
			t.Errorf("system notification exposes social field %q", forbidden)
		}
	}
	systemPayload := system["system"].(map[string]any)
	if !reflect.DeepEqual(sortedKeys(systemPayload), []string{"count", "countCapped", "destination"}) {
		t.Errorf("system payload keys = %v", sortedKeys(systemPayload))
	}

	privateKeys := map[string]bool{
		"challengeDigest": true, "candidateIgsid": true, "senderIgsid": true,
		"officialAccountId": true, "messageIdDigest": true, "leaseOwner": true,
		"leaseExpiresAt": true, "attemptCount": true, "nextAttemptAt": true,
		"conflictParty": true, "pdsOperationKey": true, "rawBody": true,
		"messageText": true, "plaintextChallenge": true, "profileResponse": true,
	}
	for id, body := range allServerResponseBodies(corpus) {
		value := decodeWireValue(t, body)
		assertCamelCaseObject(t, value, id)
		assertNoPrivateWireKeys(t, value, privateKeys, id)
	}

	for _, unknown := range corpus.NotificationContract.UnknownClientCases {
		if unknown.ExpectedClientVariant != "genericSystem" && unknown.ExpectedClientVariant != "genericSocial" {
			t.Errorf("unknown fixture %s has unsafe expectation %q", unknown.ID, unknown.ExpectedClientVariant)
		}
	}
}

func loadInstagramWireCorpus(t *testing.T) instagramWireCorpus {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test path")
	}
	path := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "docs", "changes", "2026-07-11-instagram-dm-verification", "fixtures", "instagram_wire", "corpus.json")
	raw, err := osReadFile(path)
	if err != nil {
		t.Fatalf("read shared Instagram wire corpus: %v", err)
	}
	var corpus instagramWireCorpus
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		t.Fatalf("decode shared Instagram wire corpus: %v", err)
	}
	return corpus
}

// osReadFile is a variable only to keep the fixture loader easy to identify in
// privacy reviews; tests always use the standard library implementation.
var osReadFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func assertSuccessMetadata(t *testing.T, fixture instagramWireResponse, allowed map[int]bool) {
	t.Helper()
	if fixture.Method == "" || fixture.Path == "" || !allowed[fixture.Status] || len(fixture.Body) == 0 {
		t.Fatalf("invalid success fixture metadata: %+v", fixture)
	}
}

func assertWireRoundTrip[T any](t *testing.T, raw json.RawMessage) {
	t.Helper()
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode production wire type: %v\n%s", err, raw)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode production wire type: %v", err)
	}
	assertSemanticJSONEqual(t, raw, encoded)
}

func assertSemanticJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	wantValue := decodeWireValue(t, want)
	gotValue := decodeWireValue(t, got)
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Fatalf("wire mismatch\nwant: %s\n got: %s", want, got)
	}
}

func decodeWireMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	value := decodeWireValue(t, raw)
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("fixture is %T, want object", value)
	}
	return result
}

func decodeWireValue(t *testing.T, raw []byte) any {
	t.Helper()
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode fixture JSON: %v", err)
	}
	return value
}

func mustStrictDecodeAndRoundTrip(t *testing.T, request *http.Request, destination any, raw []byte) {
	t.Helper()
	if err := decodeStrictJSONObject(request, destination); err != nil {
		t.Fatalf("strict decode: %v", err)
	}
	encoded, err := json.Marshal(destination)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	assertSemanticJSONEqual(t, raw, encoded)
}

func assertStringSet(t *testing.T, label string, got map[string]struct{}, want []string) {
	t.Helper()
	wantSet := make(map[string]struct{}, len(want))
	for _, value := range want {
		wantSet[value] = struct{}{}
	}
	if !reflect.DeepEqual(got, wantSet) {
		t.Fatalf("%s = %v, want %v", label, got, wantSet)
	}
}

func assertReplayGroupsAreIdentical(t *testing.T, fixtures []instagramWireResponse) {
	t.Helper()
	groups := map[string]json.RawMessage{}
	counts := map[string]int{}
	for _, fixture := range fixtures {
		if fixture.ReplayGroup == "" {
			continue
		}
		counts[fixture.ReplayGroup]++
		if first, ok := groups[fixture.ReplayGroup]; ok {
			assertSemanticJSONEqual(t, first, fixture.Body)
		} else {
			groups[fixture.ReplayGroup] = fixture.Body
		}
	}
	for group, count := range counts {
		if count < 2 {
			t.Errorf("replay group %q has only %d fixture", group, count)
		}
	}
}

func assertCamelCaseObject(t *testing.T, value any, path string) {
	t.Helper()
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if strings.ContainsRune(key, '_') || key == "" || !unicode.IsLower(rune(key[0])) {
				t.Errorf("%s contains non-camelCase key %q", path, key)
			}
			assertCamelCaseObject(t, child, path+"."+key)
		}
	case []any:
		for index, child := range value {
			assertCamelCaseObject(t, child, path+"["+strconv.Itoa(index)+"]")
		}
	}
}

func assertNoPrivateWireKeys(t *testing.T, value any, forbidden map[string]bool, path string) {
	t.Helper()
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if forbidden[key] {
				t.Errorf("%s exposes private field %q", path, key)
			}
			assertNoPrivateWireKeys(t, child, forbidden, path+"."+key)
		}
	case []any:
		for index, child := range value {
			assertNoPrivateWireKeys(t, child, forbidden, path+"["+strconv.Itoa(index)+"]")
		}
	}
}

func allServerResponseBodies(corpus instagramWireCorpus) map[string]json.RawMessage {
	result := map[string]json.RawMessage{}
	appendFixtures := func(fixtures []instagramWireResponse) {
		for _, fixture := range fixtures {
			result[fixture.ID] = fixture.Body
		}
	}
	appendFixtures(corpus.VerificationResponses)
	appendFixtures(corpus.ConfirmationResponses)
	appendFixtures(corpus.AccountResponses)
	appendFixtures(corpus.ImportCreateResponses)
	appendFixtures(corpus.ImportResponses)
	appendFixtures(corpus.SuggestionResponses)
	appendFixtures(corpus.SuggestionActions)
	appendFixtures(corpus.ErrorContracts)
	for _, fixture := range corpus.PageContracts {
		result[fixture.ID] = fixture.Body
	}
	result[corpus.NotificationContract.ID] = corpus.NotificationContract.Body
	return result
}

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func findWireRequest(t *testing.T, fixtures []instagramWireRequest, id string) instagramWireRequest {
	t.Helper()
	for _, fixture := range fixtures {
		if fixture.ID == id {
			return fixture
		}
	}
	t.Fatalf("request fixture %q not found", id)
	return instagramWireRequest{}
}

type wireImportRepository struct {
	limit int
}

func (*wireImportRepository) CreateImport(context.Context, instagram.CreateImportParams) (instagram.CreateImportResult, error) {
	return instagram.CreateImportResult{}, nil
}

func (r *wireImportRepository) ListImports(_ context.Context, _ syntax.DID, limit int, _ *instagram.ImportCursor, _ time.Time) ([]instagram.GraphImport, *instagram.ImportCursor, error) {
	r.limit = limit
	return nil, nil, nil
}

func (*wireImportRepository) GetImport(context.Context, syntax.DID, uuid.UUID, time.Time) (instagram.GraphImport, error) {
	return instagram.GraphImport{}, nil
}

func (*wireImportRepository) UpdateImport(context.Context, syntax.DID, uuid.UUID, instagram.UpdateImportParams) (instagram.GraphImport, error) {
	return instagram.GraphImport{}, nil
}

func (*wireImportRepository) DeleteImport(context.Context, syntax.DID, uuid.UUID, time.Time) error {
	return nil
}

type wireSuggestionRepository struct {
	limit int
}

func (r *wireSuggestionRepository) ListPendingSuggestions(_ context.Context, _ syntax.DID, limit int, _ *instagram.SuggestionCursor) ([]instagram.SuggestionEvidence, *instagram.SuggestionCursor, error) {
	r.limit = limit
	return nil, nil, nil
}

func (*wireSuggestionRepository) DismissSuggestion(context.Context, syntax.DID, uuid.UUID, time.Time) error {
	return nil
}

func (*wireSuggestionRepository) ClaimSuggestionAcceptance(context.Context, syntax.DID, uuid.UUID, string, time.Time) (instagram.AcceptanceClaim, error) {
	return instagram.AcceptanceClaim{}, nil
}

func (*wireSuggestionRepository) CompleteSuggestionAcceptance(context.Context, syntax.DID, uuid.UUID, instagram.InstagramSuggestionState, time.Time) (instagram.Suggestion, error) {
	return instagram.Suggestion{}, nil
}

func (*wireSuggestionRepository) ResetSuggestionAcceptance(context.Context, syntax.DID, uuid.UUID, string, time.Time) error {
	return nil
}

func (*wireSuggestionRepository) InvalidateSuggestion(context.Context, syntax.DID, uuid.UUID, time.Time) error {
	return nil
}

type wireEligibilityPolicy struct{}

func (wireEligibilityPolicy) Evaluate(context.Context, instagram.EligibilityStage, instagram.SuggestionEligibilityRequest) (instagram.EligibilityDecision, error) {
	return instagram.EligibilityDecision{Eligible: true}, nil
}
