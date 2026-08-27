package scheduledposts

const lockScheduledPostOwnerSQL = `
	SELECT did
	FROM craftsky_profiles
	WHERE did=$1
	FOR UPDATE
`

const lockActiveScheduledOwnerGenerationSQL = `
	SELECT generation
	FROM owner_lifecycles
	WHERE owner_did=$1 AND state='active'
	FOR SHARE
`

const countScheduledPostsSQL = `
	SELECT count(*)
	FROM scheduled_posts
	WHERE owner_did=$1 AND owner_generation=$2
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
	FROM scheduled_posts AS posts
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=posts.owner_did
	 AND lifecycle.generation=posts.owner_generation
	 AND lifecycle.state='active'
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
		id, owner_did, owner_generation, operation_id, request_hash, status,
		scheduled_at, next_attempt_at, payload_bytes, payload_hash, payload_version
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, $9, $10)
`

const selectScheduledPostByOperationSQL = `
	SELECT id, owner_did, owner_generation, operation_id, status, scheduled_at, payload_version, request_hash
	FROM scheduled_posts
	WHERE owner_did=$1 AND owner_generation=$2 AND operation_id=$3
`

const scheduledPostResourceColumns = `
	posts.id, posts.owner_did, posts.owner_generation, posts.operation_id,
	posts.status, posts.scheduled_at, posts.payload_bytes,
	posts.payload_version, posts.needs_attention_expires_at
`

const listScheduledPostsSQL = `
	SELECT ` + scheduledPostResourceColumns + `
	FROM scheduled_posts AS posts
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=posts.owner_did
	 AND lifecycle.generation=posts.owner_generation
	 AND lifecycle.state='active'
	WHERE posts.owner_did=$1
	ORDER BY scheduled_at, id
`

const getScheduledPostSQL = `
	SELECT ` + scheduledPostResourceColumns + `
	FROM scheduled_posts AS posts
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=posts.owner_did
	 AND lifecycle.generation=posts.owner_generation
	 AND lifecycle.state='active'
	WHERE posts.owner_did=$1 AND posts.id=$2
`

const selectPublicationTombstoneByOperationSQL = `
	SELECT schedule_id, owner_did, owner_generation, operation_id, request_hash,
	       publication_uri, publication_cid, published_at
	FROM scheduled_post_publication_tombstones
	WHERE owner_did=$1 AND owner_generation=$2 AND operation_id=$3
`

const selectScheduledMediaForClaimSQL = `
	SELECT state, schedule_id, mime_type, size_bytes
	FROM scheduled_post_media
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
	FOR UPDATE
`

const attachScheduledMediaSQL = `
	UPDATE scheduled_post_media
	SET schedule_id=$4, ordinal=$5, updated_at=now()
	WHERE owner_did=$1
	  AND id=$2
	  AND owner_generation=$3
	  AND state='ready'
	  AND schedule_id IS NULL
`

const selectPrivateMediaByIDSQL = `
	SELECT media.owner_did, media.owner_generation, media.upload_generation,
	       media.upload_attempt_id, media.object_key, media.state,
	       media.schedule_id, media.mime_type,
	       media.size_bytes, media.sha256, media.blob_cid,
	       media.unclaimed_expires_at, attempts.remote_deadline,
	       attempts.settlement_not_before
	FROM scheduled_post_media AS media
	JOIN scheduled_post_object_attempts AS attempts
	  ON attempts.upload_attempt_id=media.upload_attempt_id
	WHERE media.id=$1
	FOR UPDATE OF media, attempts
`

const selectCleanupJobStateForObjectSQL = `
	SELECT state
	FROM scheduled_post_cleanup_jobs
	WHERE object_key=$1
	FOR UPDATE
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

const insertPrivateObjectAttemptSQL = `
	INSERT INTO scheduled_post_object_attempts (
		upload_attempt_id, media_id, owner_did, owner_generation,
		upload_generation, object_key, request_fingerprint,
		remote_started_at, remote_deadline, settlement_not_before,
		created_at, updated_at
	) VALUES ($1, $2, $3, $4, $4, $5, $6, $7, $8, $9, $7, $7)
	ON CONFLICT DO NOTHING
