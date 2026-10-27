import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart' as comments;
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/edit_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_edit_seed_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../search/fakes/fake_search_repository.dart';
import '../../test_support/session_registry_storage.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  test('IT-008 replacement video exposes a missing blob to the '
      'bounded recovery coordinator', () async {
    final seed = _seed();
    const error = ApiBadRequest(
      'video_blob_missing',
      details: ApiFailureDetails(appViewError: 'video_blob_missing'),
    );
    final stack = StackTrace.fromString('original missing video blob stack');
    final container = ProviderContainer.test(
      overrides: [
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(
            onEdit: (
              did,
              key, {
              required changes,
              required expectedCid,
              required operationKey,
            }) async => Error.throwWithStackTrace(error, stack),
          ),
        ),
      ],
    );
    final subscription = container.listen(editPostProvider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(editPostProvider.future);
    await expectLater(
      container
          .read(editPostProvider.notifier)
          .save(
            seed: seed,
            changes: PostEditChanges({'text': 'After'}),
            allowVideoBlobRecovery: true,
          ),
      throwsA(same(error)),
    );
    expect(container.read(editPostProvider).hasError, isFalse);
  });
  test(
    'AT-006 unresolved edit keeps its exact key, body and diagnostic cause',
    () async {
      final seed = _seed();
      final stack = StackTrace.fromString('original edit transport stack');
      final cause = PdsMutationAmbiguousException(
        retryAfterSeconds: 1,
        diagnosticCause: const ApiNetworkError('connection lost'),
        diagnosticStack: stack,
      );
      final keys = <String>[];
      final bases = <Object>[];
      final bodies = <Map<String, Object?>>[];
      var accept = false;
      final acceptedPost = seed.post.copyWith(
        cid: 'bafyaccepted',
        text: 'After',
      );
      final container = ProviderContainer.test(
        overrides: [
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onEdit:
                  (
                    did,
                    key, {
                    required changes,
                    required expectedCid,
                    required operationKey,
                  }) async {
                    keys.add(operationKey);
                    bases.add(expectedCid);
                    bodies.add(changes.toMap());
                    if (!accept) Error.throwWithStackTrace(cause, stack);
                    return PostEditResult(
                      post: acceptedPost,
                      acceptedCid: acceptedPost.cid,
                      changed: true,
                      projectionPending: false,
                      editedAt: DateTime.utc(2026),
                      editTimeSource: PostEditTimeSource.confirmed,
                    );
                  },
            ),
          ),
        ],
      );
      final subscription = container.listen(editPostProvider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(editPostProvider.future);
      final notifier = container.read(editPostProvider.notifier);
      final changes = PostEditChanges({'text': 'After'});
      expect(await notifier.save(seed: seed, changes: changes), isNull);
      expect(keys, hasLength(7));
      final state = container.read(editPostProvider);
      final unresolved = state.error! as PdsMutationUnresolvedException;
      expect(unresolved.diagnosticCause, same(cause));
      expect(unresolved.diagnosticStack, same(stack));
      expect(state.stackTrace, same(stack));
      accept = true;
      final result = await notifier.save(seed: seed, changes: changes);
      expect(result?.post?.text, 'After');
      expect(keys, hasLength(8));
      expect(keys.toSet(), hasLength(1));
      expect(bases, everyElement(seed.baseCid));
      expect(bodies, everyElement({'text': 'After'}));
    },
  );
  test(
    'IT-015 saves the exact edit base through the existing repository',
    () async {
      final seed = _seed();
      final original = seed.post;
      var calls = 0;
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onEdit:
                  (
                    did,
                    key, {
                    required changes,
                    required expectedCid,
                    required operationKey,
                  }) async {
                    calls++;
                    expect(did, original.author.did);
                    expect(key, original.rkey);
                    expect(expectedCid, seed.baseCid);
                    expect(changes.toMap(), {'text': 'After'});
                    expect(operationKey, isNotEmpty);
                    return PostEditResult(
                      acceptedCid: Cid.parse('bafyaccepted'),
                      changed: true,
                      projectionPending: false,
                      post: original.copyWith(
                        cid: 'bafyaccepted',
                        text: 'After',
                      ),
                      editedAt: DateTime.utc(2026, 10, 26),
                      editTimeSource: PostEditTimeSource.confirmed,
                    );
                  },
            ),
          ),
        ],
      );
      final subscription = container.listen(editPostProvider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(editPostProvider.future);
      final result = await container
          .read(editPostProvider.notifier)
          .save(seed: seed, changes: PostEditChanges({'text': 'After'}));
      expect(calls, 1);
      expect(result?.post?.text, 'After');
      expect(result?.post?.likeCount, 7);
      expect(container.read(editPostProvider).value?.changed, isTrue);
    },
  );
  test('UT-007 accepted edit replaces existing rows without insertion or '
      'count rollback', () async {
    final seed = _seed();
    final savedKey = SavedPostListKey(
      account: AccountKey('did:plc:alice'),
      scope: const SavedPostScope.all(),
      sort: SavedPostSort.newest,
    );
    var timelineReads = 0;
    final repository = FakePostRepository(
      onCommentSection:
          (
            did,
            key, {
            sort = comments.CommentSort.oldest,
            focus,
            cursor,
            limit,
          }) async => comments.PostCommentSection(
            post: seed.post.copyWith(
              uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
              rkey: 'root',
            ),
            sort: sort ?? comments.CommentSort.oldest,
            comments: comments.CommentPage(
              items: [
                comments.CommentItem(
                  post: seed.post.copyWith(likeCount: 11),
                  placement: comments.CommentPlacement.normal,
                  replies: const comments.ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
      onListTimeline: ({cursor, limit}) async {
        timelineReads++;
        return TimelinePage(
          items: [
            TimelineItem(itemKey: 'post:${seed.post.uri}', post: seed.post),
          ],
        );
      },
      onFetch: (_, _) async => seed.post.copyWith(likeCount: 9),
      onEdit:
          (
            did,
            key, {
            required changes,
            required expectedCid,
            required operationKey,
          }) async => PostEditResult(
            acceptedCid: Cid.parse('bafyaccepted'),
            changed: true,
            projectionPending: false,
            post: seed.post.copyWith(
              cid: 'bafyaccepted',
              text: 'After',
              likeCount: 7,
            ),
            editedAt: DateTime.utc(2026, 10, 26),
            editTimeSource: PostEditTimeSource.confirmed,
            recordRevision: '3aaaaaaaaaaa3',
          ),
    );
    final container = ProviderContainer.test(
      overrides: [
        postRepositoryProvider.overrideWithValue(repository),
        accountSavedPostRepositoryProvider(savedKey.account)
            .overrideWith((ref) async => _SavedEditReadRepository(seed.post)),
        searchRepositoryProvider.overrideWithValue(
          FakeSearchRepository(
            onSearchPosts: ({required q, limit, cursor}) async =>
                SearchPostPage(items: [seed.post]),
          ),
        ),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
      ],
    );
    final subscription = container.listen(editPostProvider, (_, _) {});
    addTearDown(subscription.close);
    final savedSubscription = container.listen(
      savedPostsProvider(savedKey),
      (_, _) {},
    );
    addTearDown(savedSubscription.close);
    await container.read(savedPostsProvider(savedKey).future);
    final searchProvider = postSearchProvider(
      const PostSearchQuery(q: 'shirt'),
    );
    final searchSubscription = container.listen(searchProvider, (_, _) {});
    addTearDown(searchSubscription.close);
    await container.read(searchProvider.future);
    final timelineSubscription = container.listen(timelineProvider, (_, _) {});
    addTearDown(timelineSubscription.close);
    await container.read(timelineProvider.future);
    await container.read(editPostProvider.future);
    await container
        .read(editPostProvider.notifier)
        .save(seed: seed, changes: PostEditChanges({'text': 'After'}));
    final saved = await container.read(savedPostsProvider(savedKey).future);
    expect(saved.items.single.post.text, 'After');
    expect(saved.items.single.savedAt, DateTime.utc(2025));
    expect(saved.items.single.folderId, 'original-folder');
    final search = await container.read(searchProvider.future);
    expect(search.items.single.text, 'After');
    final timeline = await container.read(timelineProvider.future);
    expect(timelineReads, greaterThan(1));
    expect(timeline.items.single.post.text, 'After');
    final visible = Provider(
      (ref) => applyPostRecordListOverlays(ref, [
        seed.post.copyWith(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/newer',
          rkey: 'newer',
          quote: PostRef(uri: seed.post.uri.value, cid: seed.post.cid.value),
          quoteView: QuoteView(
            state: 'visible',
            post: QuotePreviewPost(
              uri: seed.post.uri.value,
              cid: seed.post.cid.value,
              text: seed.post.text,
              author: seed.post.author,
              createdAt: seed.post.createdAt,
              sponsored: false,
            ),
          ),
        ),
        seed.post.copyWith(likeCount: 9),
      ], includes: (_) => true),
    );
    final detail = await container.read(
      postProvider(seed.post.author.did, seed.post.rkey).future,
    );
    expect(detail.text, 'After');
    expect(detail.likeCount, 9);
    final thread = await container.read(
      postCommentSectionProvider(
        seed.post.author.did,
        RecordKey.parse('root'),
      ).future,
    );
    expect(thread.comments.items.single.post.text, 'After');
    expect(thread.comments.items.single.post.likeCount, 11);
    final posts = container.read(visible);
    expect(posts.map((post) => post.rkey), ['newer', 'base']);
    expect(posts.first.quoteView?.post?.text, 'After');
    expect(posts.first.quote?.cid, seed.post.cid);
    expect(posts.last.text, 'After');
    expect(posts.last.likeCount, 9);
    expect(posts.last.editedAt, DateTime.utc(2026, 10, 26));
    final filtered = Provider(
      (ref) =>
          applyPostRecordListOverlays(ref, const [], includes: (_) => true),
    );
    expect(container.read(filtered), isEmpty);
  });
  test(
    'UT-007 later authoritative revision supersedes an accepted edit',
    () async {
      final seed = _seed();
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onEdit:
                  (
                    did,
                    key, {
                    required changes,
                    required expectedCid,
                    required operationKey,
                  }) async => PostEditResult(
                    acceptedCid: Cid.parse('bafyaccepted'),
                    changed: true,
                    projectionPending: false,
                    post: seed.post.copyWith(
                      cid: 'bafyaccepted',
                      text: 'After',
                    ),
                    editedAt: DateTime.utc(2026, 10, 26),
                    editTimeSource: PostEditTimeSource.confirmed,
                    recordRevision: '3aaaaaaaaaaa3',
                  ),
            ),
          ),
        ],
      );
      final subscription = container.listen(editPostProvider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(editPostProvider.future);
      await container
          .read(editPostProvider.notifier)
          .save(seed: seed, changes: PostEditChanges({'text': 'After'}));
      final newer = Provider(
        (ref) => applyPostRecordOverlay(
          ref,
          seed.post.copyWith(
            cid: 'bafylater',
            text: 'Later external version',
            recordRevision: '3aaaaaaaaaaa4',
          ),
        ),
      );
      expect(container.read(newer)?.text, 'Later external version');
    },
  );
  test('IT-015 edit loading discards a completion after the active account '
      'changes', () async {
    final seed = _seed();
    final read = Completer<PostEditSeed>();
    final initial = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.example',
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          InMemorySessionRegistryStorage(initial),
        ),
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(onFetchEdit: (did, key) => read.future),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final provider = postEditSeedProvider(
      initial.activeLease!,
      seed.post.author.did,
      seed.post.rkey,
    );
    final sub = container.listen(provider, (_, _) {});
    addTearDown(sub.close);
    final loading = container.read(provider.future);
    final rejected = expectLater(
      loading,
      throwsA(isA<AccountOperationDiscarded>()),
    );
    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.example',
        );
    read.complete(seed);
    await rejected;
  });
  test(
    'IT-015 accepted pending edit refreshes reads without resending the write',
    () async {
      final seed = _seed();
      var writes = 0;
      var reads = 0;
      final accepted = PostEditResult(
        acceptedCid: Cid.parse('bafyaccepted'),
        changed: true,
        projectionPending: true,
        editedAt: DateTime.utc(2026, 10, 26),
        editTimeSource: PostEditTimeSource.confirmed,
        recordRevision: '3aaaaaaaaaaa3',
      );
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onEdit:
                  (
                    did,
                    key, {
                    required changes,
                    required expectedCid,
                    required operationKey,
                  }) async {
                    writes++;
                    return accepted;
                  },
              onFetchEdit: (did, key) async {
                reads++;
                if (reads == 1) return seed;
                final current = seed.post.copyWith(
                  cid: 'bafyaccepted',
                  text: 'After',
                  recordRevision: '3aaaaaaaaaaa3',
                );
                return PostEditSeed(
                  post: current,
                  baseCid: current.cid,
                  recordRevision: '3aaaaaaaaaaa3',
                  editable: {'text': current.text, 'sponsored': false},
                );
              },
            ),
          ),
        ],
      );
      final sub = container.listen(editPostProvider, (_, _) {});
      addTearDown(sub.close);
      await container.read(editPostProvider.future);
      final notifier = container.read(editPostProvider.notifier);
      await notifier.save(
        seed: seed,
        changes: PostEditChanges({'text': 'After'}),
      );
      expect(
        await notifier.refreshAccepted(seed: seed, accepted: accepted),
        isNull,
      );
      expect(
        (await notifier.refreshAccepted(seed: seed, accepted: accepted))?.text,
        'After',
      );
      expect(writes, 1);
      expect(reads, 2);
    },
  );
}

PostEditSeed _seed() {
  final original = Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/base',
    cid: 'bafybase',
    rkey: 'base',
    text: 'Before',
    tags: const [],
    createdAt: DateTime.utc(2026),
    indexedAt: DateTime.utc(2026),
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
    likeCount: 7,
    repostCount: 2,
    replyCount: 1,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: true,
    sponsored: false,
  );
  return PostEditSeed(
    post: original,
    baseCid: original.cid,
    recordRevision: '3aaaaaaaaaaa2',
    editable: {'text': original.text, 'sponsored': false},
  );
}

final class _SavedEditReadRepository extends Fake
    implements SavedPostRepository {
  new(this.post);
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
        post: post,
        savedAt: DateTime.utc(2025),
        folderId: 'original-folder',
      ),
    ],
  );
}
