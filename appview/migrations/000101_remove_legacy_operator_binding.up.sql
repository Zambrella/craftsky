-- Human console access is bound to verified Atmosphere DIDs. The retired
-- external UI identifier is unused; stable operator IDs and audits remain.
ALTER TABLE safety_operators DROP COLUMN retool_user_id;
