import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

const _did = 'did:plc:alice';
const _rkey = 'root';

void main() {
  setUpAll(initializeMappers);

  group('UT-004 account interaction lists', () {
    test(
      'exposes initial pending, success, and empty states by kind',
      () async {
        final pending = Completer<ProfileAccountPage>();
        final repository = FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) => pending.future,
          onListReposts: (did, rkey, {cursor, limit}) async =>
              const ProfileAccountPage(items: [], totalCount: 0),
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final likes = _accountsProvider(PostInteractionAccountKind.likes);
        final likesSubscription = container.listen(likes, (_, _) {});
        addTearDown(likesSubscription.close);

        expect(container.read(likes).isLoading, isTrue);
        expect(container.read(likes).value, isNull);

        pending.complete(
          ProfileAccountPage(
            items: [_account('did:plc:dana')],
            totalCount: 7,
            cursor: 'likes-next',
          ),
        );
        final loaded = await container.read(likes.future);
        final empty = await container.read(
          _accountsProvider(PostInteractionAccountKind.reposts).future,
        );

        expect(loaded.items.map((item) => item.did.toString()), [
          'did:plc:dana',
        ]);
        expect(loaded.totalCount, 7);
        expect(loaded.cursor, 'likes-next');
        expect(empty.items, isEmpty);
        expect(empty.totalCount, 0);
        expect(empty.cursor, isNull);
      },
    );

    test(
      'keeps transient and post_not_found initial failures distinct',
      () async {
        final errors = <Object>[
          const ApiNetworkError('offline'),
          const ApiBadRequest('post_not_found'),
        ];
        final repository = FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) =>
              Future<ProfileAccountPage>.error(errors.removeAt(0)),
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = _accountsProvider(PostInteractionAccountKind.likes);

        await expectLater(
          container.read(provider.future),
          throwsA(isA<ApiNetworkError>()),
        );
        container.invalidate(provider);
        await expectLater(
          container.read(provider.future),
          throwsA(
            isA<ApiBadRequest>().having(
              (error) => error.code,
              'code',
              'post_not_found',
            ),
          ),
        );
      },
    );

    test('appends once and deduplicates accounts by DID', () async {
      final duplicate = _account('did:plc:dana');
      final calls = <String?>[];
      final repository = FakePostRepository(
        onListLikes: (did, rkey, {cursor, limit}) async {
          calls.add(cursor);
          return cursor == null
              ? ProfileAccountPage(
                  items: [duplicate],
                  totalCount: 2,
                  cursor: 'next',
                )
              : ProfileAccountPage(
                  items: [
                    _account('did:plc:dana', displayName: 'duplicate'),
                    _account('did:plc:carol'),
                  ],
                  totalCount: 2,
                );
        },
      );
      final container = _container(repository);
      addTearDown(container.dispose);
      final provider = _accountsProvider(PostInteractionAccountKind.likes);
      await container.read(provider.future);

      await container.read(provider.notifier).loadMore();
      final state = container.read(provider).requireValue;

      expect(state.items.map((item) => item.did.toString()), [
        'did:plc:dana',
        'did:plc:carol',
      ]);
      expect(state.items.first, same(duplicate));
      expect(state.cursor, isNull);
      expect(state.totalCount, 2);
      expect(calls, [null, 'next']);
    });

    test(
      'retains data and retries a failed continuation with the same cursor',
      () async {
        var continuationCalls = 0;
        final repository = FakePostRepository(
          onListReposts: (did, rkey, {cursor, limit}) async {
            if (cursor == null) {
              return ProfileAccountPage(
                items: [_account('did:plc:dana')],
                totalCount: 2,
                cursor: 'retry-me',
              );
            }
            continuationCalls++;
            if (continuationCalls == 1) {
              throw const ApiNetworkError('offline');
            }
            return ProfileAccountPage(
              items: [_account('did:plc:carol')],
              totalCount: 2,
            );
          },
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = _accountsProvider(PostInteractionAccountKind.reposts);
        await container.read(provider.future);

        await container.read(provider.notifier).loadMore();
        final failed = container.read(provider);
        expect(failed.hasError, isTrue);
        expect(failed.value!.items.single.did.toString(), 'did:plc:dana');
        expect(failed.value!.cursor, 'retry-me');

        await container.read(provider.notifier).loadMore();
        final retried = container.read(provider).requireValue;
        expect(retried.items.map((item) => item.did.toString()), [
          'did:plc:dana',
          'did:plc:carol',
        ]);
        expect(continuationCalls, 2);
      },
    );

    test(
      'invalid_cursor replaces data only after a cursorless restart succeeds',
      () async {
        final restart = Completer<ProfileAccountPage>();
        var calls = 0;
        final repository = FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) {
            calls++;
            if (calls == 1) {
              return Future.value(
                ProfileAccountPage(
                  items: [_account('did:plc:old')],
                  totalCount: 1,
                  cursor: 'expired',
                ),
              );
            }
            if (cursor == 'expired') {
              return Future.error(const ApiBadRequest('invalid_cursor'));
            }
            return restart.future;
          },
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = _accountsProvider(PostInteractionAccountKind.likes);
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        await container.read(provider.future);

        final operation = container.read(provider.notifier).loadMore();
        await _flush();
        expect(
          container.read(provider).value!.items.single.did.toString(),
          'did:plc:old',
        );

        restart.complete(
          ProfileAccountPage(
            items: [_account('did:plc:new')],
            totalCount: 1,
          ),
        );
        await operation;

        expect(
          container.read(provider).requireValue.items.single.did.toString(),
          'did:plc:new',
        );
        expect(calls, 3);
      },
    );

    test('refresh and account fences reject stale account pages', () async {
      final continuation = Completer<ProfileAccountPage>();
      final switchContinuation = Completer<ProfileAccountPage>();
      final refresh = Completer<ProfileAccountPage>();
      var initialRequest = true;
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final repository = FakePostRepository(
        onListLikes: (did, rkey, {cursor, limit}) {
          if (initialRequest) {
            initialRequest = false;
            return Future.value(
              ProfileAccountPage(
                items: [_account('did:plc:old')],
                totalCount: 1,
                cursor: 'next',
              ),
            );
          }
          if (cursor == null) return refresh.future;
          return cursor == 'next'
              ? continuation.future
              : switchContinuation.future;
        },
      );
      final container = _container(repository, registry: registry);
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      final provider = _accountsProvider(PostInteractionAccountKind.likes);
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(provider.future);

      final staleContinuation = container.read(provider.notifier).loadMore();
      await _flush();
      final currentRefresh = container.read(provider.notifier).refresh();
      refresh.complete(
        ProfileAccountPage(
          items: [_account('did:plc:fresh')],
          totalCount: 1,
          cursor: 'fresh-next',
        ),
      );
      await currentRefresh;
      continuation.complete(
        ProfileAccountPage(
          items: [_account('did:plc:stale')],
          totalCount: 2,
        ),
      );
      await staleContinuation;
      expect(
        container.read(provider).requireValue.items.single.did.toString(),
        'did:plc:fresh',
      );

      final staleAfterSwitch = container.read(provider.notifier).loadMore();
      await _flush();
      final bob = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bob);
      switchContinuation.complete(
        ProfileAccountPage(
          items: [_account('did:plc:wrong-account')],
          totalCount: 2,
        ),
      );
      await staleAfterSwitch;
      expect(
        container.read(provider).value!.items.single.did.toString(),
        'did:plc:fresh',
      );
    });

    test(
      'IR-001 account activation clears rows and restarts without the old '
      'cursor',
      () async {
        final continuation = Completer<ProfileAccountPage>();
        final cursors = <String?>[];
        var firstPage = 0;
        final registry = SessionRegistry.empty()
            .upsertAndActivate(
              token: 'token-b',
              did: 'did:plc:bob',
              handle: 'bob.test',
            )
            .upsertAndActivate(
              token: 'token-a',
              did: 'did:plc:alice',
              handle: 'alice.test',
            );
        final repository = FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) {
            cursors.add(cursor);
            if (cursor != null) return continuation.future;
            firstPage++;
            return Future.value(
              ProfileAccountPage(
                items: [_account('did:plc:account-$firstPage')],
                totalCount: 1,
                cursor: firstPage == 1 ? 'account-a-next' : null,
              ),
            );
          },
        );
        final container = _container(repository, registry: registry);
        addTearDown(container.dispose);
        await container.read(sessionRegistryProvider.future);
        final provider = _accountsProvider(PostInteractionAccountKind.likes);
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        await container.read(provider.future);

        final staleLoadMore = container.read(provider.notifier).loadMore();
        await _flush();
        final bob = container
            .read(sessionRegistryProvider)
            .requireValue
            .leaseFor(AccountKey('did:plc:bob'))!;
        await container.read(sessionRegistryProvider.notifier).activate(bob);
        await _flush();
        continuation.complete(
          ProfileAccountPage(
            items: [_account('did:plc:stale-account-a')],
            totalCount: 2,
          ),
        );
        await staleLoadMore;
        final reloaded = await container.read(provider.future);

        expect(reloaded.items.single.did.toString(), 'did:plc:account-2');
        expect(cursors, [null, 'account-a-next', null]);
      },
    );
  });

  group('UT-004 quote interaction lists', () {
    test(
      'exposes pending, success, empty, transient, and post_not_found',
      () async {
        final firstPending = Completer<PostPage>();
        final results = <Object>[
          firstPending,
          const PostPage(items: []),
          const ApiNetworkError('offline'),
          const ApiBadRequest('post_not_found'),
        ];
        final repository = FakePostRepository(
          onListQuotes: (did, rkey, {cursor, limit}) {
            final result = results.removeAt(0);
            if (result is Completer<PostPage>) return result.future;
            if (result is PostPage) return Future.value(result);
            return Future.error(result);
          },
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = _quotesProvider();
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);

        expect(container.read(provider).isLoading, isTrue);
        firstPending.complete(
          PostPage(items: [_post('first')], cursor: 'next'),
        );
        expect(
          (await container.read(provider.future)).items.single.rkey.toString(),
          'first',
        );

        container.invalidate(provider);
        expect((await container.read(provider.future)).items, isEmpty);
        container.invalidate(provider);
        await expectLater(
          container.read(provider.future),
          throwsA(isA<ApiNetworkError>()),
        );
        container.invalidate(provider);
        await expectLater(
          container.read(provider.future),
          throwsA(
            isA<ApiBadRequest>().having(
              (error) => error.code,
              'code',
              'post_not_found',
            ),
          ),
        );
      },
    );

    test(
      'retries the same cursor, dedupes by URI, and supports replace/remove',
      () async {
        var continuationCalls = 0;
        final first = _post('first');
        final second = _post('second');
        final repository = FakePostRepository(
          onListQuotes: (did, rkey, {cursor, limit}) async {
            if (cursor == null) return PostPage(items: [first], cursor: 'next');
            continuationCalls++;
            if (continuationCalls == 1) throw StateError('offline');
            return PostPage(items: [_post('first'), second]);
          },
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = _quotesProvider();
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        await container.read(provider.future);

        await container.read(provider.notifier).loadMore();
        expect(container.read(provider).hasError, isTrue);
        expect(container.read(provider).value!.cursor, 'next');
        await container.read(provider.notifier).loadMore();
        expect(
          container.read(provider).requireValue.items.map((post) => post.rkey),
          ['first', 'second'],
        );

        final replacement = _post('second', text: 'updated');
        container.read(provider.notifier).replace(replacement);
        expect(
          container.read(provider).requireValue.items.last,
          same(replacement),
        );
        container.read(provider.notifier).remove(first.uri);
        expect(container.read(provider).requireValue.items, [
          same(replacement),
        ]);
        expect(continuationCalls, 2);
      },
    );

    test(
      'IR-003 late continuation preserves quote replacement and removal',
      () async {
        final continuation = Completer<PostPage>();
        final first = _post('first');
        final second = _post('second');
        final repository = FakePostRepository(
          onListQuotes: (did, rkey, {cursor, limit}) => cursor == null
              ? Future.value(
                  PostPage(items: [first, second], cursor: 'next'),
                )
              : continuation.future,
        );
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = _quotesProvider();
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        await container.read(provider.future);

        final loadMore = container.read(provider.notifier).loadMore();
        await _flush();
        final replacement = _post('second', text: 'updated');
        container.read(provider.notifier).replace(replacement);
        container.read(provider.notifier).remove(first.uri);
        continuation.complete(PostPage(items: [_post('third')]));
        await loadMore;

        final items = container.read(provider).requireValue.items;
        expect(items.map((post) => post.rkey), ['second', 'third']);
        expect(items.first, same(replacement));
      },
    );

    test('refresh generation rejects a late continuation completion', () async {
      final continuation = Completer<PostPage>();
      final refresh = Completer<PostPage>();
      var first = true;
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) {
          if (first) {
            first = false;
            return Future.value(
              PostPage(items: [_post('old')], cursor: 'old-next'),
            );
          }
          return cursor == null ? refresh.future : continuation.future;
        },
      );
      final container = _container(repository);
      addTearDown(container.dispose);
      final provider = _quotesProvider();
      await container.read(provider.future);

      final lateLoadMore = container.read(provider.notifier).loadMore();
      await _flush();
      final currentRefresh = container.read(provider.notifier).refresh();
      refresh.complete(PostPage(items: [_post('fresh')], cursor: 'fresh-next'));
      await currentRefresh;
      continuation.complete(PostPage(items: [_post('stale')]));
      await lateLoadMore;

      final state = container.read(provider).requireValue;
      expect(state.items.map((post) => post.rkey), ['fresh']);
      expect(state.cursor, 'fresh-next');
    });

    test('invalid_cursor restarts quote traversal without old items', () async {
      var calls = 0;
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) async {
          calls++;
          if (calls == 1) return PostPage(items: [_post('old')], cursor: 'bad');
          if (cursor != null) {
            throw const ApiBadRequest('invalid_cursor');
          }
          return PostPage(items: [_post('restarted')]);
        },
      );
      final container = _container(repository);
      addTearDown(container.dispose);
      final provider = _quotesProvider();
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(provider.future);

      await container.read(provider.notifier).loadMore();

      expect(
        container.read(provider).requireValue.items.single.rkey.toString(),
        'restarted',
      );
      expect(calls, 3);
    });

    test(
      'IR-001 quote account activation reloads without the old cursor',
      () async {
        final continuation = Completer<PostPage>();
        final cursors = <String?>[];
        var firstPage = 0;
        final initial = SessionRegistry.empty()
            .upsertAndActivate(
              token: 'token-b',
              did: 'did:plc:bob',
              handle: 'bob.test',
            )
            .upsertAndActivate(
              token: 'token-a',
              did: 'did:plc:alice',
              handle: 'alice.test',
            );
        final repository = FakePostRepository(
          onListQuotes: (did, rkey, {cursor, limit}) {
            cursors.add(cursor);
            if (cursor != null) return continuation.future;
            firstPage++;
            return Future.value(
              PostPage(
                items: [_post('account-$firstPage')],
                cursor: firstPage == 1 ? 'next' : null,
              ),
            );
          },
        );
        final container = _container(repository, registry: initial);
        addTearDown(container.dispose);
        await container.read(sessionRegistryProvider.future);
        final provider = _quotesProvider();
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        await container.read(provider.future);

        final lateLoadMore = container.read(provider.notifier).loadMore();
        await _flush();
        final bob = container
            .read(sessionRegistryProvider)
            .requireValue
            .leaseFor(AccountKey('did:plc:bob'))!;
        await container.read(sessionRegistryProvider.notifier).activate(bob);
        await _flush();
        continuation.complete(PostPage(items: [_post('stale-account-a')]));
        await lateLoadMore;
        final reloaded = await container.read(provider.future);

        expect(reloaded.items.single.rkey.toString(), 'account-2');
        expect(cursors, [null, 'next', null]);
      },
    );

    test('watches the active quote language policy', () async {
      var calls = 0;
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) async {
          calls++;
          return PostPage(items: [_post('language-$calls')]);
        },
      );
      final container = _container(repository);
      addTearDown(container.dispose);
      final provider = _quotesProvider();
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(provider.future);

      container
          .read(_testLanguagePreferencesProvider.notifier)
          .setContentLanguages(['fr']);
      await _flush();
      final reloaded = await container.read(provider.future);

      expect(calls, 2);
      expect(reloaded.items.single.rkey.toString(), 'language-2');
    });
  });
}

