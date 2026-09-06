package ingestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/identity"
	atrepo "github.com/bluesky-social/indigo/atproto/repo"
	"github.com/bluesky-social/indigo/atproto/repo/mst"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	blockstore "github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipld/go-car"
	carutil "github.com/ipld/go-car/util"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

var errRepairInterrupted = errors.New("injected repair interruption")

func TestRepositoryRepairVerifiedSnapshotConvergesThroughDurableProjection(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE repair_projection_audit(
			uri TEXT PRIMARY KEY, owner_did TEXT NOT NULL, action TEXT NOT NULL,
			applications INTEGER NOT NULL DEFAULT 1
		)
	`); err != nil {
		t.Fatalf("create repair projection audit: %v", err)
	}
	ctx := context.Background()
	store, lifecycles, service := repairIntegrationService(t, pool, nil)
	dispatcher := repairIntegrationDispatcher(t)
	owner := syntax.DID("did:plc:repair-integration")

	profile := repairFixtureRecord(owner, "social.craftsky.actor.profile", "self", "profile-old")
	seedRepairSource(t, service, store, dispatcher, profile.event(1, "3aaaaaaaaaaa2", "create"))
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,$2)`, owner, profile.cid); err != nil {
		t.Fatalf("seed membership projection: %v", err)
	}
	for i, collection := range dispatcher.Collections() {
		if collection == profile.collection {
			continue
		}
		record := repairFixtureRecord(owner, collection, repairFixtureRkey(collection, i), "old")
		seedRepairSource(t, service, store, dispatcher, record.event(uint64(i+2), "3aaaaaaaaaaa2", "create"))
	}
	if _, err := pool.Exec(ctx, `DELETE FROM repair_projection_audit`); err != nil {
		t.Fatalf("clear seed audit: %v", err)
	}

	records := make([]repairFixture, 0, 110)
	for i, collection := range dispatcher.Collections() {
		rkey := repairFixtureRkey(collection, i)
		switch collection {
		case "social.craftsky.feed.post": // authoritative create
			rkey += "-new"
		case "social.craftsky.business.event": // authoritative update
		case "social.craftsky.feed.repost": // authoritative delete: omit the seeded path
			continue
		}
		value := "old"
		if collection == "social.craftsky.business.event" {
			value = "updated"
		}
		if collection == "social.craftsky.actor.profile" {
			value = "profile-old"
		}
		records = append(records, repairFixtureRecord(owner, collection, rkey, value))
	}
	for i := 0; i < 101; i++ {
		records = append(records, repairFixtureRecord(owner, "social.craftsky.feed.post", syntax.RecordKey(fmt.Sprintf("3repair%06d", i)), fmt.Sprintf("new-%03d", i)))
	}

	// A newer Tap source must remain authoritative over this older snapshot.
	defended := repairFixtureRecord(owner, "social.craftsky.feed.post", "3repairdefend", "snapshot-old")
	newer := repairFixtureRecord(owner, defended.collection, defended.rkey, "tap-newer")
	seedRepairSource(t, service, store, dispatcher, newer.event(900, "3aaaaaaaaaaaz", "create"))
	if _, err := pool.Exec(ctx, `DELETE FROM repair_projection_audit WHERE uri=$1`, defended.uri); err != nil {
		t.Fatalf("clear defended seed audit: %v", err)
	}
	records = append(records, defended)

	snapshot := fetchSignedRepairSnapshot(t, owner, "3aaaaaaaaaaay", records)
	snapshotRecords, err := ingestion.DescribeRepositoryRepair(snapshot, nil, dispatcher)
	if err != nil {
		t.Fatalf("describe signed snapshot: %v", err)
	}
	collectionsSeen := make(map[syntax.NSID]int)
	for _, description := range snapshotRecords {
		collectionsSeen[description.Collection]++
	}
	if len(collectionsSeen) != len(dispatcher.Collections())-1 {
		t.Fatalf("signed snapshot collections=%v", collectionsSeen)
	}
	profileSource, err := store.Source(ctx, profile.uri)
	if err != nil {
		t.Fatalf("read profile source before repair: %v", err)
	}
	profileComparison, err := ingestion.DescribeRepositoryRepair(snapshot, []ingestion.SourceRecord{profileSource}, dispatcher)
	if err != nil {
		t.Fatalf("compare profile before repair: %v", err)
	}
	for _, description := range profileComparison {
		if description.URI == profile.uri && description.Action == ingestion.RepositoryRepairDelete {
			t.Fatalf("verified snapshot unexpectedly omitted profile: %+v", description)
		}
	}
	if err := store.EnqueueRepositoryJob(ctx, owner, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue repair: %v", err)
	}
	claim := claimRepairRepositoryJob(t, store)
	var projected atomic.Int32
	interruptedProjector := func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
		if projected.Add(1) == ingestion.RepositoryRepairBatchSize+1 {
			return tap.Outcome{}, errRepairInterrupted
		}
		return dispatcher.Project(ctx, tx, source)
	}
	repair, err := ingestion.NewRepositoryRepair(ingestion.RepositoryRepairConfig{
		Store: store, Ingestor: service, Projector: interruptedProjector,
	})
	if err != nil {
		t.Fatalf("new repair: %v", err)
	}
	err = store.RunRepositoryJob(ctx, claim, func(ctx context.Context, _ ingestion.RepositoryClaim) (string, error) {
		return repair.Apply(ctx, snapshot, dispatcher)
	})
	if !errors.Is(err, errRepairInterrupted) {
		t.Fatalf("interrupted repair error = %v", err)
	}
	job, err := store.RepositoryJob(ctx, owner, ingestion.RepositoryJobPDSReconcile)
	if err != nil || job.State != "pending" || projected.Load() != ingestion.RepositoryRepairBatchSize+1 {
		t.Fatalf("interrupted job=%+v projected=%d err=%v", job, projected.Load(), err)
	}

	repair, err = ingestion.NewRepositoryRepair(ingestion.RepositoryRepairConfig{
		Store: store, Ingestor: service, Projector: dispatcher.Project,
	})
	if err != nil {
		t.Fatalf("new retry repair: %v", err)
	}
	claim = claimRepairRepositoryJob(t, store)
	if err := store.RunRepositoryJob(ctx, claim, func(ctx context.Context, _ ingestion.RepositoryClaim) (string, error) {
		return repair.Apply(ctx, snapshot, dispatcher)
	}); err != nil {
		t.Fatalf("retry repair: %v", err)
	}

	// A duplicate completed job must converge without applying any projection twice.
	if err := store.EnqueueRepositoryJob(ctx, owner, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue duplicate repair: %v", err)
	}
	claim = claimRepairRepositoryJob(t, store)
	if err := store.RunRepositoryJob(ctx, claim, func(ctx context.Context, _ ingestion.RepositoryClaim) (string, error) {
		return repair.Apply(ctx, snapshot, dispatcher)
	}); err != nil {
		t.Fatalf("duplicate repair: %v", err)
	}

	job, err = store.RepositoryJob(ctx, owner, ingestion.RepositoryJobPDSReconcile)
	if err != nil || job.State != "complete" || job.AuthoritativeRevision != "3aaaaaaaaaaay" {
		t.Fatalf("completed job=%+v err=%v", job, err)
	}
	var duplicateApplications, wrongOwner int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM repair_projection_audit WHERE applications<>1`).Scan(&duplicateApplications); err != nil {
		t.Fatalf("count duplicate applications: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM repair_projection_audit WHERE owner_did<>$1`, owner).Scan(&wrongOwner); err != nil {
		t.Fatalf("count wrong owners: %v", err)
	}
	if duplicateApplications != 0 || wrongOwner != 0 {
		t.Fatalf("projection audit duplicate=%d wrongOwner=%d", duplicateApplications, wrongOwner)
	}
	var creates, updates, deletes int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE action='create'),
		       count(*) FILTER (WHERE action='update'),
		       count(*) FILTER (WHERE action='delete')
		FROM repair_projection_audit
	`).Scan(&creates, &updates, &deletes); err != nil {
		t.Fatalf("count repair actions: %v", err)
	}
	if creates != 101 || updates != 2 || deletes != 1 {
		t.Fatalf("repair actions create/update/delete=%d/%d/%d, want 101/2/1", creates, updates, deletes)
	}
	var unchangedApplications int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM repair_projection_audit WHERE uri=$1`,
		syntax.ATURI("at://"+owner.String()+"/app.bsky.actor.profile/self")).Scan(&unchangedApplications); err != nil || unchangedApplications != 0 {
		t.Fatalf("unchanged projection applications=%d err=%v", unchangedApplications, err)
	}
	missingCollection := syntax.NSID("social.craftsky.feed.repost")
	missingURI := syntax.ATURI("at://" + owner.String() + "/" + missingCollection.String() + "/" + repairFixtureRkey(missingCollection, slices.Index(dispatcher.Collections(), missingCollection)).String())
	assertRepairProjection(t, pool, missingURI, "delete", owner)
	updatedCollection := syntax.NSID("social.craftsky.business.event")
	updatedURI := syntax.ATURI("at://" + owner.String() + "/" + updatedCollection.String() + "/" + repairFixtureRkey(updatedCollection, slices.Index(dispatcher.Collections(), updatedCollection)).String())
	assertRepairProjection(t, pool, updatedURI, "update", owner)
	defendedSource, err := store.Source(ctx, defended.uri)
	if err != nil || defendedSource.Revision != "3aaaaaaaaaaaz" || defendedSource.CID != newer.cid {
		t.Fatalf("newer Tap source overwritten: source=%+v err=%v", defendedSource, err)
	}
	lifecycle, err := lifecycles.Get(ctx, owner)
	if err != nil || lifecycle.State != ownerlifecycle.StateActive {
		t.Fatalf("repair owner lifecycle=%+v err=%v", lifecycle, err)
	}
}

