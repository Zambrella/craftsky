package ownerlifecycle

// TerminalDIDAction declares the physical treatment of one persisted DID
// role after the terminal tombstone has made the row invisible/ineffective.
type TerminalDIDAction string

const (
	TerminalDeleteRow       TerminalDIDAction = "delete_row"
	TerminalAnonymizeRow    TerminalDIDAction = "anonymize_row"
	TerminalRetainTombstone TerminalDIDAction = "retain_tombstone"
	TerminalRetainCleanup   TerminalDIDAction = "retain_cleanup"
)

// TerminalDIDEntry is one schema-backed DID role. Component and Role form the
// fixed owner_purge_components key installed before a terminal Tap event may
// be acknowledged. KeyColumns define the stable delete order used by bounded
// component workers.
type TerminalDIDEntry struct {
	Table      string
	Column     string
	Component  string
	Role       string
	Action     TerminalDIDAction
	KeyColumns []string
	Rationale  string
}

func deleteDID(table, column, role string, keyColumns ...string) TerminalDIDEntry {
	return TerminalDIDEntry{
		Table: table, Column: column, Component: table, Role: role,
		Action: TerminalDeleteRow, KeyColumns: keyColumns,
	}
}

func retainDID(table, column, role, rationale string, keyColumns ...string) TerminalDIDEntry {
	return TerminalDIDEntry{
		Table: table, Column: column, Component: table, Role: role,
		Action: TerminalRetainTombstone, KeyColumns: keyColumns, Rationale: rationale,
	}
}

func cleanupDID(table, column, role, rationale string, keyColumns ...string) TerminalDIDEntry {
	return TerminalDIDEntry{
		Table: table, Column: column, Component: table, Role: role,
		Action: TerminalRetainCleanup, KeyColumns: keyColumns, Rationale: rationale,
	}
}

func anonymizeDID(table, column, role, rationale string, keyColumns ...string) TerminalDIDEntry {
	return TerminalDIDEntry{
		Table: table, Column: column, Component: table, Role: role,
		Action: TerminalAnonymizeRow, KeyColumns: keyColumns, Rationale: rationale,
	}
}

