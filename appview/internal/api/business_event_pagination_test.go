package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

func TestAT006PaginateUpcomingEventsWithFrozenTimeEligibility(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	owner := syntax.DID("did:plc:at006eventowner")
	visitor := syntax.DID("did:plc:at006eventvisitor")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	clock := asOf
	seedEligibleBusinessOwner(t, pool, owner)

	type expectedEvent struct {
		startsAt time.Time
		uri      string
	}
	want := make([]expectedEvent, 0, 56)
	var expiresAfterFirstPage syntax.ATURI
	for i := 55; i >= 0; i-- {
		startsAt := asOf.Add(time.Duration(1+(i-12)/8) * time.Hour)
		endsAt := startsAt.Add(time.Hour)
		if i < 12 {
			startsAt = asOf.Add(-time.Hour)
			endsAt = asOf.Add(24 * time.Hour)
		}
		if i == 11 {
			endsAt = asOf.Add(30 * time.Minute)
		}
		uri := seedEventFixture(t, pool, eventFixture{
			Owner: owner, Rkey: syntax.RecordKey(fmt.Sprintf("3msat006%03d", i)),
			Name: fmt.Sprintf("AT-006 Event %03d", i), StartsAt: startsAt, EndsAt: endsAt,
		})
		if i == 11 {
			expiresAfterFirstPage = uri
		}
		want = append(want, expectedEvent{startsAt: startsAt, uri: uri.String()})
	}
	sort.Slice(want, func(i, j int) bool {
		if want[i].startsAt.Equal(want[j].startsAt) {
			return want[i].uri < want[j].uri
		}
		return want[i].startsAt.Before(want[j].startsAt)
	})

	handler := api.GetProfileBusinessEventsHandler(
		business.NewStore(pool),
		fakeResolver{didFor: owner},
		testEventCursorCodec(t),
		func() time.Time { return clock },
	)

	var got []string
	cursor := ""
	for pageNumber := 0; ; pageNumber++ {
		path := "/v1/profiles/@events.example/events"
		if cursor != "" {
			path += "?cursor=" + url.QueryEscape(cursor)
		}
		page := serveBusinessEventPage(t, handler, visitor, "@events.example", path)
		if pageNumber == 0 {
			if len(page.Items) != 10 {
				t.Fatalf("default page size = %d, want 10", len(page.Items))
			}
			clock = asOf.Add(time.Hour)
		}
		for _, event := range page.Items {
			got = append(got, event.URI)
		}
		if page.Cursor == "" {
			break
		}
		if strings.Contains(page.Cursor, "at://") || strings.Contains(page.Cursor, asOf.Format(time.RFC3339)) {
			t.Fatalf("cursor exposes pagination state: %q", page.Cursor)
		}
		cursor = page.Cursor
		if pageNumber > 10 {
			t.Fatal("cursor traversal did not terminate")
		}
	}

	wantURIs := make([]string, len(want))
	for i, event := range want {
		wantURIs[i] = event.uri
	}
	if !reflect.DeepEqual(got, wantURIs) {
		t.Fatalf("startsAt/URI traversal = %v, want %v", got, wantURIs)
	}
	if !containsString(got, expiresAfterFirstPage.String()) {
		t.Fatalf("traversal omitted %s after clock advanced beyond its endsAt", expiresAfterFirstPage)
	}

	capped := serveBusinessEventPage(t, handler, visitor, owner.String(), "/v1/profiles/"+owner.String()+"/events?limit=999")
	if len(capped.Items) != 50 || capped.Cursor == "" {
		t.Fatalf("capped page = %d items cursor %q, want 50 items and cursor", len(capped.Items), capped.Cursor)
	}
}

