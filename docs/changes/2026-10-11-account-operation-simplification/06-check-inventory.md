# Account Ownership Check Inventory

Completed inventory and disposition for the account-operation refactor. The baseline search found ownership candidates in 80 files. Counts include declarations and session lookups; they are not counts of removable checks. Necessary checks now often delegate to shared context methods rather than disappear.

The follow-up scan covered direct registry comparisons, session lookups, widget helpers, private-state boundary predicates and mutation revision guards. Synchronous selectors and target resolution remain separate from asynchronous authority checks.

| File | Baseline matches | Final disposition |
|---|---|---|
| `app/lib/auth/models/account_deletion.dart` | 1 | Retained: persisted deletion fence targets the exact session; reconciliation can outlive active UI. |
| `app/lib/auth/models/account_switcher_state.dart` | 1 | Retained: synchronous account selection/display state, not an async operation check. |
| `app/lib/auth/models/session_registry.dart` | 5 | Retained: source of session and activation generations, registry transitions and deletion acceptance. |
| `app/lib/auth/providers/account_activation_coordinator.dart` | 2 | Retained: validate the target retained session before/after unsaved-work confirmation and activation. |
| `app/lib/auth/providers/account_boundary_provider.dart` | 2 | Retained: central invalidation, private-data cleanup and deletion/session removal boundary. |
| `app/lib/auth/providers/account_operation_guard.dart` | 2 | Implemented: captured context, guarded acquisition/steps and synchronous publication; existing predicate remains authoritative. |
| `app/lib/business/pages/event_editor_dialog.dart` | 1 | Migrated: shared widget context; mounted UI, confirmation and form/record identity fences retained. |
| `app/lib/business/providers/business_event_detail_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/business/providers/business_event_mutation_controller.dart` | 8 | Migrated: captured repository and shared retry decisions; token fences and feature-specific exhaustion/error results retained. |
| `app/lib/business/providers/owner_business_events_provider.dart` | 5 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/business/providers/products_controller.dart` | 5 | Migrated: captured repository and shared retry decisions; token fences and feature-specific exhaustion/error results retained. |
| `app/lib/business/providers/profile_business_events_provider.dart` | 5 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/business/providers/report_business_event_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/business/widgets/event_form.dart` | 1 | Migrated: shared widget context; mounted UI, confirmation and form/record identity fences retained. |
| `app/lib/business/widgets/product_editor.dart` | 1 | Migrated: shared widget context; mounted UI, confirmation and form/record identity fences retained. |
| `app/lib/drafts/pages/drafts_page.dart` | 1 | Migrated: shared widget context; mounted UI, confirmation and form/record identity fences retained. |
| `app/lib/drafts/providers/draft_save_controller.dart` | 4 | Migrated: guarded acquisition/steps and publication; null discard and original failure propagation retained. |
| `app/lib/drafts/providers/local_post_drafts_provider.dart` | 3 | Migrated: shared active context; account family, account-match failure and cleanup retained. |
| `app/lib/feed/composer/composer_media_uploader.dart` | 6 | Migrated: shared composer ownership assertion; bound clients, step order and diagnostic error retained. |
| `app/lib/feed/composer/composer_submission_coordinator.dart` | 5 | Migrated: shared composer ownership assertion; bound clients, step order and diagnostic error retained. |
| `app/lib/feed/providers/create_post_provider.dart` | 5 | Migrated: captured repository and shared retry decisions; token fences and feature-specific exhaustion/error results retained. |
| `app/lib/feed/providers/delete_post_provider.dart` | 4 | Migrated: captured repository and shared retry decisions; token fences and feature-specific exhaustion/error results retained. |
| `app/lib/feed/providers/post_comment_section_provider.dart` | 4 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/post_interaction_lists_provider.dart` | 4 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/profile_pins_provider.dart` | 2 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/report_post_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/timeline_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/toggle_like_post_provider.dart` | 3 | Migrated: captured repository and shared active predicate; optimistic state, token revision and overlays retained. |
| `app/lib/feed/providers/toggle_repost_post_provider.dart` | 3 | Migrated: captured repository and shared active predicate; optimistic state, token revision and overlays retained. |
| `app/lib/feed/providers/user_comments_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/user_posts_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/providers/user_reposts_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/feed/widgets/post_composer_sheet.dart` | 29 | Migrated: captured context for submission/scheduling; widget lifetime, step checkpoints and origin cleanup retained. |
| `app/lib/instagram_migration/pages/instagram_migration_page.dart` | 1 | Migrated: shared active context; lease family, domain discard, polling/cleanup and action outcomes retained. |
| `app/lib/instagram_migration/providers/instagram_account_provider.dart` | 6 | Migrated: shared active context; lease family, domain discard, polling/cleanup and action outcomes retained. |
| `app/lib/instagram_migration/providers/instagram_imports_provider.dart` | 9 | Migrated: shared active context; lease family, domain discard, polling/cleanup and action outcomes retained. |
| `app/lib/instagram_migration/providers/instagram_migration_repository_provider.dart` | 3 | Migrated: shared active context; lease family, domain discard, polling/cleanup and action outcomes retained. |
| `app/lib/instagram_migration/providers/instagram_suggestions_provider.dart` | 8 | Migrated: shared active context; lease family, domain discard, polling/cleanup and action outcomes retained. |
| `app/lib/instagram_migration/providers/instagram_verification_provider.dart` | 14 | Migrated: shared active context; lease family, domain discard, polling/cleanup and action outcomes retained. |
| `app/lib/notifications/providers/notification_runtime_provider.dart` | 1 | Retained: recipient AccountSessionLease may target an inactive account; session resolution and device/open generations remain authoritative. |
| `app/lib/notifications/providers/notifications_provider.dart` | 1 | Retained: recipient AccountSessionLease may target an inactive account; session resolution and device/open generations remain authoritative. |
| `app/lib/notifications/services/notification_routing_storage.dart` | 4 | Retained: recipient AccountSessionLease may target an inactive account; session resolution and device/open generations remain authoritative. |
| `app/lib/onboarding/providers/onboarding_status_provider.dart` | 2 | Retained: session-scoped completion/retry state; active activation is deliberately not its authority. |
| `app/lib/profile/pages/edit_profile_dialog.dart` | 2 | Migrated: shared widget context; mounted UI, confirmation and form/record identity fences retained. |
| `app/lib/profile/providers/profile_customisation_provider.dart` | 2 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/profile/providers/profile_relationship_provider.dart` | 2 | Retained: AccountKey family uses AccountSessionLease; inactive retained-account reconciliation remains valid. |
| `app/lib/profile/providers/report_profile_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/profile/providers/save_profile_provider.dart` | 3 | Migrated: captured repository and shared retry decisions; token fences and feature-specific exhaustion/error results retained. |
| `app/lib/profile/providers/toggle_block_profile_provider.dart` | 3 | Migrated: captured repository and shared active predicate; optimistic state, token revision and overlays retained. |
| `app/lib/profile/providers/toggle_follow_profile_provider.dart` | 3 | Migrated: captured repository and shared active predicate; optimistic state, token revision and overlays retained. |
| `app/lib/profile/providers/user_profile_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/projects/providers/project_feed_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/projects/providers/user_projects_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/projects/widgets/project_composer_sheet.dart` | 26 | Migrated: captured context for submission/scheduling; widget lifetime, step checkpoints and origin cleanup retained. |
| `app/lib/saved_posts/providers/account_saved_post_state_provider.dart` | 1 | Retained: existing AccountKey family plus shared boundary epoch; private cleanup, pagination restart and entity revisions remain distinct. |
| `app/lib/saved_posts/providers/save_post_dialog_controller.dart` | 3 | Retained: existing AccountKey family plus shared boundary epoch; private cleanup, pagination restart and entity revisions remain distinct. |
| `app/lib/saved_posts/providers/saved_post_folders_provider.dart` | 8 | Retained: existing AccountKey family plus shared boundary epoch; private cleanup, pagination restart and entity revisions remain distinct. |
| `app/lib/saved_posts/providers/saved_post_repository_provider.dart` | 1 | Retained: existing AccountKey family plus shared boundary epoch; private cleanup, pagination restart and entity revisions remain distinct. |
| `app/lib/saved_posts/providers/saved_posts_provider.dart` | 5 | Retained: existing AccountKey family plus shared boundary epoch; private cleanup, pagination restart and entity revisions remain distinct. |
| `app/lib/scheduled_posts/pages/scheduled_posts_page.dart` | 1 | Migrated: shared widget context; mounted UI, confirmation and form/record identity fences retained. |
| `app/lib/scheduled_posts/providers/scheduled_posts_provider.dart` | 2 | Migrated: shared active context; account family, account-match failure and cleanup retained. |
| `app/lib/search/providers/hashtag_result_search_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/search/providers/hashtag_search_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/search/providers/post_search_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/search/providers/profile_search_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/search/providers/project_search_provider.dart` | 1 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/search/providers/recent_searches_provider.dart` | 2 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/settings/providers/account_deletion_controller.dart` | 2 | Retained: fresh deletion confirmation and exact-session acceptance/reconciliation fence, including inactive-owner cleanup. |
| `app/lib/settings/providers/relationship_list_provider.dart` | 4 | Migrated: captured active predicate/guarded publication; existing invalidation, request revisions and result/error contracts retained. |
| `app/lib/settings/services/account_deletion_acceptance_coordinator.dart` | 2 | Retained: fresh deletion confirmation and exact-session acceptance/reconciliation fence, including inactive-owner cleanup. |
| `app/lib/subscriptions/models/subscription_rules.dart` | 1 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/pages/subscription_page.dart` | 5 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/providers/customer_center_coordinator.dart` | 4 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/providers/subscription_access_provider.dart` | 1 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/providers/subscription_assignment_controller.dart` | 2 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/providers/subscription_page_model_provider.dart` | 3 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/providers/subscription_purchase_controller.dart` | 2 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/providers/subscription_restore_controller.dart` | 3 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/services/billing_owner_guard.dart` | 1 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |
| `app/lib/subscriptions/services/billing_owner_setup_coordinator.dart` | 1 | Retained: billing-owner identity, target session, SDK ownership, foreground and reconciliation generations are domain authority. |