`

const selectPrivateObjectAttemptIdentitySQL = `
	SELECT owner_did, owner_generation, media_id, object_key,
	       request_fingerprint, remote_outcome, remote_deadline
	FROM scheduled_post_object_attempts
	WHERE upload_attempt_id=$1
	FOR UPDATE
`

const selectObjectAttemptOutcomeSQL = `
	SELECT remote_outcome
	FROM scheduled_post_object_attempts
	WHERE upload_attempt_id=$1
`

const insertUploadingPrivateMediaSQL = `
	INSERT INTO scheduled_post_media (
		id, owner_did, owner_generation, upload_generation,
		upload_attempt_id, object_key, state, mime_type,
		size_bytes, sha256, unclaimed_expires_at, created_at, updated_at
	) VALUES ($1, $2, $3, $3, $4, $5, 'uploading', $6, $7, $8, $9, $10, $10)
`

const markPrivateObjectAttemptDispatchedSQL = `
	UPDATE scheduled_post_object_attempts
	SET remote_outcome='dispatched', dispatched_at=$2, updated_at=$2
	WHERE upload_attempt_id=$1 AND remote_outcome='prepared'
	  AND remote_deadline>$2
`

const markPrivateObjectAttemptAcceptedSQL = `
	UPDATE scheduled_post_object_attempts
	SET remote_outcome='accepted', completed_at=$2, updated_at=$2
	WHERE upload_attempt_id=$1 AND remote_outcome='dispatched'
`

const markPrivateMediaReadySQL = `
	UPDATE scheduled_post_media
	SET state='ready', blob_cid=$5, updated_at=$6
	WHERE owner_did=$1 AND owner_generation=$2 AND id=$3
	  AND upload_attempt_id=$4 AND state='uploading'
`

const selectReadyPrivateMediaSQL = `
	SELECT media.object_key, media.owner_generation, media.upload_generation,
	       media.upload_attempt_id, media.mime_type, media.size_bytes,
	       media.sha256, media.blob_cid
	FROM scheduled_post_media AS media
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=media.owner_did
	 AND lifecycle.generation=media.owner_generation
	 AND lifecycle.state='active'
	WHERE media.owner_did=$1 AND media.id=$2 AND media.state='ready'
`

const selectPrivateMediaIdentitySQL = `
	SELECT object_key, upload_generation, upload_attempt_id
	FROM scheduled_post_media
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
`

const deleteUploadingPrivateMediaSQL = `
	DELETE FROM scheduled_post_media
	WHERE owner_did=$1 AND owner_generation=$2 AND id=$3
	  AND upload_attempt_id=$4 AND state='uploading'
`

const deleteUnclaimedPrivateMediaSQL = `
	DELETE FROM scheduled_post_media
	WHERE owner_did=$1 AND owner_generation=$2 AND id=$3
	  AND schedule_id IS NULL
`

const selectScheduledPostForUpdateSQL = `
	SELECT status, payload_version
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
	FOR UPDATE
`

const selectAttachedScheduledMediaForUpdateSQL = `
	SELECT id, object_key, owner_generation
	FROM scheduled_post_media
	WHERE owner_did=$1 AND schedule_id=$2 AND owner_generation=$3
	ORDER BY ordinal
	FOR UPDATE
`

const detachScheduledMediaForUpdateSQL = `
	UPDATE scheduled_post_media
	SET schedule_id=NULL, ordinal=NULL, updated_at=now()
	WHERE owner_did=$1 AND schedule_id=$2 AND owner_generation=$3
`

const updateScheduledPostSQL = `
	UPDATE scheduled_posts
	SET scheduled_at=$4,
	    next_attempt_at=$4,
	    payload_bytes=$5,
	    payload_hash=$6,
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
	    updated_at=$7
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
	RETURNING payload_version
`

const lockScheduledPostOwnerForTransactionSQL = `
	SELECT pg_advisory_xact_lock(
		hashtextextended('scheduled-post-owner:' || $1::text, 0)
	)
`

const updateAndClaimManualPublicationSQL = `
	UPDATE scheduled_posts
	SET scheduled_at=$4,
	    next_attempt_at=$4,
	    payload_bytes=$5,
	    payload_hash=$6,
	    payload_version=payload_version+1,
	    status='publishing',
	    lease_token=$7,
	    lease_expires_at=$8,
	    attempt_count=6,
	    last_error_code=NULL,
	    publication_rkey=$9,
	    publication_created_at=$4,
	    publication_record_bytes=NULL,
	    publication_record_hash=NULL,
	    needs_attention_at=NULL,
	    needs_attention_expires_at=NULL,
	    updated_at=$4
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
	RETURNING payload_version
