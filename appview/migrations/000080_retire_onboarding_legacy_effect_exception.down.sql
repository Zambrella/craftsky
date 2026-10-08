-- Restore the narrow onboarding allowance if rolling back the command writer.
CREATE OR REPLACE FUNCTION reject_new_pds_record_effect_attempts()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.effect_kind = 'pds_record' THEN
        IF NEW.effect_action = 'put_record'
            AND NEW.operation_id = 'oauth-onboarding-profile:' || NEW.owner_did || ':' || NEW.owner_generation
            AND NEW.mutation_key = NEW.operation_id
            AND NEW.deterministic_key = 'at://' || NEW.owner_did || '/social.craftsky.actor.profile/self'
            AND NEW.expected_cid IS NULL
            AND NEW.remote_outcome = 'prepared'
            AND EXISTS (
                SELECT 1 FROM owner_lifecycles
                WHERE owner_did = NEW.owner_did
                  AND generation = NEW.owner_generation
                  AND state = 'departed'
            ) THEN
            RETURN NEW;
        END IF;
        RAISE EXCEPTION 'new pds_record owner effect attempts are not supported'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
