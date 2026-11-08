-- Unknown is deliberately NULL for both existing and new profiles. Do not
-- populate these columns from historical records or inferred defaults.
ALTER TABLE bluesky_profiles
    ADD COLUMN guest_visibility_hidden BOOLEAN,
    ADD COLUMN guest_visibility_revision TEXT COLLATE "C",
    ADD COLUMN guest_visibility_record_cid TEXT,
    ADD COLUMN guest_visibility_evidence TEXT,
    ADD COLUMN guest_visibility_hold_key UUID,
    ADD CONSTRAINT bluesky_profiles_guest_visibility_evidence CHECK (
        guest_visibility_evidence IS NULL OR
        guest_visibility_evidence IN ('source', 'writeReceipt', 'observation')
    ),
    ADD CONSTRAINT bluesky_profiles_guest_visibility_known CHECK (
        guest_visibility_hidden IS NULL OR (
            guest_visibility_revision IS NOT NULL AND
            guest_visibility_revision <> '' AND
            guest_visibility_record_cid IS NOT NULL AND
            guest_visibility_evidence IS NOT NULL
        )
    );

CREATE INDEX bluesky_profiles_guest_visibility_hold_idx
    ON bluesky_profiles(guest_visibility_hold_key)
    WHERE guest_visibility_hold_key IS NOT NULL;
