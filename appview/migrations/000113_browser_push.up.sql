ALTER TABLE push_installations DROP CONSTRAINT push_installations_platform_check;
ALTER TABLE push_installations ADD CONSTRAINT push_installations_platform_check CHECK (platform IN ('ios', 'android', 'web'));
