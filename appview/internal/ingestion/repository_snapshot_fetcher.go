package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/identity"
	atrepo "github.com/bluesky-social/indigo/atproto/repo"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"

	_ "social.craftsky/appview/internal/lexicon/craftsky"
)

const (
	MaxRepositorySnapshotBytes int64         = 64 << 20
	RepositorySnapshotTimeout  time.Duration = 2 * time.Minute
)

type RepositorySnapshotFetcher struct {
	directory identity.Directory
	client    *http.Client
}

func NewRepositorySnapshotFetcher(directory identity.Directory, client *http.Client) (*RepositorySnapshotFetcher, error) {
	if directory == nil || client == nil {
		return nil, errors.New("repository snapshot fetcher dependencies are unavailable")
	}
	return &RepositorySnapshotFetcher{directory: directory, client: client}, nil
}

func (fetcher *RepositorySnapshotFetcher) Fetch(
	ctx context.Context,
	did syntax.DID,
	registry RepositoryCollectionRegistry,
) (VerifiedRepositorySnapshot, error) {
	if fetcher == nil || fetcher.directory == nil || fetcher.client == nil || did == "" || registry == nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("invalid_request", errors.New("repository snapshot request is invalid"))
	}
	ctx, cancel := context.WithTimeout(ctx, RepositorySnapshotTimeout)
	defer cancel()

	before, err := fetcher.resolveSource(ctx, did)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("source_unavailable", err)
	}
	repositoryURL, err := repositorySnapshotURL(before.pds, did)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("source_invalid", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repositoryURL, nil)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("request_invalid", err)
	}
	req.Header.Set("Accept", "application/vnd.ipld.car")
	response, err := fetcher.client.Do(req)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("download_failed", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return VerifiedRepositorySnapshot{}, snapshotFailure("download_failed", fmt.Errorf("getRepo status %d", response.StatusCode))
	}
	carBytes, err := io.ReadAll(io.LimitReader(response.Body, MaxRepositorySnapshotBytes+1))
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("download_incomplete", err)
	}
	if int64(len(carBytes)) > MaxRepositorySnapshotBytes {
		return VerifiedRepositorySnapshot{}, snapshotFailure("download_oversized", errors.New("repository snapshot exceeds size limit"))
	}

	verified, err := verifyRepositoryCAR(ctx, carBytes, did, before, registry.Collections())
	if err != nil {
		return VerifiedRepositorySnapshot{}, err
	}
	after, err := fetcher.resolveSource(ctx, did)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("source_unavailable", err)
	}
	if before.pds != after.pds || before.key != after.key {
		return VerifiedRepositorySnapshot{}, snapshotFailure("source_changed", errors.New("repository authority changed during fetch"))
	}
	return verified, nil
}

type repositorySnapshotSource struct {
	pds string
	key identity.VerificationMethod
}

func (fetcher *RepositorySnapshotFetcher) resolveSource(ctx context.Context, did syntax.DID) (repositorySnapshotSource, error) {
	resolved, err := fetcher.directory.LookupDID(ctx, did)
	if err != nil {
		return repositorySnapshotSource{}, err
	}
	if resolved == nil || resolved.DID != did {
		return repositorySnapshotSource{}, errors.New("resolved repository identity is invalid")
	}
	pds, err := canonicalRepositoryOrigin(resolved.PDSEndpoint())
	if err != nil {
		return repositorySnapshotSource{}, err
	}
	key, ok := resolved.Keys["atproto"]
	if !ok {
		return repositorySnapshotSource{}, identity.ErrKeyNotDeclared
	}
	if _, err := resolved.PublicKey(); err != nil {
		return repositorySnapshotSource{}, err
	}
	return repositorySnapshotSource{pds: pds, key: key}, nil
}

func canonicalRepositoryOrigin(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("repository PDS endpoint is not an origin")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path, parsed.RawPath, parsed.RawQuery, parsed.Fragment = "", "", "", ""
	return parsed.String(), nil
}

