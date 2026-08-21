CREATE INDEX account_deletion_operations_intent_expiry_idx
    ON account_deletion_operations(intent_expires_at, id)
    WHERE state = 'intent';