func TestOwnerBusinessEventsUpcomingTraversalIsCompleteAtFrozenCutoff(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	owner := syntax.DID("did:plc:ownerupcomingtraversal")
	cutoff := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	clock := cutoff
	seedEligibleBusinessOwner(t, pool, owner)

	fixtures := []eventFixture{
		{Owner: owner, Rkey: "3msupcoming000", Name: "Ongoing", StartsAt: cutoff.Add(-time.Hour), EndsAt: cutoff.Add(30 * time.Minute)},
		{Owner: owner, Rkey: "3msupcoming001", Name: "Tie low URI", StartsAt: cutoff.Add(time.Hour), EndsAt: cutoff.Add(2 * time.Hour)},
		{Owner: owner, Rkey: "3msupcoming002", Name: "Tie high URI", StartsAt: cutoff.Add(time.Hour), EndsAt: cutoff.Add(2 * time.Hour)},
		{Owner: owner, Rkey: "3msupcoming003", Name: "Third start", StartsAt: cutoff.Add(2 * time.Hour), EndsAt: cutoff.Add(3 * time.Hour)},
		{Owner: owner, Rkey: "3msupcoming004", Name: "Fourth start", StartsAt: cutoff.Add(3 * time.Hour), EndsAt: cutoff.Add(4 * time.Hour)},
		{Owner: owner, Rkey: "3msupcoming005", Name: "Fifth start", StartsAt: cutoff.Add(4 * time.Hour), EndsAt: cutoff.Add(5 * time.Hour)},
		{Owner: owner, Rkey: "3msupcoming006", Name: "Sixth start", StartsAt: cutoff.Add(5 * time.Hour), EndsAt: cutoff.Add(6 * time.Hour)},
	}
	want := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		want = append(want, seedEventFixture(t, pool, fixture).String())
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_business_events SET cid=$1 WHERE owner_did=$2`, businessEventCID1, owner); err != nil {
		t.Fatalf("set canonical event CIDs: %v", err)
	}

	handler := api.GetOwnerBusinessEventsHandler(business.NewStore(pool), testEventCursorCodec(t), func() time.Time { return clock })
	limits := []int{2, 1, 4}
	wantPages := [][]string{
		want[:2],
		want[2:3],
		want[3:],
	}
	var got []string
	seen := make(map[string]bool, len(want))
	cursor := ""
	for pageNumber, limit := range limits {
		target := fmt.Sprintf("/v1/events?filter=upcoming&limit=%d", limit)
		if cursor != "" {
			target += "&cursor=" + url.QueryEscape(cursor)
		}
		response := serveAT007OwnerEvents(handler, owner, target)
		if response.Code != http.StatusOK {
			t.Fatalf("upcoming page %d status=%d body=%s", pageNumber, response.Code, response.Body.String())
		}
		var page api.BusinessEventPage
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode upcoming page %d: %v", pageNumber, err)
		}
		pageURIs := make([]string, len(page.Items))
		for index, event := range page.Items {
			uri := event.URI.String()
			pageURIs[index] = uri
			if seen[uri] {
				t.Fatalf("upcoming page %d duplicated URI %s", pageNumber, uri)
			}
			seen[uri] = true
			got = append(got, uri)
		}
		if !reflect.DeepEqual(pageURIs, wantPages[pageNumber]) {
			t.Fatalf("upcoming page %d URIs = %v, want %v", pageNumber, pageURIs, wantPages[pageNumber])
		}
		if pageNumber == 0 {
			clock = cutoff.Add(time.Hour)
		}
		if pageNumber < len(limits)-1 && page.Cursor == "" {
			t.Fatalf("upcoming page %d omitted continuation cursor", pageNumber)
		}
		if pageNumber == len(limits)-1 && page.Cursor != "" {
			t.Fatalf("final upcoming page cursor=%q, want empty", page.Cursor)
		}
		cursor = page.Cursor
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upcoming traversal URIs = %v, want exact order %v", got, want)
	}
}

func TestOwnerBusinessEventsHistoryTraversalIsCompleteAtIndependentFrozenCutoff(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	owner := syntax.DID("did:plc:ownerhistorytraversal")
	historyCutoff := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	clock := historyCutoff.Add(-2 * time.Hour)
	seedEligibleBusinessOwner(t, pool, owner)

	fixtures := []eventFixture{
		{Owner: owner, Rkey: "3mshistory000", Name: "Oldest ended", StartsAt: historyCutoff.Add(-time.Hour), EndsAt: historyCutoff.Add(-30 * time.Minute)},
		{Owner: owner, Rkey: "3mshistory001", Name: "Recently ended", StartsAt: historyCutoff.Add(time.Hour), EndsAt: historyCutoff.Add(-time.Minute)},
		{Owner: owner, Rkey: "3mshistory002", Name: "Ended at cutoff", StartsAt: historyCutoff.Add(2 * time.Hour), EndsAt: historyCutoff},
		{Owner: owner, Rkey: "3mshistory003", Name: "Tie low URI", StartsAt: historyCutoff.Add(4 * time.Hour), EndsAt: historyCutoff.Add(5 * time.Hour), Status: "postponed"},
		{Owner: owner, Rkey: "3mshistory004", Name: "Tie high URI", StartsAt: historyCutoff.Add(4 * time.Hour), EndsAt: historyCutoff.Add(5 * time.Hour), Status: "postponed"},
		{Owner: owner, Rkey: "3mshistory005", Name: "Cancelled", StartsAt: historyCutoff.Add(5 * time.Hour), EndsAt: historyCutoff.Add(6 * time.Hour), Status: "cancelled"},
		{Owner: owner, Rkey: "3mshistory006", Name: "Unknown", StartsAt: historyCutoff.Add(6 * time.Hour), EndsAt: historyCutoff.Add(7 * time.Hour), Status: "rescheduled-externally"},
		{Owner: owner, Rkey: "3mshistory007", Name: "Crosses later clock only", StartsAt: historyCutoff.Add(3 * time.Hour), EndsAt: historyCutoff.Add(time.Hour)},
	}
	seeded := make(map[syntax.RecordKey]string, len(fixtures))
	for _, fixture := range fixtures {
		seeded[fixture.Rkey] = seedEventFixture(t, pool, fixture).String()
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_business_events SET cid=$1 WHERE owner_did=$2`, businessEventCID1, owner); err != nil {
		t.Fatalf("set canonical event CIDs: %v", err)
	}
	want := []string{
		seeded["3mshistory006"],
		seeded["3mshistory005"],
		seeded["3mshistory004"],
		seeded["3mshistory003"],
		seeded["3mshistory002"],
		seeded["3mshistory001"],
		seeded["3mshistory000"],
	}

	codec := testEventCursorCodec(t)
	handler := api.GetOwnerBusinessEventsHandler(business.NewStore(pool), codec, func() time.Time { return clock })
	upcomingResponse := serveAT007OwnerEvents(handler, owner, "/v1/events?filter=upcoming&limit=1")
	var upcomingPage api.BusinessEventPage
	if upcomingResponse.Code != http.StatusOK || json.Unmarshal(upcomingResponse.Body.Bytes(), &upcomingPage) != nil || upcomingPage.Cursor == "" {
		t.Fatalf("earlier upcoming page status=%d cursor=%q body=%s", upcomingResponse.Code, upcomingPage.Cursor, upcomingResponse.Body.String())
	}
	upcomingCursor, err := codec.Decode(upcomingPage.Cursor, api.EventCursorOwnerUpcoming, owner)
	if err != nil || !upcomingCursor.AsOf.Equal(clock) {
		t.Fatalf("upcoming cursor cutoff=%s error=%v, want %s", upcomingCursor.AsOf, err, clock)
	}

	clock = historyCutoff
	limits := []int{3, 1, 3}
	wantPages := [][]string{
		want[:3],
		want[3:4],
		want[4:],
	}
	var got []string
	seen := make(map[string]bool, len(want))
	cursor := ""
	for pageNumber, limit := range limits {
		target := fmt.Sprintf("/v1/events?filter=history&limit=%d", limit)
		if cursor != "" {
			target += "&cursor=" + url.QueryEscape(cursor)
		}
		response := serveAT007OwnerEvents(handler, owner, target)
		if response.Code != http.StatusOK {
			t.Fatalf("history page %d status=%d body=%s", pageNumber, response.Code, response.Body.String())
		}
		var page api.BusinessEventPage
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode history page %d: %v", pageNumber, err)
		}
		pageURIs := make([]string, len(page.Items))
		for index, event := range page.Items {
			uri := event.URI.String()
			pageURIs[index] = uri
			if seen[uri] {
				t.Fatalf("history page %d duplicated URI %s", pageNumber, uri)
			}
			seen[uri] = true
			got = append(got, uri)
		}
		if !reflect.DeepEqual(pageURIs, wantPages[pageNumber]) {
			t.Fatalf("history page %d URIs = %v, want %v", pageNumber, pageURIs, wantPages[pageNumber])
		}
		if pageNumber == 0 {
			historyCursor, err := codec.Decode(page.Cursor, api.EventCursorOwnerHistory, owner)
			if err != nil || !historyCursor.AsOf.Equal(historyCutoff) || historyCursor.AsOf.Equal(upcomingCursor.AsOf) {
				t.Fatalf("history cursor cutoff=%s error=%v, want independent %s", historyCursor.AsOf, err, historyCutoff)
			}
			clock = historyCutoff.Add(2 * time.Hour)
		}
		if pageNumber < len(limits)-1 && page.Cursor == "" {
			t.Fatalf("history page %d omitted continuation cursor", pageNumber)
		}
		if pageNumber == len(limits)-1 && page.Cursor != "" {
			t.Fatalf("final history page cursor=%q, want empty", page.Cursor)
		}
		cursor = page.Cursor
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("history traversal URIs = %v, want exact order %v", got, want)
	}
	if containsString(got, seeded["3mshistory007"]) {
		t.Fatalf("history traversal admitted event that crossed the cutoff only after page one: %v", got)
	}
}