func repositorySnapshotURL(origin string, did syntax.DID) (string, error) {
	parsed, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	parsed.Path = "/xrpc/com.atproto.sync.getRepo"
	query := parsed.Query()
	query.Set("did", did.String())
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func verifyRepositoryCAR(
	ctx context.Context,
	carBytes []byte,
	did syntax.DID,
	source repositorySnapshotSource,
	collections []syntax.NSID,
) (VerifiedRepositorySnapshot, error) {
	reader, err := car.NewCarReader(bytes.NewReader(carBytes))
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("car_malformed", err)
	}
	if len(reader.Header.Roots) != 1 {
		return VerifiedRepositorySnapshot{}, snapshotFailure("car_ambiguous_root", fmt.Errorf("repository CAR has %d roots", len(reader.Header.Roots)))
	}
	root := reader.Header.Roots[0]
	for {
		if _, err := reader.Next(); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return VerifiedRepositorySnapshot{}, snapshotFailure("car_malformed", err)
		}
	}
	commit, repository, err := atrepo.LoadRepoFromCAR(ctx, bytes.NewReader(carBytes))
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("repository_invalid", err)
	}
	if commit.DID != did.String() || repository.DID != did {
		return VerifiedRepositorySnapshot{}, snapshotFailure("commit_wrong_did", errors.New("repository commit DID does not match request"))
	}
	revision, err := syntax.ParseTID(commit.Rev)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("commit_invalid", err)
	}
	publicKey, err := identityKey(source.key)
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("signing_key_invalid", err)
	}
	if err := commit.VerifySignature(publicKey); err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("signature_invalid", err)
	}
	if err := repository.MST.Verify(); err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("mst_invalid", err)
	}
	dataRoot, err := repository.MST.RootCID()
	if err != nil || dataRoot == nil || !dataRoot.Equals(commit.Data) {
		return VerifiedRepositorySnapshot{}, snapshotFailure("commit_wrong_root", errors.New("repository MST root does not match commit"))
	}

	slices.Sort(collections)
	collections = slices.Compact(collections)
	registered := make(map[syntax.NSID]struct{}, len(collections))
	for _, collection := range collections {
		if _, err := syntax.ParseNSID(collection.String()); err != nil {
			return VerifiedRepositorySnapshot{}, snapshotFailure("registry_invalid", err)
		}
		registered[collection] = struct{}{}
	}
	records := make([]RepositorySnapshotRecord, 0)
	err = repository.MST.Walk(func(key []byte, recordCID cid.Cid) error {
		collection, rkey, err := syntax.ParseRepoPath(string(key))
		if err != nil {
			return err
		}
		block, err := repository.RecordStore.Get(ctx, recordCID)
		if err != nil {
			return err
		}
		if _, wanted := registered[collection]; !wanted {
			return nil
		}
		decoded, err := lexutil.CborDecodeValue(block.RawData())
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return err
		}
		records = append(records, RepositorySnapshotRecord{
			URI: syntax.ATURI("at://" + did.String() + "/" + collection.String() + "/" + rkey.String()),
			CID: syntax.CID(recordCID.String()), Record: encoded,
		})
		return nil
	})
	if err != nil {
		return VerifiedRepositorySnapshot{}, snapshotFailure("mst_invalid", err)
	}
	slices.SortFunc(records, func(a, b RepositorySnapshotRecord) int { return strings.Compare(a.URI.String(), b.URI.String()) })
	return newVerifiedRepositorySnapshot(did, revision, syntax.CID(root.String()), records), nil
}

func identityKey(method identity.VerificationMethod) (atcrypto.PublicKey, error) {
	ident := identity.Identity{Keys: map[string]identity.VerificationMethod{"atproto": method}}
	return ident.PublicKey()
}

type RepositorySnapshotError struct {
	code string
	err  error
}

func snapshotFailure(code string, err error) error {
	return &RepositorySnapshotError{code: code, err: err}
}

func (err *RepositorySnapshotError) Error() string {
	return "repository snapshot verification: " + err.code
}
func (err *RepositorySnapshotError) Unwrap() error      { return err.err }
func (err *RepositorySnapshotError) ReasonCode() string { return err.code }
