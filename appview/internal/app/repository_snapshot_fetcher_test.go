package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/identity"
	atrepo "github.com/bluesky-social/indigo/atproto/repo"
	"github.com/bluesky-social/indigo/atproto/repo/mst"
	"github.com/bluesky-social/indigo/atproto/syntax"
	blockstore "github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipld/go-car"
	carutil "github.com/ipld/go-car/util"

	"social.craftsky/appview/internal/ingestion"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
)

func TestRepositorySnapshotFetcherRejectsTruncatedRepositoryBeforeComparison(t *testing.T) {
	did := syntax.DID("did:plc:repositorysnapshot")
	key := repositorySnapshotTestKey(t, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ipld.car")
		w.Header().Set("Content-Length", "64")
		_, _ = w.Write([]byte{0x3a, 0xa2, 0x65, 0x72, 0x6f, 0x6f, 0x74, 0x73})
	}))
	t.Cleanup(server.Close)

	directory := repositorySnapshotDirectoryFunc(func(context.Context, syntax.DID) (*identity.Identity, error) {
		return &identity.Identity{
			DID: did,
			Keys: map[string]identity.VerificationMethod{
				"atproto": repositorySnapshotVerificationMethod(t, key),
			},
			Services: map[string]identity.ServiceEndpoint{
				"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: server.URL},
			},
		}, nil
	})
	fetcher, err := newRepositorySnapshotFetcher(directory, server.Client())
	if err != nil {
		t.Fatalf("new snapshot fetcher: %v", err)
	}

	snapshot, err := fetcher.Fetch(context.Background(), did, repositorySnapshotRegistry{"social.craftsky.feed.post"})
	if err == nil {
		t.Fatal("truncated repository unexpectedly verified")
	}
	actions, compareErr := ingestion.DescribeRepositoryRepair(snapshot, nil, repositorySnapshotRegistry{"social.craftsky.feed.post"})
	if !errors.Is(compareErr, ingestion.ErrRepositorySnapshotUnverified) || len(actions) != 0 {
		t.Fatalf("comparison after truncated fetch = (%v, %v), want no actions and unverified error", actions, compareErr)
	}
}

func TestRepositorySnapshotFetcherTrustBoundary(t *testing.T) {
	did := syntax.DID("did:plc:repositorysnapshot")
	key := repositorySnapshotTestKey(t, 1)
	valid := buildRepositorySnapshotCAR(t, did, key)
	wrongDID := buildRepositorySnapshotCAR(t, "did:plc:wrongrepositoryowner", key)
	badSignature := buildRepositorySnapshotCAR(t, did, repositorySnapshotTestKey(t, 2))

	tests := []struct {
		name string
		body []byte
	}{
		{name: "malformed", body: []byte("not a car")},
		{name: "duplicate root", body: repositorySnapshotCARWithRoots(t, valid, valid.root, valid.root)},
		{name: "wrong root", body: repositorySnapshotCARWithRoots(t, valid, valid.dataRoot)},
		{name: "wrong DID", body: wrongDID.data},
		{name: "bad signature", body: badSignature.data},
		{name: "invalid MST", body: repositorySnapshotCARWithCommitData(t, valid, key, valid.root)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot, err := fetchRepositorySnapshotFromBytes(t, did, key, test.body, nil)
			assertUnverifiedRepositorySnapshot(t, snapshot, err)
		})
	}

	t.Run("source changed", func(t *testing.T) {
		t.Run("signing key", func(t *testing.T) {
			var lookups atomic.Int32
			changedKey := repositorySnapshotTestKey(t, 3)
			snapshot, err := fetchRepositorySnapshotFromBytes(t, did, key, valid.data, func(ident *identity.Identity) {
				if lookups.Add(1) == 2 {
					ident.Keys["atproto"] = repositorySnapshotVerificationMethod(t, changedKey)
				}
			})
			assertUnverifiedRepositorySnapshot(t, snapshot, err)
		})
		t.Run("PDS", func(t *testing.T) {
			var lookups atomic.Int32
			snapshot, err := fetchRepositorySnapshotFromBytes(t, did, key, valid.data, func(ident *identity.Identity) {
				if lookups.Add(1) == 2 {
					ident.Services["atproto_pds"] = identity.ServiceEndpoint{
						Type: "AtprotoPersonalDataServer", URL: "https://moved.example",
					}
				}
			})
			assertUnverifiedRepositorySnapshot(t, snapshot, err)
		})
	})

	t.Run("valid deterministic signed repository", func(t *testing.T) {
		snapshot, err := fetchRepositorySnapshotFromBytes(t, did, key, valid.data, nil)
		if err != nil {
			t.Fatalf("fetch valid repository: %v: %v", err, errors.Unwrap(err))
		}
		actions, err := ingestion.DescribeRepositoryRepair(snapshot, nil, repositorySnapshotRegistry{"social.craftsky.actor.profile"})
		if err != nil || len(actions) != 1 || actions[0].Action != ingestion.RepositoryRepairCreate {
			t.Fatalf("verified repository actions = (%v, %v), want one create", actions, err)
		}
	})
}

