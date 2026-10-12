-- Author-supplied historical dates remain intact; future dates cannot pin a post.
UPDATE craftsky_posts SET profile_sort_at=LEAST(profile_sort_at,indexed_at)
WHERE profile_sort_at>indexed_at;
CREATE INDEX craftsky_posts_chronological_sort_idx ON craftsky_posts
    (LEAST(created_at,indexed_at) DESC,uri DESC);
