package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/app"
	internalTap "social.craftsky/appview/internal/tap"
)

const craftskyPostCollection = "social.craftsky.feed.post"

var tapRepoCheckOpts tapRepoCheckOptions

var tapRepoCheckCmd = &cobra.Command{
	Use:   "repo-check DID",
	Short: "Compare one DID's Tap/PDS/AppView post state",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTapRepoCheck(cmd.Context(), args[0], tapRepoCheckOpts, cmd.OutOrStdout())
	},
}

func init() {
	tapRepoCheckCmd.Flags().StringVar(&tapRepoCheckOpts.collection, "collection", craftskyPostCollection, "collection to compare; currently only social.craftsky.feed.post is supported")
	tapRepoCheckCmd.Flags().DurationVar(&tapRepoCheckOpts.wait, "wait", 3*time.Second, "time to wait after requesting Tap tracking/backfill")
	tapRepoCheckCmd.Flags().BoolVar(&tapRepoCheckOpts.json, "json", false, "print machine-readable JSON")
	tapRepoCheckCmd.Flags().BoolVar(&tapRepoCheckOpts.repairStale, "repair-stale", false, "delete stale local post rows that are absent from the PDS; dev only")
	tapRepoCheckCmd.Flags().BoolVar(&tapRepoCheckOpts.yes, "yes", false, "required with --repair-stale")
	tapCmd.AddCommand(tapRepoCheckCmd)
}

type tapRepoCheckOptions struct {
	collection  string
	wait        time.Duration
	json        bool
	repairStale bool
	yes         bool
}

type repoRecord struct {
	URI       string    `json:"uri"`
	CID       string    `json:"cid"`
	Rkey      string    `json:"rkey"`
	IndexedAt time.Time `json:"indexedAt,omitempty"`
}

type repoDiff struct {
	StaleLocal   []repoRecord  `json:"staleLocal"`
	MissingLocal []repoRecord  `json:"missingLocal"`
	CIDMismatch  []cidMismatch `json:"cidMismatch"`
}

type cidMismatch struct {
	URI      string `json:"uri"`
	LocalCID string `json:"localCid"`
	PDSCID   string `json:"pdsCid"`
}

type repoCheckReport struct {
	DID                 string          `json:"did"`
	Collection          string          `json:"collection"`
	TapBaseURL          string          `json:"tapBaseUrl"`
	TapAddRequested     bool            `json:"tapAddRequested"`
	TapInfoBefore       json.RawMessage `json:"tapInfoBefore,omitempty"`
	TapInfoAfter        json.RawMessage `json:"tapInfoAfter,omitempty"`
	TapOutboxBuffer     json.RawMessage `json:"tapOutboxBuffer,omitempty"`
	TapResyncBuffer     json.RawMessage `json:"tapResyncBuffer,omitempty"`
	PDSRecords          int             `json:"pdsRecords"`
	AppViewRecords      int             `json:"appViewRecords"`
	Diff                repoDiff        `json:"diff"`
	RepairRequested     bool            `json:"repairRequested"`
	RepairDeleted       int             `json:"repairDeleted"`
	RepairSkippedReason string          `json:"repairSkippedReason,omitempty"`
}