func TestRepositoryRepairProfileAbsenceRequiresVerificationAndAppliesDeparture(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:repair-profile-absence")
	if _, err := pool.Exec(ctx, `
		CREATE TABLE repair_projection_audit(uri TEXT PRIMARY KEY, owner_did TEXT NOT NULL, action TEXT NOT NULL, applications INTEGER NOT NULL DEFAULT 1);
		CREATE TABLE repair_member_work(owner_did TEXT PRIMARY KEY, state TEXT NOT NULL)
	`); err != nil {
		t.Fatalf("create member work fixture: %v", err)
	}
	participant := func(ctx context.Context, tx pgx.Tx, _, after ownerlifecycle.Lifecycle) error {
		_, err := tx.Exec(ctx, `UPDATE repair_member_work SET state='stopped' WHERE owner_did=$1`, after.Owner)
		return err
	}
	store, lifecycles, service := repairIntegrationService(t, pool, participant)
	dispatcher := repairIntegrationDispatcher(t)
	profile := repairFixtureRecord(owner, "social.craftsky.actor.profile", "self", "profile")
	seedRepairSource(t, service, store, dispatcher, profile.event(1, "3aaaaaaaaaaa2", "create"))
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,$2)`, owner, profile.cid); err != nil {
		t.Fatalf("seed membership projection: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO repair_member_work(owner_did,state) VALUES($1,'scheduled')`, owner); err != nil {
		t.Fatalf("seed member work: %v", err)
	}
	repair, err := ingestion.NewRepositoryRepair(ingestion.RepositoryRepairConfig{
		Store: store, Ingestor: service, Projector: dispatcher.Project,
	})
	if err != nil {
		t.Fatalf("new repair: %v", err)
	}

	if _, err := repair.Apply(ctx, ingestion.VerifiedRepositorySnapshot{}, dispatcher); !errors.Is(err, ingestion.ErrRepositorySnapshotUnverified) {
		t.Fatalf("unverified absence error=%v", err)
	}
	assertRepairLifecycleAndWork(t, lifecycles, pool, owner, ownerlifecycle.StateActive, "scheduled")

	snapshot := fetchSignedRepairSnapshot(t, owner, "3aaaaaaaaaaa3", []repairFixture{
		repairFixtureRecord(owner, "social.craftsky.business.profile", "self", "still-public"),
	})
	if _, err := repair.Apply(ctx, snapshot, dispatcher); err != nil {
		t.Fatalf("verified profile absence repair: %v", err)
	}
	assertRepairLifecycleAndWork(t, lifecycles, pool, owner, ownerlifecycle.StateDeparted, "stopped")
	lifecycle, err := lifecycles.Get(ctx, owner)
	if err != nil || lifecycle.TerminalAt != nil || lifecycle.TransitionReason != "profileDeleted" {
		t.Fatalf("verified absence terminalized owner: lifecycle=%+v err=%v", lifecycle, err)
	}
}

