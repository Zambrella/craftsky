package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestAT007DiagnoseReportAndModerateBusinessEvent(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t)+businessEventModerationDDL+businessEventModerationMigrationDDL(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:event-owner")
	visitor := syntax.DID("did:plc:event-visitor")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)

	moderation, lifecycles := newModerationStore(t, pool, func() time.Time { return asOf })
	for _, did := range []syntax.DID{owner, visitor} {
		if _, err := lifecycles.EnsureOnboardingOwner(ctx, did); err != nil {
			t.Fatalf("seed lifecycle for %s: %v", did, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES ($1)`, did); err != nil {
			t.Fatalf("seed member %s: %v", did, err)
		}
		if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
			Owner: did, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileCreated",
		}); err != nil {
			t.Fatalf("activate member %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed business account type: %v", err)
	}

	fixtures := []eventFixture{
		{
			Owner: owner, Rkey: "3msfuture0001", Name: "Future", StartsAt: asOf.Add(8 * time.Hour), EndsAt: asOf.Add(9 * time.Hour),
			RawFields: map[string]any{"image": map[string]any{
				"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"}, "mimeType": "image/png", "size": 0},
				"alt":   "Future event", "aspectRatio": map[string]any{"width": 3, "height": 2},
			}},
		},
		{Owner: owner, Rkey: "3mspast000001", Name: "Past", StartsAt: asOf.Add(-2 * time.Hour), EndsAt: asOf.Add(-time.Hour)},
		{Owner: owner, Rkey: "3mscancel0001", Name: "Cancelled", StartsAt: asOf.Add(7 * time.Hour), EndsAt: asOf.Add(8 * time.Hour), Status: "cancelled"},
		{Owner: owner, Rkey: "3mspostpone01", Name: "Postponed", StartsAt: asOf.Add(6 * time.Hour), EndsAt: asOf.Add(7 * time.Hour), Status: "postponed"},
		{Owner: owner, Rkey: "3msinvalid001", Name: "Invalid", StartsAt: asOf.Add(5 * time.Hour), EndsAt: asOf.Add(5 * time.Hour)},
		{
			Owner: owner, Rkey: "3msoverlong01", Name: "Independent over duration",
			StartsAt: asOf.Add(4 * time.Hour), EndsAt: asOf.Add(32*24*time.Hour + 4*time.Hour),
			RawFields: map[string]any{"independentExtension": map[string]any{"retained": true}},
		},
		{Owner: owner, Rkey: "3msmoderateaa", Name: "Moderated", StartsAt: asOf.Add(3 * time.Hour), EndsAt: asOf.Add(4 * time.Hour)},
		{Owner: owner, Rkey: "3msreportabcd", Name: "Reportable", StartsAt: asOf.Add(2 * time.Hour), EndsAt: asOf.Add(3 * time.Hour)},
	}
	for _, fixture := range fixtures {
		seedEventFixture(t, pool, fixture)
	}
	for index := 0; index < 53; index++ {
		start := asOf.Add(time.Duration(100+index) * time.Hour)
		seedEventFixture(t, pool, eventFixture{
			Owner: owner, Rkey: syntax.RecordKey(fmt.Sprintf("3msbulk%06d", index)),
			Name: fmt.Sprintf("Bulk %02d", index), StartsAt: start, EndsAt: start.Add(time.Hour),
		})
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_business_events SET cid=$1 WHERE owner_did=$2`, businessEventCID1, owner); err != nil {
		t.Fatalf("set canonical event CIDs: %v", err)
	}

	moderationInput := func(rkey syntax.RecordKey, action string) api.ModerationOutputInput {
		t.Helper()
		input, err := api.DecodeSyntheticModerationRequest(strings.NewReader(`{
			"subject":{"type":"event","did":"`+owner.String()+`","rkey":"`+rkey.String()+`"},
			"value":"hide","action":"`+action+`"
		}`), api.ModerationRequestConfig{
			DefaultSourceDID: "did:plc:labeler", TrustedSourceDIDs: []string{"did:plc:labeler"},
		})
		if err != nil {
			t.Fatalf("decode %s moderation for %s: %v", action, rkey, err)
		}
		return input
	}
	if _, err := moderation.InsertOutput(ctx, "at007-initial-hide", moderationInput("3msmoderateaa", "apply")); err != nil {
		t.Fatalf("seed moderated event: %v", err)
	}

	store := business.NewStore(pool)
	ownerList := api.GetOwnerBusinessEventsHandler(store, testEventCursorCodec(t), func() time.Time { return asOf })
	seen := make(map[syntax.ATURI]bool)
	var traversed []business.EventView
	var ownerImage any
	cursor := ""
	for pageNumber := 0; ; pageNumber++ {
		target := "/v1/events"
		if cursor != "" {
			target += "?cursor=" + cursor
		}
		response := serveAT007OwnerEvents(ownerList, owner, target)
		if response.Code != http.StatusOK {
			t.Fatalf("owner page %d status=%d body=%s", pageNumber, response.Code, response.Body.String())
		}
		var page api.BusinessEventPage
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode owner page %d: %v", pageNumber, err)
		}
		if len(page.Items) > 20 {
			t.Fatalf("owner page %d has %d items, want at most 20", pageNumber, len(page.Items))
		}
		var pageWire struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &pageWire); err != nil {
			t.Fatalf("decode owner image page %d: %v", pageNumber, err)
		}
		for _, event := range pageWire.Items {
			if event["name"] == "Future" {
				ownerImage = event["image"]
			}
		}
		for _, event := range page.Items {
			if seen[event.URI] {
				t.Fatalf("owner traversal returned duplicate %s", event.URI)
			}
			seen[event.URI] = true
			traversed = append(traversed, event)
		}
		if page.Cursor == "" {
			break
		}
		if strings.Contains(page.Cursor, "at://") || page.Cursor == traversed[len(traversed)-1].URI.String() {
			t.Fatalf("owner cursor exposes seek state: %q", page.Cursor)
		}
		cursor = page.Cursor
	}
	if len(traversed) != len(fixtures)+53 {
		t.Fatalf("owner traversal returned %d events, want %d", len(traversed), len(fixtures)+53)
	}
	wantOwnerImage := map[string]any{
		"cid":      "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq",
		"mime":     "image/png",
		"size":     float64(0),
		"alt":      "Future event",
		"thumb":    "https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:event-owner/bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq@png",
		"fullsize": "https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:event-owner/bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq@png",
		"aspectRatio": map[string]any{
			"width":  float64(3),
			"height": float64(2),
		},
	}
	if !jsonObjectsEqual(ownerImage, wantOwnerImage) {
		t.Errorf("owner event image = %#v, want exact normalized image %#v", ownerImage, wantOwnerImage)
	}
	for index := 1; index < len(traversed); index++ {
		previous, err := time.Parse(time.RFC3339Nano, traversed[index-1].StartsAt)
		if err != nil {
			t.Fatalf("parse previous startsAt: %v", err)
		}
		current, err := time.Parse(time.RFC3339Nano, traversed[index].StartsAt)
		if err != nil {
			t.Fatalf("parse current startsAt: %v", err)
		}
		if previous.Before(current) || (previous.Equal(current) && traversed[index-1].URI < traversed[index].URI) {
			t.Fatalf("owner ordering at %d: %s/%s then %s/%s", index, previous, traversed[index-1].URI, current, traversed[index].URI)
		}
	}
	capped := serveAT007OwnerEvents(ownerList, owner, "/v1/events?limit=500")
	var cappedPage api.BusinessEventPage
	if capped.Code != http.StatusOK || json.Unmarshal(capped.Body.Bytes(), &cappedPage) != nil {
		t.Fatalf("decode capped owner page: status=%d body=%s", capped.Code, capped.Body.String())
	}
	if len(cappedPage.Items) != 50 || cappedPage.Cursor == "" {
		t.Fatalf("capped owner page = %d items, cursor %q; want 50 and a cursor", len(cappedPage.Items), cappedPage.Cursor)
	}

	wantDiagnostics := map[string]struct {
		public   []string
		upcoming []string
	}{
		"Future":                    {[]string{}, []string{}},
		"Past":                      {[]string{}, []string{"ended"}},
		"Cancelled":                 {[]string{}, []string{"cancelled"}},
		"Postponed":                 {[]string{}, []string{"postponed"}},
		"Invalid":                   {[]string{"invalid-time-range"}, []string{"invalid-time-range"}},
		"Independent over duration": {[]string{"duration-exceeds-limit"}, []string{"duration-exceeds-limit"}},
		"Moderated":                 {[]string{"record-moderated"}, []string{"record-moderated"}},
		"Reportable":                {[]string{}, []string{}},
	}
	for _, event := range traversed {
		want, special := wantDiagnostics[event.Name]
		if !special {
			want = struct {
				public   []string
				upcoming []string
			}{[]string{}, []string{}}
		}
		if event.PublicSuppressionReasons == nil || event.UpcomingExclusionReasons == nil ||
			!reflect.DeepEqual(event.PublicSuppressionReasons, want.public) ||
			!reflect.DeepEqual(event.UpcomingExclusionReasons, want.upcoming) {
			t.Fatalf("%s diagnostics = public %v upcoming %v, want %v/%v", event.Name,
				event.PublicSuppressionReasons, event.UpcomingExclusionReasons, want.public, want.upcoming)
		}
	}

	overRkey := syntax.RecordKey("3msoverlong01")
	overURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.business.event/" + overRkey.String())
	var rawOverDuration map[string]any
	if err := pool.QueryRow(ctx, `SELECT raw_record FROM craftsky_business_events WHERE uri=$1`, overURI).Scan(&rawOverDuration); err != nil {
		t.Fatalf("read raw independent over-duration event: %v", err)
	}
	if !reflect.DeepEqual(rawOverDuration["independentExtension"], map[string]any{"retained": true}) {
		t.Fatalf("raw independent extension = %#v", rawOverDuration["independentExtension"])
	}
	direct := api.GetBusinessEventHandler(store, func() time.Time { return asOf })
	assertBusinessEventAcceptanceError(t, serveBusinessEventAcceptanceRead(direct, visitor, owner, overRkey), http.StatusNotFound, "event_not_found")
	upcoming := api.GetProfileBusinessEventsHandler(store, nil, testEventCursorCodec(t), func() time.Time { return asOf })
	upcomingResponse := serveBusinessEventAcceptanceUpcoming(upcoming, visitor, owner)
	var upcomingPage api.BusinessEventPage
	if upcomingResponse.Code != http.StatusOK || json.Unmarshal(upcomingResponse.Body.Bytes(), &upcomingPage) != nil {
		t.Fatalf("decode visitor upcoming page: status=%d body=%s", upcomingResponse.Code, upcomingResponse.Body.String())
	}
	for _, event := range upcomingPage.Items {
		if event.URI == overURI {
			t.Fatalf("visitor upcoming list exposed over-duration event %s", overURI)
		}
	}

	reportRkey := syntax.RecordKey("3msreportabcd")
	reportURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.business.event/" + reportRkey.String())
	visibleBeforeHide := decodeBusinessEventAcceptanceEvent(t, serveBusinessEventAcceptanceRead(direct, visitor, owner, reportRkey), http.StatusOK)
	if visibleBeforeHide.URI != reportURI {
		t.Fatalf("visible report target URI = %s, want %s", visibleBeforeHide.URI, reportURI)
	}
	reportHandler := api.ReportBusinessEventHandler(store, api.NewReportStore(pool),
		api.NewPlaceholderReportForwarder(func() time.Time { return asOf }), nilLogger(), func() time.Time { return asOf })
	reportRequest := httptest.NewRequest(http.MethodPost, "/v1/events/"+owner.String()+"/"+reportRkey.String()+"/reports", strings.NewReader(`{"reasonType":"spam"}`))
	reportRequest.SetPathValue("did", owner.String())
	reportRequest.SetPathValue("rkey", reportRkey.String())
	reportContext := middleware.WithDID(reportRequest.Context(), visitor)
	reportContext = middleware.WithDeviceID(reportContext, "at007-device")
	reportContext = ownerlifecycle.WithExpectedGeneration(reportContext, 2)
	reportRequest = reportRequest.WithContext(reportContext)
	reportResponse := httptest.NewRecorder()
	reportHandler.ServeHTTP(reportResponse, reportRequest)
	if reportResponse.Code != http.StatusCreated {
		t.Fatalf("report event status=%d body=%s", reportResponse.Code, reportResponse.Body.String())
	}
	var storedReportURI string
	if err := pool.QueryRow(ctx, `SELECT subject_uri FROM moderation_reports`).Scan(&storedReportURI); err != nil || storedReportURI != reportURI.String() {
		t.Fatalf("stored report URI=%q error=%v, want %s", storedReportURI, err, reportURI)
	}

	if _, err := moderation.InsertOutput(ctx, "at007-reportable-hide", moderationInput(reportRkey, "apply")); err != nil {
		t.Fatalf("hide reportable event: %v", err)
	}
	assertBusinessEventAcceptanceError(t, serveBusinessEventAcceptanceRead(direct, visitor, owner, reportRkey), http.StatusNotFound, "event_not_found")
	if at007UpcomingContains(t, upcoming, visitor, owner, reportURI) {
		t.Fatalf("visitor upcoming list exposed hidden event %s", reportURI)
	}
	hiddenOwner := decodeBusinessEventAcceptanceEvent(t, serveBusinessEventAcceptanceRead(direct, owner, owner, reportRkey), http.StatusOK)
	if !reflect.DeepEqual(hiddenOwner.PublicSuppressionReasons, []string{"record-moderated"}) ||
		!reflect.DeepEqual(hiddenOwner.UpcomingExclusionReasons, []string{"record-moderated"}) {
		t.Fatalf("hidden owner diagnostics = %v/%v", hiddenOwner.PublicSuppressionReasons, hiddenOwner.UpcomingExclusionReasons)
	}

	if _, err := moderation.InsertOutput(ctx, "at007-reportable-negate", moderationInput(reportRkey, "negate")); err != nil {
		t.Fatalf("negate reportable event hide: %v", err)
	}
	restoredVisitor := decodeBusinessEventAcceptanceEvent(t, serveBusinessEventAcceptanceRead(direct, visitor, owner, reportRkey), http.StatusOK)
	restoredOwner := decodeBusinessEventAcceptanceEvent(t, serveBusinessEventAcceptanceRead(direct, owner, owner, reportRkey), http.StatusOK)
	if restoredVisitor.URI != reportURI || restoredOwner.PublicSuppressionReasons == nil || restoredOwner.UpcomingExclusionReasons == nil ||
		len(restoredOwner.PublicSuppressionReasons) != 0 || len(restoredOwner.UpcomingExclusionReasons) != 0 ||
		!at007UpcomingContains(t, upcoming, visitor, owner, reportURI) {
		t.Fatalf("negated visibility/diagnostics = visitor %s owner %v/%v", restoredVisitor.URI,
			restoredOwner.PublicSuppressionReasons, restoredOwner.UpcomingExclusionReasons)
	}
}

func serveAT007OwnerEvents(handler http.Handler, owner syntax.DID, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request = request.WithContext(middleware.WithDID(request.Context(), owner))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func at007UpcomingContains(t *testing.T, handler http.Handler, caller, owner syntax.DID, uri syntax.ATURI) bool {
	t.Helper()
	response := serveBusinessEventAcceptanceUpcoming(handler, caller, owner)
	if response.Code != http.StatusOK {
		t.Fatalf("upcoming status=%d body=%s", response.Code, response.Body.String())
	}
	var page api.BusinessEventPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode upcoming page: %v", err)
	}
	for _, event := range page.Items {
		if event.URI == uri {
			return true
		}
	}
	return false
}
