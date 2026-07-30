ALTER TABLE craftsky_posts
    ADD COLUMN langs TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX craftsky_posts_langs_gin
    ON craftsky_posts USING GIN (langs);

CREATE TABLE account_language_preferences (
    account_did       TEXT        NOT NULL PRIMARY KEY,
    primary_language  TEXT        NOT NULL,
    content_languages TEXT[]      NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
