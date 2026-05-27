-- appview/migrations/000013_profile_social_summary_indexes.up.sql
-- Supporting ordered predicates for profile social-summary graph lists and
-- root-post summary counts. See docs/changes/2026-05-27-profile-social-summary/.
CREATE INDEX atproto_follows_subject_created_uri_desc_idx
    ON atproto_follows (subject_did, created_at DESC, uri DESC);

CREATE INDEX atproto_follows_did_created_uri_desc_idx
    ON atproto_follows (did, created_at DESC, uri DESC);

CREATE INDEX craftsky_posts_root_did_created_idx
    ON craftsky_posts (did, created_at DESC)
    WHERE reply_root_uri IS NULL AND reply_parent_uri IS NULL;
