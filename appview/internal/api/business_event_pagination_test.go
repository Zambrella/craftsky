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
