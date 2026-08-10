import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/profile_pins_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_result_search_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/profile_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_suggestions_provider.dart';
import 'package:craftsky_app/settings/providers/relationship_list_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Provider families whose retained values can contain a public identity.
///
/// Invalidation is intentionally centralized so adding a new identity-bearing
/// cache requires one auditable inventory change instead of another mutation-
/// specific list.
final Map<Object, void Function(Ref)> _profileIdentityInvalidations = {
  postProvider: (ref) => ref.invalidate(postProvider),
  postCommentSectionProvider: (ref) =>
      ref.invalidate(postCommentSectionProvider),
  postCommentPageLoaderProvider: (ref) =>
      ref.invalidate(postCommentPageLoaderProvider),
  postCommentRepliesLoaderProvider: (ref) =>
      ref.invalidate(postCommentRepliesLoaderProvider),
  profilePinsProvider: (ref) => ref.invalidate(profilePinsProvider),
  timelineProvider: (ref) => ref.invalidate(timelineProvider),
  userPostsProvider: (ref) => ref.invalidate(userPostsProvider),
  userCommentsProvider: (ref) => ref.invalidate(userCommentsProvider),
  notificationsProvider: (ref) => ref.invalidate(notificationsProvider),
  accountNotificationsProvider: (ref) =>
      ref.invalidate(accountNotificationsProvider),
  projectFeedProvider: (ref) => ref.invalidate(projectFeedProvider),
  userProjectsProvider: (ref) => ref.invalidate(userProjectsProvider),
  savedPostsProvider: (ref) => ref.invalidate(savedPostsProvider),
  blankSearchProvider: (ref) => ref.invalidate(blankSearchProvider),
  postSearchProvider: (ref) => ref.invalidate(postSearchProvider),
  profileSearchProvider: (ref) => ref.invalidate(profileSearchProvider),
  projectSearchProvider: (ref) => ref.invalidate(projectSearchProvider),
  hashtagSearchProvider: (ref) => ref.invalidate(hashtagSearchProvider),
  hashtagResultSearchProvider: (ref) =>
      ref.invalidate(hashtagResultSearchProvider),
  searchSuggestionsProvider: (ref) => ref.invalidate(searchSuggestionsProvider),
  recentSearchPageProvider: (ref) => ref.invalidate(recentSearchPageProvider),
  relationshipListProvider: (ref) => ref.invalidate(relationshipListProvider),
};

Set<Object> get profileIdentityBearingProviderFamilies =>
    _profileIdentityInvalidations.keys.toSet();

final profileIdentityCacheInvalidatorProvider = Provider<void Function()>(
  (ref) => () {
    for (final invalidate in _profileIdentityInvalidations.values) {
      invalidate(ref);
    }
  },
);