func TestRepositorySnapshotFetcherRejectsOversizedRepository(t *testing.T) {
	did := syntax.DID("did:plc:repositorysnapshot")
	key := repositorySnapshotTestKey(t, 1)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(io.LimitReader(repositorySnapshotRepeatingReader{}, ingestion.MaxRepositorySnapshotBytes+1)),
			Header:     make(http.Header),
		}, nil
	})}
	directory := staticRepositorySnapshotDirectory(t, did, "https://pds.example", key, nil)
	fetcher, err := newRepositorySnapshotFetcher(directory, client)
	if err != nil {
		t.Fatalf("new snapshot fetcher: %v", err)
	}
	snapshot, err := fetcher.Fetch(context.Background(), did, repositorySnapshotRegistry{"social.craftsky.actor.profile"})
	assertUnverifiedRepositorySnapshot(t, snapshot, err)
}

type repositorySnapshotCAR struct {
	data      []byte
	root      cid.Cid
	dataRoot  cid.Cid
	recordCID cid.Cid
}

type repositorySnapshotFixtureRecord struct {
	path   string
	record interface{ MarshalCBOR(io.Writer) error }
}

func buildRepositorySnapshotCAR(t *testing.T, did syntax.DID, key atcrypto.PrivateKey) repositorySnapshotCAR {
	return buildRepositorySnapshotCARWithRecords(t, did, key, "3aaaaaaaaaaa2", []repositorySnapshotFixtureRecord{{
		path:   "social.craftsky.actor.profile/self",
		record: &craftskylex.ActorProfile{LexiconTypeID: "social.craftsky.actor.profile", Crafts: []string{"knitting"}},
	}})
}

func buildRepositorySnapshotCARWithRecords(
	t *testing.T,
	did syntax.DID,
	key atcrypto.PrivateKey,
	revision string,
	records []repositorySnapshotFixtureRecord,
) repositorySnapshotCAR {
	t.Helper()
	ctx := context.Background()
	store := blockstore.NewBlockstore(datastore.NewMapDatastore())
	tree := mst.NewEmptyTree()
	recordCIDs := make([]cid.Cid, 0, len(records))
	for _, fixture := range records {
		var recordBytes bytes.Buffer
		if err := fixture.record.MarshalCBOR(&recordBytes); err != nil {
			t.Fatalf("marshal fixture record %s: %v", fixture.path, err)
		}
		recordCID, err := cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: 0x12, MhLength: -1}.Sum(recordBytes.Bytes())
		if err != nil {
			t.Fatalf("CID fixture record %s: %v", fixture.path, err)
		}
		recordBlock, err := blocks.NewBlockWithCid(recordBytes.Bytes(), recordCID)
		if err != nil || store.Put(ctx, recordBlock) != nil {
			t.Fatalf("store fixture record %s: %v", fixture.path, err)
		}
		if _, err := tree.Insert([]byte(fixture.path), recordCID); err != nil {
			t.Fatalf("insert fixture MST record %s: %v", fixture.path, err)
		}
		recordCIDs = append(recordCIDs, recordCID)
	}
	dataRoot, err := tree.WriteDiffBlocks(ctx, store)
	if err != nil {
		t.Fatalf("write fixture MST: %v", err)
	}
	commit := atrepo.Commit{DID: did.String(), Version: atrepo.ATPROTO_REPO_VERSION, Data: *dataRoot, Rev: revision}
	if err := commit.Sign(key); err != nil {
		t.Fatalf("sign fixture commit: %v", err)
	}
	root := putRepositorySnapshotCommit(t, store, &commit)
	required := append([]cid.Cid{*dataRoot}, recordCIDs...)
	var firstRecord cid.Cid
	if len(recordCIDs) > 0 {
		firstRecord = recordCIDs[0]
	}
	return repositorySnapshotCAR{
		data: writeRepositorySnapshotCAR(t, store, []cid.Cid{root}, required...),
		root: root, dataRoot: *dataRoot, recordCID: firstRecord,
	}
}

