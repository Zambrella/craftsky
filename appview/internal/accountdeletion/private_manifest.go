package accountdeletion

type ManifestKind string

const (
	ManifestTable   ManifestKind = "table"
	ManifestService ManifestKind = "service"
)

func (kind ManifestKind) Valid() bool {
	return kind == ManifestTable || kind == ManifestService
}

type ManifestPolicy string

const (
	ManifestDelete       ManifestPolicy = "delete"
	ManifestIndexerOwned ManifestPolicy = "indexerOwned"
	ManifestRetainShared ManifestPolicy = "retainShared"
)

func (policy ManifestPolicy) Valid() bool {
	return policy == ManifestDelete || policy == ManifestIndexerOwned || policy == ManifestRetainShared
}

type ManifestEntry struct {
	Kind              ManifestKind
	Name              string
	Policy            ManifestPolicy
	Rule              string
	OwnershipPath     string
	CleanupComponent  string
	VerificationQuery string
}

func PrivateDataManifest() []ManifestEntry {
	entries := manifestTableEntries(ManifestDelete, "deleteOwnerRows", []string{
		"account_deletion_cleanup_artifacts",
		"account_deletion_cleanup_steps",
		"account_deletion_expected_records",
		"account_deletion_index_receipts",
		"account_deletion_operations",
		"account_deletion_recovery_credentials",
		"account_deletion_status_credentials",
		"account_language_preferences",
		"actor_mutes",
		"craftsky_recent_searches",
		"craftsky_sessions",
		"instagram_account_links",
		"instagram_automatic_follow_ledger",
		"instagram_automatic_follow_sources",
		"instagram_audit_events",
		"instagram_graph_imports",
		"instagram_identity_claims",
		"instagram_link_conflicts",
		"instagram_reconciliation_jobs",
		"instagram_verification_attempts",
		"instagram_webhook_work",
		"moderation_outputs",
		"moderation_reports",
		"notification_events",
		"notification_preferences",
		"notification_seen_state",
		"oauth_auth_requests",
		"oauth_sessions",
		"pds_follow_operations",
		"profile_customisations",
		"profile_pins",
		"push_account_subscriptions",
		"push_deliveries",
		"saved_post_folders",
		"saved_posts",
		"scheduled_post_cleanup_jobs",
		"scheduled_post_media",
		"scheduled_post_publication_tombstones",
		"scheduled_posts",
	})
	entries = append(entries,
		newManifestEntry(ManifestTable, "account_deletion_audits", ManifestDelete, "expireAtTerminalPlus30Days"),
		newManifestEntry(ManifestTable, "instagram_graph_handles", ManifestDelete, "deleteOnlyWhenNoImportsReferenceHandle"),
		newManifestEntry(ManifestTable, "instagram_rate_limit_buckets", ManifestDelete, "deleteOwnerScopedBuckets"),
	)
	entries = append(entries, manifestTableEntries(ManifestIndexerOwned, "existingIndexerDeleteEventsOnly", []string{
		"craftsky_likes",
		"craftsky_post_mentions",
		"craftsky_posts",
		"craftsky_profiles",
		"craftsky_project_posts",
		"craftsky_reposts",
	})...)
	entries = append(entries, manifestTableEntries(ManifestRetainShared, "preserveSharedAtprotoData", []string{
		"atproto_blocks",
		"atproto_follows",
		"atproto_identity_cache",
		"bluesky_profiles",
	})...)
	entries = append(entries,
		newManifestEntry(ManifestTable, "push_installations", ManifestRetainShared, "deleteOnlyWhenNoSubscriptionsRemain"),
		newManifestEntry(ManifestService, "scheduledPostObjectStore", ManifestDelete, "deleteOwnerStagedObjects"),
		newManifestEntry(ManifestService, "instagramPrivateDataService", ManifestDelete, "purgeOwnerIncludingOrphans"),
		newManifestEntry(ManifestService, "craftskyPublicIndexers", ManifestIndexerOwned, "existingIndexerDeleteEventsOnly"),
		newManifestEntry(ManifestService, "sharedIdentityAndBlueskyCaches", ManifestRetainShared, "preserveSharedAtprotoData"),
	)
	return entries
}

func manifestTableEntries(policy ManifestPolicy, rule string, names []string) []ManifestEntry {
	entries := make([]ManifestEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, newManifestEntry(ManifestTable, name, policy, rule))
	}
	return entries
}

func newManifestEntry(kind ManifestKind, name string, policy ManifestPolicy, rule string) ManifestEntry {
	component := "databasePrivate"
	verificationQuery := "gate:" + name + ":" + rule
	if kind == ManifestTable {
		verificationQuery = "SELECT count(*) FROM " + name
	}
	switch {
	case policy == ManifestIndexerOwned:
		component = "craftskyPublicIndexers"
	case policy == ManifestRetainShared:
		component = "sharedIdentityAndBlueskyCaches"
	case kind == ManifestService && name == "scheduledPostObjectStore":
		component = "scheduledPosts"
	case kind == ManifestService && name == "instagramPrivateDataService":
		component = "instagramPrivate"
	case kind == ManifestService:
		component = name
	case name == "account_deletion_audits":
		component = "auditSweeper"
	case len(name) >= len("account_deletion_") && name[:len("account_deletion_")] == "account_deletion_":
		component = "terminalFinalization"
	case len(name) >= len("instagram_") && name[:len("instagram_")] == "instagram_":
		component = "instagramPrivate"
	case len(name) >= len("scheduled_post_") && name[:len("scheduled_post_")] == "scheduled_post_":
		component = "scheduledPosts"
	}
	return ManifestEntry{
		Kind: kind, Name: name, Policy: policy, Rule: rule,
		OwnershipPath: rule, CleanupComponent: component,
		VerificationQuery: verificationQuery,
	}
}
