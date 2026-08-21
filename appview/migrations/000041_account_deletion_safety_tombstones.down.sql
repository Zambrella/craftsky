DROP TABLE IF EXISTS account_deletion_safety_tombstones;

ALTER TABLE account_deletion_operations
    DROP CONSTRAINT IF EXISTS account_deletion_operations_id_owner_generation_key,
    DROP CONSTRAINT IF EXISTS account_deletion_operations_owner_generation_check,
    DROP COLUMN IF EXISTS owner_generation;