func TestOwnerBusinessEventsUpcomingFilter(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	owner := syntax.DID("did:plc:ownerfilter")
	cutoff := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	seedEligibleBusinessOwner(t, pool, owner)

	fixtures := []eventFixture{
		{Owner: owner, Rkey: "3msongoing001", Name: "Ongoing", StartsAt: cutoff.Add(-time.Hour), EndsAt: cutoff.Add(time.Hour)},
		{Owner: owner, Rkey: "3msinvalid001", Name: "Suppressed invalid range", StartsAt: cutoff.Add(3 * time.Hour), EndsAt: cutoff.Add(2 * time.Hour)},
		{Owner: owner, Rkey: "3msfuture0001", Name: "Future", StartsAt: cutoff.Add(4 * time.Hour), EndsAt: cutoff.Add(5 * time.Hour)},
		{Owner: owner, Rkey: "3msended00001", Name: "Ended", StartsAt: cutoff.Add(-2 * time.Hour), EndsAt: cutoff},
		{Owner: owner, Rkey: "3mscancel0001", Name: "Cancelled", StartsAt: cutoff.Add(6 * time.Hour), EndsAt: cutoff.Add(7 * time.Hour), Status: "cancelled"},
		{Owner: owner, Rkey: "3msunknown001", Name: "Unknown", StartsAt: cutoff.Add(8 * time.Hour), EndsAt: cutoff.Add(9 * time.Hour), Status: "rescheduled-externally"},
	}
	for _, fixture := range fixtures {
		seedEventFixture(t, pool, fixture)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_business_events SET cid=$1 WHERE owner_did=$2`, businessEventCID1, owner); err != nil {
		t.Fatalf("set canonical event CIDs: %v", err)
	}

	clock := cutoff
	handler := api.GetOwnerBusinessEventsHandler(business.NewStore(pool), testEventCursorCodec(t), func() time.Time { return clock })
	response := serveAT007OwnerEvents(handler, owner, "/v1/events?filter=upcoming")
	if response.Code != http.StatusOK {
		t.Fatalf("upcoming filter status=%d body=%s", response.Code, response.Body.String())
	}
	var page api.BusinessEventPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode upcoming owner page: %v", err)
	}
	got := make([]string, len(page.Items))
	for index, event := range page.Items {
		got[index] = event.Name
	}
	want := []string{"Ongoing", "Suppressed invalid range", "Future"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upcoming owner events = %v, want %v", got, want)
	}

	response = serveAT007OwnerEvents(handler, owner, "/v1/events?filter=history")
	if response.Code != http.StatusOK {
		t.Fatalf("history filter status=%d body=%s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode history owner page: %v", err)
	}
	got = make([]string, len(page.Items))
	for index, event := range page.Items {
		got[index] = event.Name
	}
	want = []string{"Unknown", "Cancelled", "Ended"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("history owner events = %v, want %v", got, want)
	}

	first := serveAT007OwnerEvents(handler, owner, "/v1/events?filter=upcoming&limit=1")
	if first.Code != http.StatusOK {
		t.Fatalf("first filtered page status=%d body=%s", first.Code, first.Body.String())
	}
	if err := json.Unmarshal(first.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode first filtered page: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "Ongoing" || page.Cursor == "" {
		t.Fatalf("first filtered page = %+v", page)
	}
	clock = cutoff.Add(2 * time.Hour)
	second := serveAT007OwnerEvents(handler, owner, "/v1/events?filter=upcoming&limit=2&cursor="+url.QueryEscape(page.Cursor))
	if second.Code != http.StatusOK {
		t.Fatalf("second filtered page status=%d body=%s", second.Code, second.Body.String())
	}
	if err := json.Unmarshal(second.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode second filtered page: %v", err)
	}
	got = make([]string, len(page.Items))
	for index, event := range page.Items {
		got[index] = event.Name
	}
	want = []string{"Suppressed invalid range", "Future"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen filtered continuation = %v, want %v", got, want)
	}

	for _, target := range []string{
		"/v1/events?filter=unknown",
		"/v1/events?filter=",
		"/v1/events?filter=upcoming&filter=history",
	} {
		invalid := serveAT007OwnerEvents(handler, owner, target)
		if invalid.Code != http.StatusBadRequest {
			t.Errorf("GET %s status=%d body=%s, want 400 invalid_filter", target, invalid.Code, invalid.Body.String())
			continue
		}
		var body envelope.Error
		if err := json.Unmarshal(invalid.Body.Bytes(), &body); err != nil || body.Error != "invalid_filter" {
			t.Errorf("GET %s error=%+v decode=%v, want invalid_filter", target, body, err)
		}
	}

	clock = cutoff
	firstCursor := func(target string) string {
		t.Helper()
		response := serveAT007OwnerEvents(handler, owner, target)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", target, response.Code, response.Body.String())
		}
		var cursorPage api.BusinessEventPage
		if err := json.Unmarshal(response.Body.Bytes(), &cursorPage); err != nil || cursorPage.Cursor == "" {
			t.Fatalf("GET %s cursor=%q decode=%v", target, cursorPage.Cursor, err)
		}
		return cursorPage.Cursor
	}
	unfilteredCursor := firstCursor("/v1/events?limit=1")
	upcomingCursor := firstCursor("/v1/events?filter=upcoming&limit=1")
	historyCursor := firstCursor("/v1/events?filter=history&limit=1")
	for _, target := range []string{
		"/v1/events?cursor=malformed",
		"/v1/events?filter=upcoming&cursor=" + url.QueryEscape(unfilteredCursor),
		"/v1/events?cursor=" + url.QueryEscape(upcomingCursor),
		"/v1/events?filter=history&cursor=" + url.QueryEscape(upcomingCursor),
		"/v1/events?filter=upcoming&cursor=" + url.QueryEscape(historyCursor),
	} {
		invalid := serveAT007OwnerEvents(handler, owner, target)
		if invalid.Code != http.StatusBadRequest {
			t.Errorf("GET %s status=%d body=%s, want 400 invalid_cursor", target, invalid.Code, invalid.Body.String())
			continue
		}
		var body envelope.Error
		if err := json.Unmarshal(invalid.Body.Bytes(), &body); err != nil || body.Error != "invalid_cursor" {
			t.Errorf("GET %s error=%+v decode=%v, want invalid_cursor", target, body, err)
		}
	}
	ignored := serveAT007OwnerEvents(handler, owner, "/v1/events?filter=history&unknown=ignored")
	if ignored.Code != http.StatusOK {
		t.Errorf("unknown query parameter status=%d body=%s, want ignored", ignored.Code, ignored.Body.String())
	}
}

func TestBusinessEventPaginationRealQuery(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	ctx := context.Background()
	visitor := syntax.DID("did:plc:eventpagevisitor")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	clock := asOf
	store := business.NewStore(pool)

	owner := syntax.DID("did:plc:eventpageowner")
	seedEligibleBusinessOwner(t, pool, owner)
	equalStart := asOf.Add(24*time.Hour + 123456789*time.Nanosecond)
	wantURIs := make([]string, 0, 55)
	for i := 54; i >= 0; i-- {
		rkey := syntax.RecordKey(fmt.Sprintf("3mspage%03d", i))
		uri := seedEventFixture(t, pool, eventFixture{
			Owner: owner, Rkey: rkey, Name: fmt.Sprintf("Event %03d", i),
			StartsAt: equalStart, EndsAt: equalStart.Add(time.Hour),
			RawFields: map[string]any{"startsAt": equalStart.Format(time.RFC3339Nano)},
		})
		wantURIs = append(wantURIs, uri.String())
	}
	sort.Strings(wantURIs)

	handler := api.GetProfileBusinessEventsHandler(
		store,
		fakeResolver{didFor: owner},
		testEventCursorCodec(t),
		func() time.Time { return clock },
	)

	t.Run("unchanged traversal is complete and stable", func(t *testing.T) {
		var gotURIs []string
		cursor := ""
		for pageNumber := 0; ; pageNumber++ {
			if pageNumber > 10 {
				t.Fatal("seek traversal did not terminate")
			}
			path := "/v1/profiles/@events.example/events"
			if cursor != "" {
				path += "?cursor=" + url.QueryEscape(cursor)
			}
			page := serveBusinessEventPage(t, handler, visitor, "@events.example", path)
			if pageNumber == 0 && len(page.Items) != 10 {
				t.Fatalf("default first page size = %d, want 10", len(page.Items))
			}
			for _, event := range page.Items {
				gotURIs = append(gotURIs, event.URI)
			}
			if page.Cursor == "" {
				break
			}
			if strings.Contains(page.Cursor, "at://") {
				t.Fatalf("cursor exposes ordering key: %q", page.Cursor)
			}
			cursor = page.Cursor
		}
		if !reflect.DeepEqual(gotURIs, wantURIs) {
			t.Fatalf("traversal URIs = %v, want %v", gotURIs, wantURIs)
		}
	})

	t.Run("limit is capped at fifty", func(t *testing.T) {
		page := serveBusinessEventPage(t, handler, visitor, owner.String(), "/v1/profiles/"+owner.String()+"/events?limit=51")
		if len(page.Items) != 50 || page.Cursor == "" {
			t.Fatalf("max page = %d items cursor %q, want 50 items and cursor", len(page.Items), page.Cursor)
		}
	})

	t.Run("first page freezes time eligibility", func(t *testing.T) {
		frozenOwner := syntax.DID("did:plc:eventfrozenowner")
		seedEligibleBusinessOwner(t, pool, frozenOwner)
		for i := 0; i < 12; i++ {
			seedEventFixture(t, pool, eventFixture{
				Owner: frozenOwner, Rkey: syntax.RecordKey(fmt.Sprintf("3msfrozen%03d", i)),
				Name: fmt.Sprintf("Long event %03d", i), StartsAt: asOf.Add(-2 * time.Hour), EndsAt: asOf.Add(24 * time.Hour),
			})
		}
		soonURI := seedEventFixture(t, pool, eventFixture{
			Owner: frozenOwner, Rkey: "3msfrozenso0n", Name: "Ends after first page cutoff",
			StartsAt: asOf.Add(-time.Hour), EndsAt: asOf.Add(30 * time.Minute),
		})

		clock = asOf
		first := serveBusinessEventPage(t, handler, visitor, frozenOwner.String(), "/v1/profiles/"+frozenOwner.String()+"/events")
		if len(first.Items) != 10 || first.Cursor == "" {
			t.Fatalf("frozen first page = %+v", first)
		}
		clock = asOf.Add(time.Hour)
		second := serveBusinessEventPage(t, handler, visitor, frozenOwner.String(), "/v1/profiles/"+frozenOwner.String()+"/events?cursor="+url.QueryEscape(first.Cursor))
		if !eventPageContainsURI(second, soonURI) {
			t.Fatalf("second page at changed clock omitted event eligible at first asOf: %+v", second.Items)
		}
	})

	t.Run("mutation follows seek rather than snapshot semantics", func(t *testing.T) {
		mutationOwner := syntax.DID("did:plc:eventmutationowner")
		seedEligibleBusinessOwner(t, pool, mutationOwner)
		start := asOf.Add(48 * time.Hour)
		for _, rkey := range []syntax.RecordKey{"3msmutation100", "3msmutation200", "3msmutation300", "3msmutation400"} {
			seedEventFixture(t, pool, eventFixture{Owner: mutationOwner, Rkey: rkey, Name: rkey.String(), StartsAt: start, EndsAt: start.Add(time.Hour)})
		}

		clock = asOf
		first := serveBusinessEventPage(t, handler, visitor, mutationOwner.String(), "/v1/profiles/"+mutationOwner.String()+"/events?limit=2")
		if got := eventPageRkeys(first); !reflect.DeepEqual(got, []string{"3msmutation100", "3msmutation200"}) {
			t.Fatalf("mutation first page rkeys = %v", got)
		}
		seedEventFixture(t, pool, eventFixture{Owner: mutationOwner, Rkey: "3msmutation150", Name: "inserted before seek", StartsAt: start, EndsAt: start.Add(time.Hour)})
		seedEventFixture(t, pool, eventFixture{Owner: mutationOwner, Rkey: "3msmutation250", Name: "inserted after seek", StartsAt: start, EndsAt: start.Add(time.Hour)})
		if _, err := pool.Exec(ctx, `DELETE FROM craftsky_business_events WHERE owner_did=$1 AND rkey='3msmutation300'`, mutationOwner); err != nil {
			t.Fatalf("delete unseen event: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE craftsky_business_events SET starts_at=$2 WHERE owner_did=$1 AND rkey='3msmutation400'`, mutationOwner, start.Add(-time.Hour)); err != nil {
			t.Fatalf("move unseen event before seek: %v", err)
		}

		second := serveBusinessEventPage(t, handler, visitor, mutationOwner.String(), "/v1/profiles/"+mutationOwner.String()+"/events?limit=2&cursor="+url.QueryEscape(first.Cursor))
		if got := eventPageRkeys(second); !reflect.DeepEqual(got, []string{"3msmutation250"}) || second.Cursor != "" {
			t.Fatalf("mutation second page = %v cursor %q, want only post-seek insert", got, second.Cursor)
		}
	})

	t.Run("malformed pagination inputs use standard errors", func(t *testing.T) {
		validCursor := serveBusinessEventPage(t, handler, visitor, owner.String(), "/v1/profiles/"+owner.String()+"/events?limit=1").Cursor
		if validCursor == "" {
			t.Fatal("expected a valid cursor to tamper with")
		}
		tamperedCursor := validCursor[:len(validCursor)-1] + "A"
		if strings.HasSuffix(validCursor, "A") {
			tamperedCursor = validCursor[:len(validCursor)-1] + "B"
		}
		for _, test := range []struct {
			query string
			code  string
		}{
			{query: "cursor=bad@@", code: "invalid_cursor"},
			{query: "cursor=" + url.QueryEscape(tamperedCursor), code: "invalid_cursor"},
			{query: "limit=zero", code: "invalid_limit"},
			{query: "limit=0", code: "invalid_limit"},
		} {
			response := serveBusinessEventPageResponse(handler, visitor, owner.String(), "/v1/profiles/"+owner.String()+"/events?"+test.query)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("%s status = %d, want 400", test.query, response.Code)
			}
			var body envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode %s error: %v", test.query, err)
			}
			if body.Error != test.code || body.Message == "" || body.RequestID != "business-event-pagination" {
				t.Fatalf("%s error = %+v", test.query, body)
			}
		}
	})
}