type repairRecordIngestor interface {
	IngestRecord(context.Context, tap.Event) (tap.Outcome, error)
}

type repairAuditIndexer struct{}

func (repairAuditIndexer) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO repair_projection_audit(uri,owner_did,action) VALUES($1,$2,$3)
		ON CONFLICT(uri) DO UPDATE SET owner_did=EXCLUDED.owner_did,
			action=EXCLUDED.action,applications=repair_projection_audit.applications+1
	`, event.URI, event.DID, event.Action)
	return tap.Applied(), err
}

func repairIntegrationDispatcher(t *testing.T) *index.TransactionalDispatcher {
	t.Helper()
	dispatcher := index.NewTransactionalDispatcher()
	for _, collection := range repairIntegrationCollections() {
		dispatcher.Register(collection, repairAuditIndexer{})
	}
	return dispatcher
}

func repairIntegrationService(t *testing.T, pool *pgxpool.Pool, participant ownerlifecycle.TransitionParticipant) (*ingestion.Store, *ownerlifecycle.Store, *ingestion.Service) {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new repair owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("new repair lifecycle store: %v", err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new repair ingestion store: %v", err)
	}
	if participant == nil {
		participant = func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error { return nil }
	}
	service, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: store, Lifecycles: lifecycles, ProfileParticipant: participant,
	})
	if err != nil {
		t.Fatalf("new repair ingestion service: %v", err)
	}
	return store, lifecycles, service
}

func seedRepairSource(t *testing.T, ingestor repairRecordIngestor, store *ingestion.Store, dispatcher *index.TransactionalDispatcher, event tap.Event) {
	t.Helper()
	if outcome, err := ingestor.IngestRecord(context.Background(), event); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("seed source %s: outcome=%+v err=%v", event.URI, outcome, err)
	}
	claim := claimOneProjection(t, store, "repair-seed")
	if claim.SourceURI != event.URI {
		t.Fatalf("seed projection claim=%s want=%s", claim.SourceURI, event.URI)
	}
	if err := store.Project(context.Background(), claim, dispatcher.Project); err != nil {
		t.Fatalf("seed projection %s: %v", event.URI, err)
	}
}

func claimRepairRepositoryJob(t *testing.T, store *ingestion.Store) ingestion.RepositoryClaim {
	t.Helper()
	claims, err := store.ClaimRepositoryJobs(context.Background(), ingestion.RepositoryClaimRequest{
		Worker: "repair-integration", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim repair repository job: claims=%d err=%v", len(claims), err)
	}
	return claims[0]
}

func assertRepairProjection(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, uri syntax.ATURI, action string, owner syntax.DID) {
	t.Helper()
	var gotAction string
	var gotOwner syntax.DID
	var applications int
	if err := pool.QueryRow(context.Background(), `SELECT action,owner_did,applications FROM repair_projection_audit WHERE uri=$1`, uri).Scan(&gotAction, &gotOwner, &applications); err != nil {
		t.Fatalf("read repair projection %s: %v", uri, err)
	}
	if gotAction != action || gotOwner != owner || applications != 1 {
		t.Fatalf("repair projection %s = %s/%s/%d", uri, gotAction, gotOwner, applications)
	}
}

func assertRepairLifecycleAndWork(t *testing.T, lifecycles *ownerlifecycle.Store, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, owner syntax.DID, state ownerlifecycle.State, work string) {
	t.Helper()
	lifecycle, err := lifecycles.Get(context.Background(), owner)
	if err != nil || lifecycle.State != state {
		t.Fatalf("owner lifecycle=%+v want=%s err=%v", lifecycle, state, err)
	}
	var got string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM repair_member_work WHERE owner_did=$1`, owner).Scan(&got); err != nil || got != work {
		t.Fatalf("member work=%q want=%q err=%v", got, work, err)
	}
}