## Additional Areas Reviewed

| Area | Disposition and reason |
|---|---|
| `shared/api/providers/dio_provider.dart` and `auth/services/session_validation_coordinator.dart` | Retained: fixed captured token/session target, client disposal, and lease-specific validation/401 invalidation. V-01 proves a replacement session cannot pass guarded acquisition. No new client cache or keep-alive was introduced. |
| `auth/models/account_session_lease.dart` and `pending_account_deletion.dart` | Retained: session and activation generations are different identities; deletion confirmation adds its own expiry/authority fence. |
| `auth/providers/active_account_identity_provider.dart`, `active_account_initialization_provider.dart`, `auth_session_provider.dart` | Retained: session identity publication and initialization/validation selectors intentionally use their existing authority and dependencies. These are not ordinary mutation retry guards. |
| `account_eligibility/providers/account_eligibility_provider.dart` | Retained: existing ActiveAccountLease family and gate failure contract. Adding another operation context to this small read would not remove its initialization dependencies or error boundaries. |
| `feed/providers/post_record_overlay.dart`, `like_post_overlay.dart`, profile follow/block overlays | Retained: session-owned mutation scopes and same-entity revisions are needed for reconciliation, independent of active completion suppression. |
| `shared/mutations/pds_record_operation_controller.dart` | Migrated retry policy orchestration into `PdsMutationRetry`, reused by the existing runner and outcome adapters. Controller `_isCurrent(token)`, immutable body/key validation, overlays, expiry and reset remain unchanged. |
| `notifications/services/notification_open_coordinator.dart`, `notification_runtime.dart`, registration coordinator | Retained: inactive-recipient resolution, latest-open work and installation/registration generations cannot use a generic active-operation predicate. |
| `notifications/widgets/notification_row.dart`, `notification_effect_host.dart` and notification pages | Retained: exact recipient/session presentation and effect delivery checks; the active session selector is part of routing/display authority. |
| `subscriptions/services/reconciliation_coordinator.dart` | Retained: reconciliation request/result generations and combined owner/foreground cancellation; not a PDS keyed-write retry. |
| Router, shell, saved-post/profile/business/settings page selectors | Retained: synchronous active-session selection, initialization gates and access checks; there is no captured async operation to replace these with. |
| Composer image/video/link-preview providers and coordinators | Retained existing client binding, media cache identity, cancellation and request-sequence guards. Shared uploader/coordinator assertions were consolidated without changing those transfer boundaries. |

## Retained-Guard Decisions

- **Session versus activation:** session-scoped relationship reconciliation, onboarding status, notification recipients and billing targets may legitimately refer to inactive accounts. Substituting active authority would change behavior.
- **Private state:** saved-post AccountKey families already share a boundary epoch that rebuilds providers and prevents stale publication. A new session family would duplicate that mechanism and change retention without reducing the revision checks. Draft and scheduled families likewise keep existing storage/cleanup semantics.
- **Same-account ordering:** request generations, current-state identity comparisons, mutation tokens and per-entity revisions remain in feed/search/business/relationship state. Capturing account authority does not replace them.
- **Widget continuation:** a guarded helper checks its own work. Its caller also checks authority before later dispatch, state publication or navigation, because awaiting a helper introduces a continuation boundary. Explicit `mounted` checks remain where BuildContext is used.
- **Specialized work:** auth switching/removal, notification delivery, billing SDK ownership and permanent-account deletion keep domain-specific fences. Their existing regression suites are part of the full Flutter verification.

No backend, API, lexicon, storage schema, dependency or generated-provider declaration changed. Existing client disposal and cancellation are authoritative; the refactor neither rolls back remote work nor retains obsolete clients.
