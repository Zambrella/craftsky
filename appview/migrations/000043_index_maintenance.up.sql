CREATE INDEX saved_posts_post_uri_idx
    ON saved_posts (post_uri);

CREATE INDEX push_deliveries_account_subscription_id_idx
    ON push_deliveries (account_subscription_id);

CREATE INDEX craftsky_likes_subject_uri_idx
    ON craftsky_likes (subject_uri);

CREATE INDEX craftsky_reposts_subject_uri_idx
    ON craftsky_reposts (subject_uri);

DROP INDEX craftsky_likes_active_did_subject_uri;
DROP INDEX craftsky_reposts_active_did_subject_uri;
DROP INDEX atproto_follows_did_subject_did_idx;