type repairFixture struct {
	collection syntax.NSID
	rkey       syntax.RecordKey
	uri        syntax.ATURI
	cid        syntax.CID
	record     json.RawMessage
}

func (record repairFixture) event(id uint64, revision syntax.TID, action string) tap.Event {
	event := tap.Event{ID: id, URI: record.uri, DID: record.uri.Authority().DID(), Collection: record.collection, Rkey: record.rkey, Rev: revision, Action: action}
	if action != "delete" {
		event.CID, event.Record = record.cid, record.record
	}
	return event
}

func repairFixtureRkey(collection syntax.NSID, index int) syntax.RecordKey {
	if collection == "social.craftsky.actor.profile" || collection == "social.craftsky.business.profile" || collection == "app.bsky.actor.profile" {
		return "self"
	}
	if collection == "social.craftsky.business.event" {
		return "3meventrecord"
	}
	return syntax.RecordKey(fmt.Sprintf("3repair%06d", index))
}

func repairFixtureRecord(owner syntax.DID, collection syntax.NSID, rkey syntax.RecordKey, value string) repairFixture {
	created := "2026-09-03T12:00:00Z"
	record := map[string]any{"$type": collection.String()}
	switch collection {
	case "social.craftsky.actor.profile":
		record["crafts"] = []string{value}
	case "social.craftsky.feed.post":
		record["text"], record["createdAt"] = value, created
	case "social.craftsky.feed.like", "social.craftsky.feed.repost":
		record["subject"] = map[string]any{"uri": "at://did:plc:subject/social.craftsky.feed.post/3subject0000", "cid": "bafysubject"}
		record["createdAt"] = created
	case "social.craftsky.business.profile":
		record["tagline"] = value
	case "social.craftsky.business.event":
		record["name"], record["startsAt"], record["endsAt"], record["createdAt"] = value, created, "2026-09-03T13:00:00Z", created
		record["roles"] = []string{"vendor"}
	case "app.bsky.actor.profile":
		record["displayName"] = value
	case "app.bsky.graph.follow", "app.bsky.graph.block":
		record["subject"], record["createdAt"] = "did:plc:subject", created
	}
	encoded, _ := json.Marshal(record)
	return repairFixture{
		collection: collection, rkey: rkey,
		uri: syntax.ATURI("at://" + owner.String() + "/" + collection.String() + "/" + rkey.String()),
		cid: repairFixtureCID(encoded), record: encoded,
	}
}