`

const selectWorkerFenceSQL = `
	SELECT status, lease_token, payload_version
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
	FOR UPDATE
`

const saveFrozenRecordSQL = `
	UPDATE scheduled_posts
	SET publication_record_bytes=$6,
	    publication_record_hash=$7,
	    updated_at=$8
	WHERE owner_did=$1
	  AND id=$2
	  AND owner_generation=$3
	  AND status='publishing'
	  AND lease_token=$4
	  AND payload_version=$5
`

const selectPublicationSnapshotSQL = `
	SELECT posts.payload_bytes, posts.publication_record_bytes,
	       posts.attempt_count, posts.scheduled_at
	FROM scheduled_posts AS posts
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=posts.owner_did
	 AND lifecycle.generation=posts.owner_generation
	 AND lifecycle.state='active'
	WHERE posts.owner_did=$1 AND posts.id=$2 AND posts.owner_generation=$3
	  AND posts.status='publishing' AND posts.lease_token=$4
	  AND posts.payload_version=$5
`

const selectPublicationMediaSQL = `
	SELECT id, object_key, mime_type, size_bytes, sha256, blob_cid
	FROM scheduled_post_media
	WHERE owner_did=$1 AND schedule_id=$2 AND owner_generation=$3 AND state='ready'
	ORDER BY ordinal
`

const failScheduledPublicationSQL = `
	UPDATE scheduled_posts
	SET status=$6,
	    next_attempt_at=$7,
	    last_error_code=$8,
	    lease_token=NULL,
	    lease_expires_at=NULL,
	    needs_attention_at=$10,
	    needs_attention_expires_at=$11,
	    updated_at=$9
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3 AND status='publishing'
	  AND lease_token=$4 AND payload_version=$5
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
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
`

const selectScheduledPostIDsForDepartureSQL = `
	SELECT id
	FROM scheduled_posts
	WHERE owner_did=$1 AND owner_generation=$2
	ORDER BY id
	LIMIT $3
	FOR UPDATE
`

const selectScheduledMediaForDepartureSQL = `
	SELECT id,object_key
	FROM scheduled_post_media
	WHERE owner_did=$1 AND owner_generation=$2
	  AND schedule_id=ANY($3::uuid[])
	ORDER BY id
	LIMIT $4
	FOR UPDATE
`

const deleteScheduledMediaForDepartureSQL = `
	DELETE FROM scheduled_post_media
	WHERE owner_did=$1 AND owner_generation=$2
	  AND schedule_id IS NOT NULL
`

const deleteScheduledPostsForDepartureSQL = `
	DELETE FROM scheduled_posts
	WHERE owner_did=$1 AND owner_generation=$2
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
	UPDATE scheduled_posts AS posts
	SET status='retrying',
	    lease_token=NULL,
	    lease_expires_at=NULL,
	    next_attempt_at=$1,
	    last_error_code='leaseExpired',
	    updated_at=$1
	FROM owner_lifecycles AS lifecycle
	WHERE posts.status='publishing'
	  AND posts.lease_expires_at <= $1
	  AND posts.attempt_count < 6
	  AND lifecycle.owner_did=posts.owner_did
	  AND lifecycle.generation=posts.owner_generation
	  AND lifecycle.state='active'
`

const selectExpiredFinalPublishingSQL = `
	SELECT posts.id, posts.owner_did, posts.owner_generation, posts.payload_version,
	       posts.publication_rkey, posts.publication_created_at
	FROM scheduled_posts AS posts
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=posts.owner_did
	 AND lifecycle.generation=posts.owner_generation
	 AND lifecycle.state='active'
	WHERE posts.status='publishing'
	  AND posts.lease_expires_at <= $2
	  AND posts.attempt_count >= 6
	ORDER BY posts.lease_expires_at, posts.id
	LIMIT $1
	FOR UPDATE OF posts SKIP LOCKED