func putRepositorySnapshotCommit(t *testing.T, store blockstore.Blockstore, commit *atrepo.Commit) cid.Cid {
	t.Helper()
	var encoded bytes.Buffer
	if err := commit.MarshalCBOR(&encoded); err != nil {
		t.Fatalf("marshal fixture commit: %v", err)
	}
	root, err := cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: 0x12, MhLength: -1}.Sum(encoded.Bytes())
	if err != nil {
		t.Fatalf("CID fixture commit: %v", err)
	}
	block, err := blocks.NewBlockWithCid(encoded.Bytes(), root)
	if err != nil || store.Put(context.Background(), block) != nil {
		t.Fatalf("store fixture commit: %v", err)
	}
	return root
}

func writeRepositorySnapshotCAR(t *testing.T, store blockstore.Blockstore, roots []cid.Cid, required ...cid.Cid) []byte {
	t.Helper()
	var output bytes.Buffer
	if err := car.WriteHeader(&car.CarHeader{Roots: roots, Version: 1}, &output); err != nil {
		t.Fatalf("write fixture CAR header: %v", err)
	}
	written := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		block, err := store.Get(context.Background(), root)
		if err != nil {
			t.Fatalf("load fixture root block: %v", err)
		}
		if err := carutil.LdWrite(&output, root.Bytes(), block.RawData()); err != nil {
			t.Fatalf("write fixture root block: %v", err)
		}
		written[root.KeyString()] = struct{}{}
	}
	for _, key := range required {
		if _, exists := written[key.KeyString()]; exists {
			continue
		}
		block, err := store.Get(context.Background(), key)
		if err != nil {
			t.Fatalf("load required fixture block: %v", err)
		}
		if err := carutil.LdWrite(&output, key.Bytes(), block.RawData()); err != nil {
			t.Fatalf("write required fixture block: %v", err)
		}
		written[key.KeyString()] = struct{}{}
	}
	keys, err := store.AllKeysChan(context.Background())
	if err != nil {
		t.Fatalf("list fixture blocks: %v", err)
	}
	var cids []cid.Cid
	for key := range keys {
		cids = append(cids, key)
	}
	slices.SortFunc(cids, func(a, b cid.Cid) int { return strings.Compare(a.String(), b.String()) })
	for _, key := range cids {
		if _, exists := written[key.KeyString()]; exists {
			continue
		}
		block, err := store.Get(context.Background(), key)
		if err != nil {
			t.Fatalf("load fixture block: %v", err)
		}
		if err := carutil.LdWrite(&output, key.Bytes(), block.RawData()); err != nil {
			t.Fatalf("write fixture block: %v", err)
		}
	}
	return output.Bytes()
}

func repositorySnapshotCARWithRoots(t *testing.T, fixture repositorySnapshotCAR, roots ...cid.Cid) []byte {
	t.Helper()
	reader, err := car.NewCarReader(bytes.NewReader(fixture.data))
	if err != nil {
		t.Fatalf("read fixture CAR: %v", err)
	}
	var output bytes.Buffer
	if err := car.WriteHeader(&car.CarHeader{Roots: roots, Version: 1}, &output); err != nil {
		t.Fatalf("rewrite fixture CAR header: %v", err)
	}
	for {
		block, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read fixture CAR block: %v", err)
		}
		if err := carutil.LdWrite(&output, block.Cid().Bytes(), block.RawData()); err != nil {
			t.Fatalf("rewrite fixture CAR block: %v", err)
		}
	}
	return output.Bytes()
}

func repositorySnapshotCARWithCommitData(t *testing.T, fixture repositorySnapshotCAR, key atcrypto.PrivateKey, data cid.Cid) []byte {
	t.Helper()
	reader, err := car.NewCarReader(bytes.NewReader(fixture.data))
	if err != nil {
		t.Fatalf("read fixture CAR: %v", err)
	}
	store := blockstore.NewBlockstore(datastore.NewMapDatastore())
	var commit atrepo.Commit
	for {
		block, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read fixture CAR block: %v", err)
		}
		if block.Cid().Equals(fixture.root) {
			if err := commit.UnmarshalCBOR(bytes.NewReader(block.RawData())); err != nil {
				t.Fatalf("decode fixture commit: %v", err)
			}
			continue
		}
		if err := store.Put(context.Background(), block); err != nil {
			t.Fatalf("copy fixture block: %v", err)
		}
	}
	commit.Data = data
	commit.Sig = nil
	if err := commit.Sign(key); err != nil {
		t.Fatalf("sign rewritten fixture commit: %v", err)
	}
	root := putRepositorySnapshotCommit(t, store, &commit)
	return writeRepositorySnapshotCAR(t, store, []cid.Cid{root}, fixture.dataRoot, fixture.recordCID)
}