func repairFixtureCID(encoded []byte) syntax.CID {
	block, _ := repairFixtureBlock(encoded)
	return syntax.CID(block.Cid().String())
}

func fetchSignedRepairSnapshot(t *testing.T, did syntax.DID, revision syntax.TID, records []repairFixture) ingestion.VerifiedRepositorySnapshot {
	t.Helper()
	key := repairSnapshotKey(t)
	body := buildSignedRepairCAR(t, did, revision, key, records)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/xrpc/com.atproto.sync.getRepo" || r.URL.Query().Get("did") != did.String() {
			t.Errorf("unexpected getRepo request %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/vnd.ipld.car")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	public, err := key.PublicKey()
	if err != nil {
		t.Fatalf("snapshot public key: %v", err)
	}
	directory := repairDirectoryFunc(func(context.Context, syntax.DID) (*identity.Identity, error) {
		return &identity.Identity{
			DID:      did,
			Keys:     map[string]identity.VerificationMethod{"atproto": {Type: "Multikey", PublicKeyMultibase: public.Multibase()}},
			Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: server.URL}},
		}, nil
	})
	fetcher, err := ingestion.NewRepositorySnapshotFetcher(directory, server.Client())
	if err != nil {
		t.Fatalf("new snapshot fetcher: %v", err)
	}
	registry := repairRegistry(repairIntegrationCollections())
	snapshot, err := fetcher.Fetch(context.Background(), did, registry)
	if err != nil {
		t.Fatalf("fetch signed repair snapshot: %v: %v", err, errors.Unwrap(err))
	}
	return snapshot
}

