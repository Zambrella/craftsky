ALTER TABLE tap_source_records
    RENAME COLUMN owner_generation TO projection_generation;

ALTER TABLE tap_source_records
    RENAME CONSTRAINT tap_source_records_owner_generation_check
    TO tap_source_records_projection_generation_check;

COMMENT ON COLUMN tap_source_records.projection_generation IS
    'Current lifecycle generation authorizing projection; effect_operation_id carries historical mutation provenance.';
