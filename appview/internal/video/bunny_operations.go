package video

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/pdseffects"
)

// ImportExisting only copies an indexed owner's referenced video. It never
// changes the owner's repository or asks a client to read directly from a PDS.
func (b *Bunny) ImportExisting(ctx context.Context, owner syntax.DID, contentCID syntax.CID, directory identity.Directory, boundary *federatedhttp.Boundary) (string, error) {
	var generation, size int64
	err := b.pool.QueryRow(ctx, `SELECT o.generation,(p.record#>>'{embed,video,size}')::bigint
 FROM craftsky_posts p JOIN owner_lifecycles o ON o.owner_did=p.did
 WHERE p.did=$1 AND o.state='active' AND p.record#>>'{embed,$type}'='app.bsky.embed.video'
 AND p.record#>>'{embed,video,ref,$link}'=$2 LIMIT 1`, owner, contentCID).Scan(&generation, &size)
	if err != nil {
		return "", ErrVideoJobNotFound
	}
	var existing string
	if err = b.pool.QueryRow(ctx, "SELECT id::text FROM bunny_video_jobs WHERE owner_did=$1 AND blob_cid=$2 AND library_id=$3 AND state IN ('created','ready') LIMIT 1", owner, contentCID, b.library).Scan(&existing); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	account, err := directory.LookupDID(ctx, owner)
	if err != nil || account == nil {
		return "", ErrBunnyUnavailable
	}
	origin, err := boundary.ValidateOrigin(ctx, account.PDSEndpoint())
	if err != nil {
		return "", ErrBunnyUnavailable
	}
	endpoint := origin.JoinPath("xrpc", "com.atproto.sync.getBlob")
	q := endpoint.Query()
	q.Set("did", owner.String())
	q.Set("cid", contentCID.String())
	endpoint.RawQuery = q.Encode()
	profile, err := federatedhttp.DefaultProfile(federatedhttp.PurposePDSVideo)
	if err != nil {
		return "", err
	}
	client, err := boundary.Client(profile)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", ErrBunnyUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || size <= 0 || size > maxVideoBlobBytes {
		return "", ErrBunnyUnavailable
	}
	file, err := os.CreateTemp("", "craftsky-video-import-*.mp4")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	n, err := io.Copy(file, io.LimitReader(res.Body, size+1))
	if err != nil || n != size {
		return "", ErrBunnyUnavailable
	}
	// Use the same content identity validation as a normal streamed PDS upload.
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	predicted, err := streamCID(file)
	if err != nil || predicted != contentCID {
		return "", errors.New("PDS video content does not match its CID")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	job := uuid.NewString()
	allocation, err := b.createOrganizedVideo(ctx, owner, job, time.Now())
	if err != nil {
		return "", err
	}
	_, err = b.pool.Exec(ctx, `INSERT INTO bunny_video_jobs(id,owner_did,owner_generation,library_id,video_id,blob_cid,blob_size,imported)
 VALUES($1,$2,$3,$4,$5,$6,$7,true)`, job, owner, generation, b.library, allocation.ID, contentCID, size)
	if err != nil {
		return "", err
	}
	uploaded := false
	defer func() {
		if !uploaded {
			finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			b.pool.Exec(finish, "UPDATE bunny_video_jobs SET state='failed',updated_at=now() WHERE id=$1", job)
		}
	}()
	if err = b.setVideoMetadata(ctx, allocation.ID, job, "pds-import", contentCID.String()); err != nil {
		return "", err
	}
	upload, err := http.NewRequestWithContext(ctx, "PUT", "https://video.bunnycdn.com/library/"+b.library+"/videos/"+allocation.ID, file)
	if err != nil {
		return "", err
	}
	upload.ContentLength = size
	upload.Header.Set("AccessKey", b.apiKey)
	upload.Header.Set("Content-Type", "video/mp4")
	response, err := b.client.Do(upload)
	if err != nil {
		return "", ErrBunnyUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", ErrBunnyUnavailable
	}
	uploaded = true
	return job, nil
}

func streamCID(file *os.File) (syntax.CID, error) {
	// Hash the spool without reading the complete original into memory.
	return pdseffects.PredictBlobStreamCID(file)
}

// Cleanup previews or durably queues aged, unreferenced copies. The same
// background processor always performs origin removal followed by CDN purge.
func (b *Bunny) Cleanup(ctx context.Context, apply bool) (int, error) {
	const eligible = `library_id=$1 AND state IN ('created','ready','failed')
 AND created_at<now()-interval '7 days' AND NOT EXISTS(SELECT 1 FROM craftsky_posts p
 WHERE p.did=j.owner_did AND p.record#>>'{embed,video,ref,$link}'=j.blob_cid)`
	if !apply {
		var count int
		err := b.pool.QueryRow(ctx, "SELECT count(*) FROM bunny_video_jobs j WHERE "+eligible, b.library).Scan(&count)
		return count, err
	}
	result, err := b.pool.Exec(ctx, `UPDATE bunny_video_jobs j SET cleanup_requested_at=COALESCE(cleanup_requested_at,now()),
 cleanup_next_attempt_at=now() WHERE `+eligible, b.library)
	return int(result.RowsAffected()), err
}

func (b *Bunny) ImportedStatus(ctx context.Context, owner syntax.DID, id string) (BunnyJobStatus, error) {
	j, err := b.job(ctx, owner, id)
	if err != nil {
		return BunnyJobStatus{}, err
	}
	if !j.imported {
		return BunnyJobStatus{}, errors.New("use the authenticated app to complete an upload")
	}
	return b.Status(ctx, owner, "", id)
}