func fetchRepositorySnapshotFromBytes(
	t *testing.T,
	did syntax.DID,
	key atcrypto.PrivateKey,
	body []byte,
	mutateIdentity func(*identity.Identity),
) (ingestion.VerifiedRepositorySnapshot, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.sync.getRepo" || r.URL.Query().Get("did") != did.String() {
			t.Errorf("getRepo request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/vnd.ipld.car")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	directory := staticRepositorySnapshotDirectory(t, did, server.URL, key, mutateIdentity)
	fetcher, err := newRepositorySnapshotFetcher(directory, server.Client())
	if err != nil {
		t.Fatalf("new snapshot fetcher: %v", err)
	}
	return fetcher.Fetch(context.Background(), did, repositorySnapshotRegistry{"social.craftsky.actor.profile"})
}

func staticRepositorySnapshotDirectory(
	t *testing.T,
	did syntax.DID,
	pds string,
	key atcrypto.PrivateKey,
	mutate func(*identity.Identity),
) identity.Directory {
	t.Helper()
	method := repositorySnapshotVerificationMethod(t, key)
	return repositorySnapshotDirectoryFunc(func(context.Context, syntax.DID) (*identity.Identity, error) {
		ident := &identity.Identity{
			DID: did,
			Services: map[string]identity.ServiceEndpoint{
				"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: pds},
			},
			Keys: map[string]identity.VerificationMethod{"atproto": method},
		}
		if mutate != nil {
			mutate(ident)
		}
		return ident, nil
	})
}

func repositorySnapshotTestKey(t *testing.T, scalar byte) atcrypto.PrivateKey {
	t.Helper()
	bytes := make([]byte, 32)
	bytes[len(bytes)-1] = scalar
	key, err := atcrypto.ParsePrivateBytesP256(bytes)
	if err != nil {
		t.Fatalf("parse deterministic fixture key: %v", err)
	}
	return key
}

func repositorySnapshotVerificationMethod(t *testing.T, key atcrypto.PrivateKey) identity.VerificationMethod {
	t.Helper()
	publicKey, err := key.PublicKey()
	if err != nil {
		t.Fatalf("fixture public key: %v", err)
	}
	return identity.VerificationMethod{Type: "Multikey", PublicKeyMultibase: publicKey.Multibase()}
}

func assertUnverifiedRepositorySnapshot(t *testing.T, snapshot ingestion.VerifiedRepositorySnapshot, fetchErr error) {
	t.Helper()
	if fetchErr == nil {
		t.Fatal("untrusted repository unexpectedly verified")
	}
	var reasoned interface{ ReasonCode() string }
	if !errors.As(fetchErr, &reasoned) || reasoned.ReasonCode() == "" || len(reasoned.ReasonCode()) > 64 {
		t.Fatalf("snapshot failure has no bounded retry reason: %v", fetchErr)
	}
	actions, compareErr := ingestion.DescribeRepositoryRepair(snapshot, nil, repositorySnapshotRegistry{"social.craftsky.actor.profile"})
	if !errors.Is(compareErr, ingestion.ErrRepositorySnapshotUnverified) || len(actions) != 0 {
		t.Fatalf("untrusted comparison = (%v, %v), want no actions and unverified error", actions, compareErr)
	}
}

type repositorySnapshotRepeatingReader struct{}

func (repositorySnapshotRepeatingReader) Read(buffer []byte) (int, error) {
	for i := range buffer {
		buffer[i] = 'x'
	}
	return len(buffer), nil
}

type repositorySnapshotDirectoryFunc func(context.Context, syntax.DID) (*identity.Identity, error)

func (resolve repositorySnapshotDirectoryFunc) LookupDID(ctx context.Context, did syntax.DID) (*identity.Identity, error) {
	return resolve(ctx, did)
}

func (resolve repositorySnapshotDirectoryFunc) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	return nil, errors.New("unexpected handle lookup")
}

func (resolve repositorySnapshotDirectoryFunc) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return nil, errors.New("unexpected identifier lookup")
}

func (resolve repositorySnapshotDirectoryFunc) Purge(context.Context, syntax.AtIdentifier) error {
	return nil
}

type repositorySnapshotRegistry []syntax.NSID

func (registry repositorySnapshotRegistry) Collections() []syntax.NSID {
	return append([]syntax.NSID(nil), registry...)
}
