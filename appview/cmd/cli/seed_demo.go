package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/app"
)

const (
	demoFollowCollection = "app.bsky.graph.follow"
	demoLikeCollection   = "social.craftsky.feed.like"
	demoRepostCollection = "social.craftsky.feed.repost"
)

type demoSeedArgs struct {
	UserDID    string
	ViewerDIDs []string
	Reset      bool
	Seed       string
	Now        time.Time
}

type demoSeedStats struct {
	Profiles int
	Follows  int
	Posts    int
	Projects int
	Comments int
	Likes    int
	Reposts  int
	Deleted  int64
}

type demoProfile struct {
	DID         string
	Handle      string
	DisplayName string
	Description string
	Crafts      []string
	AvatarName  string
	BannerName  string
}

type demoPost struct {
	Rkey           string
	AuthorDID      string
	Text           string
	Images         []demoImage
	Tags           []string
	Project        *demoProject
	ReplyRootURI   string
	ReplyRootCID   string
	ReplyParentURI string
	ReplyParentCID string
	CreatedAt      time.Time
	IndexedAt      time.Time
}

type demoImage struct {
	Name   string
	Alt    string
	Width  int
	Height int
}

type demoProject struct {
	CraftType         string
	Status            string
	Title             string
	Duration          string
	PatternName       string
	PatternDifficulty string
	PatternDesigner   string
	Materials         []string
	Colors            []string
	DesignTags        []string
	Tags              []string
	DetailsType       string
	Details           map[string]any
}

var demoSeedFlags = demoSeedArgs{
	Seed: "demo",
}

var seedDemoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Seed screenshot-friendly demo profiles, projects, media, and engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := parseEnvFlag()
		if err != nil {
			return err
		}
		if env != app.EnvDev {
			return fmt.Errorf("seed demo is dev-only; refusing to run with --env %s", env)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
		defer cancel()

		deps, cleanup, err := loadDeps(ctx)
		if err != nil {
			return err
		}
		defer cleanup()

		seedArgs := demoSeedFlags
		if strings.TrimSpace(seedArgs.UserDID) == "" {
			viewerDIDs, err := collectDemoViewerDIDs(ctx, deps.DB, deps.Config.DevDID)
			if err != nil {
				return err
			}
			seedArgs.ViewerDIDs = viewerDIDs
		}
		seedArgs.Now = time.Now().UTC()
		stats, err := runDemoSeed(ctx, deps.DB, seedArgs)
		if err != nil {
			return err
		}
		printDemoSeedStats(os.Stdout, stats)
		return nil
	},
}

func init() {
	seedDemoCmd.Flags().StringVar(&demoSeedFlags.UserDID, "user", "", "viewer DID to seed follows/timeline around; defaults to CRAFTSKY_DEV_DID plus active local sessions")
	seedDemoCmd.Flags().BoolVar(&demoSeedFlags.Reset, "reset", false, "delete previous demo data for this seed before inserting")
	seedDemoCmd.Flags().StringVar(&demoSeedFlags.Seed, "seed", demoSeedFlags.Seed, "deterministic demo-data namespace")
	seedCmd.AddCommand(seedDemoCmd)
}