`

const reclaimExpiredFinalPublishingSQL = `
	UPDATE scheduled_posts
	SET lease_token=$4,
	    lease_expires_at=$5,
	    last_error_code='leaseExpired',
	    updated_at=$6
	WHERE owner_did=$1
	  AND id=$2
	  AND owner_generation=$3
	  AND status='publishing'
	  AND attempt_count >= 6
	  AND lease_expires_at <= $6
`

const selectDueScheduledPostsSQL = `
	SELECT posts.id, posts.owner_did, posts.owner_generation, posts.payload_version,
	       posts.publication_rkey, posts.publication_created_at
	FROM scheduled_posts AS posts
	JOIN owner_lifecycles AS lifecycle
	  ON lifecycle.owner_did=posts.owner_did
	 AND lifecycle.generation=posts.owner_generation
	 AND lifecycle.state='active'
	WHERE posts.status IN ('scheduled', 'retrying')
	  AND posts.next_attempt_at <= $2
	  AND posts.attempt_count < 6
	ORDER BY posts.next_attempt_at, posts.id
	LIMIT $1
	FOR UPDATE OF posts SKIP LOCKED
`

const claimScheduledPostSQL = `
	UPDATE scheduled_posts
	SET status='publishing',
	    lease_token=$4,
	    lease_expires_at=$5,
	    attempt_count=attempt_count+1,
	    publication_rkey=COALESCE(publication_rkey, $6),
	    publication_created_at=COALESCE(publication_created_at, $7),
	    updated_at=$7
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
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
	  ON posts.owner_did=media.owner_did
	 AND posts.owner_generation=media.owner_generation
	 AND posts.id=media.schedule_id
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
		id, object_key, owner_did, owner_generation, upload_generation,
		source_attempt_id, outcome_uncertain, settlement_not_before,
		next_attempt_at, created_at, updated_at
	)
	SELECT $1, attempts.object_key, attempts.owner_did,
	       attempts.owner_generation, attempts.upload_generation,
	       attempts.upload_attempt_id,
	       attempts.remote_outcome = 'dispatched',
	       attempts.settlement_not_before,
	       $3, $3, $3
	FROM scheduled_post_object_attempts AS attempts
	WHERE attempts.object_key=$2
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
	SELECT jobs.id, jobs.object_key, jobs.owner_did,
	       jobs.owner_generation, jobs.upload_generation,
	       jobs.source_attempt_id, jobs.outcome_uncertain,
	       attempts.remote_deadline, jobs.settlement_not_before,
	       jobs.attempt_count
	FROM scheduled_post_cleanup_jobs AS jobs
	JOIN scheduled_post_object_attempts AS attempts
	  ON attempts.upload_attempt_id=jobs.source_attempt_id
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

const recordCleanupAbsenceSQL = `
	UPDATE scheduled_post_cleanup_jobs
	SET last_absence_at=$3, updated_at=$3
	WHERE id=$1 AND state='deleting' AND lease_token=$2
	RETURNING outcome_uncertain, settlement_not_before, last_absence_at
`

const selectPublicationTombstoneSQL = `
	SELECT publication_uri, publication_cid, published_at, expires_at
	FROM scheduled_post_publication_tombstones
	WHERE schedule_id=$1 AND owner_did=$2 AND owner_generation=$3
`

const selectScheduledPostForFinalizationSQL = `
	SELECT operation_id, request_hash, status, lease_token, payload_version
	FROM scheduled_posts
	WHERE owner_did=$1 AND id=$2 AND owner_generation=$3
	FOR UPDATE
`

const selectScheduledMediaForFinalizationSQL = `
	SELECT object_key
	FROM scheduled_post_media
	WHERE owner_did=$1 AND schedule_id=$2 AND owner_generation=$3
	ORDER BY ordinal
	FOR UPDATE
`

const insertPublicationTombstoneSQL = `
	INSERT INTO scheduled_post_publication_tombstones (
		schedule_id, owner_did, owner_generation, operation_id, request_hash,
		publication_uri, publication_cid, published_at, expires_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

const deleteFinalizedScheduledPostSQL = `
	DELETE FROM scheduled_posts
	WHERE owner_did=$1
	  AND id=$2
	  AND owner_generation=$3
	  AND status='publishing'
	  AND lease_token=$4
	  AND payload_version=$5
`
