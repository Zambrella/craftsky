ALTER TABLE profile_customisations
    ADD COLUMN profile_border TEXT NOT NULL DEFAULT 'medium';

ALTER TABLE profile_customisations
    ALTER COLUMN profile_border DROP DEFAULT;
