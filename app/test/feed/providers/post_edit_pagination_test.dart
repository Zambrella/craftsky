import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/edit_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/feed/providers/user_reposts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../projects/fakes/fake_project_repository.dart';
import '../../search/fakes/fake_search_repository.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  for (final surface in [
    'profile',
    'comments',
    'reposts',
    'projects',
    'browse',
    'search',
    'projectSearch',
    'hashtag',
    'saved',
    'quotes',
  ]) {
    test(
      'REG-004 IR-006 $surface retains page two and cursor after edit',
      () async {
        final project = [
          'projects',
          'browse',
          'projectSearch',
        ].contains(surface);
        final post = _post(project: project).copyWith(
          reply: surface == 'comments'
              ? PostReply(
                  root: PostRef(
                    uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
                    cid: 'bafyroot',
                  ),
                  parent: PostRef(
                    uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
                    cid: 'bafyroot',
                  ),
                )
              : null,
          quote: surface == 'quotes'
              ? PostRef(
                  uri: 'at://did:plc:alice/social.craftsky.feed.post/target',
                  cid: 'bafytarget',
                )
              : null,
        );
        final first = post.copyWith(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/first',
          rkey: 'first',
        );
        PostPage page(String? cursor) => PostPage(
          items: [if (cursor == null) first else post],
          cursor: cursor == null ? 'second' : 'third',
        );
        SearchPostPage search(String? cursor) => SearchPostPage(
          items: page(cursor).items,
          cursor: page(cursor).cursor,
        );
        var writes = 0;
        final c = ProviderContainer.test(
          overrides: [
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            postRepositoryProvider.overrideWithValue(
              FakePostRepository(
                onFetch: (_, _) async => post,
                onListByAuthor: (_, {cursor, limit}) async => page(cursor),
                onListCommentsByAuthor: (_, {cursor, limit}) async =>
                    page(cursor),
                onListRepostsByAuthor: (_, {cursor, limit}) async =>
                    page(cursor),
                onListProjectsByAuthor: (_, {cursor, limit}) async =>
                    page(cursor),
                onListQuotes: (_, _, {cursor, limit}) async => page(cursor),
                onEdit:
                    (
                      did,
                      key, {
                      required changes,
                      required expectedCid,
                      required operationKey,
                    }) async {
                      writes++;
                      return PostEditResult(
                        acceptedCid: Cid.parse('bafyaccepted'),
                        changed: true,
                        editedAt: DateTime.utc(2026),
                        recordRevision: '3aaaaaaaaaaa3',
                      );
                    },
              ),
            ),
            projectRepositoryProvider.overrideWithValue(
              FakeProjectRepository(
                onListProjects: ({required query, cursor, limit}) async =>
                    page(cursor),
              ),
            ),
            searchRepositoryProvider.overrideWithValue(
              FakeSearchRepository(
                onSearchPosts: ({required q, cursor, limit}) async =>
                    search(cursor),
                onSearchProjects: ({required q, cursor, limit}) async =>
                    search(cursor),
                onSearchHashtagPosts: (_, {sort, cursor, limit}) async =>
                    search(cursor),
              ),
            ),
            accountSavedPostRepositoryProvider.overrideWith(
              (ref, account) async => _Saved(first, post),
            ),
          ],
        );
        final probe = _probe(c, surface, post);
        addTearDown(probe.close);
        final edits = c.listen(editPostProvider, (_, _) {});
        addTearDown(edits.close);
        await probe.load();
        await probe.more();
        expect(probe.posts().map((p) => p.rkey.value), ['first', 'base']);
        await c.read(editPostProvider.future);
        await c
            .read(editPostProvider.notifier)
            .save(
              seed: PostEditSeed(
                post: post,
                baseCid: post.cid,
                recordRevision: post.recordRevision,
                editable: {
                  'text': post.text,
                  'sponsored': false,
                  if (post.project != null) 'project': post.project!.toMap(),
                  if (project)
                    'images': [
                      {
                        'existingIndex': 0,
                        'alt': 'Existing image',
                        'image': {
                          r'$type': 'blob',
                          'ref': {r'$link': 'bafyimage'},
                          'mimeType': 'image/jpeg',
                          'size': 100,
                        },
                      },
                    ],
                },
              ),
              changes: PostEditChanges({'text': 'After'}),
            );
        expect(probe.posts().map((p) => p.rkey.value), ['first', 'base']);
        expect(probe.posts().last.text, 'After');
        expect(probe.posts().last.likeCount, 7);
        expect(probe.cursor(), 'third');
        expect(writes, 1);
      },
    );
  }
}

class _Probe {
  new({
    required this.load,
    required this.more,
    required this.posts,
    required this.cursor,
    required this.close,
  });
  final Future<void> Function() load;
  final Future<void> Function() more;
  final List<Post> Function() posts;
  final String? Function() cursor;
  final void Function() close;
}

_Probe _probe(ProviderContainer c, String surface, Post post) {
  if (surface == 'profile') {
    final p = userPostsProvider(post.author.did);
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'comments') {
    final p = userCommentsProvider(post.author.did);
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'reposts') {
    final p = userRepostsProvider(post.author.did);
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'projects') {
    final p = userProjectsProvider(post.author.did);
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'browse') {
    final p = projectFeedProvider(const ProjectBrowseQuery());
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'search') {
    final p = postSearchProvider(const PostSearchQuery(q: 'caption'));
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'projectSearch') {
    final p = projectSearchProvider(const ProjectSearchQuery(q: 'caption'));
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'hashtag') {
    final p = hashtagSearchProvider(const HashtagSearchQuery(tag: 'craft'));
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'saved') {
    final p = savedPostsProvider(
      SavedPostListKey(
        account: AccountKey('did:plc:alice'),
        scope: const SavedPostScope.all(),
        sort: SavedPostSort.newest,
      ),
    );
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () =>
          c.read(p).requireValue.items.map((item) => item.post).toList(),
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  if (surface == 'quotes') {
    final p = postQuotesProvider(post.author.did, RecordKey.parse('target'));
    final sub = c.listen(p, (_, _) {});
    return _Probe(
      load: () async {
        await c.read(p.future);
      },
      more: () => c.read(p.notifier).loadMore(),
      posts: () => c.read(p).requireValue.items,
      cursor: () => c.read(p).requireValue.cursor,
      close: sub.close,
    );
  }
  throw StateError('Unknown test surface');
}

Post _post({bool project = false}) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/base',
  cid: 'bafybase',
  rkey: 'base',
  text: 'Before',
  tags: const [],
  createdAt: DateTime.utc(2020),
  indexedAt: DateTime.utc(2020),
  author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
  likeCount: 7,
  repostCount: 2,
  replyCount: 1,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: true,
  sponsored: false,
  recordRevision: '3aaaaaaaaaaa2',
  images: project
      ? [
          PostImage(
            cid: 'bafyimage',
            mime: 'image/jpeg',
            size: 100,
            alt: 'Existing image',
            fullsize: 'https://cdn.example/image.jpg',
          ),
        ]
      : null,
  project: project
      ? const Project(
          common: ProjectCommon(
            craftType: 'social.craftsky.feed.defs#sewing',
            title: 'Shirt',
          ),
        )
      : null,
);

class _Saved extends Fake implements SavedPostRepository {
  new(this.first, this.post);
  final Post first;
  final Post post;
  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async => SavedPostPage(
    items: [
      SavedPostItem(
        post: cursor == null ? first : post,
        savedAt: DateTime.utc(2020),
        folderId: 'original',
      ),
    ],
    cursor: cursor == null ? 'second' : 'third',
  );
}
