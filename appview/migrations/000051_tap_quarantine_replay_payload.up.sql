ALTER TABLE tap_quarantined_events
    ADD COLUMN replay_envelope BYTEA;

ALTER TABLE tap_quarantined_events
    ADD CONSTRAINT tap_quarantined_events_replay_envelope_check
    CHECK (
        replay_envelope IS NULL
        OR octet_length(replay_envelope) BETWEEN 1 AND 2097152
    );

COMMENT ON COLUMN tap_quarantined_events.replay_envelope IS
    'Exact bounded Tap frame used only by leased replay; intentionally excluded from operator listings';
