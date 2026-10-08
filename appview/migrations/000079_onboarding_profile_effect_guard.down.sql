CREATE OR REPLACE FUNCTION reject_new_pds_record_effect_attempts()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.effect_kind = 'pds_record' THEN
        RAISE EXCEPTION 'new pds_record owner effect attempts are not supported'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
