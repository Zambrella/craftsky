package video

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

var ErrBunnyUnavailable = errors.New("video service unavailable")
var ErrVideoJobNotFound = errors.New("video job not found")
var ErrInvalidVideo = errors.New("invalid video media")

var ErrVideoQuota = errors.New("video upload quota exhausted")

type BunnyBlobEffects interface {
	UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error)
}
type BunnyEffectFactory func(context.Context, syntax.DID, string) (BunnyBlobEffects, error)

type activeBunnyJob struct {
	owner  syntax.DID
	cancel context.CancelFunc
}

type Bunny struct {
	active               sync.Map
	sessions             auth.OAuthSessionRunner
	pool                 *pgxpool.Pool
	library, apiKey, cdn string
	client               *http.Client
	effects              BunnyEffectFactory
	slots                chan struct{}
	fallback             *PlaybackURLBuilder
}

// NewBunny uses fixed vendor endpoints, never client-supplied destinations.
func NewBunny(pool *pgxpool.Pool, library, key, cdn string, effects BunnyEffectFactory, fallback *PlaybackURLBuilder, sessions ...auth.OAuthSessionRunner) (*Bunny, error) {
	if _, err := strconv.ParseUint(library, 10, 64); err != nil || library == "0" {
		return nil, errors.New("invalid Bunny library ID")
	}
	u, err := url.Parse("https://" + cdn)
	if err != nil || u.Hostname() != cdn || !strings.HasSuffix(cdn, ".b-cdn.net") || u.Port() != "" || key == "" || pool == nil || effects == nil {
		return nil, errors.New("invalid Bunny configuration")
	}
	var runner auth.OAuthSessionRunner
	if len(sessions) > 0 {
		runner = sessions[0]
	}
	return &Bunny{sessions: runner, pool: pool, library: library, apiKey: key, cdn: cdn, effects: effects, slots: make(chan struct{}, 1), fallback: fallback, client: &http.Client{Timeout: 90 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (b *Bunny) api(ctx context.Context, method, path string, input, output any) error {
	return b.libraryAPI(ctx, method, "/videos"+path, input, output)
}

func (b *Bunny) libraryAPI(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://video.bunnycdn.com/library/"+b.library+path, body)
	if err != nil {
		return ErrBunnyUnavailable
	}
	req.Header.Set("AccessKey", b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return ErrBunnyUnavailable
	}
	defer resp.Body.Close()
	if method == "DELETE" && resp.StatusCode == 404 {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrBunnyUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(data) > 65536 {
		return ErrBunnyUnavailable
	}
	if output != nil && json.Unmarshal(data, output) != nil {
		return ErrBunnyUnavailable
	}
	return nil
}

// Allocation and per-owner quota are serialized before the billable vendor call.
func (b *Bunny) IssueUpload(ctx context.Context, owner syntax.DID, session string) (UploadAuthorization, error) {
	if b.sessions != nil {
		err := b.sessions.WithActiveSession(ctx, owner, session, func(context.Context, *oauth.ClientSession) error { return nil })
		if err != nil {
			return UploadAuthorization{}, err
		}
	}
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return UploadAuthorization{}, err
	}
	defer tx.Rollback(ctx)
	var generation int64
	if err = tx.QueryRow(ctx, "SELECT generation FROM owner_lifecycles WHERE owner_did=$1 AND state='active' FOR UPDATE", owner).Scan(&generation); err != nil {
		return UploadAuthorization{}, ErrVideoJobNotFound
	}
	var count int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM bunny_video_jobs WHERE owner_did=$1 AND created_at>now()-interval '24 hours'", owner).Scan(&count); err != nil {
		return UploadAuthorization{}, err
	}
	if count >= 20 {
		return UploadAuthorization{}, ErrVideoQuota
	}
	job := uuid.NewString()
	_, err = tx.Exec(ctx, "INSERT INTO bunny_video_jobs(id,owner_did,owner_generation,library_id) VALUES($1,$2,$3,$4)", job, owner, generation, b.library)
	if err != nil {
		return UploadAuthorization{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return UploadAuthorization{}, err
	}
	allocation, err := b.createOrganizedVideo(ctx, owner, job, time.Now())
	if err != nil {
		return UploadAuthorization{}, err
	}
	if _, err = b.pool.Exec(ctx, "UPDATE bunny_video_jobs SET video_id=$2,updated_at=now() WHERE id=$1", job, allocation.ID); err != nil {
		return UploadAuthorization{}, err
	}
	if err = b.setVideoMetadata(ctx, allocation.ID, job, "upload", ""); err != nil {
		return UploadAuthorization{}, err
	}
	expires := time.Now().UTC().Add(30 * time.Minute)
	digest := sha256.Sum256([]byte(b.library + b.apiKey + strconv.FormatInt(expires.Unix(), 10) + allocation.ID))
	return UploadAuthorization{Token: hex.EncodeToString(digest[:]), ExpiresAt: expires, Provider: "bunny", JobID: job, VideoID: allocation.ID, LibraryID: b.library, Title: allocation.Title, CollectionID: allocation.CollectionID}, nil
}

func (b *Bunny) Get(ctx context.Context, owner syntax.DID, session string) (UploadLimits, error) {
	var count int64
	err := b.pool.QueryRow(ctx, "SELECT count(*) FROM bunny_video_jobs WHERE owner_did=$1 AND created_at>now()-interval '24 hours'", owner).Scan(&count)
	if err != nil {
		return UploadLimits{}, err
	}
	remaining := max(int64(0), 20-count)
	result := UploadLimits{CanUpload: remaining > 0, RemainingDailyVideos: &remaining}
	if remaining == 0 {
		result.Reason = "quota_exhausted"
	}
	return result, nil
}

type BunnyJobStatus struct {
	JobID    string         `json:"jobId"`
	State    string         `json:"state"`
	Progress int            `json:"progress"`
	Blob     map[string]any `json:"blob,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type bunnyJob struct {
	createdAt         time.Time
	imported          bool
	video, state, cid string
	failureCode       string
	size, generation  int64
}

func (b *Bunny) job(ctx context.Context, owner syntax.DID, id string) (bunnyJob, error) {
	var j bunnyJob
	if _, err := uuid.Parse(id); err != nil {
		return j, ErrVideoJobNotFound
	}
	err := b.pool.QueryRow(ctx, `SELECT COALESCE(video_id::text,''),j.state,COALESCE(blob_cid,''),COALESCE(blob_size,0),owner_generation,j.imported,j.created_at,j.failure_code
 FROM bunny_video_jobs j JOIN owner_lifecycles o ON o.owner_did=j.owner_did
 WHERE j.id=$1 AND j.owner_did=$2 AND j.library_id=$3 AND o.state='active' AND o.generation=j.owner_generation`, id, owner, b.library).Scan(&j.video, &j.state, &j.cid, &j.size, &j.generation, &j.imported, &j.createdAt, &j.failureCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return j, ErrVideoJobNotFound
	}
	return j, err
}
func completedBunnyJob(id string, j bunnyJob) BunnyJobStatus {
	return BunnyJobStatus{JobID: id, State: "JOB_STATE_COMPLETED", Progress: 100, Blob: map[string]any{"$type": "blob", "ref": map[string]string{"$link": j.cid}, "mimeType": "video/mp4", "size": j.size}}
}

// Status is a POST: when encoding finishes it may perform the durable PDS handoff.
// One local spool and a database lease bound resource use and serialize retries.
func (b *Bunny) Status(ctx context.Context, owner syntax.DID, session, id string) (BunnyJobStatus, error) {
	pending := BunnyJobStatus{JobID: id, State: "JOB_STATE_PROCESSING"}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	running := &activeBunnyJob{owner: owner, cancel: cancel}
	if value, loaded := b.active.LoadOrStore(id, running); loaded {
		if value.(*activeBunnyJob).owner != owner {
			return pending, ErrVideoJobNotFound
		}
		return pending, nil
	}
	defer b.active.CompareAndDelete(id, running)
	j, err := b.job(ctx, owner, id)
	if err != nil {
		return pending, err
	}
	if j.state == "ready" {
		return completedBunnyJob(id, j), nil
	}
	if j.state == "failed" || j.state == "deleted" || j.state == "deleting" || j.state == "purging" {
		pending.State = "JOB_STATE_FAILED"
		pending.Error = j.failureCode
		if pending.Error == "" {
			pending.Error = "invalid_video"
		}
		return pending, nil
	}
	if j.video == "" {
		return pending, nil
	}
	var remote struct {
		GUID        string `json:"guid"`
		Library     int64  `json:"videoLibraryId"`
		Status      int    `json:"status"`
		Progress    int    `json:"encodeProgress"`
		Length      int    `json:"length"`
		MP4         bool   `json:"hasMP4Fallback"`
		Resolutions string `json:"availableResolutions"`
	}
	if err = b.api(ctx, "GET", "/"+j.video, nil, &remote); err != nil {
		return pending, err
	}
	if remote.GUID != j.video || strconv.FormatInt(remote.Library, 10) != b.library {
		return pending, ErrBunnyUnavailable
	}
	pending.Progress = min(99, max(0, remote.Progress))
	if remote.Status == 5 || remote.Status == 6 || remote.Length > 600 {
		_, err = b.pool.Exec(ctx, "UPDATE bunny_video_jobs SET state='failed',updated_at=now() WHERE id=$1", id)
		pending.State = "JOB_STATE_FAILED"
		pending.Error = "invalid_video"
		return pending, err
	}
	if remote.Status != 4 || !remote.MP4 {
		return pending, nil
	}
	if j.imported {
		_, err = b.pool.Exec(ctx, "UPDATE bunny_video_jobs SET state='ready',updated_at=now() WHERE id=$1 AND state='created' AND cleanup_requested_at IS NULL", id)
		return completedBunnyJob(id, j), err
	}
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
	default:
		return pending, nil
	}
	claim, err := b.pool.Exec(ctx, `UPDATE bunny_video_jobs SET lease_until=now()+interval '8 minutes' WHERE id=$1 AND state='created' AND cleanup_requested_at IS NULL AND lease_until<now()`, id)
	if err != nil {
		return pending, err
	}
	if claim.RowsAffected() == 0 {
		return pending, nil
	}
	defer func() {
		release, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		b.pool.Exec(release, "UPDATE bunny_video_jobs SET lease_until='-infinity' WHERE id=$1", id)
	}()
	resolution := ""
	for _, wanted := range []string{"720p", "1080p", "360p", "480p", "240p"} {
		for _, r := range strings.Split(remote.Resolutions, ",") {
			if strings.TrimSpace(r) == wanted {
				resolution = wanted
				break
			}
		}
		if resolution != "" {
			break
		}
	}
	if resolution == "" {
		return pending, ErrBunnyUnavailable
	}
	file, err := b.download(ctx, j.video, resolution)
	if errors.Is(err, ErrInvalidVideo) {
		b.pool.Exec(ctx, "UPDATE bunny_video_jobs SET state='failed',updated_at=now() WHERE id=$1", id)
		pending.State = "JOB_STATE_FAILED"
		pending.Error = "invalid_video"
		return pending, nil
	}
	if err != nil {
		return pending, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	effects, err := b.effects(ctx, owner, session)
	if err != nil {
		return pending, err
	}
	operation := "bunny-video:" + id
	blob, err := effects.UploadBlob(ctx, pdseffects.UploadBlobRequest{OperationID: operation, MutationKey: operation, Owner: owner, OwnerGeneration: j.generation, ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: j.generation}}, MIME: "video/mp4", Body: file})
	if err != nil {
		if errors.Is(err, auth.ErrPDSSessionExpired) || errors.Is(err, auth.ErrPDSPermissionsRequired) {
			return pending, err
		}
		if errors.Is(err, pdseffects.ErrOutcomeAmbiguous) || errors.Is(err, pdseffects.ErrEffectConflict) {
			code := "processing_failed"
			var timeout net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
				code = "video_upload_timeout"
			}
			saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer saveCancel()
			if _, saveErr := b.pool.Exec(saveCtx, "UPDATE bunny_video_jobs SET state='failed',failure_code=$2,updated_at=now() WHERE id=$1 AND state='created'", id, code); saveErr != nil {
				return pending, saveErr
			}
			pending.State = "JOB_STATE_FAILED"
			pending.Error = code
			return pending, nil
		}
		return pending, err
	}
	// The durable PDS effect is replay-safe if a metadata request must retry.
	if err = b.setVideoMetadata(ctx, j.video, id, "upload", blob.CID); err != nil {
		return pending, err
	}
	result, err := b.pool.Exec(ctx, `UPDATE bunny_video_jobs j SET state='ready',blob_cid=$2,blob_size=$3,updated_at=now()
 WHERE id=$1 AND state='created' AND cleanup_requested_at IS NULL AND EXISTS(SELECT 1 FROM owner_lifecycles o WHERE o.owner_did=j.owner_did AND o.state='active' AND o.generation=j.owner_generation)`, id, blob.CID, blob.Size)
	if err != nil {
		return pending, err
	}
	if result.RowsAffected() != 1 {
		return pending, ErrVideoJobNotFound
	}
	j.cid = blob.CID
	j.size = blob.Size
	return completedBunnyJob(id, j), nil
}

func (b *Bunny) download(ctx context.Context, video, resolution string) (*os.File, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://"+b.cdn+"/"+video+"/play_"+resolution+".mp4", nil)
	if err != nil {
		return nil, ErrBunnyUnavailable
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, ErrBunnyUnavailable
	}
	defer resp.Body.Close()
	if resp.ContentLength > maxVideoBlobBytes {
		return nil, ErrInvalidVideo
	}
	if resp.StatusCode != 200 {
		return nil, ErrBunnyUnavailable
	}
	file, err := os.CreateTemp("", "craftsky-video-*.mp4")
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			file.Close()
			os.Remove(file.Name())
		}
	}()
	n, err := io.Copy(file, io.LimitReader(resp.Body, maxVideoBlobBytes+1))
	if err != nil || n < 12 || n > maxVideoBlobBytes || (resp.ContentLength >= 0 && n != resp.ContentLength) {
		return nil, ErrBunnyUnavailable
	}
	var header [12]byte
	if _, err = file.ReadAt(header[:], 0); err != nil || string(header[4:8]) != "ftyp" {
		return nil, ErrBunnyUnavailable
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	success = true
	return file, nil
}

func (b *Bunny) Verify(ctx context.Context, owner syntax.DID, id string, blob Blob) (Blob, error) {
	j, err := b.job(ctx, owner, id)
	if err != nil || time.Since(j.createdAt) > 24*time.Hour || j.state != "ready" || j.cid != blob.CID.String() || j.size != blob.Size || blob.MIMEType != "video/mp4" {
		return Blob{}, &VerificationError{Kind: VerificationRejected}
	}
	// Reserve a short publication window before releasing the proof. Cleanup
	// shares this row lock and cannot claim it while a new post is being prepared.
	result, err := b.pool.Exec(ctx, `UPDATE bunny_video_jobs SET lease_until=GREATEST(lease_until,now()+interval '2 minutes')
 WHERE id=$1 AND state='ready' AND cleanup_requested_at IS NULL`, id)
	if err != nil {
		return Blob{}, err
	}
	if result.RowsAffected() != 1 {
		return Blob{}, &VerificationError{Kind: VerificationRejected}
	}
	return blob, nil
}

// URLs resolves the durable owner/CID mapping. Existing unimported posts retain
// their previous provider URLs. The bounded lookup never exposes a vendor key.
func (b *Bunny) URLs(owner syntax.DID, cid syntax.CID) (string, string) {
	playlist, thumbnail, _ := b.URLsWithEmbed(owner, cid)
	return playlist, thumbnail
}

// URLsWithEmbed resolves all playback destinations in one owner-scoped lookup.
// The embed URL is public playback metadata, never a Stream API capability.
func (b *Bunny) URLsWithEmbed(owner syntax.DID, cid syntax.CID) (string, string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var video string
	err := b.pool.QueryRow(ctx, `SELECT video_id::text FROM bunny_video_jobs WHERE owner_did=$1 AND blob_cid=$2 AND library_id=$3 AND state='ready' ORDER BY updated_at DESC LIMIT 1`, owner, cid, b.library).Scan(&video)
	if err == nil {
		return "https://" + b.cdn + "/" + video + "/playlist.m3u8", "https://" + b.cdn + "/" + video + "/thumbnail.jpg", "https://player.mediadelivery.net/embed/" + b.library + "/" + video
	}
	if errors.Is(err, pgx.ErrNoRows) && b.fallback != nil {
		playlist, thumbnail := b.fallback.URLs(owner, cid)
		return playlist, thumbnail, ""
	}
	return "", "", ""
}

func (b *Bunny) String() string { return "Bunny(<configured>)" }

func (b *Bunny) PlaybackOrigin() string { return "https://" + b.cdn }

func (b *Bunny) GoString() string { return "Bunny(<configured>)" }