// terminalDIDInventory deliberately names every persisted plaintext DID role
// in the fully migrated schema. Do not merge entries merely because one row
// can match more than one role: role-complete accounting is what makes the
// pre-ACK ledger exhaustive and auditable.
var terminalDIDInventory = []TerminalDIDEntry{
	deleteDID("account_deletion_operations", "owner_did", "owner", "id"),
	cleanupDID("account_deletion_safety_tombstones", "owner_did", "owner", "exact-key cleanup authority remains until uncertainty converges", "id"),
	deleteDID("account_language_preferences", "account_did", "owner", "account_did"),
	deleteDID("account_onboarding_completions", "account_did", "owner", "account_did"),
	deleteDID("actor_mutes", "owner_did", "owner", "owner_did", "subject_did"),
	deleteDID("actor_mutes", "subject_did", "subject", "owner_did", "subject_did"),
	deleteDID("atproto_blocks", "blocker_did", "actor", "uri"),
	deleteDID("atproto_blocks", "subject_did", "subject", "uri"),
	deleteDID("atproto_follows", "did", "actor", "uri"),
	deleteDID("atproto_follows", "subject_did", "subject", "uri"),
	deleteDID("atproto_identity_cache", "did", "owner", "did"),
	deleteDID("atproto_identity_refresh_state", "did", "owner", "did"),
	cleanupDID("auth_auxiliary_cleanup_jobs", "owner_did", "owner", "remote revocation cleanup remains durable after local denial", "id"),
	deleteDID("bluesky_profiles", "did", "owner", "did"),
	deleteDID("craftsky_account_types", "owner_did", "owner", "owner_did"),
	deleteDID("craftsky_business_events", "owner_did", "owner", "owner_did", "rkey"),
	deleteDID("craftsky_business_profiles", "owner_did", "owner", "owner_did"),
	deleteDID("craftsky_business_record_tombstones", "owner_did", "owner", "uri"),
	deleteDID("craftsky_likes", "did", "actor", "uri"),
	deleteDID("craftsky_post_mentions", "mentioned_did", "mentioned", "post_uri", "mentioned_did"),
	deleteDID("craftsky_posts", "did", "owner", "uri"),
	deleteDID("craftsky_profiles", "did", "owner", "did"),
	deleteDID("craftsky_recent_searches", "viewer_did", "owner", "id"),
	deleteDID("craftsky_reposts", "did", "actor", "uri"),
	deleteDID("craftsky_sessions", "account_did", "owner", "token_hash"),
	deleteDID("follower_growth_snapshots", "profile_did", "owner", "profile_did", "snapshot_date"),
	deleteDID("instagram_account_links", "owner_did", "owner", "id"),
	anonymizeDID("instagram_audit_events", "owner_did", "owner", "DID-free security history is retained without a subject identifier", "id"),
	deleteDID("instagram_automatic_follow_ledger", "importer_did", "actor", "id"),
	deleteDID("instagram_automatic_follow_ledger", "target_did", "target", "id"),
	deleteDID("instagram_graph_imports", "owner_did", "owner", "id"),
	deleteDID("instagram_identity_claims", "owner_did", "owner", "id"),
	deleteDID("instagram_private_suggestions", "importer_did", "actor", "id"),
	deleteDID("instagram_private_suggestions", "target_did", "target", "id"),
	deleteDID("instagram_reconciliation_jobs", "owner_did", "owner", "id"),
	deleteDID("instagram_reconciliation_jobs", "target_did", "target", "id"),
	deleteDID("instagram_verification_attempts", "owner_did", "owner", "id"),
	deleteDID("moderation_outputs", "source_did", "source", "id"),
	deleteDID("moderation_outputs", "subject_did", "subject", "id"),
	deleteDID("moderation_reports", "reporter_did", "reporter", "id"),
	deleteDID("moderation_reports", "subject_did", "subject", "id"),
	deleteDID("moderation_restoration_outbox", "target_did", "target", "moderation_output_id"),
	deleteDID("notification_events", "actor_did", "actor", "id"),
	deleteDID("notification_events", "recipient_did", "recipient", "id"),
	deleteDID("notification_preferences", "account_did", "owner", "account_did", "category"),
	deleteDID("notification_seen_state", "account_did", "owner", "account_did"),
	deleteDID("oauth_auth_requests", "account_deletion_owner_did", "deletion_owner", "state"),
	deleteDID("oauth_auth_requests", "owner_did", "owner", "state"),
	deleteDID("oauth_handoff_exchanges", "owner_did", "owner", "id"),
	deleteDID("oauth_sessions", "account_did", "owner", "account_did", "session_id"),
	deleteDID("owner_effect_attempts", "owner_did", "owner", "operation_id"),
	retainDID("owner_lifecycles", "owner_did", "owner", "irreversible terminal visibility tombstone", "owner_did"),
	retainDID("owner_purge_components", "owner_did", "owner", "fixed terminal purge completion ledger", "owner_did", "owner_generation", "component", "did_role"),
	deleteDID("pds_follow_operations", "owner_did", "actor", "id"),
	deleteDID("pds_follow_operations", "target_did", "target", "id"),
	deleteDID("profile_customisations", "owner_did", "owner", "owner_did"),
	deleteDID("profile_pins", "owner_did", "owner", "owner_did", "slot"),
	deleteDID("push_account_subscriptions", "account_did", "recipient", "id"),
	deleteDID("saved_post_folders", "owner_did", "owner", "id"),
	deleteDID("saved_posts", "owner_did", "owner", "owner_did", "post_uri"),
	cleanupDID("scheduled_post_cleanup_jobs", "owner_did", "owner", "object deletion intent remains until absence is proven", "id"),
	deleteDID("scheduled_post_media", "owner_did", "owner", "id"),
	deleteDID("scheduled_post_object_attempts", "owner_did", "owner", "upload_attempt_id"),
	deleteDID("scheduled_post_publication_tombstones", "owner_did", "owner", "schedule_id"),
	deleteDID("scheduled_posts", "owner_did", "owner", "id"),
	deleteDID("tap_repository_jobs", "did", "owner", "id"),
	deleteDID("tap_source_records", "did", "owner", "uri"),
}

// TerminalDIDInventory returns the complete immutable-by-convention schema
// inventory. The slice and nested key-column slices are copied so callers
// cannot weaken the catalogue used by Terminalize.
func TerminalDIDInventory() []TerminalDIDEntry {
	inventory := make([]TerminalDIDEntry, len(terminalDIDInventory))
	for index, entry := range terminalDIDInventory {
		inventory[index] = entry
		inventory[index].KeyColumns = append([]string(nil), entry.KeyColumns...)
	}
	return inventory
}

// TerminalPurgeCatalogue is the one package-owned, fixed-size ledger
// catalogue. Terminalize never accepts caller-selected security coverage.
func TerminalPurgeCatalogue() []PurgeComponent {
	components := make([]PurgeComponent, 0, len(terminalDIDInventory))
	for _, entry := range terminalDIDInventory {
		components = append(components, PurgeComponent{
			Component: entry.Component,
			DIDRole:   entry.Role,
		})
	}
	canonical, err := canonicalPurgeComponents(components)
	if err != nil || len(canonical) != len(components) {
		panic("ownerlifecycle: invalid or duplicate terminal DID inventory")
	}
	return canonical
}
