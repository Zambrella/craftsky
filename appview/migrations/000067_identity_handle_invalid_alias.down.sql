DROP INDEX atproto_identity_cache_valid_handle_lower_unique;

DELETE FROM atproto_identity_cache cache
USING atproto_identity_cache duplicate
WHERE cache.handle_lower = 'handle.invalid'
  AND duplicate.handle_lower = cache.handle_lower
  AND duplicate.did < cache.did;

ALTER TABLE atproto_identity_cache
    ADD CONSTRAINT atproto_identity_cache_handle_lower_key UNIQUE (handle_lower);