func buildSignedRepairCAR(t *testing.T, did syntax.DID, revision syntax.TID, key atcrypto.PrivateKey, records []repairFixture) []byte {
	t.Helper()
	ctx := context.Background()
	store := blockstore.NewBlockstore(datastore.NewMapDatastore())
	tree := mst.NewEmptyTree()
	blockCIDs := make([]cid.Cid, 0, len(records)+8)
	for _, record := range records {
		block, err := repairFixtureBlock(record.record)
		if err != nil {
			t.Fatalf("encode repair record %s: %v", record.uri, err)
		}
		if err := store.Put(ctx, block); err != nil {
			t.Fatalf("store repair record: %v", err)
		}
		blockCIDs = append(blockCIDs, block.Cid())
		if _, err := tree.Insert([]byte(record.collection.String()+"/"+record.rkey.String()), block.Cid()); err != nil {
			t.Fatalf("insert repair MST record: %v", err)
		}
	}
	walked := 0
	if err := tree.Walk(func([]byte, cid.Cid) error { walked++; return nil }); err != nil || walked != len(records) {
		t.Fatalf("repair fixture MST records=%d want=%d err=%v", walked, len(records), err)
	}
	dataRoot, err := writeCompleteRepairMST(ctx, store, tree.Root, &blockCIDs)
	if err != nil {
		t.Fatalf("write repair MST: %v", err)
	}
	commit := atrepo.Commit{DID: did.String(), Version: atrepo.ATPROTO_REPO_VERSION, Data: *dataRoot, Rev: revision.String()}
	if err := commit.Sign(key); err != nil {
		t.Fatalf("sign repair commit: %v", err)
	}
	var commitBytes bytes.Buffer
	if err := commit.MarshalCBOR(&commitBytes); err != nil {
		t.Fatalf("marshal repair commit: %v", err)
	}
	commitCID, err := cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: 0x12, MhLength: -1}.Sum(commitBytes.Bytes())
	if err != nil {
		t.Fatalf("CID repair commit: %v", err)
	}
	commitBlock, err := blocks.NewBlockWithCid(commitBytes.Bytes(), commitCID)
	if err != nil {
		t.Fatalf("build repair commit block: %v", err)
	}
	if err := store.Put(ctx, commitBlock); err != nil {
		t.Fatalf("store repair commit: %v", err)
	}
	var output bytes.Buffer
	if err := car.WriteHeader(&car.CarHeader{Roots: []cid.Cid{commitBlock.Cid()}, Version: 1}, &output); err != nil {
		t.Fatalf("write repair CAR header: %v", err)
	}
	all := blockCIDs
	slices.SortFunc(all, func(a, b cid.Cid) int { return strings.Compare(a.String(), b.String()) })
	all = slices.CompactFunc(all, func(a, b cid.Cid) bool { return a.Equals(b) })
	all = append([]cid.Cid{commitBlock.Cid()}, all...)
	for _, blockCID := range all {
		block, err := store.Get(ctx, blockCID)
		if err != nil {
			t.Fatalf("load repair CAR block %s: %v", blockCID, err)
		}
		if err := carutil.LdWrite(&output, blockCID.Bytes(), block.RawData()); err != nil {
			t.Fatalf("write repair CAR block %s: %v", blockCID, err)
		}
	}
	loadedCommit, loadedRepo, err := atrepo.LoadRepoFromCAR(ctx, bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("load generated repair CAR: %v", err)
	}
	loaded := 0
	if err := loadedRepo.MST.Walk(func([]byte, cid.Cid) error { loaded++; return nil }); err != nil || loaded != len(records) {
		t.Fatalf("generated repair CAR records=%d want=%d commit=%s err=%v", loaded, len(records), loadedCommit.Rev, err)
	}
	return output.Bytes()
}

