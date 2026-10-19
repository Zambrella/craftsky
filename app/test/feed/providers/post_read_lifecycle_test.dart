import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart' as model;
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart' as single;
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_reposts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../projects/fakes/fake_project_repository.dart';
import '../fakes/fake_post_repository.dart';

final _did = Did.parse('did:plc:alice');
final _rkey = RecordKey.parse('root');
const _query = ProjectBrowseQuery();

Post _post() => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
  cid: 'bafy_root',
  rkey: 'root',
  text: 'root',
  tags: const [],
  createdAt: DateTime.utc(2026),
  indexedAt: DateTime.utc(2026),
  author: PostAuthor(did: _did, handle: 'alice.craftsky.social'),
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  sponsored: false,
);

void main() {
  setUpAll(initializeMappers);

  test(
    'single post late read cancels before reading disposed overlay state',
    () async {
      final late = Completer<Post>();
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onFetch: (_, _) => late.future,
            ),
          ),
        ],
      );
      final reference = Provider<Ref>((ref) => ref);
      final subscription = container.listen(reference, (_, _) {});
      final pending = single.post(container.read(reference), _did, _rkey);
      final canceled = expectLater(pending, throwsA(isA<ApiCanceled>()));
      subscription.close();
      container.dispose();
      late.complete(_post());
      await canceled;
    },
  );

  for (final kind in ['comments', 'projects', 'reposts', 'browse', 'thread']) {
    test(
      '$kind late build cancels before reading disposed overlay state',
      () async {
        final late = Completer<PostPage>();
        var requests = 0;
        Future<PostPage> fetch() async {
          if (requests++ == 0) return const PostPage(items: [], cursor: 'next');
          return await late.future;
        }

        final repository = FakePostRepository(
          onListCommentsByAuthor: (did, {cursor, limit}) => fetch(),
          onListProjectsByAuthor: (did, {cursor, limit}) => fetch(),
          onListRepostsByAuthor: (did, {cursor, limit}) => fetch(),
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
            await fetch();
            return model.PostCommentSection(
              post: _post(),
              sort: model.CommentSort.oldest,
              comments: const model.CommentPage(items: []),
            );
          },
        );
        final container = ProviderContainer.test(
          overrides: [
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: [],
              ),
            ),
            postRepositoryProvider.overrideWithValue(repository),
            projectRepositoryProvider.overrideWithValue(
              FakeProjectRepository(
                onListProjects: ({required query, cursor, limit}) => fetch(),
              ),
            ),
          ],
        );
        final subscription = switch (kind) {
          'comments' => container.listen(userCommentsProvider(_did), (_, _) {}),
          'projects' => container.listen(userProjectsProvider(_did), (_, _) {}),
          'reposts' => container.listen(userRepostsProvider(_did), (_, _) {}),
          'browse' => container.listen(projectFeedProvider(_query), (_, _) {}),
          _ => container.listen(
            postCommentSectionProvider(_did, _rkey),
            (_, _) {},
          ),
        };
        await switch (kind) {
          'comments' => container.read(userCommentsProvider(_did).future),
          'projects' => container.read(userProjectsProvider(_did).future),
          'reposts' => container.read(userRepostsProvider(_did).future),
          'browse' => container.read(projectFeedProvider(_query).future),
          _ => container.read(postCommentSectionProvider(_did, _rkey).future),
        };
        final pending = switch (kind) {
          'comments' =>
            container.read(userCommentsProvider(_did).notifier).build(_did),
          'projects' =>
            container.read(userProjectsProvider(_did).notifier).build(_did),
          'reposts' =>
            container.read(userRepostsProvider(_did).notifier).build(_did),
          'browse' =>
            container.read(projectFeedProvider(_query).notifier).build(_query),
          _ =>
            container
                .read(postCommentSectionProvider(_did, _rkey).notifier)
                .build(_did, _rkey),
        };
        final canceled = expectLater(pending, throwsA(isA<ApiCanceled>()));
        subscription.close();
        await container.pump();
        late.complete(PostPage(items: [_post()]));
        await canceled;
      },
    );
  }
}
