-- appview/migrations/000060_follower_growth_snapshots.down.sql
DROP TABLE IF EXISTS follower_growth_snapshot_runs;
DROP TABLE IF EXISTS follower_growth_snapshots;
DROP VIEW IF EXISTS craftsky_profile_follower_counts;
