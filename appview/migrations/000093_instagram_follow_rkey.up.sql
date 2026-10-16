-- Reserve a lexicon-valid TID before dispatch so acceptance retries use the
-- same public follow record identity. Existing terminal results stay intact.
ALTER TABLE instagram_private_suggestions
    ADD COLUMN follow_rkey TEXT CHECK (follow_rkey ~ '^[234567abcdefghij][234567abcdefghijklmnopqrstuvwxyz]{12}$');
