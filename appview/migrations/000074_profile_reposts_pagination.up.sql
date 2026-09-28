CREATE INDEX pds_set_aggregates_repost_actor_pagination_idx
    ON pds_set_aggregates (actor_did, activated_at DESC, subject_uri DESC)
    WHERE kind = 'repost';
