package ownerlifecycle

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// terminalQueryInventory is the reviewed list of serving, effect, and
// projection boundaries that can make an owner visible or behaviorally
// effective while physical terminal purge is still pending. Keeping the list
// here makes that review executable: moving or replacing a boundary requires
// an explicit update to this security inventory.
var terminalQueryInventory = []struct {
	surface   string
	path      string
	fragments []string
}{
	{"profile reads and relationship counts", "api/profile_store.go", []string{"profileVisibleModerationPredicate", "accountProfileExists", "ListMutualFollowers", "appview_owner_is_terminal"}},
	{"post reads, relationships, and engagement", "api/post_store.go", []string{"postVisibleModerationPredicate", "RelationshipStates", "CountActiveLikes", "CountDescendantReplies", "appview_owner_is_terminal"}},
	{"timeline feed assembly", "api/timeline_store.go", []string{"ListTimelineWithLanguages", "appview_owner_is_terminal(f.did)", "appview_owner_is_terminal(r.did)"}},
	{"profile and post search", "api/search_store.go", []string{"SearchProfiles", "SearchPostsWithLanguages", "relationshipTopLevelPredicate", "appview_owner_is_terminal"}},
	{"mention search and resolution", "api/facet_store.go", []string{"SearchMentionSuggestions", "ResolveMention", "appview_owner_is_terminal"}},
	{"follow lookup and list", "api/follow_store.go", []string{"FindActiveFollow", "ListActiveFollowedDIDs", "appview_owner_is_terminal"}},
	{"identity cache reads and backfill", "api/identity_cache_store.go", []string{"FreshByHandle", "BackfillCandidateDIDs", "appview_owner_is_terminal"}},
	{"profile customisation hydration", "api/profile_customisation_store.go", []string{"ProfileCustomisationBatchQuery", "ReadBatch", "appview_owner_is_terminal"}},
	{"notification feed hydration", "api/notification_store.go", []string{"ListNotifications", "NotificationHandles", "appview_owner_is_terminal"}},
	{"notification newness", "api/notification_newness.go", []string{"NotificationNewCount", "appview_owner_is_terminal(event.recipient_did)", "appview_owner_is_terminal(event.actor_did)"}},
	{"moderation active policy", "api/moderation_store.go", []string{"ActivePolicyForSubject", "appview_owner_is_terminal(source_did)", "appview_owner_is_terminal(subject_did)"}},
	{"relationship reads", "relationships/store.go", []string{"IsMuted", "State", "ListMutes", "ListBlocks", "appview_owner_is_terminal"}},
	{"current-member admission", "relationships/membership.go", []string{"IsCurrentMember", "appview_owner_is_active"}},
	{"push claim and pre-send recheck", "push/dispatcher.go", []string{"claimOne", "ownsCurrentDelivery", "appview_owner_is_terminal(n.recipient_did)", "appview_owner_is_terminal(n.actor_did)"}},
	{"transactional Tap projection", "index/transactional_projectors.go", []string{"projectionMemberReady", "ReasonOwnerTerminal", "appview_owner_is_terminal"}},
	{"legacy follow indexing", "index/bluesky_follow.go", []string{"appview_owner_is_terminal"}},
	{"legacy post indexing", "index/craftsky_post.go", []string{"appview_owner_is_terminal"}},
	{"legacy interaction indexing", "index/craftsky_interaction.go", []string{"appview_owner_is_terminal"}},
	{"durable Tap admission", "ingestion/service.go", []string{"IngestRecord", "IngestIdentity", "StateTerminal", "ReasonOwnerTerminal"}},
	{"durable projection queue", "ingestion/store.go", []string{"initialProjectionOutcome", "projectionEligibility", "ReasonOwnerTerminal"}},
	{"uncertain Tap reconciliation", "ingestion/reconciliation.go", []string{"ReconcileSource", "StateTerminal", "ReasonOwnerTerminal"}},
	{"owner effect reconciliation", "ownerlifecycle/effect_store.go", []string{"ReconcileEffectAttempt", "StateTerminal"}},
	{"authoritative lifecycle predicate", "ownerlifecycle/lifecycle_store.go", []string{"IsTerminal", "appview_owner_is_terminal", "WithActiveEffects", "StateTerminal"}},
	{"fixed terminal component ledger", "ownerlifecycle/purge_store.go", []string{"TerminalPurgeCatalogue", "TerminalizeWith"}},
	{"OAuth continuation authority", "auth/session_lifecycle.go", []string{"StateTerminal", "AuthEpoch"}},
	{"OAuth handoff authority", "auth/handoff.go", []string{"StateTerminal", "AuthEpoch"}},
	{"OAuth cleanup authority", "auth/session_expiry_processor.go", []string{"StateTerminal"}},
}

func TestTerminalQueryInventoryCoversReviewedServingAndEffectBoundaries(t *testing.T) {
	seenSurface := make(map[string]struct{}, len(terminalQueryInventory))
	seenPath := make(map[string]struct{}, len(terminalQueryInventory))
	var failures []string
	for _, entry := range terminalQueryInventory {
		if strings.TrimSpace(entry.surface) == "" || strings.TrimSpace(entry.path) == "" || len(entry.fragments) == 0 {
			failures = append(failures, "terminal query inventory has an empty entry")
			continue
		}
		if _, duplicate := seenSurface[entry.surface]; duplicate {
			failures = append(failures, "duplicate terminal query surface: "+entry.surface)
		}
		seenSurface[entry.surface] = struct{}{}
		seenPath[entry.path] = struct{}{}

		contents, err := os.ReadFile(filepath.Join("..", entry.path))
		if err != nil {
			failures = append(failures, entry.surface+": "+err.Error())
			continue
		}
		for _, fragment := range entry.fragments {
			if !strings.Contains(string(contents), fragment) {
				failures = append(failures, entry.surface+" missing "+fragment+" in "+entry.path)
			}
		}
	}
	if len(seenPath) < 20 {
		failures = append(failures, "terminal query inventory unexpectedly shrank below 20 reviewed files")
	}
	if len(failures) != 0 {
		sort.Strings(failures)
		t.Fatalf("terminal query inventory failures:\n%s", strings.Join(failures, "\n"))
	}
}