func runTapRepoCheck(ctx context.Context, didRaw string, opts tapRepoCheckOptions, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
	if opts.collection == "" {
		opts.collection = craftskyPostCollection
	}
	if opts.collection != craftskyPostCollection {
		return fmt.Errorf("unsupported collection %q: only %s is supported", opts.collection, craftskyPostCollection)
	}
	if opts.repairStale && !opts.yes {
		return errors.New("--repair-stale requires --yes")
	}

	did, err := syntax.ParseDID(didRaw)
	if err != nil {
		return fmt.Errorf("invalid DID %q: %w", didRaw, err)
	}
	env, err := parseEnvFlag()
	if err != nil {
		return err
	}
	if opts.repairStale && env != app.EnvDev {
		return errors.New("--repair-stale is only allowed with --env dev")
	}

	deps, cleanup, err := loadDeps(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	tapBase, err := tapHTTPBaseURL(deps.Config.TapWSURL)
	if err != nil {
		return err
	}
	tapClient := http.Client{Timeout: 5 * time.Second}

	report := repoCheckReport{
		DID:             did.String(),
		Collection:      opts.collection,
		TapBaseURL:      tapBase,
		RepairRequested: opts.repairStale,
	}
	report.TapInfoBefore, _ = tapGET(ctx, &tapClient, tapBase+"/info/"+url.PathEscape(did.String()))
	report.TapOutboxBuffer, _ = tapGET(ctx, &tapClient, tapBase+"/stats/outbox-buffer")
	report.TapResyncBuffer, _ = tapGET(ctx, &tapClient, tapBase+"/stats/resync-buffer")

	if err := tapAddRepo(ctx, &tapClient, tapBase, did); err != nil {
		return fmt.Errorf("tap repos/add: %w", err)
	}
	report.TapAddRequested = true
	if opts.wait > 0 {
		select {
		case <-time.After(opts.wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	report.TapInfoAfter, _ = tapGET(ctx, &tapClient, tapBase+"/info/"+url.PathEscape(did.String()))

	pdsRecords, err := listPDSRecords(ctx, did, opts.collection)
	if err != nil {
		return fmt.Errorf("list PDS records: %w", err)
	}
	localRecords, err := listLocalPostRecords(ctx, deps.DB, did)
	if err != nil {
		return fmt.Errorf("list AppView post records: %w", err)
	}
	report.PDSRecords = len(pdsRecords)
	report.AppViewRecords = len(localRecords)
	report.Diff = diffRepoRecords(localRecords, pdsRecords)

	if opts.repairStale {
		deleted, err := repairStalePosts(ctx, deps.DB, report.Diff.StaleLocal)
		if err != nil {
			return fmt.Errorf("repair stale posts: %w", err)
		}
		report.RepairDeleted = deleted
	} else if len(report.Diff.StaleLocal) > 0 {
		report.RepairSkippedReason = "run again with --repair-stale --yes to delete stale local post rows in dev"
	}

	if opts.json {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	printRepoCheckReport(out, report)
	return nil
}

func tapHTTPBaseURL(wsURL string) (string, error) {
	return internalTap.HTTPBaseURL(wsURL)
}

func tapGET(ctx context.Context, client *http.Client, url string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.RawMessage(body), nil
}

func tapAddRepo(ctx context.Context, client *http.Client, tapBase string, did syntax.DID) error {
	admin, err := internalTap.NewAdminClient(tapBase, client)
	if err != nil {
		return err
	}
	return admin.AddRepo(ctx, did)
}

func listPDSRecords(ctx context.Context, did syntax.DID, collection string) (map[string]repoRecord, error) {
	dir := identity.DefaultDirectory()
	ident, err := dir.LookupDID(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("resolve did %s: %w", did, err)
	}
	host := ident.PDSEndpoint()
	if host == "" {
		return nil, fmt.Errorf("did %s: no atproto_pds service endpoint in DID doc", did)
	}
	api := atclient.NewAPIClient(host)
	api.Client = &http.Client{Timeout: 10 * time.Second}
	nsid, err := syntax.ParseNSID("com.atproto.repo.listRecords")
	if err != nil {
		return nil, err
	}

	records := map[string]repoRecord{}
	cursor := ""
	for {
		var resp struct {
			Cursor  string `json:"cursor"`
			Records []struct {
				URI string `json:"uri"`
				CID string `json:"cid"`
			} `json:"records"`
		}
		params := map[string]any{
			"repo":       did.String(),
			"collection": collection,
			"limit":      100,
		}
		if cursor != "" {
			params["cursor"] = cursor
		}
		if err := api.Get(ctx, nsid, params, &resp); err != nil {
			return nil, err
		}
		for _, rec := range resp.Records {
			if rec.URI == "" {
				continue
			}
			records[rec.URI] = repoRecord{URI: rec.URI, CID: rec.CID, Rkey: rkeyFromURI(rec.URI)}
		}
		if resp.Cursor == "" || resp.Cursor == cursor {
			break
		}
		cursor = resp.Cursor
	}
	return records, nil
}

func listLocalPostRecords(ctx context.Context, pool *pgxpool.Pool, did syntax.DID) (map[string]repoRecord, error) {
	rows, err := pool.Query(ctx, `
		SELECT uri, cid, rkey, indexed_at
		FROM craftsky_posts
		WHERE did = $1
		ORDER BY indexed_at DESC, uri DESC`, did.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := map[string]repoRecord{}
	for rows.Next() {
		var rec repoRecord
		if err := rows.Scan(&rec.URI, &rec.CID, &rec.Rkey, &rec.IndexedAt); err != nil {
			return nil, err
		}
		records[rec.URI] = rec
	}
	return records, rows.Err()
}

func diffRepoRecords(local, remote map[string]repoRecord) repoDiff {
	diff := repoDiff{}
	for uri, localRec := range local {
		remoteRec, ok := remote[uri]
		if !ok {
			diff.StaleLocal = append(diff.StaleLocal, localRec)
			continue
		}
		if localRec.CID != remoteRec.CID {
			diff.CIDMismatch = append(diff.CIDMismatch, cidMismatch{
				URI:      uri,
				LocalCID: localRec.CID,
				PDSCID:   remoteRec.CID,
			})
		}
	}
	for uri, remoteRec := range remote {
		if _, ok := local[uri]; !ok {
			diff.MissingLocal = append(diff.MissingLocal, remoteRec)
		}
	}
	sort.Slice(diff.StaleLocal, func(i, j int) bool { return diff.StaleLocal[i].URI < diff.StaleLocal[j].URI })
	sort.Slice(diff.MissingLocal, func(i, j int) bool { return diff.MissingLocal[i].URI < diff.MissingLocal[j].URI })
	sort.Slice(diff.CIDMismatch, func(i, j int) bool { return diff.CIDMismatch[i].URI < diff.CIDMismatch[j].URI })
	return diff
}

func repairStalePosts(ctx context.Context, pool *pgxpool.Pool, stale []repoRecord) (int, error) {
	if len(stale) == 0 {
		return 0, nil
	}
	uris := make([]string, 0, len(stale))
	for _, rec := range stale {
		uris = append(uris, rec.URI)
	}
	cmd, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri = ANY($1)`, uris)
	if err != nil {
		return 0, err
	}
	return int(cmd.RowsAffected()), nil
}

func printRepoCheckReport(out io.Writer, report repoCheckReport) {
	fmt.Fprintf(out, "did:              %s\n", report.DID)
	fmt.Fprintf(out, "collection:       %s\n", report.Collection)
	fmt.Fprintf(out, "tap_base_url:     %s\n", report.TapBaseURL)
	fmt.Fprintf(out, "tap_add_requested:%t\n", report.TapAddRequested)
	fmt.Fprintf(out, "tap_info_before:  %s\n", compactRawJSON(report.TapInfoBefore))
	fmt.Fprintf(out, "tap_info_after:   %s\n", compactRawJSON(report.TapInfoAfter))
	fmt.Fprintf(out, "tap_outbox:       %s\n", compactRawJSON(report.TapOutboxBuffer))
	fmt.Fprintf(out, "tap_resync:       %s\n", compactRawJSON(report.TapResyncBuffer))
	fmt.Fprintf(out, "pds_records:      %d\n", report.PDSRecords)
	fmt.Fprintf(out, "appview_records:  %d\n", report.AppViewRecords)
	fmt.Fprintf(out, "stale_local:      %d\n", len(report.Diff.StaleLocal))
	fmt.Fprintf(out, "missing_local:    %d\n", len(report.Diff.MissingLocal))
	fmt.Fprintf(out, "cid_mismatch:     %d\n", len(report.Diff.CIDMismatch))
	if len(report.Diff.StaleLocal) > 0 {
		fmt.Fprintln(out, "\nstale local rows:")
		for _, rec := range report.Diff.StaleLocal {
			fmt.Fprintf(out, "- %s cid=%s indexed_at=%s\n", rec.URI, rec.CID, rec.IndexedAt.Format(time.RFC3339))
		}
	}
	if len(report.Diff.MissingLocal) > 0 {
		fmt.Fprintln(out, "\nmissing local rows:")
		for _, rec := range report.Diff.MissingLocal {
			fmt.Fprintf(out, "- %s cid=%s\n", rec.URI, rec.CID)
		}
	}
	if len(report.Diff.CIDMismatch) > 0 {
		fmt.Fprintln(out, "\ncid mismatches:")
		for _, rec := range report.Diff.CIDMismatch {
			fmt.Fprintf(out, "- %s local=%s pds=%s\n", rec.URI, rec.LocalCID, rec.PDSCID)
		}
	}
	if report.RepairDeleted > 0 {
		fmt.Fprintf(out, "\nrepair: deleted %d stale craftsky_posts rows\n", report.RepairDeleted)
	} else if report.RepairSkippedReason != "" {
		fmt.Fprintf(out, "\nrepair skipped: %s\n", report.RepairSkippedReason)
	}
}

func rkeyFromURI(uri string) string {
	idx := strings.LastIndex(uri, "/")
	if idx < 0 || idx == len(uri)-1 {
		return ""
	}
	return uri[idx+1:]
}

func compactRawJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "unavailable"
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return buf.String()
}
