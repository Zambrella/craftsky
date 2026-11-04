-- Browser installations cannot be represented by the previous constraint.
-- Keep existing rows and require operators to explicitly remove them before rollback.
ALTER TABLE push_installations DROP CONSTRAINT push_installations_platform_check;
ALTER TABLE push_installations ADD CONSTRAINT push_installations_platform_check CHECK (platform IN ('ios', 'android'));
