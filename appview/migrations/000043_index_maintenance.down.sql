CREATE INDEX craftsky_likes_active_did_subject_uri
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;

CREATE INDEX craftsky_reposts_active_did_subject_uri
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;

CREATE INDEX atproto_follows_did_subject_did_idx
    ON atproto_follows (did, subject_did);

DROP INDEX saved_posts_post_uri_idx;
DROP INDEX push_deliveries_account_subscription_id_idx;
DROP INDEX craftsky_likes_subject_uri_idx;
DROP INDEX craftsky_reposts_subject_uri_idx;
