package video

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/observability"
)

// BunnyCleanup keeps origin deletion and edge invalidation independently durable.
// It never holds PDS credentials or deletes the portable PDS blob.
type BunnyCleanup struct {
	bunny    *Bunny
	purgeKey string
	observer *observability.Observer
	logger   *slog.Logger
}

func NewBunnyCleanup(b *Bunny, purgeKey string, observer *observability.Observer) *BunnyCleanup {
	return &BunnyCleanup{bunny: b, purgeKey: purgeKey, observer: observer, logger: slog.Default()}
}

func (*BunnyCleanup) String() string   { return "BunnyCleanup(<configured>)" }
func (*BunnyCleanup) GoString() string { return "BunnyCleanup(<configured>)" }

var errBunnyPurgeCredential = errors.New("Bunny CDN purge credential is unavailable")

func (w *BunnyCleanup) purge(ctx context.Context, video string) error {
	if w.purgeKey == "" {
		return errBunnyPurgeCredential
	}
	q := url.Values{"url": {"https://" + w.bunny.cdn + "/" + video + "/*"}, "async": {"false"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.bunny.net/purge?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("AccessKey", w.purgeKey)
	resp, err := w.bunny.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		return ErrBunnyUnavailable
	}
	return nil
}

// ProcessBatch processes one bounded item. Row locks serialize concurrent workers
// and post-reference changes. The committing deletion claim is the point after
// which this vendor asset cannot be reused for a new publication.
func (w *BunnyCleanup) ProcessBatch(ctx context.Context) (int, error) {
	b := w.bunny
	opctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// Bound each sweep and skip rows held by publishers or other workers.
	_, err := b.pool.Exec(opctx, `WITH candidates AS (
 SELECT j.id FROM bunny_video_jobs j WHERE library_id=$1
 AND state IN ('created','ready','failed') AND cleanup_requested_at IS NULL
 AND (created_at<now()-interval '7 days' OR
  ((state!='created' OR created_at<now()-interval '5 minutes') AND
   EXISTS(SELECT 1 FROM owner_lifecycles o WHERE o.owner_did=j.owner_did AND o.state='terminal')))
 AND NOT EXISTS(SELECT 1 FROM craftsky_posts p WHERE p.did=j.owner_did AND p.record#>>'{embed,video,ref,$link}'=j.blob_cid)
 ORDER BY created_at,id LIMIT 100 FOR UPDATE SKIP LOCKED)
 UPDATE bunny_video_jobs j SET cleanup_requested_at=now(),cleanup_next_attempt_at=now()
 FROM candidates c WHERE j.id=c.id`, b.library)
	if err != nil {
		return 0, err
	}
	// Keep quota tombstones for one day; terminal owners no longer need them.
	_, err = b.pool.Exec(opctx, `WITH candidates AS (
 SELECT j.id FROM bunny_video_jobs j WHERE library_id=$1 AND state='deleted'
 AND (created_at<now()-interval '24 hours' OR EXISTS(SELECT 1 FROM owner_lifecycles o
 WHERE o.owner_did=j.owner_did AND o.state='terminal'))
 ORDER BY created_at,id LIMIT 100 FOR UPDATE SKIP LOCKED)
 DELETE FROM bunny_video_jobs j USING candidates c WHERE j.id=c.id`, b.library)
	if err != nil {
		return 0, err
	}
	tx, err := b.pool.Begin(opctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	var id, video, state string
	var owner syntax.DID
	var attempt int
	err = tx.QueryRow(opctx, `SELECT id::text,owner_did,COALESCE(video_id::text,''),state,cleanup_attempts
 FROM bunny_video_jobs WHERE library_id=$1 AND cleanup_requested_at IS NOT NULL AND state!='deleted'
 AND cleanup_next_attempt_at<=now() AND (state IN ('deleting','purging') OR lease_until<now())
 ORDER BY cleanup_next_attempt_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, b.library).Scan(&id, &owner, &video, &state, &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if state != "deleting" && state != "purging" {
		var referenced bool
		err = tx.QueryRow(opctx, `SELECT EXISTS(SELECT 1 FROM craftsky_posts p JOIN bunny_video_jobs j
   ON p.did=j.owner_did AND p.record#>>'{embed,video,ref,$link}'=j.blob_cid WHERE j.id=$1)`, id).Scan(&referenced)
		if err != nil {
			return 0, err
		}
		if referenced {
			_, err = tx.Exec(opctx, `UPDATE bunny_video_jobs SET cleanup_requested_at=NULL,cleanup_next_attempt_at=NULL WHERE id=$1`, id)
		} else {
			// Accepted indexed posts are checked above. Durable unresolved writes
			// also protect the blob after the short request-preparation lease ends.
			var publishing bool
			err = tx.QueryRow(opctx, `SELECT EXISTS(SELECT 1 FROM pds_command_steps s
 JOIN pds_commands c ON c.id=s.command_id JOIN bunny_video_jobs j ON j.id=$1
 WHERE c.owner_did=j.owner_did AND c.state IN ('prepared','dispatching','ambiguous')
 AND s.repo_did=j.owner_did AND s.collection='social.craftsky.feed.post'
 AND s.action IN ('create','update') AND s.record::jsonb#>>'{embed,video,ref,$link}'=j.blob_cid)`, id).Scan(&publishing)
			if err != nil {
				return 0, err
			}
			if publishing {
				_, err = tx.Exec(opctx, `UPDATE bunny_video_jobs SET cleanup_next_attempt_at=now()+interval '1 minute' WHERE id=$1`, id)
				if err != nil {
					return 0, err
				}
				return 1, tx.Commit(opctx)
			}
			// Persist intent before the irreversible external call, including ambiguous
			// network outcomes. A restarted worker repeats DELETE safely (404 is success).
			_, err = tx.Exec(opctx, `UPDATE bunny_video_jobs SET state='deleting',cleanup_next_attempt_at=now(),updated_at=now() WHERE id=$1`, id)
		}
		if err != nil {
			return 0, err
		}
		return 1, tx.Commit(opctx)
	}
	attempt++
	stage := "delete_origin"
	var operationErr error
	if state == "deleting" {
		if video != "" {
			operationErr = b.api(opctx, http.MethodDelete, "/"+video, nil, nil)
		}
	} else {
		stage = "purge_cdn"
		if video != "" {
			operationErr = w.purge(opctx, video)
		}
	}
	if opctx.Err() != nil {
		// Record retries after an operation timeout without using the canceled call.
		_ = tx.Rollback(context.WithoutCancel(ctx))
		retryCtx, retryCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer retryCancel()
		_, err = b.pool.Exec(retryCtx, `UPDATE bunny_video_jobs SET cleanup_attempts=cleanup_attempts+1,cleanup_next_attempt_at=now()+interval '1 minute' WHERE id=$1 AND state=$2`, id, state)
	} else if operationErr != nil {
		delay := min(time.Hour, time.Second*time.Duration(1<<min(attempt, 12)))
		_, err = tx.Exec(opctx, `UPDATE bunny_video_jobs SET cleanup_attempts=$2,cleanup_next_attempt_at=now()+$3 * interval '1 second' WHERE id=$1`, id, attempt, delay.Seconds())
		if err == nil {
			err = tx.Commit(opctx)
		}
	} else {
		next := "purging"
		if state == "purging" {
			next = "deleted"
		}
		_, err = tx.Exec(opctx, `UPDATE bunny_video_jobs SET state=$2,cleanup_attempts=0,cleanup_next_attempt_at=now(),updated_at=now() WHERE id=$1`, id, next)
		if err == nil {
			err = tx.Commit(opctx)
		}
	}
	if operationErr != nil && ctx.Err() == nil {
		diagnostic := observability.DiagnosticInput{Error: operationErr, Context: observability.EventContext{
			"component": "bunny_video_cleanup", "operation": "video.cleanup", "failure_stage": stage, "result": "retry", "attempt": attempt,
		}}
		diagnostic.Workflow = observability.PrivateFailureContext(operationErr, owner, id)
		if attempt >= 5 {
			diagnostic.Context["result"] = "exhausted"
		}
		observability.LogDiagnostic(ctx, w.logger, diagnostic)
		if attempt >= 5 && w.observer != nil {
			w.observer.CaptureDiagnostic(ctx, diagnostic)
		}
	}
	return 1, err
}