func collectDemoViewerDIDs(ctx context.Context, pool *pgxpool.Pool, configDevDID string) ([]string, error) {
	viewerDIDs := []string{configDevDID}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT account_did
		FROM craftsky_sessions
		WHERE revoked_at IS NULL
		ORDER BY account_did
	`)
	if err != nil {
		return nil, fmt.Errorf("list active local sessions for demo seed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			return nil, fmt.Errorf("scan active local session DID: %w", err)
		}
		viewerDIDs = append(viewerDIDs, did)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active local session DIDs: %w", err)
	}
	return normalizeDemoViewerDIDList(viewerDIDs)
}

func normalizeDemoViewerDIDs(args demoSeedArgs) ([]string, error) {
	if strings.TrimSpace(args.UserDID) != "" {
		return normalizeDemoViewerDIDList([]string{args.UserDID})
	}
	return normalizeDemoViewerDIDList(args.ViewerDIDs)
}

func normalizeDemoViewerDIDList(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		did, err := syntax.ParseDID(value)
		if err != nil {
			return nil, fmt.Errorf("demo viewer DID %q: %w", value, err)
		}
		if seen[did.String()] {
			continue
		}
		seen[did.String()] = true
		out = append(out, did.String())
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("demo seed requires at least one viewer DID")
	}
	return out, nil
}

func runDemoSeed(ctx context.Context, pool *pgxpool.Pool, args demoSeedArgs) (demoSeedStats, error) {
	if pool == nil {
		return demoSeedStats{}, fmt.Errorf("db pool is required")
	}
	if args.Now.IsZero() {
		args.Now = time.Now().UTC()
	}
	seed, err := normalizeFakeSeed(args.Seed)
	if err != nil {
		return demoSeedStats{}, err
	}
	viewerDIDs, err := normalizeDemoViewerDIDs(args)
	if err != nil {
		return demoSeedStats{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return demoSeedStats{}, fmt.Errorf("begin demo seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stats := demoSeedStats{}
	if args.Reset {
		deleted, err := resetDemoSeed(ctx, tx, seed)
		if err != nil {
			return demoSeedStats{}, err
		}
		stats.Deleted = deleted
	}

	profiles := demoProfiles(seed)
	for _, viewerDID := range viewerDIDs {
		viewer := demoProfile{
			DID:         viewerDID,
			Handle:      localDemoHandle(viewerDID),
			DisplayName: "You",
			Description: "Local Craftsky demo viewer.",
			Crafts:      []string{"social.craftsky.feed.defs#knitting", "social.craftsky.feed.defs#sewing", "social.craftsky.feed.defs#quilting"},
			AvatarName:  "avatar-viewer",
			BannerName:  "banner-viewer",
		}
		if err := upsertDemoProfile(ctx, tx, viewer, false); err != nil {
			return demoSeedStats{}, err
		}
		stats.Profiles++
	}
	for _, profile := range profiles {
		if err := upsertDemoProfile(ctx, tx, profile, true); err != nil {
			return demoSeedStats{}, err
		}
		stats.Profiles++
	}

	for viewerIndex, viewerDID := range viewerDIDs {
		for i, profile := range profiles {
			if err := upsertDemoFollow(ctx, tx, viewerDID, profile.DID, fmt.Sprintf("demo-%s-follow-viewer-%02d-%02d", seed, viewerIndex+1, i+1), args.Now.Add(-time.Duration(i+viewerIndex)*time.Hour)); err != nil {
				return demoSeedStats{}, err
			}
			stats.Follows++
		}
	}
	for i := 0; i < len(profiles); i++ {
		subject := profiles[(i+1)%len(profiles)].DID
		if err := upsertDemoFollow(ctx, tx, profiles[i].DID, subject, fmt.Sprintf("demo-%s-follow-maker-%02d", seed, i+1), args.Now.Add(-time.Duration(i+12)*time.Hour)); err != nil {
			return demoSeedStats{}, err
		}
		stats.Follows++
	}

	posts := demoRootPosts(seed, profiles, args.Now)
	rootRefs := make([]demoPostRef, 0, len(posts))
	for _, post := range posts {
		ref, err := upsertDemoPost(ctx, tx, post)
		if err != nil {
			return demoSeedStats{}, err
		}
		stats.Posts++
		if post.Project != nil {
			stats.Projects++
		}
		rootRefs = append(rootRefs, ref)
	}

	comments := demoComments(seed, profiles, rootRefs, args.Now)
	for _, post := range comments {
		if _, err := upsertDemoPost(ctx, tx, post); err != nil {
			return demoSeedStats{}, err
		}
		stats.Posts++
		stats.Comments++
	}

	likes, reposts, err := seedDemoEngagement(ctx, tx, seed, profiles, rootRefs, args.Now)
	if err != nil {
		return demoSeedStats{}, err
	}
	stats.Likes = likes
	stats.Reposts = reposts

	if err := tx.Commit(ctx); err != nil {
		return demoSeedStats{}, fmt.Errorf("commit demo seed transaction: %w", err)
	}
	return stats, nil
}

func resetDemoSeed(ctx context.Context, tx pgx.Tx, seed string) (int64, error) {
	rkeyPattern := "demo-" + seed + "-%"
	var deleted int64
	for _, q := range []string{
		`DELETE FROM craftsky_likes WHERE rkey LIKE $1`,
		`DELETE FROM craftsky_reposts WHERE rkey LIKE $1`,
		`DELETE FROM atproto_follows WHERE rkey LIKE $1`,
		`DELETE FROM craftsky_posts WHERE rkey LIKE $1`,
	} {
		tag, err := tx.Exec(ctx, q, rkeyPattern)
		if err != nil {
			return 0, fmt.Errorf("reset demo data: %w", err)
		}
		deleted += tag.RowsAffected()
	}
	didPattern := demoDIDPrefix(seed) + "%"
	for _, q := range []string{
		`DELETE FROM atproto_identity_cache WHERE did LIKE $1`,
		`DELETE FROM bluesky_profiles WHERE did LIKE $1`,
		`DELETE FROM craftsky_profiles WHERE did LIKE $1`,
	} {
		if _, err := tx.Exec(ctx, q, didPattern); err != nil {
			return 0, fmt.Errorf("reset demo profiles: %w", err)
		}
	}
	return deleted, nil
}

func upsertDemoProfile(ctx context.Context, tx pgx.Tx, profile demoProfile, overwrite bool) error {
	if _, err := syntax.ParseDID(profile.DID); err != nil {
		return fmt.Errorf("demo profile DID %q: %w", profile.DID, err)
	}
	if _, err := syntax.ParseHandle(profile.Handle); err != nil {
		return fmt.Errorf("demo profile handle %q: %w", profile.Handle, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid, created_at, indexed_at)
		VALUES ($1, $2, $3, now(), now())
		ON CONFLICT (did) DO UPDATE SET
			crafts = EXCLUDED.crafts,
			record_cid = EXCLUDED.record_cid,
			indexed_at = now()
	`, profile.DID, profile.Crafts, fakeCID("demo-profile", profile.DID)); err != nil {
		return fmt.Errorf("upsert demo craftsky profile %s: %w", profile.DID, err)
	}

	conflict := "DO NOTHING"
	if overwrite {
		conflict = `DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			avatar_cid = EXCLUDED.avatar_cid,
			avatar_mime = EXCLUDED.avatar_mime,
			banner_cid = EXCLUDED.banner_cid,
			banner_mime = EXCLUDED.banner_mime,
			record_cid = EXCLUDED.record_cid,
			indexed_at = now()`
	}
	q := `
		INSERT INTO bluesky_profiles (did, display_name, description, avatar_cid, avatar_mime, banner_cid, banner_mime, record_cid, indexed_at)
		VALUES ($1, $2, $3, $4, 'image/jpeg', $5, 'image/jpeg', $6, now())
		ON CONFLICT (did) ` + conflict
	if _, err := tx.Exec(ctx, q, profile.DID, profile.DisplayName, profile.Description, devMediaCID(profile.AvatarName), devMediaCID(profile.BannerName), fakeCID("demo-bsky-profile", profile.DID)); err != nil {
		return fmt.Errorf("upsert demo bluesky profile %s: %w", profile.DID, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at, updated_at)
		VALUES ($1, $2, lower($2), now(), now())
		ON CONFLICT (did) DO UPDATE SET
			handle = EXCLUDED.handle,
			handle_lower = EXCLUDED.handle_lower,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = now()
	`, profile.DID, profile.Handle); err != nil {
		return fmt.Errorf("upsert demo identity %s: %w", profile.DID, err)
	}
	return nil
}

func upsertDemoFollow(ctx context.Context, tx pgx.Tx, did, subjectDID, rkey string, createdAt time.Time) error {
	record, err := json.Marshal(map[string]any{"$type": demoFollowCollection, "subject": subjectDID, "createdAt": createdAt.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	uri := "at://" + did + "/" + demoFollowCollection + "/" + rkey
	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (did, subject_did) DO UPDATE SET
			uri = EXCLUDED.uri,
			rkey = EXCLUDED.rkey,
			cid = EXCLUDED.cid,
			record = EXCLUDED.record,
			created_at = EXCLUDED.created_at,
			indexed_at = EXCLUDED.indexed_at
	`, uri, did, rkey, fakeCID("demo-follow", did, subjectDID), subjectDID, record, createdAt.UTC()); err != nil {
		return fmt.Errorf("upsert demo follow %s -> %s: %w", did, subjectDID, err)
	}
	return nil
}

