DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM billing_web_checkouts WHERE embedded) THEN
  RAISE EXCEPTION 'cannot remove embedded checkout state while operations remain';
 END IF;
END $$;
ALTER TABLE billing_web_checkouts DROP COLUMN embedded;
