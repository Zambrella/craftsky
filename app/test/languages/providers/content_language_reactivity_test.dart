import 'dart:io';

import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../feed/fakes/fake_post_repository.dart';
import '../../projects/fakes/fake_project_repository.dart';
import '../../search/fakes/fake_search_repository.dart';

final _preferencesProvider =
    NotifierProvider<_Preferences, LanguagePreferences>(_Preferences.new);

final class _Preferences extends Notifier<LanguagePreferences> {
  @override
  LanguagePreferences build() => const LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en'],
  );

  LanguagePreferences get preferences => state;

  set preferences(LanguagePreferences preferences) => state = preferences;
}

void main() {
  test(
    'all eight filtered families refetch only for content-language changes',
    () async {
      final calls = <String, int>{};
      void called(String name) => calls.update(
        name,
        (count) => count + 1,
        ifAbsent: () => 1,
      );
      final postRepository = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          called('timeline');
          return const TimelinePage(items: []);
        },
        onListByAuthor: (id, {cursor, limit}) async {
          called('userPosts');
          return const PostPage(items: []);
        },
        onListProjectsByAuthor: (id, {cursor, limit}) async {
          called('userProjects');
          return const PostPage(items: []);
        },
        onListCommentsByAuthor: (id, {cursor, limit}) async {
          called('userComments');
          return const PostPage(items: []);
        },
      );
      final projectRepository = FakeProjectRepository(
        onListProjects: ({required query, cursor, limit}) async {
          called('projectBrowse');
          return const PostPage(items: []);
        },
      );
      final searchRepository = FakeSearchRepository(
        onSearchHashtagPosts: (tag, {sort, cursor, limit}) async {
          called('hashtagSearch');
          return const SearchPostPage(items: []);
        },
        onSearchPosts: ({required q, cursor, limit}) async {
          called('postSearch');
          return const SearchPostPage(items: []);
        },
        onSearchProjects: ({required q, cursor, limit}) async {
          called('projectSearch');
          return const SearchPostPage(items: []);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => ref.watch(_preferencesProvider),
          ),
          postRepositoryProvider.overrideWithValue(postRepository),
          projectRepositoryProvider.overrideWithValue(projectRepository),
          searchRepositoryProvider.overrideWithValue(searchRepository),
        ],
      );
      final subscriptions = <ProviderSubscription<Object?>>[
        container.listen<Object?>(timelineProvider, (_, _) {}),
        container.listen<Object?>(
          projectFeedProvider(const ProjectBrowseQuery()),
          (_, _) {},
        ),
        container.listen<Object?>(
          postSearchProvider(const PostSearchQuery(q: 'query')),
          (_, _) {},
        ),
        container.listen<Object?>(
          projectSearchProvider(const ProjectSearchQuery(q: 'query')),
          (_, _) {},
        ),
        container.listen<Object?>(
          hashtagSearchProvider(const HashtagSearchQuery(tag: 'tag')),
          (_, _) {},
        ),
        container.listen<Object?>(userPostsProvider('alice.test'), (_, _) {}),
        container.listen<Object?>(
          userProjectsProvider('alice.test'),
          (_, _) {},
        ),
        container.listen<Object?>(
          userCommentsProvider('alice.test'),
          (_, _) {},
        ),
      ];
      addTearDown(() {
        for (final subscription in subscriptions) {
          subscription.close();
        }
      });

      await Future.wait([
        container.read(timelineProvider.future),
        container.read(
          projectFeedProvider(const ProjectBrowseQuery()).future,
        ),
        container.read(
          postSearchProvider(const PostSearchQuery(q: 'query')).future,
        ),
        container.read(
          projectSearchProvider(const ProjectSearchQuery(q: 'query')).future,
        ),
        container.read(
          hashtagSearchProvider(const HashtagSearchQuery(tag: 'tag')).future,
        ),
        container.read(userPostsProvider('alice.test').future),
        container.read(userProjectsProvider('alice.test').future),
        container.read(userCommentsProvider('alice.test').future),
      ]);
      expect(calls.values, everyElement(1));
      expect(calls, hasLength(8));

      container
          .read(_preferencesProvider.notifier)
          .preferences = const LanguagePreferences(
        primaryLanguage: 'fr',
        contentLanguages: ['en'],
      );
      await pumpEventQueue();
      expect(calls.values, everyElement(1));

      container
          .read(_preferencesProvider.notifier)
          .preferences = const LanguagePreferences(
        primaryLanguage: 'fr',
        contentLanguages: ['fr'],
      );
      await _waitFor(() => calls.values.every((count) => count == 2));
      expect(calls.values, everyElement(2));
    },
  );

  test(
    'filtered providers use no future policy barrier or cache invalidator',
    () {
      const providerFiles = [
        'lib/feed/providers/timeline_provider.dart',
        'lib/feed/providers/user_comments_provider.dart',
        'lib/feed/providers/user_posts_provider.dart',
        'lib/projects/providers/project_feed_provider.dart',
        'lib/projects/providers/user_projects_provider.dart',
        'lib/search/providers/hashtag_search_provider.dart',
        'lib/search/providers/post_search_provider.dart',
        'lib/search/providers/project_search_provider.dart',
      ];

      for (final path in providerFiles) {
        final source = File(path).readAsStringSync();
        expect(
          source,
          contains('ref.watch(activeContentLanguagePolicyProvider)'),
        );
        expect(
          source,
          isNot(contains('activeContentLanguagePolicyProvider.future')),
        );
      }
      expect(
        File(
          'lib/languages/providers/content_language_invalidation.dart',
        ).existsSync(),
        isFalse,
      );
    },
  );
}

Future<void> _waitFor(bool Function() condition) async {
  for (var attempt = 0; attempt < 20; attempt++) {
    if (condition()) return;
    await pumpEventQueue();
  }
  fail('Condition did not become true');
}
