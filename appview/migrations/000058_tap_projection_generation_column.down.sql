ALTER TABLE tap_source_records
    RENAME CONSTRAINT tap_source_records_projection_generation_check
    TO tap_source_records_owner_generation_check;

ALTER TABLE tap_source_records
    RENAME COLUMN projection_generation TO owner_generation;
