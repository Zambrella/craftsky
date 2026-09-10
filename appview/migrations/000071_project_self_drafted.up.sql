ALTER TABLE craftsky_project_posts
    ADD COLUMN pattern_self_drafted BOOLEAN;

UPDATE craftsky_project_posts
SET pattern_self_drafted = CASE raw_project #>> '{common,pattern,selfDrafted}'
    WHEN 'true' THEN true
    WHEN 'false' THEN false
    ELSE NULL
END;

CREATE INDEX craftsky_project_posts_self_drafted_true_idx
    ON craftsky_project_posts (uri)
    WHERE pattern_self_drafted IS TRUE;
