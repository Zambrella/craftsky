package scheduledposts

const lockScheduledPostOwnerSQL = `
	SELECT did
	FROM craftsky_profiles
	WHERE did=$1
	FOR UPDATE
`

const countScheduledPostsSQL = `
	SELECT count(*)
	FROM scheduled_posts
	WHERE owner_did=$1
`

const scheduledQueueSnapshotSQL = `
	SELECT
		count(*) FILTER (WHERE status='scheduled'),
		count(*) FILTER (WHERE status='publishing'),
		count(*) FILTER (WHERE status='retrying'),
		count(*) FILTER (WHERE status='needs_attention'),
		count(*) FILTER (
			WHERE status IN ('scheduled', 'retrying') AND next_attempt_at <= $1
		),
		count(*) FILTER (
			WHERE status IN ('scheduled', 'retrying')
			  AND next_attempt_at <= $1 - interval '60 seconds'
		),
		COALESCE(EXTRACT(EPOCH FROM ($1 - min(next_attempt_at) FILTER (
			WHERE status IN ('scheduled', 'retrying') AND next_attempt_at <= $1
		))), 0)
	FROM scheduled_posts
`

const cleanupQueueSnapshotSQL = `
	SELECT
		count(*) FILTER (WHERE state IN ('pending', 'deleting')),
		COALESCE(EXTRACT(EPOCH FROM ($1 - min(created_at) FILTER (
			WHERE state IN ('pending', 'deleting')
		))), 0)
	FROM scheduled_post_cleanup_jobs
`

const insertScheduledPostSQL = `
	INSERT INTO scheduled_posts (
		id, owner_did, operation_id, request_hash, status,
		scheduled_at, next_attempt_at, payload_bytes, payload_hash, payload_version
	) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8, $9)
`

const selectScheduledPostByOperationSQL = `
	SELECT id, owner_did, operation_id, status, scheduled_at, payload_version, request_hash
	FROM scheduled_posts
	WHERE owner_did=$1 AND operation_id=$2
`

const scheduledPostResourceColumns = `
	id, owner_did, operation_id, status, scheduled_at,
	payload_bytes, payload_version, needs_attention_expires_at
`

const listScheduledPostsSQL = `
	SELECT ` + scheduledPostResourceColumns + `
	FROM scheduled_posts
	WHERE owner_did=$1
	ORDER BY scheduled_at, id
`

const getScheduledPostSQL = `
	SELECT ` + scheduledPostResourceColumns + `
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2
`

const selectPublicationTombstoneByOperationSQL = `
	SELECT schedule_id, owner_did, operation_id, request_hash,
	       publication_uri, publication_cid, published_at
	FROM scheduled_post_publication_tombstones
	WHERE owner_did=$1 AND operation_id=$2
`

const selectScheduledMediaForClaimSQL = `
	SELECT state, schedule_id
	FROM scheduled_post_media
	WHERE owner_did=$1 AND id=$2
	FOR UPDATE
`

const attachScheduledMediaSQL = `
	UPDATE scheduled_post_media
	SET schedule_id=$3, ordinal=$4, updated_at=now()
	WHERE owner_did=$1
	  AND id=$2
	  AND state='ready'
	  AND schedule_id IS NULL
`

const selectPrivateMediaByIDSQL = `
	SELECT owner_did, object_key, state, schedule_id, mime_type,
	       size_bytes, sha256, blob_cid, unclaimed_expires_at
	FROM scheduled_post_media
	WHERE id=$1
	FOR UPDATE
`

const selectCleanupJobStateForObjectSQL = `
	SELECT state
	FROM scheduled_post_cleanup_jobs
	WHERE object_key=$1
	FOR UPDATE
`

const deletePendingCleanupJobForObjectSQL = `
	DELETE FROM scheduled_post_cleanup_jobs
	WHERE object_key=$1 AND state='pending'
`

const lockCleanupObjectForTransactionSQL = `
	SELECT pg_advisory_xact_lock(
		hashtextextended('scheduled-cleanup-object:' || $1::text, 0)
	)
`

const lockCleanupEffectForSessionSQL = `
	SELECT pg_advisory_lock(
		hashtextextended('scheduled-cleanup-object:' || $1::text, 0)
	)
`

const unlockCleanupEffectForSessionSQL = `
	SELECT pg_advisory_unlock(
		hashtextextended('scheduled-cleanup-object:' || $1::text, 0)
	)
`

