-- appview/migrations/000013_profile_social_summary_indexes.down.sql
DROP INDEX IF EXISTS craftsky_posts_root_did_created_idx;
DROP INDEX IF EXISTS atproto_follows_did_created_uri_desc_idx;
DROP INDEX IF EXISTS atproto_follows_subject_created_uri_desc_idx;