type businessEventPageResponse struct {
	Items  []businessEventPageItem `json:"items"`
	Cursor string                  `json:"cursor,omitempty"`
}

type businessEventPageItem struct {
	Rkey     string `json:"rkey"`
	URI      string `json:"uri"`
	StartsAt string `json:"startsAt"`
}

func seedEligibleBusinessOwner(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed business owner membership %s: %v", owner, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed business owner account type %s: %v", owner, err)
	}
}

func serveBusinessEventPage(t *testing.T, handler http.Handler, visitor syntax.DID, handleOrDID, target string) businessEventPageResponse {
	t.Helper()
	response := serveBusinessEventPageResponse(handler, visitor, handleOrDID, target)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d body = %s", target, response.Code, response.Body.String())
	}
	var page businessEventPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode GET %s: %v", target, err)
	}
	return page
}

func serveBusinessEventPageResponse(handler http.Handler, visitor syntax.DID, handleOrDID, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.SetPathValue("handleOrDid", handleOrDID)
	ctx := middleware.WithDID(request.Context(), visitor)
	ctx = ctxkeys.WithRunID(ctx, "business-event-pagination")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func eventPageContainsURI(page businessEventPageResponse, uri syntax.ATURI) bool {
	for _, event := range page.Items {
		if event.URI == uri.String() {
			return true
		}
	}
	return false
}

func eventPageRkeys(page businessEventPageResponse) []string {
	rkeys := make([]string, 0, len(page.Items))
	for _, event := range page.Items {
		rkeys = append(rkeys, event.Rkey)
	}
	return rkeys
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
