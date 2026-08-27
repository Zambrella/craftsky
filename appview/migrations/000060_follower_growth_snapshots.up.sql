-- appview/migrations/000060_follower_growth_snapshots.up.sql
CREATE VIEW craftsky_profile_follower_counts AS
SELECT
    profile.did AS profile_did,
    COUNT(follower.did)::BIGINT AS follower_count
FROM craftsky_profiles profile
LEFT JOIN atproto_follows follow
    ON follow.subject_did = profile.did
    AND NOT appview_owner_is_terminal(follow.did)
    AND NOT appview_owner_is_terminal(follow.subject_did)
LEFT JOIN craftsky_profiles follower
    ON follower.did = follow.did
    AND NOT appview_owner_is_terminal(follower.did)
WHERE NOT appview_owner_is_terminal(profile.did)
GROUP BY profile.did;

CREATE TABLE follower_growth_snapshots (
    profile_did    TEXT        NOT NULL,
    snapshot_date DATE        NOT NULL,
    follower_count BIGINT     NOT NULL,
    captured_at   TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (profile_did, snapshot_date),
    CONSTRAINT follower_growth_snapshots_follower_count_check
        CHECK (follower_count >= 0)
);

CREATE TABLE follower_growth_snapshot_runs (
    snapshot_date          DATE        NOT NULL PRIMARY KEY,
    completed_at           TIMESTAMPTZ NOT NULL,
    captured_profile_count BIGINT      NOT NULL,

    CONSTRAINT follower_growth_snapshot_runs_captured_profile_count_check
        CHECK (captured_profile_count >= 0)
);
