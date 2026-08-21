ALTER TABLE tap_quarantined_events
    DROP CONSTRAINT IF EXISTS tap_quarantined_events_replay_envelope_check;

ALTER TABLE tap_quarantined_events
    DROP COLUMN IF EXISTS replay_envelope;
