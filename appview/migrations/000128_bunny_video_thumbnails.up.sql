ALTER TABLE bunny_video_jobs
    ADD COLUMN pending_thumbnail bytea CHECK (octet_length(pending_thumbnail) <= 1000000),
    ADD COLUMN thumbnail_filename text;
