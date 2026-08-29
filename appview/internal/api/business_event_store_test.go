package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/testdb"
)

const businessEventStoreDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE craftsky_account_types (
    owner_did TEXT PRIMARY KEY,
    account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
);
CREATE TABLE craftsky_business_events (
    uri TEXT PRIMARY KEY,
    owner_did TEXT NOT NULL,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    raw_record JSONB NOT NULL,
    source_revision TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    status TEXT,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_did, rkey)
);
CREATE TABLE atproto_blocks (
    uri TEXT PRIMARY KEY,
    blocker_did TEXT NOT NULL,
    subject_did TEXT NOT NULL
);
CREATE TABLE moderation_outputs (
    id TEXT PRIMARY KEY,
    source_did TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_uri TEXT,
    value TEXT NOT NULL,
    action TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestBusinessEventStoreServesEligibleVisitorDirectAndUpcoming(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:event-owner")
	visitor := syntax.DID("did:plc:event-visitor")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	rkey := syntax.RecordKey("3msevent00001")
	uri := syntax.ATURI("at://did:plc:event-owner/social.craftsky.business.event/3msevent00001")

	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed eligible owner membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed eligible owner account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_events
			(uri, owner_did, rkey, cid, raw_record, source_revision, starts_at, ends_at, created_at, status)
		VALUES ($2, $1, $3, 'event-cid', $4::jsonb, '3msrevision001', $5, $6, $7, NULL)
	`, owner, uri, rkey, `{
		"$type":"social.craftsky.business.event",
		"name":"No declaration needed",
		"startsAt":"2026-08-30T10:00:00Z",
		"endsAt":"2026-08-30T12:00:00Z",
		"roles":["vendor"],
		"createdAt":"2026-08-01T09:00:00Z"
	}`, asOf.Add(22*time.Hour), asOf.Add(24*time.Hour), asOf.Add(-28*24*time.Hour)); err != nil {
		t.Fatalf("seed eligible event: %v", err)
	}

	store := business.NewStore(pool)
	got, err := store.ReadEvent(ctx, business.EventReadInput{
		CallerDID: visitor,
		OwnerDID:  owner,
		Rkey:      rkey,
		AsOf:      asOf,
	})
	if err != nil {
		t.Fatalf("read eligible event: %v", err)
	}
	if got.URI != uri || got.Name != "No declaration needed" || got.Status.Value != "scheduled" || !got.Status.Known || got.IsAllDay {
		t.Fatalf("direct event = %+v", got)
	}
	if got.PublicSuppressionReasons == nil || got.UpcomingExclusionReasons == nil {
		t.Fatalf("diagnostics must be non-nil: %+v", got)
	}
	wire, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal eligible event: %v", err)
	}
	var wireFields map[string]any
	if err := json.Unmarshal(wire, &wireFields); err != nil || wireFields["isAllDay"] != false {
		t.Fatalf("omitted isAllDay wire value = %v, decode error %v", wireFields["isAllDay"], err)
	}

	upcoming, err := store.ListUpcomingEvents(ctx, business.UpcomingEventListInput{
		CallerDID: visitor,
		OwnerDID:  owner,
		AsOf:      asOf,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("list upcoming events: %v", err)
	}
	if len(upcoming) != 1 || upcoming[0].URI != uri {
		t.Fatalf("upcoming = %+v", upcoming)
	}
}

func TestBusinessEventStoreCallerAwarePolicyMatrix(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	ctx := context.Background()
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	visitor := syntax.DID("did:plc:event-visitor")
	owner := syntax.DID("did:plc:event-owner")
	regular := syntax.DID("did:plc:regular-owner")
	departed := syntax.DID("did:plc:departed-owner")
	blockedOwner := syntax.DID("did:plc:blocked-owner")

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did, record_cid) VALUES
			($1, 'owner-profile'), ($2, 'regular-profile'), ($3, 'blocked-profile')
	`, owner, regular, blockedOwner); err != nil {
		t.Fatalf("seed owner memberships: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_account_types(owner_did, account_type) VALUES
			($1, 'business'), ($2, 'regular'), ($3, 'business'), ($4, 'business')
	`, owner, regular, blockedOwner, departed); err != nil {
		t.Fatalf("seed owner account types: %v", err)
	}

	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3msfuture0001", Name: "Future", StartsAt: asOf.Add(time.Hour), EndsAt: asOf.Add(2 * time.Hour)})
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3msongoing001", Name: "Ongoing", StartsAt: asOf.Add(-time.Hour), EndsAt: asOf.Add(time.Hour)})
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3mspast000001", Name: "Past", StartsAt: asOf.Add(-3 * time.Hour), EndsAt: asOf.Add(-2 * time.Hour)})
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3mscancel0001", Name: "Cancelled", StartsAt: asOf.Add(3 * time.Hour), EndsAt: asOf.Add(4 * time.Hour), Status: "cancelled"})
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3mspostpone01", Name: "Postponed", StartsAt: asOf.Add(5 * time.Hour), EndsAt: asOf.Add(6 * time.Hour), Status: "postponed"})
	invalidURI := seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3msinvalid001", Name: "Invalid", StartsAt: asOf.Add(7 * time.Hour), EndsAt: asOf.Add(7 * time.Hour)})
	seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3msoverlong01", Name: "Over duration", StartsAt: asOf.Add(8 * time.Hour), EndsAt: asOf.Add(32*24*time.Hour + 8*time.Hour)})
	moderatedURI := seedEventFixture(t, pool, eventFixture{Owner: owner, Rkey: "3msmoderate01", Name: "Moderated", StartsAt: asOf.Add(9 * time.Hour), EndsAt: asOf.Add(10 * time.Hour)})
	unsafeURI := seedEventFixture(t, pool, eventFixture{
		Owner: owner, Rkey: "3msunknown001", Name: "Federated", StartsAt: asOf.Add(11 * time.Hour), EndsAt: asOf.Add(12 * time.Hour),
		RawFields: map[string]any{
			"roles":           []string{"vendor", "community-mystery"},
			"mode":            "telepresence",
			"status":          "rescheduled-externally",
			"timeZone":        "Not/A-TimeZone",
			"eventUri":        "http://unsafe.example/event",
			"registrationUri": "https://safe.example/register",
			"image": map[string]any{
				"image": map[string]any{"mimeType": "image/gif", "size": 12},
			},
			"independentExtension": map[string]any{"retained": true},
		},
	})
	seedEventFixture(t, pool, eventFixture{Owner: regular, Rkey: "3msregular001", Name: "Regular", StartsAt: asOf.Add(time.Hour), EndsAt: asOf.Add(2 * time.Hour)})
	seedEventFixture(t, pool, eventFixture{Owner: departed, Rkey: "3msdeparted01", Name: "Departed", StartsAt: asOf.Add(time.Hour), EndsAt: asOf.Add(2 * time.Hour)})
	blockedURI := seedEventFixture(t, pool, eventFixture{Owner: blockedOwner, Rkey: "3msblocked001", Name: "Blocked", StartsAt: asOf.Add(time.Hour), EndsAt: asOf.Add(2 * time.Hour)})

	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks(uri, blocker_did, subject_did)
		VALUES ('at://did:plc:blocked-owner/app.bsky.graph.block/visitor', $1, $2)
	`, blockedOwner, visitor); err != nil {
		t.Fatalf("seed block policy: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(id, source_did, subject_type, subject_did, subject_uri, value, action)
		VALUES ('moderated-event', 'did:plc:labeler', 'event', $1, $2, 'hide', 'apply')
	`, owner, moderatedURI); err != nil {
		t.Fatalf("seed moderation policy: %v", err)
	}

	store := business.NewStore(pool)

	for _, rkey := range []syntax.RecordKey{"3mspast000001", "3mscancel0001", "3mspostpone01"} {
		got, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: visitor, OwnerDID: owner, Rkey: rkey, AsOf: asOf})
		if err != nil {
			t.Errorf("visitor direct read %s: %v", rkey, err)
			continue
		}
		if rkey == "3mspast000001" && !got.Past {
			t.Errorf("past event did not derive past state: %+v", got)
		}
	}

	ownerInvalid, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: owner, OwnerDID: owner, Rkey: "3msinvalid001", AsOf: asOf})
	if err != nil {
		t.Fatalf("owner read invalid retained event: %v", err)
	}
	if ownerInvalid.URI != invalidURI || !reflect.DeepEqual(ownerInvalid.PublicSuppressionReasons, []string{"invalid-time-range"}) ||
		!reflect.DeepEqual(ownerInvalid.UpcomingExclusionReasons, []string{"invalid-time-range"}) {
		t.Fatalf("invalid event diagnostics = %+v", ownerInvalid)
	}

	ownerModerated, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: owner, OwnerDID: owner, Rkey: "3msmoderate01", AsOf: asOf})
	if err != nil || !reflect.DeepEqual(ownerModerated.PublicSuppressionReasons, []string{"record-moderated"}) {
		t.Fatalf("moderated owner event = (%+v, %v)", ownerModerated, err)
	}
	regularManagement, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: regular, OwnerDID: regular, Rkey: "3msregular001", AsOf: asOf})
	if err != nil || !reflect.DeepEqual(regularManagement.PublicSuppressionReasons, []string{"owner-not-business"}) {
		t.Fatalf("regular owner management event = (%+v, %v)", regularManagement, err)
	}

	for _, test := range []struct {
		name  string
		owner syntax.DID
		rkey  syntax.RecordKey
	}{
		{name: "regular owner", owner: regular, rkey: "3msregular001"},
		{name: "departed owner", owner: departed, rkey: "3msdeparted01"},
		{name: "blocked caller", owner: blockedOwner, rkey: "3msblocked001"},
		{name: "invalid range", owner: owner, rkey: "3msinvalid001"},
		{name: "over duration", owner: owner, rkey: "3msoverlong01"},
		{name: "moderated record", owner: owner, rkey: "3msmoderate01"},
	} {
		t.Run("visitor_not_found_"+test.name, func(t *testing.T) {
			_, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: visitor, OwnerDID: test.owner, Rkey: test.rkey, AsOf: asOf})
			if !errors.Is(err, business.ErrEventNotFound) {
				t.Fatalf("ReadEvent error = %v, want ErrEventNotFound", err)
			}
		})
	}

	unknown, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: visitor, OwnerDID: owner, Rkey: "3msunknown001", AsOf: asOf})
	if err != nil {
		t.Fatalf("read safely narrowed independent event: %v", err)
	}
	if unknown.URI != unsafeURI || unknown.Status != (business.OpenValue{Value: "rescheduled-externally"}) || unknown.Mode == nil || unknown.Mode.Known ||
		unknown.EventURI != "" || unknown.RegistrationURI != "https://safe.example/register" || unknown.TimeZone != "" || len(unknown.Image) != 0 {
		t.Fatalf("independent hydration = %+v", unknown)
	}
	if len(unknown.Roles) != 2 || !unknown.Roles[0].Known || unknown.Roles[1].Known {
		t.Fatalf("independent roles = %+v", unknown.Roles)
	}
	var retained map[string]any
	if err := pool.QueryRow(ctx, `SELECT raw_record FROM craftsky_business_events WHERE uri=$1`, unsafeURI).Scan(&retained); err != nil {
		t.Fatalf("read retained raw event: %v", err)
	}
	if _, ok := retained["independentExtension"]; !ok || retained["eventUri"] != "http://unsafe.example/event" {
		t.Fatalf("raw event was narrowed in storage: %+v", retained)
	}

	upcoming, err := store.ListUpcomingEvents(ctx, business.UpcomingEventListInput{CallerDID: visitor, OwnerDID: owner, AsOf: asOf, Limit: 50})
	if err != nil {
		t.Fatalf("list upcoming policy matrix: %v", err)
	}
	gotNames := make([]string, 0, len(upcoming))
	for _, event := range upcoming {
		gotNames = append(gotNames, event.Name)
	}
	if want := []string{"Ongoing", "Future", "Federated"}; !reflect.DeepEqual(gotNames, want) {
		t.Fatalf("upcoming names = %v, want %v", gotNames, want)
	}
	for _, listOwner := range []syntax.DID{regular, departed, blockedOwner} {
		got, err := store.ListUpcomingEvents(ctx, business.UpcomingEventListInput{CallerDID: visitor, OwnerDID: listOwner, AsOf: asOf, Limit: 50})
		if err != nil || len(got) != 0 {
			t.Errorf("suppressed upcoming owner %s = (%+v, %v)", listOwner, got, err)
		}
	}
	if blockedURI == "" {
		t.Fatal("blocked fixture URI must be populated")
	}
}

type eventFixture struct {
	Owner     syntax.DID
	Rkey      syntax.RecordKey
	Name      string
	StartsAt  time.Time
	EndsAt    time.Time
	Status    string
	RawFields map[string]any
}

func seedEventFixture(t *testing.T, pool *pgxpool.Pool, fixture eventFixture) syntax.ATURI {
	t.Helper()
	uri := syntax.ATURI("at://" + fixture.Owner.String() + "/social.craftsky.business.event/" + fixture.Rkey.String())
	record := map[string]any{
		"$type":     "social.craftsky.business.event",
		"name":      fixture.Name,
		"startsAt":  fixture.StartsAt.UTC().Format(time.RFC3339),
		"endsAt":    fixture.EndsAt.UTC().Format(time.RFC3339),
		"roles":     []string{"vendor"},
		"createdAt": "2026-08-01T09:00:00Z",
	}
	for key, value := range fixture.RawFields {
		record[key] = value
	}
	var status any
	if fixture.Status != "" {
		record["status"] = fixture.Status
		status = fixture.Status
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal event %s: %v", fixture.Name, err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_business_events
			(uri, owner_did, rkey, cid, raw_record, source_revision, starts_at, ends_at, created_at, status)
		VALUES ($1, $2, $3, $4, $5, '3msrevision001', $6, $7, '2026-08-01T09:00:00Z', $8)
	`, uri, fixture.Owner, fixture.Rkey, "cid-"+fixture.Rkey.String(), raw, fixture.StartsAt, fixture.EndsAt, status); err != nil {
		t.Fatalf("seed event %s: %v", fixture.Name, err)
	}
	return uri
}
