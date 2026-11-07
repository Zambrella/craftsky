package video

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrVideoCancellationTooLate = errors.New("video is already being published")

// Cancel fences future finalisation and durably requests the existing vendor
// cleanup worker. Published or reserved PDS records and blobs are never deleted.
func (b *Bunny) Cancel(ctx context.Context, owner syntax.DID, id string) error {
	j, err := b.job(ctx, owner, id)
	if err != nil {
		return err
	}
	if running, ok := b.active.Load(id); ok {
		active := running.(*activeBunnyJob)
		if active.owner == owner {
			active.cancel()
		}
	}
	if j.failureCode == "video_upload_canceled" || j.state == "deleted" || j.state == "deleting" || j.state == "purging" {
		return nil
	}
	result, err := b.pool.Exec(ctx, `UPDATE bunny_video_jobs j
 SET state='failed',failure_code='video_upload_canceled',cleanup_requested_at=COALESCE(cleanup_requested_at,now()),cleanup_next_attempt_at=now(),updated_at=now()
 WHERE id=$1 AND owner_did=$2 AND library_id=$3 AND state IN ('created','failed','ready')
 AND (state!='ready' OR lease_until<now())
 AND NOT EXISTS(SELECT 1 FROM craftsky_posts p WHERE p.did=j.owner_did AND p.record#>>'{embed,video,ref,$link}'=j.blob_cid)
 AND NOT EXISTS(SELECT 1 FROM pds_command_steps s JOIN pds_commands c ON c.id=s.command_id
 WHERE c.owner_did=j.owner_did AND c.state IN ('prepared','dispatching','ambiguous') AND s.repo_did=j.owner_did
 AND s.collection='social.craftsky.feed.post' AND s.action IN ('create','update') AND s.record::jsonb#>>'{embed,video,ref,$link}'=j.blob_cid)`, id, owner, b.library)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrVideoCancellationTooLate
	}
	return nil
}