func writeCompleteRepairMST(ctx context.Context, store blockstore.Blockstore, node *mst.Node, blockCIDs *[]cid.Cid) (*cid.Cid, error) {
	if node == nil {
		return nil, errors.New("nil repair MST node")
	}
	for i := range node.Entries {
		entry := &node.Entries[i]
		if entry.Child == nil {
			continue
		}
		childCID, err := writeCompleteRepairMST(ctx, store, entry.Child, blockCIDs)
		if err != nil {
			return nil, err
		}
		entry.ChildCID = childCID
	}
	nodeData := node.NodeData()
	encoded, nodeCID, err := nodeData.Bytes()
	if err != nil {
		return nil, err
	}
	block, err := blocks.NewBlockWithCid(encoded, *nodeCID)
	if err != nil {
		return nil, err
	}
	if err := store.Put(ctx, block); err != nil {
		return nil, err
	}
	*blockCIDs = append(*blockCIDs, *nodeCID)
	return nodeCID, nil
}

func repairFixtureBlock(raw []byte) (blocks.Block, error) {
	builder := basicnode.Prototype.Any.NewBuilder()
	if err := dagjson.Decode(builder, bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	if err := dagcbor.Encode(builder.Build(), &encoded); err != nil {
		return nil, err
	}
	recordCID, err := cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: 0x12, MhLength: -1}.Sum(encoded.Bytes())
	if err != nil {
		return nil, err
	}
	return blocks.NewBlockWithCid(encoded.Bytes(), recordCID)
}

func repairSnapshotKey(t *testing.T) atcrypto.PrivateKey {
	t.Helper()
	raw := make([]byte, 32)
	raw[len(raw)-1] = 7
	key, err := atcrypto.ParsePrivateBytesP256(raw)
	if err != nil {
		t.Fatalf("parse repair snapshot key: %v", err)
	}
	return key
}

type repairDirectoryFunc func(context.Context, syntax.DID) (*identity.Identity, error)

func (resolve repairDirectoryFunc) LookupDID(ctx context.Context, did syntax.DID) (*identity.Identity, error) {
	return resolve(ctx, did)
}
func (repairDirectoryFunc) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	return nil, errors.New("unexpected handle lookup")
}
func (repairDirectoryFunc) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return nil, errors.New("unexpected identifier lookup")
}
func (repairDirectoryFunc) Purge(context.Context, syntax.AtIdentifier) error { return nil }

type repairRegistry []syntax.NSID

func (registry repairRegistry) Collections() []syntax.NSID {
	return append([]syntax.NSID(nil), registry...)
}

func repairIntegrationCollections() []syntax.NSID {
	return []syntax.NSID{
		"social.craftsky.actor.profile", "social.craftsky.feed.post", "social.craftsky.feed.like",
		"social.craftsky.feed.repost", "social.craftsky.business.profile", "social.craftsky.business.event",
		"app.bsky.actor.profile", "app.bsky.graph.follow", "app.bsky.graph.block",
	}
}
