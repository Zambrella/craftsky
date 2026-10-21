import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart' as model;
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_reposts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderBase;
import 'package:flutter_test/flutter_test.dart';

import '../../projects/fakes/fake_project_repository.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);
  final did = Did.parse('did:plc:alice');
  final post = PostMapper.fromMap({
    'uri': 'at://did:plc:alice/social.craftsky.feed.post/post',
    'cid': 'bafy_post',
    'rkey': 'post',
    'text': 'test',
    'tags': <String>[],
    'likeCount': 0,
    'repostCount': 0,
    'replyCount': 0,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'viewerHasSaved': false,
    'sponsored': false,
    'createdAt': '2026-05-04T18:23:45.000Z',
    'indexedAt': '2026-05-04T18:23:47.000Z',
    'author': {'did': 'did:plc:alice', 'handle': 'alice.test'},
  });
  for (final kind in ['projects', 'comments', 'reposts', 'thread']) {
    for (final leave in [false, true]) {
      final behavior = leave
          ? 'discards after leaving'
          : 'stays alive while observed';
      test(
        '$kind $behavior during a pending read',
        () async {
          final release = Completer<void>();
          final started = Completer<void>();
          Future<PostPage> page() async {
            started.complete();
            await release.future;
            return PostPage(items: [post]);
          }

          final container = ProviderContainer.test(
            overrides: [
              activeLanguagePreferencesProvider.overrideWith(
                (ref) => const LanguagePreferences(
                  primaryLanguage: 'en',
                  contentLanguages: ['en'],
                ),
              ),
              projectRepositoryProvider.overrideWithValue(
                FakeProjectRepository(
                  onListProjects: ({required query, limit, cursor}) => page(),
                ),
              ),
              postRepositoryProvider.overrideWithValue(
                FakePostRepository(
                  onListCommentsByAuthor: (_, {limit, cursor}) => page(),
                  onListRepostsByAuthor: (_, {limit, cursor}) => page(),
                  onCommentSection: (_, _, {cursor, sort, focus, limit}) async {
                    await page();
                    return model.PostCommentSection(
                      post: post,
                      comments: const model.CommentPage(items: []),
                      sort: model.CommentSort.oldest,
                    );
                  },
                ),
              ),
            ],
          );
          // Generated families otherwise infer Object instead of their shared
          // provider interface, which container.listen/exists require.
          // ignore: omit_local_variable_types
          final ProviderBase<AsyncValue<Object?>> provider = switch (kind) {
            'projects' => projectFeedProvider(const ProjectBrowseQuery()),
            'comments' => userCommentsProvider(did),
            'reposts' => userRepostsProvider(did),
            _ => postCommentSectionProvider(did, RecordKey.parse('post')),
          };
          final subscription = container.listen(provider, (_, _) {});
          final result = switch (kind) {
            'projects' => container.read(
              projectFeedProvider(const ProjectBrowseQuery()).future,
            ),
            'comments' => container.read(userCommentsProvider(did).future),
            'reposts' => container.read(userRepostsProvider(did).future),
            _ => container.read(
              postCommentSectionProvider(did, RecordKey.parse('post')).future,
            ),
          };
          final assertion = leave
              ? expectLater(result, throwsA(isA<ApiCanceled>()))
              : expectLater(result, completes);
          await started.future;
          if (leave) subscription.close();
          await container.pump();
          expect(container.exists(provider), !leave);
          release.complete();
          await assertion;
          if (!leave) subscription.close();
        },
      );
    }
  }
}