const insertUploadingPrivateMediaSQL = `
	INSERT INTO scheduled_post_media (
		id, owner_did, object_key, state, mime_type,
		size_bytes, sha256, unclaimed_expires_at, created_at, updated_at
	) VALUES ($1, $2, $3, 'uploading', $4, $5, $6, $7, $8, $8)
`

const markPrivateMediaReadySQL = `
	UPDATE scheduled_post_media
	SET state='ready', blob_cid=$3, updated_at=$4
	WHERE owner_did=$1 AND id=$2 AND state IN ('uploading', 'ready')
`

const selectReadyPrivateMediaSQL = `
	SELECT object_key, mime_type, size_bytes, sha256, blob_cid
	FROM scheduled_post_media
	WHERE owner_did=$1 AND id=$2 AND state='ready'
`

const deleteUnclaimedPrivateMediaSQL = `
	DELETE FROM scheduled_post_media
	WHERE owner_did=$1 AND id=$2 AND schedule_id IS NULL
`

const selectScheduledPostForUpdateSQL = `
	SELECT status, payload_version
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2
	FOR UPDATE
`

const selectAttachedScheduledMediaForUpdateSQL = `
	SELECT id, object_key
	FROM scheduled_post_media
	WHERE owner_did=$1 AND schedule_id=$2
	ORDER BY ordinal
	FOR UPDATE
`

const detachScheduledMediaForUpdateSQL = `
	UPDATE scheduled_post_media
	SET schedule_id=NULL, ordinal=NULL, updated_at=now()
	WHERE owner_did=$1 AND schedule_id=$2
`

const updateScheduledPostSQL = `
	UPDATE scheduled_posts
	SET scheduled_at=$3,
	    next_attempt_at=$3,
	    payload_bytes=$4,
	    payload_hash=$5,
	    payload_version=payload_version+1,
	    status='scheduled',
	    attempt_count=0,
	    last_error_code=NULL,
	    publication_rkey=NULL,
	    publication_created_at=NULL,
	    publication_record_bytes=NULL,
	    publication_record_hash=NULL,
	    needs_attention_at=NULL,
	    needs_attention_expires_at=NULL,
	    updated_at=$6
	WHERE owner_did=$1 AND id=$2
	RETURNING payload_version
`

const lockScheduledPostOwnerForTransactionSQL = `
	SELECT pg_advisory_xact_lock(
		hashtextextended('scheduled-post-owner:' || $1::text, 0)
	)
`

const updateAndClaimManualPublicationSQL = `
	UPDATE scheduled_posts
	SET scheduled_at=$3,
	    next_attempt_at=$3,
	    payload_bytes=$4,
	    payload_hash=$5,
	    payload_version=payload_version+1,
	    status='publishing',
	    lease_token=$6,
	    lease_expires_at=$7,
	    attempt_count=6,
	    last_error_code=NULL,
	    publication_rkey=$8,
	    publication_created_at=$3,
	    publication_record_bytes=NULL,
	    publication_record_hash=NULL,
	    needs_attention_at=NULL,
	    needs_attention_expires_at=NULL,
	    updated_at=$3
	WHERE owner_did=$1 AND id=$2
	RETURNING payload_version
`

const selectWorkerFenceSQL = `
	SELECT status, lease_token, payload_version
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2
	FOR UPDATE
`

const saveFrozenRecordSQL = `
	UPDATE scheduled_posts
	SET publication_record_bytes=$5,
	    publication_record_hash=$6,
	    updated_at=$7
	WHERE owner_did=$1
	  AND id=$2
	  AND status='publishing'
	  AND lease_token=$3
	  AND payload_version=$4
`

const selectPublicationSnapshotSQL = `
	SELECT payload_bytes, publication_record_bytes, attempt_count, scheduled_at
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2 AND status='publishing'
	  AND lease_token=$3 AND payload_version=$4
`

const selectPublicationMediaSQL = `
	SELECT object_key, mime_type, size_bytes, sha256, blob_cid
	FROM scheduled_post_media
	WHERE owner_did=$1 AND schedule_id=$2 AND state='ready'
	ORDER BY ordinal
`

const failScheduledPublicationSQL = `
	UPDATE scheduled_posts
	SET status=$5,
	    next_attempt_at=$6,
	    last_error_code=$7,
	    lease_token=NULL,
	    lease_expires_at=NULL,
	    needs_attention_at=$9,
	    needs_attention_expires_at=$10,
	    updated_at=$8
	WHERE owner_did=$1 AND id=$2 AND status='publishing'
	  AND lease_token=$3 AND payload_version=$4
`

