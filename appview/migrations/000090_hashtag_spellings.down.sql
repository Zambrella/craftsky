ALTER TABLE craftsky_posts DROP COLUMN tag_spellings;
ALTER TABLE craftsky_posts DROP CONSTRAINT craftsky_posts_unambiguous_tag_source;
DROP FUNCTION craftsky_tag_source_unambiguous(JSONB, TEXT);
DROP FUNCTION craftsky_post_tag_spellings(JSONB, BOOLEAN);
DROP FUNCTION craftsky_facet_tag_spellings(TEXT, JSONB);
DROP FUNCTION craftsky_tag_field(JSONB, TEXT);
DROP FUNCTION craftsky_tag_trim(TEXT);
DROP FUNCTION craftsky_tag_lower(TEXT);