type demoPostRef struct {
	URI  string
	CID  string
	DID  string
	Rkey string
}

func upsertDemoPost(ctx context.Context, tx pgx.Tx, post demoPost) (demoPostRef, error) {
	uri := postURI(post.AuthorDID, post.Rkey)
	cid := fakeCID("demo-post", uri)
	imagesJSON, err := demoImagesJSON(post.Images)
	if err != nil {
		return demoPostRef{}, err
	}
	record, rawProject, err := demoPostRecord(post, imagesJSON)
	if err != nil {
		return demoPostRef{}, err
	}
	var projectCraftType any
	if post.Project != nil {
		projectCraftType = post.Project.CraftType
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, facets, images,
			 reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid,
			 quote_uri, quote_cid, tags, is_project, project_craft_type, record, created_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5, NULL, $6,
			NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
			NULL, NULL, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (uri) DO UPDATE SET
			cid = EXCLUDED.cid,
			text = EXCLUDED.text,
			facets = EXCLUDED.facets,
			images = EXCLUDED.images,
			reply_root_uri = EXCLUDED.reply_root_uri,
			reply_root_cid = EXCLUDED.reply_root_cid,
			reply_parent_uri = EXCLUDED.reply_parent_uri,
			reply_parent_cid = EXCLUDED.reply_parent_cid,
			quote_uri = EXCLUDED.quote_uri,
			quote_cid = EXCLUDED.quote_cid,
			tags = EXCLUDED.tags,
			is_project = EXCLUDED.is_project,
			project_craft_type = EXCLUDED.project_craft_type,
			record = EXCLUDED.record,
			created_at = EXCLUDED.created_at,
			indexed_at = EXCLUDED.indexed_at
	`, uri, post.AuthorDID, post.Rkey, cid, post.Text, nullableRaw(imagesJSON), post.ReplyRootURI, post.ReplyRootCID, post.ReplyParentURI, post.ReplyParentCID, post.Tags, post.Project != nil, projectCraftType, record, post.CreatedAt.UTC(), post.IndexedAt.UTC()); err != nil {
		return demoPostRef{}, fmt.Errorf("upsert demo post %s: %w", uri, err)
	}
	if post.Project != nil {
		if err := upsertDemoProject(ctx, tx, uri, post.Project, rawProject); err != nil {
			return demoPostRef{}, err
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM craftsky_project_posts WHERE uri = $1`, uri); err != nil {
		return demoPostRef{}, fmt.Errorf("delete stale demo project %s: %w", uri, err)
	}
	return demoPostRef{URI: uri, CID: cid, DID: post.AuthorDID, Rkey: post.Rkey}, nil
}

