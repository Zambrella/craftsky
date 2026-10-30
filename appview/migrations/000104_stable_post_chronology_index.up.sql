-- Match the original-observation clamp used by edit-safe read ordering.
DROP INDEX craftsky_posts_chronological_sort_idx;
CREATE INDEX craftsky_posts_chronological_sort_idx ON craftsky_posts
    (LEAST(created_at, indexed_at, profile_sort_at) DESC, uri DESC);
