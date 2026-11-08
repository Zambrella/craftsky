-- name: EnsureGuestVisibilityRow :exec
INSERT INTO bluesky_profiles(did, record_cid) VALUES ($1, $2)
ON CONFLICT (did) DO NOTHING;

-- name: LockGuestVisibility :one
SELECT guest_visibility_hidden, guest_visibility_revision, guest_visibility_record_cid,
       guest_visibility_evidence, guest_visibility_hold_key
FROM bluesky_profiles WHERE did = $1 FOR UPDATE;

-- name: SetGuestVisibilityEvidence :exec
UPDATE bluesky_profiles SET
    guest_visibility_hidden = $2,
    guest_visibility_revision = $3,
    guest_visibility_record_cid = $4,
    guest_visibility_evidence = $5
WHERE did = $1;

-- name: GuestVisibilityNeedsInitialization :one
SELECT NOT EXISTS (
    SELECT 1 FROM bluesky_profiles
    WHERE did = $1 AND guest_visibility_revision IS NOT NULL
);

-- name: HoldGuestVisibility :one
UPDATE bluesky_profiles SET guest_visibility_hold_key=$2
WHERE did=$1 AND (guest_visibility_hold_key IS NULL OR guest_visibility_hold_key=$2)
RETURNING did;

-- name: ReleaseGuestVisibilityHold :exec
UPDATE bluesky_profiles SET guest_visibility_hold_key=NULL
WHERE did=$1 AND guest_visibility_hold_key=$2;

-- name: LockGuestVisibilityCommand :one
SELECT state FROM pds_commands
WHERE id=$1 AND owner_did=$2 AND operation_kind='profile.guest_visibility'
FOR UPDATE;

-- name: LockHeldGuestVisibilityCommand :one
SELECT command.id,command.owner_generation,command.operation_key,
       command.immutable_request,command.state
FROM pds_commands command
JOIN bluesky_profiles profile ON profile.guest_visibility_hold_key=command.id
WHERE command.owner_did=$1 AND profile.did=command.owner_did
  AND command.operation_kind='profile.guest_visibility'
FOR UPDATE OF command;

-- name: ReleaseTerminalGuestVisibilityHold :execrows
UPDATE bluesky_profiles profile SET guest_visibility_hold_key=NULL
FROM pds_commands command
WHERE profile.did=$1 AND profile.guest_visibility_hold_key=$2
  AND command.id=profile.guest_visibility_hold_key
  AND command.owner_did=profile.did
  AND command.operation_kind='profile.guest_visibility'
  AND command.state IN ('accepted','rejected')
  AND NOT EXISTS (
      SELECT 1 FROM pds_command_dispatches dispatch
      WHERE dispatch.command_id=command.id
        AND dispatch.outcome IN ('dispatching','ambiguous')
  );
