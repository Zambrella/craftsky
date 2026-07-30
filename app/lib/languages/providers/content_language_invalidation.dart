import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef ContentLanguageCacheInvalidator = void Function();

enum ContentLanguageCache {
  timeline,
  projectsBrowse,
  postSearch,
  projectSearch,
  hashtagPosts,
  profilePosts,
  profileProjects,
  profileComments,
}

const List<ContentLanguageCache> contentLanguageCacheInventory =
    ContentLanguageCache.values;

final contentLanguageCacheInvalidatorProvider =
    Provider<ContentLanguageCacheInvalidator>(
      (ref) =>
          () => invalidateContentLanguageCaches(ref),
    );

void invalidateContentLanguageCaches(Ref ref) {
  for (final cache in contentLanguageCacheInventory) {
    switch (cache) {
      case ContentLanguageCache.timeline:
        ref.invalidate(timelineProvider);
      case ContentLanguageCache.projectsBrowse:
        ref.invalidate(projectFeedProvider);
      case ContentLanguageCache.postSearch:
        ref.invalidate(postSearchProvider);
      case ContentLanguageCache.projectSearch:
        ref.invalidate(projectSearchProvider);
      case ContentLanguageCache.hashtagPosts:
        ref.invalidate(hashtagSearchProvider);
      case ContentLanguageCache.profilePosts:
        ref.invalidate(userPostsProvider);
      case ContentLanguageCache.profileProjects:
        ref.invalidate(userProjectsProvider);
      case ContentLanguageCache.profileComments:
        ref.invalidate(userCommentsProvider);
    }
  }
}
