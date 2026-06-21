ALTER TABLE craftsky_recent_searches
    DROP CONSTRAINT IF EXISTS craftsky_recent_searches_search_type_check;

ALTER TABLE craftsky_recent_searches
    ADD CONSTRAINT craftsky_recent_searches_search_type_check
    CHECK (search_type IN ('query', 'hashtag', 'profile', 'post', 'project'));
