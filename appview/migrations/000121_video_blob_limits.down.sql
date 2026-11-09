UPDATE bunny_video_jobs SET failure_code='processing_failed'
 WHERE failure_code IN ('pds_blob_too_large', 'video_too_large');
ALTER TABLE bunny_video_jobs DROP CONSTRAINT bunny_video_jobs_failure_code_check;
ALTER TABLE bunny_video_jobs ADD CONSTRAINT bunny_video_jobs_failure_code_check
 CHECK (failure_code IN ('', 'processing_failed', 'video_upload_timeout', 'video_upload_canceled'));
