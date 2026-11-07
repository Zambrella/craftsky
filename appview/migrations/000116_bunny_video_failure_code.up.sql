ALTER TABLE bunny_video_jobs ADD COLUMN failure_code TEXT NOT NULL DEFAULT ''
 CHECK (failure_code IN ('', 'processing_failed', 'video_upload_timeout'));