PostInteractionAccountsProvider _accountsProvider(
  PostInteractionAccountKind kind,
) => postInteractionAccountsProvider(
  Did.parse(_did),
  RecordKey.parse(_rkey),
  kind,
);

PostQuotesProvider _quotesProvider() =>
    postQuotesProvider(Did.parse(_did), RecordKey.parse(_rkey));

ProviderContainer _container(
  FakePostRepository repository, {
  SessionRegistry? registry,
}) {
  final value = registry ?? SessionRegistry.empty();
  return ProviderContainer.test(
    overrides: [
      postRepositoryProvider.overrideWithValue(repository),
      secureSessionRegistryStorageProvider.overrideWithValue(
        _RegistryStorage(value),
      ),
      activeLanguagePreferencesProvider.overrideWith(
        (ref) => ref.watch(_testLanguagePreferencesProvider),
      ),
    ],
    retry: (_, _) => null,
  );
}

final _testLanguagePreferencesProvider =
    NotifierProvider<_TestLanguagePreferences, LanguagePreferences>(
      _TestLanguagePreferences.new,
    );

final class _TestLanguagePreferences extends Notifier<LanguagePreferences> {
  @override
  LanguagePreferences build() => const LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en'],
  );

  void setContentLanguages(List<String> languages) {
    state = LanguagePreferences(
      primaryLanguage: state.primaryLanguage,
      contentLanguages: languages,
    );
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

ProfileAccountSummary _account(String did, {String? displayName}) =>
    ProfileAccountSummary(
      did: did,
      handle: '${did.split(':').last}.test',
      displayName: displayName,
      isCraftskyProfile: true,
    );

Post _post(String rkey, {String? text}) => PostMapper.fromMap({
  'uri': 'at://did:plc:author/social.craftsky.feed.post/$rkey',
  'cid': 'bafy-$rkey',
  'rkey': rkey,
  'text': text ?? rkey,
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-09-06T12:00:00.000Z',
  'indexedAt': '2026-09-06T12:00:01.000Z',
  'author': {'did': 'did:plc:author', 'handle': 'author.test'},
});

Future<void> _flush() => Future<void>.delayed(Duration.zero);