const lockScheduleEffectForTransactionSQL = `
	SELECT pg_advisory_xact_lock(
		hashtextextended($1::text || ':' || $2::uuid::text, 0)
	)
`

const tryLockScheduledPostOwnerForTransactionSQL = `
	SELECT pg_try_advisory_xact_lock(
		hashtextextended('scheduled-post-owner:' || $1::text, 0)
	)
`

const publicationRkeyAvailableSQL = `
	SELECT
		NOT EXISTS (
			SELECT 1
			FROM scheduled_posts
			WHERE owner_did=$1 AND publication_rkey=$2
		)
		AND NOT EXISTS (
			SELECT 1
			FROM scheduled_post_publication_tombstones
			WHERE owner_did=$1 AND publication_uri LIKE '%/' || $2
		)
`

const lockScheduleEffectForSessionSQL = `
	SELECT pg_advisory_lock(
		hashtextextended($1::text || ':' || $2::uuid::text, 0)
	)
`

const unlockScheduleEffectForSessionSQL = `
	SELECT pg_advisory_unlock(
		hashtextextended($1::text || ':' || $2::uuid::text, 0)
	)
`

const deleteScheduledPostSQL = `
	DELETE FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2
`

const selectScheduledPostIDsForAccountDeletionSQL = `
	SELECT id
	FROM scheduled_posts
	WHERE owner_did=$1
	ORDER BY id
`

const selectScheduledMediaForAccountDeletionSQL = `
	SELECT object_key
	FROM scheduled_post_media
	WHERE owner_did=$1
	ORDER BY object_key
	FOR UPDATE
`

const deleteScheduledMediaForAccountSQL = `
	DELETE FROM scheduled_post_media
	WHERE owner_did=$1
`

const deleteScheduledPostsForAccountSQL = `
	DELETE FROM scheduled_posts
	WHERE owner_did=$1
`

const deleteScheduledTombstonesForAccountSQL = `
	DELETE FROM scheduled_post_publication_tombstones
	WHERE owner_did=$1
`

const recoverExpiredPublishingLeasesSQL = `
	UPDATE scheduled_posts
	SET status='retrying',
	    lease_token=NULL,
	    lease_expires_at=NULL,
	    next_attempt_at=$1,
	    last_error_code='leaseExpired',
	    updated_at=$1
	WHERE status='publishing'
	  AND lease_expires_at <= $1
	  AND attempt_count < 6
`

const selectExpiredFinalPublishingSQL = `
	SELECT id, owner_did, payload_version, publication_rkey, publication_created_at
	FROM scheduled_posts
	WHERE status='publishing'
	  AND lease_expires_at <= $2
	  AND attempt_count >= 6
	ORDER BY lease_expires_at, id
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

const reclaimExpiredFinalPublishingSQL = `
	UPDATE scheduled_posts
	SET lease_token=$3,
	    lease_expires_at=$4,
	    last_error_code='leaseExpired',
	    updated_at=$5
	WHERE owner_did=$1
	  AND id=$2
	  AND status='publishing'
	  AND attempt_count >= 6
	  AND lease_expires_at <= $5
`

const selectDueScheduledPostsSQL = `
	SELECT id, owner_did, payload_version, publication_rkey, publication_created_at
	FROM scheduled_posts
	WHERE status IN ('scheduled', 'retrying')
	  AND next_attempt_at <= $2
	  AND attempt_count < 6
	ORDER BY next_attempt_at, id
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

const claimScheduledPostSQL = `
	UPDATE scheduled_posts
	SET status='publishing',
	    lease_token=$3,
	    lease_expires_at=$4,
	    attempt_count=attempt_count+1,
	    publication_rkey=COALESCE(publication_rkey, $5),
	    publication_created_at=COALESCE(publication_created_at, $6),
	    updated_at=$6
	WHERE owner_did=$1 AND id=$2
`

const deleteReferencedPendingCleanupJobsSQL = `
	DELETE FROM scheduled_post_cleanup_jobs AS jobs
	USING scheduled_post_media AS media
	WHERE jobs.object_key=media.object_key
	  AND jobs.state='pending'
`

const selectExpiredUnclaimedMediaSQL = `
	SELECT id, object_key
	FROM scheduled_post_media
	WHERE schedule_id IS NULL AND unclaimed_expires_at <= $1
	ORDER BY unclaimed_expires_at, id
	FOR UPDATE
`

