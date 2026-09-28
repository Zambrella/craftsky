-- Public set state is now derived from tap_source_records via pds_set_sources
-- and pds_set_aggregates. These physical winner-era projections must not
-- survive as a second, stale interpretation of the same repository records.
-- This intentionally discards their data; down recreates empty compatibility
-- tables for the historical migration chain, not the discarded rows.
DROP TABLE craftsky_likes;
DROP TABLE craftsky_reposts;
DROP TABLE atproto_follows;
DROP TABLE atproto_blocks;
