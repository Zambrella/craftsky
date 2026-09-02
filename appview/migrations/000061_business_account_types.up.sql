CREATE TABLE craftsky_account_types (
    owner_did    TEXT NOT NULL,
    account_type TEXT NOT NULL,

    CONSTRAINT craftsky_account_types_pkey
        PRIMARY KEY (owner_did),
    CONSTRAINT craftsky_account_types_account_type_check
        CHECK (account_type IN ('regular', 'business'))
);
