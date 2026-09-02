package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessEventOwnerManagementListsEveryRetainedState(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:event-owner")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed owner account type: %v", err)
	}

	fixtures := []eventFixture{
		{Owner: owner, Rkey: "3msfuture0001", Name: "Future", StartsAt: asOf.Add(8 * time.Hour), EndsAt: asOf.Add(9 * time.Hour)},
		{Owner: owner, Rkey: "3mscancel0001", Name: "Cancelled", StartsAt: asOf.Add(7 * time.Hour), EndsAt: asOf.Add(8 * time.Hour), Status: "cancelled"},
		{Owner: owner, Rkey: "3mspostpone01", Name: "Postponed", StartsAt: asOf.Add(6 * time.Hour), EndsAt: asOf.Add(7 * time.Hour), Status: "postponed"},
		{Owner: owner, Rkey: "3msinvalid001", Name: "Invalid", StartsAt: asOf.Add(5 * time.Hour), EndsAt: asOf.Add(5 * time.Hour)},
		{Owner: owner, Rkey: "3msoverlong01", Name: "Over duration", StartsAt: asOf.Add(4 * time.Hour), EndsAt: asOf.Add(32*24*time.Hour + 4*time.Hour)},
		{Owner: owner, Rkey: "3msmoderate01", Name: "Moderated", StartsAt: asOf.Add(3 * time.Hour), EndsAt: asOf.Add(4 * time.Hour)},
		{Owner: owner, Rkey: "3mspast000001", Name: "Past", StartsAt: asOf.Add(-2 * time.Hour), EndsAt: asOf.Add(-time.Hour)},
	}
	for _, fixture := range fixtures {
		seedEventFixture(t, pool, fixture)
	}
	moderatedURI := "at://did:plc:event-owner/social.craftsky.business.event/3msmoderate01"
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(id, source_did, subject_type, subject_did, subject_uri, value, action)
		VALUES ('event-hide', 'did:plc:labeler', 'event', $1, $2, 'hide', 'apply')
	`, owner, moderatedURI); err != nil {
		t.Fatalf("seed event moderation: %v", err)
	}

	store := business.NewStore(pool)
	items, err := store.ListOwnerEvents(ctx, business.OwnerEventListInput{OwnerDID: owner, AsOf: asOf})
	if err != nil {
		t.Fatalf("ListOwnerEvents default: %v", err)
	}
	if got, want := eventNames(items), []string{"Future", "Cancelled", "Postponed", "Invalid", "Over duration", "Moderated", "Past"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("default owner events = %v, want %v", got, want)
	}
	assertOwnerEventDiagnostics(t, items)

	for index := 0; index < 55; index++ {
		start := asOf.Add(time.Duration(100+index) * time.Hour)
		seedEventFixture(t, pool, eventFixture{
			Owner: owner, Rkey: syntax.RecordKey("3msbulk" + strconv.Itoa(10000+index)),
			Name: "Bulk " + strconv.Itoa(index), StartsAt: start, EndsAt: start.Add(time.Hour),
		})
	}
	tieStart := asOf.Add(1000 * time.Hour)
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3mstie000001", Name: "Tie low URI", StartsAt: tieStart, EndsAt: tieStart.Add(time.Hour)})
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3mstie000002", Name: "Tie high URI", StartsAt: tieStart, EndsAt: tieStart.Add(time.Hour)})
	defaultPage, err := store.ListOwnerEvents(ctx, business.OwnerEventListInput{OwnerDID: owner, AsOf: asOf})
	if err != nil {
		t.Fatalf("ListOwnerEvents populated default: %v", err)
	}
	if len(defaultPage) != 21 {
		t.Fatalf("default fetch length = %d, want 21 for a 20-item page plus lookahead", len(defaultPage))
	}
	if defaultPage[0].Name != "Tie high URI" || defaultPage[1].Name != "Tie low URI" {
		t.Fatalf("equal-start URI order = %q then %q", defaultPage[0].Name, defaultPage[1].Name)
	}
	capped, err := store.ListOwnerEvents(ctx, business.OwnerEventListInput{OwnerDID: owner, AsOf: asOf, Limit: 500})
	if err != nil {
		t.Fatalf("ListOwnerEvents cap: %v", err)
	}
	if len(capped) != 51 {
		t.Fatalf("capped fetch length = %d, want 51 for a 50-item page plus lookahead", len(capped))
	}
	for index := 1; index < len(capped); index++ {
		previous, _ := time.Parse(time.RFC3339Nano, capped[index-1].StartsAt)
		current, _ := time.Parse(time.RFC3339Nano, capped[index].StartsAt)
		if previous.Before(current) || (previous.Equal(current) && capped[index-1].URI < capped[index].URI) {
			t.Fatalf("events not descending at %d: %s/%s then %s/%s", index, previous, capped[index-1].URI, current, capped[index].URI)
		}
	}
}

func TestGetOwnerBusinessEventsHandlerPaginatesWithDefaultAndOpaqueCursor(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:event-owner")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed regular owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'regular')`, owner); err != nil {
		t.Fatalf("seed regular owner account type: %v", err)
	}
	for index := 0; index < 23; index++ {
		start := asOf.Add(time.Duration(index) * time.Hour)
		seedEventFixture(t, pool, eventFixture{
			Owner: owner, Rkey: syntax.RecordKey("3mspage" + strconv.Itoa(10000+index)),
			Name: "Page " + strconv.Itoa(index), StartsAt: start, EndsAt: start.Add(time.Hour),
		})
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_business_events SET cid='bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq' WHERE owner_did=$1`, owner); err != nil {
		t.Fatalf("set canonical event CIDs: %v", err)
	}

	handler := api.GetOwnerBusinessEventsHandler(business.NewStore(pool), testEventCursorCodec(t), func() time.Time { return asOf })
	seen := map[syntax.ATURI]bool{}
	cursor := ""
	for pageNumber := 0; ; pageNumber++ {
		path := "/v1/events"
		if cursor != "" {
			path += "?cursor=" + cursor
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(middleware.WithDID(req.Context(), owner))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("page %d status = %d, body = %s", pageNumber, rr.Code, rr.Body.String())
		}
		var page api.BusinessEventPage
		if err := json.NewDecoder(rr.Body).Decode(&page); err != nil {
			t.Fatalf("decode page %d: %v", pageNumber, err)
		}
		if pageNumber == 0 && len(page.Items) != 20 {
			t.Fatalf("default page length = %d, want 20", len(page.Items))
		}
		for _, item := range page.Items {
			if !reflect.DeepEqual(item.PublicSuppressionReasons, []string{"owner-not-business"}) ||
				!reflect.DeepEqual(item.UpcomingExclusionReasons, []string{"owner-not-business"}) {
				t.Fatalf("regular owner diagnostics = public %v upcoming %v", item.PublicSuppressionReasons, item.UpcomingExclusionReasons)
			}
			if seen[item.URI] {
				t.Fatalf("duplicate event %s", item.URI)
			}
			seen[item.URI] = true
		}
		if page.Cursor == "" {
			break
		}
		if page.Cursor == cursor {
			t.Fatal("cursor did not advance")
		}
		cursor = page.Cursor
	}
	if len(seen) != 23 {
		t.Fatalf("traversed %d events, want 23", len(seen))
	}
}

func eventNames(items []business.EventView) []string {
	names := make([]string, len(items))
	for index := range items {
		names[index] = items[index].Name
	}
	return names
}

func assertOwnerEventDiagnostics(t *testing.T, items []business.EventView) {
	t.Helper()
	wantPublic := map[string][]string{
		"Future": {}, "Cancelled": {}, "Postponed": {}, "Past": {},
		"Invalid": {"invalid-time-range"}, "Over duration": {"duration-exceeds-limit"}, "Moderated": {"record-moderated"},
	}
	wantUpcoming := map[string][]string{
		"Future": {}, "Cancelled": {"cancelled"}, "Postponed": {"postponed"}, "Past": {"ended"},
		"Invalid": {"invalid-time-range"}, "Over duration": {"duration-exceeds-limit"}, "Moderated": {"record-moderated"},
	}
	for _, item := range items {
		if item.PublicSuppressionReasons == nil || item.UpcomingExclusionReasons == nil {
			t.Fatalf("%s diagnostics contain nil array: %+v", item.Name, item)
		}
		if !reflect.DeepEqual(item.PublicSuppressionReasons, wantPublic[item.Name]) ||
			!reflect.DeepEqual(item.UpcomingExclusionReasons, wantUpcoming[item.Name]) {
			t.Fatalf("%s diagnostics = public %v upcoming %v", item.Name, item.PublicSuppressionReasons, item.UpcomingExclusionReasons)
		}
	}
}