const selectExpiredNeedsAttentionMediaSQL = `
	SELECT media.object_key
	FROM scheduled_post_media AS media
	JOIN scheduled_posts AS posts
	  ON posts.owner_did=media.owner_did AND posts.id=media.schedule_id
	WHERE posts.status='needs_attention'
	  AND posts.needs_attention_expires_at <= $1
	ORDER BY media.object_key
	FOR UPDATE OF media, posts
`

const deleteExpiredUnclaimedMediaSQL = `
	DELETE FROM scheduled_post_media
	WHERE id=$1 AND schedule_id IS NULL AND unclaimed_expires_at <= $2
`

const deleteExpiredNeedsAttentionSQL = `
	DELETE FROM scheduled_posts
	WHERE status='needs_attention' AND needs_attention_expires_at <= $1
`

const deleteExpiredTombstonesSQL = `
	DELETE FROM scheduled_post_publication_tombstones
	WHERE expires_at <= $1
`

const insertCleanupJobSQL = `
	INSERT INTO scheduled_post_cleanup_jobs (
		id, object_key, next_attempt_at, created_at, updated_at
	) VALUES ($1, $2, $3, $3, $3)
	ON CONFLICT (object_key) DO NOTHING
`

const recoverExpiredCleanupJobsSQL = `
	UPDATE scheduled_post_cleanup_jobs
	SET state='pending',
	    lease_token=NULL,
	    lease_expires_at=NULL,
	    next_attempt_at=$1,
	    last_error_code='leaseExpired',
	    updated_at=$1
	WHERE state='deleting' AND lease_expires_at <= $1
`

const selectDueCleanupJobsSQL = `
	SELECT jobs.id, jobs.object_key, jobs.attempt_count
	FROM scheduled_post_cleanup_jobs AS jobs
	WHERE jobs.state='pending'
	  AND jobs.next_attempt_at <= $2
	  AND NOT EXISTS (
		SELECT 1 FROM scheduled_post_media AS media
		WHERE media.object_key=jobs.object_key
	  )
	ORDER BY jobs.next_attempt_at, jobs.id
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

const claimCleanupJobSQL = `
	UPDATE scheduled_post_cleanup_jobs
	SET state='deleting',
	    lease_token=$2,
	    lease_expires_at=$3,
	    attempt_count=attempt_count+1,
	    updated_at=$4
	WHERE id=$1 AND state='pending'
`

const selectCleanupJobForDeleteSQL = `
	SELECT object_key
	FROM scheduled_post_cleanup_jobs
	WHERE id=$1 AND state='deleting' AND lease_token=$2
	FOR UPDATE
`

const selectCleanupJobEffectFenceSQL = `
	SELECT object_key
	FROM scheduled_post_cleanup_jobs
	WHERE id=$1 AND state='deleting' AND lease_token=$2
`

const scheduledMediaObjectReferencedSQL = `
	SELECT EXISTS (
		SELECT 1
		FROM scheduled_post_media
		WHERE object_key=$1
	)
`

const completeCleanupJobSQL = `
	DELETE FROM scheduled_post_cleanup_jobs
	WHERE id=$1 AND state='deleting' AND lease_token=$2
`

const retryCleanupJobSQL = `
	UPDATE scheduled_post_cleanup_jobs
	SET state='pending',
	    lease_token=NULL,
	    lease_expires_at=NULL,
	    next_attempt_at=$3,
	    last_error_code=$4,
	    updated_at=$5
	WHERE id=$1 AND state='deleting' AND lease_token=$2
`

const selectPublicationTombstoneSQL = `
	SELECT publication_uri, publication_cid, published_at, expires_at
	FROM scheduled_post_publication_tombstones
	WHERE schedule_id=$1 AND owner_did=$2
`

const selectScheduledPostForFinalizationSQL = `
	SELECT operation_id, request_hash, status, lease_token, payload_version
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2
	FOR UPDATE
`

const selectScheduledMediaForFinalizationSQL = `
	SELECT object_key
	FROM scheduled_post_media
	WHERE owner_did=$1 AND schedule_id=$2
	ORDER BY ordinal
	FOR UPDATE
`

const insertPublicationTombstoneSQL = `
	INSERT INTO scheduled_post_publication_tombstones (
		schedule_id, owner_did, operation_id, request_hash,
		publication_uri, publication_cid, published_at, expires_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

const deleteFinalizedScheduledPostSQL = `
	DELETE FROM scheduled_posts
	WHERE owner_did=$1
	  AND id=$2
	  AND status='publishing'
	  AND lease_token=$3
	  AND payload_version=$4
`
