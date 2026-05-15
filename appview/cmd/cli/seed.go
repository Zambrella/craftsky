package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/app"
)

const fakePostCollection = "social.craftsky.feed.post"

type fakePostSeedArgs struct {
	UserDID           string
	RootPosts         int
	CommentsPerPost   int
	RepliesPerComment int
	Commenters        int
	Reset             bool
	Seed              string
	Since             time.Duration
	Now               time.Time
}

type fakeSeedStats struct {
	Profiles int
	Roots    int
	Comments int
	Replies  int
	Deleted  int64
}

var fakeSeedFlags = fakePostSeedArgs{
	RootPosts:         30,
	CommentsPerPost:   8,
	RepliesPerComment: 3,
	Commenters:        12,
	Seed:              "ui",
	Since:             14 * 24 * time.Hour,
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Populate local development data",
}

var seedFakePostsCmd = &cobra.Command{
	Use:   "fake-posts --user DID",
	Short: "Seed fake posts, comments, and replies for UI development",
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := parseEnvFlag()
		if err != nil {
			return err
		}
		if env != app.EnvDev {
			return fmt.Errorf("seed fake-posts is dev-only; refusing to run with --env %s", env)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
		defer cancel()

		deps, cleanup, err := loadDeps(ctx)
		if err != nil {
			return err
		}
		defer cleanup()

		seedArgs := fakeSeedFlags
		seedArgs.Now = time.Now().UTC()
		stats, err := runFakePostSeed(ctx, deps.DB, seedArgs)
		if err != nil {
			return err
		}
		printFakeSeedStats(os.Stdout, stats)
		return nil
	},
}

func init() {
	seedFakePostsCmd.Flags().StringVar(&fakeSeedFlags.UserDID, "user", "", "DID to seed root posts around")
	seedFakePostsCmd.Flags().IntVar(&fakeSeedFlags.RootPosts, "posts", fakeSeedFlags.RootPosts, "root posts to create")
	seedFakePostsCmd.Flags().IntVar(&fakeSeedFlags.CommentsPerPost, "comments", fakeSeedFlags.CommentsPerPost, "comments per root post")
	seedFakePostsCmd.Flags().IntVar(&fakeSeedFlags.RepliesPerComment, "replies", fakeSeedFlags.RepliesPerComment, "replies per comment")
	seedFakePostsCmd.Flags().IntVar(&fakeSeedFlags.Commenters, "commenters", fakeSeedFlags.Commenters, "synthetic commenter profiles to create")
	seedFakePostsCmd.Flags().BoolVar(&fakeSeedFlags.Reset, "reset", false, "delete previous fake data for this seed before inserting")
	seedFakePostsCmd.Flags().StringVar(&fakeSeedFlags.Seed, "seed", fakeSeedFlags.Seed, "deterministic fake-data namespace")
	seedFakePostsCmd.Flags().DurationVar(&fakeSeedFlags.Since, "since", fakeSeedFlags.Since, "spread timestamps back from now")
	_ = seedFakePostsCmd.MarkFlagRequired("user")

	seedCmd.AddCommand(seedFakePostsCmd)
	rootCmd.AddCommand(seedCmd)
}

