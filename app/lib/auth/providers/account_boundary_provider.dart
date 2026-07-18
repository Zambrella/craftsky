import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/report_post_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_preferences_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_seen_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/report_profile_provider.dart';
import 'package:craftsky_app/profile/providers/save_profile_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_result_search_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/profile_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/search/providers/search_suggestions_provider.dart';
import 'package:craftsky_app/search/providers/top_hashtags_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef AccountBoundaryAction = Future<void> Function();
typedef AccountSessionInvalidator =
    Future<void> Function(AccountSessionLease lease);

class AccountSessionInvalidationCoordinator {
  AccountSessionInvalidationCoordinator({
    required this.readRegistry,
    required this.invalidateLease,
    required this.invalidateAccountState,
    required this.resetHome,
  });

  final Future<SessionRegistry> Function() readRegistry;
  final AccountSessionInvalidator invalidateLease;
  final AccountBoundaryAction invalidateAccountState;
  final AccountBoundaryAction resetHome;

  Future<void> invalidate(AccountSessionLease lease) async {
    final before = await readRegistry();
    final captured = before.leaseFor(lease.account);
    final removesActive = before.activeLease?.session == lease;

    await invalidateLease(lease);
    if (captured != lease || !removesActive) return;

    await invalidateAccountState();
    final after = await readRegistry();
    if (after.activeDid != null) await resetHome();
  }
}

final accountStateInvalidatorProvider = Provider<AccountBoundaryAction>(
  (ref) => () async {
    ref
      ..invalidate(postRepositoryProvider)
      ..invalidate(timelineProvider)
      ..invalidate(postProvider)
      ..invalidate(postCommentSectionProvider)
      ..invalidate(postCommentPageLoaderProvider)
      ..invalidate(postCommentRepliesLoaderProvider)
      ..invalidate(userPostsProvider)
      ..invalidate(userCommentsProvider)
      ..invalidate(createPostProvider)
      ..invalidate(deletePostProvider)
      ..invalidate(reportPostProvider)
      ..invalidate(toggleLikePostProvider)
      ..invalidate(toggleRepostPostProvider)
      ..invalidate(profileRepositoryProvider)
      ..invalidate(userProfileProvider)
      ..invalidate(saveProfileProvider)
      ..invalidate(reportProfileProvider)
      ..invalidate(toggleFollowProfileProvider)
      ..invalidate(projectRepositoryProvider)
      ..invalidate(projectFeedProvider)
      ..invalidate(userProjectsProvider)
      ..invalidate(searchRepositoryProvider)
      ..invalidate(blankSearchProvider)
      ..invalidate(postSearchProvider)
      ..invalidate(profileSearchProvider)
      ..invalidate(projectSearchProvider)
      ..invalidate(hashtagSearchProvider)
      ..invalidate(hashtagResultSearchProvider)
      ..invalidate(searchSuggestionsProvider)
      ..invalidate(topHashtagsProvider)
      ..invalidate(recentSearchPageProvider)
      ..invalidate(saveRecentSearchProvider)
      ..invalidate(deleteRecentSearchProvider)
      ..invalidate(notificationRepositoryProvider)
      ..invalidate(notificationsProvider)
      ..invalidate(notificationPreferencesProvider)
      ..invalidate(notificationSeenProvider)
      ..invalidate(notificationNewCountProvider);
  },
);

final accountHomeResetProvider = Provider<AccountBoundaryAction>(
  (ref) =>
      () async => ref.read(goRouterProvider).go(RouteLocations.home),
);

final accountSessionInvalidatorProvider = Provider<AccountSessionInvalidator>(
  (ref) => AccountSessionInvalidationCoordinator(
    readRegistry: () => ref.read(sessionRegistryProvider.future),
    invalidateLease: ref.read(sessionRegistryProvider.notifier).invalidate,
    invalidateAccountState: ref.read(accountStateInvalidatorProvider),
    resetHome: ref.read(accountHomeResetProvider),
  ).invalidate,
);