func upsertDemoProject(ctx context.Context, tx pgx.Tx, uri string, project *demoProject, rawProject json.RawMessage) error {
	patternName, patternDifficulty, patternDesigner := nullableText(project.PatternName), nullableText(project.PatternDifficulty), nullableText(project.PatternDesigner)
	rawDetails, err := json.Marshal(project.Details)
	if err != nil {
		return fmt.Errorf("marshal demo project details %s: %w", uri, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO craftsky_project_posts (
			uri, raw_project, common_craft_type, common_status, common_title, common_duration,
			pattern_name, pattern_difficulty, pattern_designer,
			materials, colors, design_tags, project_tags, details_type, raw_details
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (uri) DO UPDATE SET
			raw_project = EXCLUDED.raw_project,
			common_craft_type = EXCLUDED.common_craft_type,
			common_status = EXCLUDED.common_status,
			common_title = EXCLUDED.common_title,
			common_duration = EXCLUDED.common_duration,
			pattern_name = EXCLUDED.pattern_name,
			pattern_difficulty = EXCLUDED.pattern_difficulty,
			pattern_designer = EXCLUDED.pattern_designer,
			materials = EXCLUDED.materials,
			colors = EXCLUDED.colors,
			design_tags = EXCLUDED.design_tags,
			project_tags = EXCLUDED.project_tags,
			details_type = EXCLUDED.details_type,
			raw_details = EXCLUDED.raw_details,
			indexed_at = now()
	`, uri, rawProject, project.CraftType, nullableText(project.Status), nullableText(project.Title), nullableText(project.Duration), patternName, patternDifficulty, patternDesigner, project.Materials, project.Colors, project.DesignTags, project.Tags, nullableText(project.DetailsType), rawDetails); err != nil {
		return fmt.Errorf("upsert demo project %s: %w", uri, err)
	}
	return nil
}

func seedDemoEngagement(ctx context.Context, tx pgx.Tx, seed string, profiles []demoProfile, roots []demoPostRef, now time.Time) (int, int, error) {
	likes := 0
	reposts := 0
	for i, root := range roots {
		for j := 0; j < min(4, len(profiles)); j++ {
			actor := profiles[(i+j)%len(profiles)]
			if actor.DID == root.DID {
				continue
			}
			if err := upsertDemoInteraction(ctx, tx, demoLikeCollection, actor.DID, fmt.Sprintf("demo-%s-like-%02d-%02d", seed, i+1, j+1), root, now.Add(-time.Duration(i+j)*time.Minute)); err != nil {
				return 0, 0, err
			}
			likes++
		}
		if i%2 == 0 {
			actor := profiles[(i+3)%len(profiles)]
			if actor.DID != root.DID {
				if err := upsertDemoInteraction(ctx, tx, demoRepostCollection, actor.DID, fmt.Sprintf("demo-%s-repost-%02d", seed, i+1), root, now.Add(-time.Duration(i)*time.Minute)); err != nil {
					return 0, 0, err
				}
				reposts++
			}
		}
	}
	return likes, reposts, nil
}

func upsertDemoInteraction(ctx context.Context, tx pgx.Tx, collection, did, rkey string, subject demoPostRef, createdAt time.Time) error {
	record, err := json.Marshal(map[string]any{"$type": collection, "subject": map[string]string{"uri": subject.URI, "cid": subject.CID}, "createdAt": createdAt.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	table := "craftsky_likes"
	if collection == demoRepostCollection {
		table = "craftsky_reposts"
	}
	uri := "at://" + did + "/" + collection + "/" + rkey
	q := fmt.Sprintf(`
		INSERT INTO %s (uri, did, rkey, cid, subject_uri, subject_cid, record, created_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (did, rkey) DO UPDATE SET
			uri = EXCLUDED.uri,
			cid = EXCLUDED.cid,
			subject_uri = EXCLUDED.subject_uri,
			subject_cid = EXCLUDED.subject_cid,
			record = EXCLUDED.record,
			created_at = EXCLUDED.created_at,
			indexed_at = EXCLUDED.indexed_at,
			deleted_at = NULL
	`, table)
	if _, err := tx.Exec(ctx, q, uri, did, rkey, fakeCID("demo-interaction", collection, did, subject.URI), subject.URI, subject.CID, record, createdAt.UTC()); err != nil {
		return fmt.Errorf("upsert demo interaction %s %s -> %s: %w", collection, did, subject.URI, err)
	}
	return nil
}

func demoPostRecord(post demoPost, imagesJSON json.RawMessage) (json.RawMessage, json.RawMessage, error) {
	record := map[string]any{"$type": fakePostCollection, "text": post.Text, "createdAt": post.CreatedAt.UTC().Format(time.RFC3339)}
	if len(imagesJSON) > 0 {
		var images any
		if err := json.Unmarshal(imagesJSON, &images); err != nil {
			return nil, nil, err
		}
		record["images"] = images
	}
	if post.ReplyRootURI != "" && post.ReplyParentURI != "" {
		record["reply"] = map[string]any{
			"root":   map[string]string{"uri": post.ReplyRootURI, "cid": post.ReplyRootCID},
			"parent": map[string]string{"uri": post.ReplyParentURI, "cid": post.ReplyParentCID},
		}
	}
	var rawProject json.RawMessage
	if post.Project != nil {
		project, err := post.Project.raw()
		if err != nil {
			return nil, nil, err
		}
		rawProject = project
		var p any
		if err := json.Unmarshal(project, &p); err != nil {
			return nil, nil, err
		}
		record["project"] = p
	}
	out, err := json.Marshal(record)
	return out, rawProject, err
}

func (p demoProject) raw() (json.RawMessage, error) {
	common := map[string]any{"craftType": p.CraftType}
	if p.Status != "" {
		common["status"] = p.Status
	}
	if p.Title != "" {
		common["title"] = p.Title
	}
	if p.Duration != "" {
		common["duration"] = p.Duration
	}
	pattern := map[string]any{}
	if p.PatternName != "" {
		pattern["name"] = p.PatternName
	}
	if p.PatternDifficulty != "" {
		pattern["difficulty"] = p.PatternDifficulty
	}
	if p.PatternDesigner != "" {
		pattern["designer"] = p.PatternDesigner
	}
	if len(pattern) > 0 {
		common["pattern"] = pattern
	}
	if len(p.Materials) > 0 {
		materials := make([]map[string]string, 0, len(p.Materials))
		for _, material := range p.Materials {
			materials = append(materials, map[string]string{"text": material})
		}
		common["materials"] = materials
	}
	if len(p.Colors) > 0 {
		common["colors"] = p.Colors
	}
	if len(p.DesignTags) > 0 {
		common["designTags"] = p.DesignTags
	}
	if len(p.Tags) > 0 {
		common["tags"] = p.Tags
	}
	out := map[string]any{"common": common}
	if len(p.Details) > 0 {
		out["details"] = p.Details
	}
	return json.Marshal(out)
}

func demoImagesJSON(images []demoImage) (json.RawMessage, error) {
	if len(images) == 0 {
		return nil, nil
	}
	out := make([]map[string]any, 0, len(images))
	for _, img := range images {
		out = append(out, map[string]any{
			"cid":  devMediaCID(img.Name),
			"mime": "image/jpeg",
			"size": int64(240000),
			"alt":  img.Alt,
			"aspectRatio": map[string]int{
				"width":  img.Width,
				"height": img.Height,
			},
		})
	}
	return json.Marshal(out)
}

func demoProfiles(seed string) []demoProfile {
	prefix := demoDIDPrefix(seed)
	handle := func(name string) string { return name + "-" + seed + ".craftsky.test" }
	profiles := []demoProfile{
		{DID: prefix + "0001", Handle: handle("mara-wool"), DisplayName: "Mara Wool", Description: "Knitter, visible mender, and chronic swatcher.", Crafts: []string{"social.craftsky.feed.defs#knitting"}, AvatarName: "avatar-mara", BannerName: "banner-mara"},
		{DID: prefix + "0002", Handle: handle("jules-sews"), DisplayName: "Jules Rivera", Description: "Sewing practical clothes from loud fabric.", Crafts: []string{"social.craftsky.feed.defs#sewing"}, AvatarName: "avatar-jules", BannerName: "banner-jules"},
		{DID: prefix + "0003", Handle: handle("nina-quilts"), DisplayName: "Nina Park", Description: "Modern quilts, tiny scraps, big opinions about binding.", Crafts: []string{"social.craftsky.feed.defs#quilting"}, AvatarName: "avatar-nina", BannerName: "banner-nina"},
		{DID: prefix + "0004", Handle: handle("sol-crochets"), DisplayName: "Sol Amari", Description: "Crochet blankets, market bags, and color experiments.", Crafts: []string{"social.craftsky.feed.defs#crochet"}, AvatarName: "avatar-sol", BannerName: "banner-sol"},
		{DID: prefix + "0005", Handle: handle("bea-mends"), DisplayName: "Bea Tan", Description: "Repairs, refashions, and making old garments useful again.", Crafts: []string{"social.craftsky.feed.defs#sewing", "social.craftsky.feed.defs#embroidery"}, AvatarName: "avatar-bea", BannerName: "banner-bea"},
		{DID: prefix + "0006", Handle: handle("ori-fiber"), DisplayName: "Ori Chen", Description: "Spinning, dye notes, and slow wool projects.", Crafts: []string{"social.craftsky.feed.defs#knitting", "social.craftsky.feed.defs#crochet"}, AvatarName: "avatar-ori", BannerName: "banner-ori"},
	}
	profiles = append(profiles, generatedDemoProfiles(seed, prefix, handle)...)
	return profiles
}

func demoRootPosts(seed string, profiles []demoProfile, now time.Time) []demoPost {
	posts := []demoPost{
		{Rkey: "demo-" + seed + "-project-001", AuthorDID: profiles[0].DID, Text: "Finished the moss cardigan after one last blocking session. The sleeves finally settled where I wanted them. #handknit #cardigan", Images: []demoImage{{Name: "knit-cardigan-moss", Alt: "Moss green hand-knit cardigan laid flat on a cream quilt", Width: 1200, Height: 900}}, Tags: []string{"handknit", "cardigan"}, CreatedAt: now.Add(-1 * time.Hour), IndexedAt: now.Add(-59 * time.Minute), Project: &demoProject{CraftType: "social.craftsky.feed.defs#knitting", Status: "social.craftsky.feed.defs#finished", Title: "Moss Cardigan", Duration: "six weeks", PatternName: "Field Notes Cardigan", PatternDifficulty: "social.craftsky.feed.defs#intermediate", PatternDesigner: "Rowan Finch", Materials: []string{"DK wool in moss", "corozo buttons"}, Colors: []string{"green", "cream"}, DesignTags: []string{"social.craftsky.project.defs#minimalist"}, Tags: []string{"handknit", "cardigan", "wool"}, DetailsType: "social.craftsky.project.knitting#details", Details: map[string]any{"$type": "social.craftsky.project.knitting#details", "projectType": "social.craftsky.project.defs#garment", "projectSubtype": "cardigan", "yarnWeight": "social.craftsky.project.defs#dk", "needleSizeMm": "4.0", "finishedSize": "42 in bust"}}},
		{Rkey: "demo-" + seed + "-project-002", AuthorDID: profiles[1].DID, Text: "The linen dress is done and already feels like summer. I used french seams throughout because this fabric frays if you look at it wrong. #memade #sewing", Images: []demoImage{{Name: "linen-dress-rust", Alt: "Rust linen sleeveless dress hanging from a wooden wardrobe", Width: 900, Height: 1200}}, Tags: []string{"memade", "sewing"}, CreatedAt: now.Add(-3 * time.Hour), IndexedAt: now.Add(-2*time.Hour - 58*time.Minute), Project: &demoProject{CraftType: "social.craftsky.feed.defs#sewing", Status: "social.craftsky.feed.defs#finished", Title: "Rust Linen Summer Dress", Duration: "a long weekend", PatternName: "Harbor Dress", PatternDifficulty: "social.craftsky.feed.defs#beginner", PatternDesigner: "North Coast Patterns", Materials: []string{"medium-weight rust linen", "cotton bias tape"}, Colors: []string{"orange", "brown"}, DesignTags: []string{"social.craftsky.project.defs#minimalist"}, Tags: []string{"memade", "linen", "summer-sewing"}, DetailsType: "social.craftsky.project.sewing#details", Details: map[string]any{"$type": "social.craftsky.project.sewing#details", "projectType": "social.craftsky.project.defs#garment", "projectSubtype": "dress", "sizeMade": "12", "fitNotes": "shortened bodice by 1 inch"}}},
		{Rkey: "demo-" + seed + "-project-003", AuthorDID: profiles[2].DID, Text: "Quilt top assembled. I am auditioning border fabrics now and trying not to overthink the navy corner blocks. #quilting #scrapquilt", Images: []demoImage{{Name: "quilt-top-navy", Alt: "Geometric quilt top with navy, cream, and gold blocks on a design wall", Width: 1200, Height: 900}}, Tags: []string{"quilting", "scrapquilt"}, CreatedAt: now.Add(-6 * time.Hour), IndexedAt: now.Add(-5*time.Hour - 55*time.Minute), Project: &demoProject{CraftType: "social.craftsky.feed.defs#quilting", Status: "social.craftsky.feed.defs#wip", Title: "Navy Window Quilt", Duration: "three months so far", PatternName: "self drafted", PatternDifficulty: "social.craftsky.feed.defs#intermediate", Materials: []string{"cotton scraps", "solid navy yardage"}, Colors: []string{"blue", "cream", "gold"}, DesignTags: []string{"social.craftsky.project.defs#geometric", "social.craftsky.project.defs#modern"}, Tags: []string{"quilting", "scrapquilt", "wip"}, DetailsType: "social.craftsky.project.quilting#details", Details: map[string]any{"$type": "social.craftsky.project.quilting#details", "projectType": "social.craftsky.project.defs#quilt", "projectSubtype": "throw", "piecingTechnique": "machine pieced", "quiltingMethod": "not quilted yet", "size": "60 x 72 in"}}},
		{Rkey: "demo-" + seed + "-project-004", AuthorDID: profiles[3].DID, Text: "Started joining the blanket squares. The palette looked chaotic in a pile but calmer once the cream borders went on. #crochet #blanket", Images: []demoImage{{Name: "crochet-blanket-squares", Alt: "Crochet blanket squares in pink, blue, ochre, and cream arranged on a sofa", Width: 1200, Height: 900}}, Tags: []string{"crochet", "blanket"}, CreatedAt: now.Add(-8 * time.Hour), IndexedAt: now.Add(-7*time.Hour - 52*time.Minute), Project: &demoProject{CraftType: "social.craftsky.feed.defs#crochet", Status: "social.craftsky.feed.defs#wip", Title: "Market Day Blanket", Duration: "on and off since spring", PatternName: "Four Round Square", PatternDifficulty: "social.craftsky.feed.defs#beginner", Materials: []string{"worsted cotton", "cream border yarn"}, Colors: []string{"pink", "blue", "yellow", "cream"}, DesignTags: []string{"social.craftsky.project.defs#colorblock", "social.craftsky.project.defs#whimsical"}, Tags: []string{"crochet", "blanket", "granny-square"}, DetailsType: "social.craftsky.project.crochet#details", Details: map[string]any{"$type": "social.craftsky.project.crochet#details", "projectType": "social.craftsky.project.defs#homeGoods", "projectSubtype": "blanket", "yarnWeight": "social.craftsky.project.defs#worsted", "hookSizeMm": "5.0", "finishedSize": "planned throw size"}}},
		{Rkey: "demo-" + seed + "-post-005", AuthorDID: profiles[4].DID, Text: "Spent lunch break reinforcing the inner thighs on two pairs of jeans. Tiny running stitches are meditative when the coffee is good. #mending", Images: []demoImage{{Name: "visible-mending-denim", Alt: "Close view of denim jeans repaired with neat rows of blue sashiko-style stitches", Width: 1200, Height: 900}}, Tags: []string{"mending"}, CreatedAt: now.Add(-10 * time.Hour), IndexedAt: now.Add(-9*time.Hour - 55*time.Minute)},
		{Rkey: "demo-" + seed + "-post-006", AuthorDID: profiles[5].DID, Text: "Dye notebook update: onion skins gave a warmer gold than expected on the alum-mordanted sample. The unmordanted skein is much softer.", Images: []demoImage{{Name: "naturally-dyed-skeins", Alt: "Small wool skeins in cream and golden yellow beside a handwritten dye notebook", Width: 1200, Height: 900}}, Tags: []string{"naturaldye", "wool"}, CreatedAt: now.Add(-13 * time.Hour), IndexedAt: now.Add(-12*time.Hour - 47*time.Minute)},
		{Rkey: "demo-" + seed + "-post-007", AuthorDID: profiles[0].DID, Text: "Swatch math says I need one more repeat than the pattern suggests. Future me is writing that down before optimism interferes.", Tags: []string{"swatching"}, CreatedAt: now.Add(-16 * time.Hour), IndexedAt: now.Add(-15*time.Hour - 30*time.Minute)},
		{Rkey: "demo-" + seed + "-post-008", AuthorDID: profiles[1].DID, Text: "Cutting layout victory: got the facings out of the offcuts without piecing them. This almost never happens.", Tags: []string{"sewing"}, CreatedAt: now.Add(-22 * time.Hour), IndexedAt: now.Add(-21*time.Hour - 45*time.Minute)},
	}
	posts = append(posts, generatedDemoRootPosts(seed, profiles, now)...)
	return posts
}

func demoComments(seed string, profiles []demoProfile, roots []demoPostRef, now time.Time) []demoPost {
	comments := []demoPost{}
	texts := []string{
		"The finishing looks so clean. Did blocking change the length much?",
		"That color combination is excellent. Saving this for palette ideas.",
		"I love seeing the process notes with the finished project.",
		"This makes me want to pull my half-finished version back out.",
		"The material notes are helpful. I always forget to write that part down.",
		"That edge finish is exactly what I was hoping to see close up.",
		"I would not have thought to combine those colors, but it works.",
		"The proportions look balanced. Did you change the original pattern?",
	}
	for i, root := range roots[:min(24, len(roots))] {
		commenter := profiles[(i+1)%len(profiles)]
		rkey := fmt.Sprintf("demo-%s-comment-%02d", seed, i+1)
		createdAt := now.Add(-time.Duration(30+i*7) * time.Minute)
		comments = append(comments, demoPost{Rkey: rkey, AuthorDID: commenter.DID, Text: texts[i%len(texts)], ReplyRootURI: root.URI, ReplyRootCID: root.CID, ReplyParentURI: root.URI, ReplyParentCID: root.CID, Tags: []string{}, CreatedAt: createdAt, IndexedAt: createdAt.Add(time.Minute)})
		if i%2 == 0 {
			replier := profiles[(i+2)%len(profiles)]
			commentURI := postURI(commenter.DID, rkey)
			commentCID := fakeCID("demo-post", commentURI)
			replyAt := createdAt.Add(8 * time.Minute)
			comments = append(comments, demoPost{Rkey: fmt.Sprintf("demo-%s-reply-%02d", seed, i+1), AuthorDID: replier.DID, Text: "I changed one small thing and immediately became convinced I should do that every time.", ReplyRootURI: root.URI, ReplyRootCID: root.CID, ReplyParentURI: commentURI, ReplyParentCID: commentCID, Tags: []string{}, CreatedAt: replyAt, IndexedAt: replyAt.Add(time.Minute)})
		}
	}
	return comments
}

func generatedDemoProfiles(seed, prefix string, handle func(string) string) []demoProfile {
	names := []struct {
		slug        string
		displayName string
		description string
		crafts      []string
	}{
		{"anya-stitches", "Anya Brooks", "Daily stitching notes, garment tweaks, and small-batch experiments.", []string{"social.craftsky.feed.defs#sewing", "social.craftsky.feed.defs#embroidery"}},
		{"caleb-knits", "Caleb Stone", "Cables, socks, and practical wool things for cold mornings.", []string{"social.craftsky.feed.defs#knitting"}},
		{"dina-crochet", "Dina Morris", "Crochet home goods with too many color charts.", []string{"social.craftsky.feed.defs#crochet"}},
		{"emi-quilts", "Emi Sato", "Scrap quilts, hand quilting, and tiny block experiments.", []string{"social.craftsky.feed.defs#quilting"}},
		{"felix-makes", "Felix Hart", "Sewing, mending, and trying to use the fabric already on the shelf.", []string{"social.craftsky.feed.defs#sewing"}},
		{"greta-fiber", "Greta Novak", "Natural dye samples and slow knitting projects.", []string{"social.craftsky.feed.defs#knitting"}},
		{"hana-hooks", "Hana Patel", "Crochet bags, blankets, and occasionally very small frogs.", []string{"social.craftsky.feed.defs#crochet"}},
		{"isla-pieces", "Isla Romero", "Quilt blocks, color studies, and machine binding practice.", []string{"social.craftsky.feed.defs#quilting"}},
		{"jo-tailors", "Jo Kim", "Fit notes, trouser adjustments, and wearable muslins.", []string{"social.craftsky.feed.defs#sewing"}},
		{"kai-wool", "Kai Morgan", "Sweaters, spinning notes, and gauge honesty.", []string{"social.craftsky.feed.defs#knitting"}},
		{"lena-loops", "Lena Ortiz", "Crochet texture tests and sturdy household projects.", []string{"social.craftsky.feed.defs#crochet"}},
		{"mina-patches", "Mina Ali", "Visible mending, patchwork repairs, and denim experiments.", []string{"social.craftsky.feed.defs#sewing", "social.craftsky.feed.defs#quilting"}},
		{"noor-thread", "Noor Haddad", "Embroidery details on handmade clothes.", []string{"social.craftsky.feed.defs#embroidery", "social.craftsky.feed.defs#sewing"}},
		{"otto-blocks", "Otto Meyer", "Traditional quilt blocks in loud modern colors.", []string{"social.craftsky.feed.defs#quilting"}},
		{"piper-purls", "Piper Evans", "Knitting socks, mitts, and anything that fits in a project bag.", []string{"social.craftsky.feed.defs#knitting"}},
		{"quinn-cuts", "Quinn Shaw", "Pattern hacking and careful cutting layouts.", []string{"social.craftsky.feed.defs#sewing"}},
		{"rhea-yarns", "Rhea Singh", "Crochet garments and yarn substitution notes.", []string{"social.craftsky.feed.defs#crochet"}},
		{"sam-seams", "Sam Carter", "Workwear sewing and repair notes.", []string{"social.craftsky.feed.defs#sewing"}},
		{"tess-quilts", "Tess Green", "Hand quilting, applique, and border indecision.", []string{"social.craftsky.feed.defs#quilting"}},
		{"uma-fiber", "Uma Wells", "Wool preparation, dye pots, and quiet knitting.", []string{"social.craftsky.feed.defs#knitting"}},
		{"vera-craft", "Vera Lin", "Whatever craft is currently taking over the table.", []string{"social.craftsky.feed.defs#knitting", "social.craftsky.feed.defs#crochet", "social.craftsky.feed.defs#sewing"}},
		{"willow-mends", "Willow Fox", "Mending baskets, practical repairs, and saved favorites.", []string{"social.craftsky.feed.defs#sewing"}},
		{"xavi-scraps", "Xavi Reed", "Scrap management disguised as quilt design.", []string{"social.craftsky.feed.defs#quilting"}},
		{"yara-needle", "Yara Bell", "Small needlework, tiny motifs, and garment embellishment.", []string{"social.craftsky.feed.defs#embroidery"}},
	}
	out := make([]demoProfile, 0, len(names))
	for i, name := range names {
		out = append(out, demoProfile{
			DID:         fmt.Sprintf("%s%04d", prefix, i+7),
			Handle:      handle(name.slug),
			DisplayName: name.displayName,
			Description: name.description,
			Crafts:      name.crafts,
			AvatarName:  fmt.Sprintf("avatar-%s", name.slug),
			BannerName:  fmt.Sprintf("banner-%s", name.slug),
		})
	}
	return out
}

func generatedDemoRootPosts(seed string, profiles []demoProfile, now time.Time) []demoPost {
	texts := []string{
		"Blocked the swatch and the fabric relaxed more than expected. I am glad I checked before casting on the full piece. #swatching",
		"Pressed every seam before moving on and the whole project behaved better. Annoying when the obvious advice is right. #sewing",
		"Laid out the next twelve quilt blocks tonight. The low-volume prints are doing more work than I expected. #quilting",
		"Trying a smaller hook for the border because the first version flared at the corners. #crochet",
		"Repaired one cuff and found two more tiny worn spots while I was there. The mending pile negotiates aggressively. #mending",
		"Made a project note card before putting this away so I do not have to decode my own choices next month.",
		"The contrast thread felt too bold on the spool but exactly right once stitched. #embroidery",
		"I used the last of this fabric for pocket bags, which feels like a tiny household victory. #memade",
	}
	tags := [][]string{
		{"swatching", "knitting"},
		{"sewing", "pressing"},
		{"quilting", "blocks"},
		{"crochet", "border"},
		{"mending", "repair"},
		{"project-notes"},
		{"embroidery", "details"},
		{"memade", "stash"},
	}
	posts := make([]demoPost, 0, 40)
	for i := 1; i <= 40; i++ {
		author := profiles[(i+5)%len(profiles)]
		createdAt := now.Add(-time.Duration(24+i*2) * time.Hour)
		post := demoPost{
			Rkey:      fmt.Sprintf("demo-%s-generated-%03d", seed, i),
			AuthorDID: author.DID,
			Text:      texts[(i-1)%len(texts)],
			Tags:      tags[(i-1)%len(tags)],
			CreatedAt: createdAt,
			IndexedAt: createdAt.Add(time.Duration(i%11) * time.Minute),
		}
		if i%6 == 0 {
			post.Images = []demoImage{{Name: fmt.Sprintf("generated-project-%02d", i), Alt: generatedImageAlt(i), Width: 1200, Height: 900}}
		}
		if i%5 == 0 {
			post.Project = generatedDemoProject(i)
			post.Text = generatedProjectText(i)
			post.Tags = post.Project.Tags
			if len(post.Images) == 0 {
				post.Images = []demoImage{{Name: fmt.Sprintf("generated-project-%02d", i), Alt: generatedImageAlt(i), Width: 1200, Height: 900}}
			}
		}
		posts = append(posts, post)
	}
	return posts
}

func generatedDemoProject(i int) *demoProject {
	switch i % 4 {
	case 0:
		return &demoProject{CraftType: "social.craftsky.feed.defs#knitting", Status: "social.craftsky.feed.defs#wip", Title: fmt.Sprintf("Everyday Pullover %02d", i), Duration: "two weeks so far", PatternName: "Workshop Pullover", PatternDifficulty: "social.craftsky.feed.defs#intermediate", PatternDesigner: "Studio Thread", Materials: []string{"worsted wool", "stitch markers"}, Colors: []string{"blue", "gray"}, DesignTags: []string{"social.craftsky.project.defs#minimalist"}, Tags: []string{"knitting", "sweater", "wip"}, DetailsType: "social.craftsky.project.knitting#details", Details: map[string]any{"$type": "social.craftsky.project.knitting#details", "projectType": "social.craftsky.project.defs#garment", "projectSubtype": "pullover", "yarnWeight": "social.craftsky.project.defs#worsted", "needleSizeMm": "4.5"}}
	case 1:
		return &demoProject{CraftType: "social.craftsky.feed.defs#sewing", Status: "social.craftsky.feed.defs#finished", Title: fmt.Sprintf("Utility Shirt %02d", i), Duration: "three evenings", PatternName: "Box Shirt", PatternDifficulty: "social.craftsky.feed.defs#beginner", PatternDesigner: "Practical Patterns", Materials: []string{"cotton twill", "recycled buttons"}, Colors: []string{"white", "blue"}, DesignTags: []string{"social.craftsky.project.defs#stripes"}, Tags: []string{"sewing", "shirt", "memade"}, DetailsType: "social.craftsky.project.sewing#details", Details: map[string]any{"$type": "social.craftsky.project.sewing#details", "projectType": "social.craftsky.project.defs#garment", "projectSubtype": "shirt", "sizeMade": "M"}}
	case 2:
		return &demoProject{CraftType: "social.craftsky.feed.defs#quilting", Status: "social.craftsky.feed.defs#wip", Title: fmt.Sprintf("Scrap Study %02d", i), Duration: "a few weekends", PatternName: "nine patch variation", PatternDifficulty: "social.craftsky.feed.defs#beginner", Materials: []string{"cotton scraps", "neutral background"}, Colors: []string{"multicolor", "cream"}, DesignTags: []string{"social.craftsky.project.defs#geometric"}, Tags: []string{"quilting", "scraps", "wip"}, DetailsType: "social.craftsky.project.quilting#details", Details: map[string]any{"$type": "social.craftsky.project.quilting#details", "projectType": "social.craftsky.project.defs#quilt", "projectSubtype": "wall hanging", "piecingTechnique": "machine pieced"}}
	default:
		return &demoProject{CraftType: "social.craftsky.feed.defs#crochet", Status: "social.craftsky.feed.defs#finished", Title: fmt.Sprintf("Market Bag %02d", i), Duration: "one weekend", PatternName: "mesh tote", PatternDifficulty: "social.craftsky.feed.defs#beginner", Materials: []string{"cotton yarn"}, Colors: []string{"green", "natural"}, DesignTags: []string{"social.craftsky.project.defs#minimalist"}, Tags: []string{"crochet", "bag", "cotton"}, DetailsType: "social.craftsky.project.crochet#details", Details: map[string]any{"$type": "social.craftsky.project.crochet#details", "projectType": "social.craftsky.project.defs#accessory", "projectSubtype": "bag", "yarnWeight": "social.craftsky.project.defs#worsted", "hookSizeMm": "4.5"}}
	}
}

func generatedProjectText(i int) string {
	return fmt.Sprintf("Project update %02d: made enough progress to write down the changes while they still make sense. The material choice is doing most of the work. #craftsky", i)
}

func generatedImageAlt(i int) string {
	return fmt.Sprintf("Generated demo craft project placeholder image %02d", i)
}

func demoDIDPrefix(seed string) string {
	return "did:plc:craftskydemo" + seed
}

func localDemoHandle(did string) string {
	parts := strings.Split(did, ":")
	last := strings.ToLower(parts[len(parts)-1])
	if len(last) > 36 {
		last = last[:36]
	}
	return last + ".craftsky.test"
}

func devMediaCID(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return "devmedia:" + name
}

func nullableRaw(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func printDemoSeedStats(out io.Writer, stats demoSeedStats) {
	fmt.Fprintf(out, "seeded demo data: profiles=%d follows=%d posts=%d projects=%d comments=%d likes=%d reposts=%d deleted=%d\n",
		stats.Profiles, stats.Follows, stats.Posts, stats.Projects, stats.Comments, stats.Likes, stats.Reposts, stats.Deleted)
}