func runFakePostSeed(ctx context.Context, pool *pgxpool.Pool, args fakePostSeedArgs) (fakeSeedStats, error) {
	if pool == nil {
		return fakeSeedStats{}, fmt.Errorf("db pool is required")
	}
	if args.Now.IsZero() {
		args.Now = time.Now().UTC()
	}
	if err := validateFakePostSeedArgs(args); err != nil {
		return fakeSeedStats{}, err
	}
	userDID, _ := syntax.ParseDID(args.UserDID)
	seed, err := normalizeFakeSeed(args.Seed)
	if err != nil {
		return fakeSeedStats{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fakeSeedStats{}, fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stats := fakeSeedStats{}
	if args.Reset {
		deleted, err := resetFakeSeed(ctx, tx, seed)
		if err != nil {
			return fakeSeedStats{}, err
		}
		stats.Deleted = deleted
	}

	if err := upsertSeedProfile(ctx, tx, userDID.String(), displayNameForDID(userDID.String()), false); err != nil {
		return fakeSeedStats{}, err
	}
	stats.Profiles++

	commenterDIDs := make([]string, 0, args.Commenters)
	for i := 1; i <= args.Commenters; i++ {
		did := fakeCommenterDID(seed, i)
		if _, err := syntax.ParseDID(did); err != nil {
			return fakeSeedStats{}, fmt.Errorf("build fake commenter DID %q: %w", did, err)
		}
		commenterDIDs = append(commenterDIDs, did)
		if err := upsertSeedProfile(ctx, tx, did, fmt.Sprintf("Fake Crafter %02d", i), true); err != nil {
			return fakeSeedStats{}, err
		}
		stats.Profiles++
	}

	batch := &pgx.Batch{}
	totalPosts := args.RootPosts * (1 + args.CommentsPerPost + args.CommentsPerPost*args.RepliesPerComment)
	step := args.Since / time.Duration(max(1, totalPosts))
	if step <= 0 {
		step = time.Second
	}
	sequence := 0

	for postIndex := 1; postIndex <= args.RootPosts; postIndex++ {
		rootRkey := fmt.Sprintf("fake-%s-root-%04d", seed, postIndex)
		rootURI := postURI(userDID.String(), rootRkey)
		rootCID := fakeCID(seed, rootRkey)
		rootCreatedAt := seededCreatedAt(args.Now, step, sequence)
		sequence++
		queuePostUpsert(batch, seedPostRow{
			URI:       rootURI,
			DID:       userDID.String(),
			Rkey:      rootRkey,
			CID:       rootCID,
			Text:      rootPostText(postIndex),
			CreatedAt: rootCreatedAt,
			IndexedAt: rootCreatedAt.Add(time.Duration(postIndex%7) * time.Second),
		})
		stats.Roots++

		for commentIndex := 1; commentIndex <= args.CommentsPerPost; commentIndex++ {
			commenter := commenterDIDs[(postIndex+commentIndex-2)%len(commenterDIDs)]
			commentRkey := fmt.Sprintf("fake-%s-comment-%04d-%04d", seed, postIndex, commentIndex)
			commentURI := postURI(commenter, commentRkey)
			commentCID := fakeCID(seed, commentRkey)
			commentCreatedAt := seededCreatedAt(args.Now, step, sequence)
			sequence++
			queuePostUpsert(batch, seedPostRow{
				URI:            commentURI,
				DID:            commenter,
				Rkey:           commentRkey,
				CID:            commentCID,
				Text:           commentText(postIndex, commentIndex),
				ReplyRootURI:   rootURI,
				ReplyRootCID:   rootCID,
				ReplyParentURI: rootURI,
				ReplyParentCID: rootCID,
				CreatedAt:      commentCreatedAt,
				IndexedAt:      commentCreatedAt.Add(time.Duration(commentIndex%5) * time.Second),
			})
			stats.Comments++

			parentURI := commentURI
			parentCID := commentCID
			for replyIndex := 1; replyIndex <= args.RepliesPerComment; replyIndex++ {
				replier := commenterDIDs[(postIndex+commentIndex+replyIndex-2)%len(commenterDIDs)]
				replyRkey := fmt.Sprintf("fake-%s-reply-%04d-%04d-%04d", seed, postIndex, commentIndex, replyIndex)
				replyURI := postURI(replier, replyRkey)
				replyCID := fakeCID(seed, replyRkey)
				replyCreatedAt := seededCreatedAt(args.Now, step, sequence)
				sequence++
				queuePostUpsert(batch, seedPostRow{
					URI:            replyURI,
					DID:            replier,
					Rkey:           replyRkey,
					CID:            replyCID,
					Text:           replyText(postIndex, commentIndex, replyIndex),
					ReplyRootURI:   rootURI,
					ReplyRootCID:   rootCID,
					ReplyParentURI: parentURI,
					ReplyParentCID: parentCID,
					CreatedAt:      replyCreatedAt,
					IndexedAt:      replyCreatedAt.Add(time.Duration(replyIndex%3) * time.Second),
				})
				stats.Replies++
				parentURI = replyURI
				parentCID = replyCID
			}
		}
	}

	results := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return fakeSeedStats{}, fmt.Errorf("insert fake post %d: %w", i+1, err)
		}
	}
	if err := results.Close(); err != nil {
		return fakeSeedStats{}, fmt.Errorf("insert fake posts: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fakeSeedStats{}, fmt.Errorf("commit seed transaction: %w", err)
	}
	return stats, nil
}

func validateFakePostSeedArgs(args fakePostSeedArgs) error {
	if _, err := syntax.ParseDID(args.UserDID); err != nil {
		return fmt.Errorf("--user must be a DID: %w", err)
	}
	if args.RootPosts <= 0 {
		return fmt.Errorf("--posts must be positive")
	}
	if args.CommentsPerPost < 0 {
		return fmt.Errorf("--comments must be zero or positive")
	}
	if args.RepliesPerComment < 0 {
		return fmt.Errorf("--replies must be zero or positive")
	}
	if args.Commenters <= 0 {
		return fmt.Errorf("--commenters must be positive")
	}
	if args.Since <= 0 {
		return fmt.Errorf("--since must be positive")
	}
	_, err := normalizeFakeSeed(args.Seed)
	return err
}

var fakeSeedPattern = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeFakeSeed(seed string) (string, error) {
	seed = strings.ToLower(strings.TrimSpace(seed))
	seed = fakeSeedPattern.ReplaceAllString(seed, "")
	if seed == "" {
		return "", fmt.Errorf("--seed must contain at least one letter or digit")
	}
	if len(seed) > 24 {
		seed = seed[:24]
	}
	return seed, nil
}

func resetFakeSeed(ctx context.Context, tx pgx.Tx, seed string) (int64, error) {
	postTag := "fake-" + seed + "-%"
	tag, err := tx.Exec(ctx, `DELETE FROM craftsky_posts WHERE rkey LIKE $1`, postTag)
	if err != nil {
		return 0, fmt.Errorf("reset fake posts: %w", err)
	}

	didTag := "did:plc:craftskyfake" + seed + "%"
	if _, err := tx.Exec(ctx, `
		DELETE FROM bluesky_profiles bp
		WHERE bp.did LIKE $1
		  AND NOT EXISTS (SELECT 1 FROM craftsky_posts p WHERE p.did = bp.did)
	`, didTag); err != nil {
		return 0, fmt.Errorf("reset fake bluesky profiles: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM craftsky_profiles cp
		WHERE cp.did LIKE $1
		  AND NOT EXISTS (SELECT 1 FROM craftsky_posts p WHERE p.did = cp.did)
	`, didTag); err != nil {
		return 0, fmt.Errorf("reset fake craftsky profiles: %w", err)
	}
	return tag.RowsAffected(), nil
}

func upsertSeedProfile(ctx context.Context, tx pgx.Tx, did, displayName string, overwriteBsky bool) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid, created_at, indexed_at)
		VALUES ($1, ARRAY['knitting', 'sewing', 'crochet'], $2, now(), now())
		ON CONFLICT (did) DO NOTHING
	`, did, fakeCID("profile", did)); err != nil {
		return fmt.Errorf("upsert craftsky profile %s: %w", did, err)
	}

	conflict := "DO NOTHING"
	if overwriteBsky {
		conflict = `DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			record_cid = EXCLUDED.record_cid,
			indexed_at = now()`
	}
	q := `
		INSERT INTO bluesky_profiles (did, display_name, description, record_cid, indexed_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (did) ` + conflict
	if _, err := tx.Exec(ctx, q, did, displayName, "Fake Craftsky UI seed profile.", fakeCID("bskyprofile", did)); err != nil {
		return fmt.Errorf("upsert bluesky profile %s: %w", did, err)
	}
	return nil
}

type seedPostRow struct {
	URI            string
	DID            string
	Rkey           string
	CID            string
	Text           string
	ReplyRootURI   string
	ReplyRootCID   string
	ReplyParentURI string
	ReplyParentCID string
	CreatedAt      time.Time
	IndexedAt      time.Time
}

func queuePostUpsert(batch *pgx.Batch, row seedPostRow) {
	record, err := json.Marshal(map[string]any{
		"$type":     fakePostCollection,
		"text":      row.Text,
		"createdAt": row.CreatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		panic(err)
	}
	batch.Queue(`
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, facets, images,
			 reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid,
			 quote_uri, quote_cid, tags, record, created_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5, NULL, NULL,
			NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''),
			NULL, NULL, ARRAY[]::text[], $10, $11, $12)
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
			record = EXCLUDED.record,
			created_at = EXCLUDED.created_at,
			indexed_at = EXCLUDED.indexed_at
	`, row.URI, row.DID, row.Rkey, row.CID, row.Text,
		row.ReplyRootURI, row.ReplyRootCID, row.ReplyParentURI, row.ReplyParentCID,
		record, row.CreatedAt.UTC(), row.IndexedAt.UTC())
}

func seededCreatedAt(now time.Time, step time.Duration, sequence int) time.Time {
	return now.Add(-step * time.Duration(sequence)).UTC()
}

func postURI(did, rkey string) string {
	return "at://" + did + "/" + fakePostCollection + "/" + rkey
}

func fakeCommenterDID(seed string, index int) string {
	return fmt.Sprintf("did:plc:craftskyfake%s%04d", seed, index)
}

func fakeCID(parts ...string) string {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("bafyfake%016x", h.Sum64())
}

func displayNameForDID(did string) string {
	parts := strings.Split(did, ":")
	last := parts[len(parts)-1]
	if len(last) > 12 {
		last = last[:12]
	}
	return "Seeded " + last
}

func rootPostText(i int) string {
	texts := []string{
		"Finished weaving in the ends and blocking this cardigan. The drape changed completely once it dried.",
		"Trying a new sleeve adjustment today. I marked the muslin heavily so I can compare both sides later.",
		"The quilt top is finally assembled. Next step is deciding whether the border needs one more contrast fabric.",
		"Swatching with two needle sizes before committing to the full sweater. The smaller gauge looks cleaner so far.",
		"Cutting project bags from leftover canvas scraps. The zipper color accidentally became the best detail.",
		"Mending a stack of jeans this afternoon. Visible patches make the old pairs feel intentional again.",
		"Started a lace repeat that looks chaotic for the first six rows and then suddenly makes sense.",
		"Testing thread tension on a slippery lining fabric before sewing the actual garment pieces.",
	}
	return texts[(i-1)%len(texts)]
}

func commentText(postIndex, commentIndex int) string {
	texts := []string{
		"That texture is lovely. What fiber did you use?",
		"The color choice really makes the stitch pattern stand out.",
		"I have been meaning to try this technique. Saving this for later.",
		"Did you change the pattern at the neckline? It sits really well.",
		"The finish looks so clean. Nice work on the details.",
		"I like how practical this is without looking plain.",
		"This makes me want to pull my half-finished version back out.",
		"Those seams look crisp. Did you press as you went?",
	}
	return texts[(postIndex+commentIndex-2)%len(texts)]
}

func replyText(postIndex, commentIndex, replyIndex int) string {
	texts := []string{
		"I used stash yarn and held two strands together for most of it.",
		"Yes, I shortened that section and moved the shaping up by about an inch.",
		"Pressing every step helped more than I expected.",
		"The tricky part was keeping the edges from stretching while finishing.",
		"I wrote notes this time so I can repeat it without guessing.",
		"The next version will probably use a quieter contrast color.",
	}
	return texts[(postIndex+commentIndex+replyIndex-3)%len(texts)]
}

func printFakeSeedStats(out io.Writer, stats fakeSeedStats) {
	fmt.Fprintf(out, "seeded fake posts: profiles=%d roots=%d comments=%d replies=%d deleted=%d\n",
		stats.Profiles, stats.Roots, stats.Comments, stats.Replies, stats.Deleted)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
