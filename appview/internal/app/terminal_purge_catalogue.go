package app

import "social.craftsky/appview/internal/ownerlifecycle"

// terminalPurgeCatalogue is the fixed, cardinality-independent inventory
// installed with every terminal owner tombstone. Component workers may split
// a role into bounded table-specific batches, but adding a new DID-bearing
// serving/effect surface requires adding its role here and to the schema/query
// inventory test before terminal events can be acknowledged safely.
func terminalPurgeCatalogue() []ownerlifecycle.PurgeComponent {
	return []ownerlifecycle.PurgeComponent{
		{Component: "account_deletion", DIDRole: "owner"},
		{Component: "auth", DIDRole: "owner"},
		{Component: "effect_attempts", DIDRole: "owner"},
		{Component: "identity_cache", DIDRole: "owner"},
		{Component: "ingestion", DIDRole: "owner"},
		{Component: "instagram", DIDRole: "owner"},
		{Component: "instagram", DIDRole: "target"},
		{Component: "moderation", DIDRole: "source"},
		{Component: "moderation", DIDRole: "subject"},
		{Component: "notifications", DIDRole: "actor"},
		{Component: "notifications", DIDRole: "recipient"},
		{Component: "private_owner_data", DIDRole: "owner"},
		{Component: "public_content", DIDRole: "owner"},
		{Component: "public_mentions", DIDRole: "mentioned"},
		{Component: "public_profiles", DIDRole: "owner"},
		{Component: "public_relationships", DIDRole: "actor"},
		{Component: "public_relationships", DIDRole: "subject"},
		{Component: "push", DIDRole: "recipient"},
		{Component: "scheduled_media", DIDRole: "owner"},
	}
}
