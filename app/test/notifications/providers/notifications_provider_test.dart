import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as auth_model;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/providers/notification_filter_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-006 notification lists remain account-scoped', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final repositories = <AccountKey, NotificationRepository>{
      alice: _FakeNotificationRepository()
        ..responses.add(
          Future.value(NotificationPage(items: [_follow('alice')])),
        ),
      bob: _FakeNotificationRepository()
        ..responses.add(
          Future.value(NotificationPage(items: [_follow('bob')])),
        ),
    };
    final registry = auth_model.SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: alice.did.value,
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'bob-token',
          did: bob.did.value,
          handle: 'bob.test',
        );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        accountNotificationRepositoryProvider.overrideWith(
          (ref, account) async => repositories[account]!,
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);

    final aliceState = await container.read(
      accountNotificationsProvider(alice).future,
    );
    final bobState = await container.read(
      accountNotificationsProvider(bob).future,
    );

    expect(_socialRkey(aliceState.items.single), 'alice');
    expect(_socialRkey(bobState.items.single), 'bob');
    expect(aliceState.owner?.account, alice);
    expect(bobState.owner?.account, bob);
  });

  test(
    'account refresh retains loaded depth and category changes reset it',
    () async {
      final alice = AccountKey('did:plc:alice');
      final registry = auth_model.SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: alice.did.value,
        handle: 'alice.test',
      );
      final repo = _FakeNotificationRepository()
        ..responses.addAll([
          Future.value(
            NotificationPage(items: [_follow('one')], cursor: 'next'),
          ),
          Future.value(NotificationPage(items: [_follow('two')])),
          Future.value(
            NotificationPage(
              items: [_follow('one-refreshed')],
              cursor: 'new-next',
            ),
          ),
          Future.value(NotificationPage(items: [_follow('two-refreshed')])),
          Future.value(
            NotificationPage(
              items: [_follow('filtered')],
              cursor: 'filtered-next',
            ),
          ),
        ]);
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          accountNotificationRepositoryProvider.overrideWith(
            (ref, account) async => repo,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final provider = accountNotificationsProvider(alice);
      container.listen(provider, (_, _) {});
      await container.read(provider.future);
      await container.read(provider.notifier).loadMore();
      container.invalidate(provider);
      final refreshed = await container.read(provider.future);
      expect(refreshed.items.map(_socialRkey), [
        'one-refreshed',
        'two-refreshed',
      ]);
      expect(refreshed.owner?.account, alice);
      container
          .read(notificationFilterProvider.notifier)
          .select(NotificationCategory.follow);
      final filtered = await container.read(provider.future);
      expect(filtered.items.map(_socialRkey), ['filtered']);
      expect(repo.calls.map((call) => call.cursor), [
        null,
        'next',
        null,
        'new-next',
        null,
      ]);
      expect(filtered.cursor, 'filtered-next');
    },
  );

  for (final accountScoped in [false, true]) {
    test(
      'disposed notification load cancels before reading overlays '
      '(account=$accountScoped)',
      () async {
        final account = AccountKey('did:plc:alice');
        final registry = auth_model.SessionRegistry.empty().upsertAndActivate(
          token: 'alice-token',
          did: account.did.value,
          handle: 'alice.test',
        );
        final page = Completer<NotificationPage>();
        final repository = _FakeNotificationRepository()
          ..responses.add(page.future);
        final container = ProviderContainer(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            notificationRepositoryProvider.overrideWithValue(repository),
            accountNotificationRepositoryProvider.overrideWith(
              (ref, account) async => repository,
            ),
          ],
        );
        await container.read(sessionRegistryProvider.future);
        final provider = accountScoped
            ? accountNotificationsProvider(account)
            : notificationsProvider;
        final pending = container.read(provider.future);
        await repository.started.future;
        expect(repository.calls, hasLength(1));
        final cancelled = expectLater(
          pending,
          throwsA(isA<AccountOperationCancellation>()),
        );
        container.dispose();
        page.complete(NotificationPage(items: [_follow('late')]));
        await cancelled;
      },
    );
  }

  for (final accountScoped in [false, true]) {
    test(
      'rebuilt notification load cannot reconcile old overlays '
      '(account=$accountScoped)',
      () async {
        final account = AccountKey('did:plc:alice');
        final registry = auth_model.SessionRegistry.empty().upsertAndActivate(
          token: 'alice-token',
          did: account.did.value,
          handle: 'alice.test',
        );
        final oldPage = Completer<NotificationPage>();
        final repository = _FakeNotificationRepository()
          ..responses.addAll([
            oldPage.future,
            Future.value(
              NotificationPage(
                items: [_follow('new', actorDid: 'did:plc:bob')],
              ),
            ),
          ]);
        final mutations = PdsRecordOperationController(schedule: (_, _) {});
        final container = ProviderContainer.test(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            notificationRepositoryProvider.overrideWithValue(repository),
            accountNotificationRepositoryProvider.overrideWith(
              (ref, account) async => repository,
            ),
            pdsRecordOperationControllerProvider.overrideWithValue(mutations),
          ],
        );
        addTearDown(mutations.reset);
        await container.read(sessionRegistryProvider.future);
        final scope = blockProfileMutationScope(
          container.read,
          Did.parse('did:plc:alice'),
        );
        final token = mutations.begin(
          scope: scope,
          operationKey: 'fixture',
          endpoint: '/fixture',
          immutableBody: '{}',
        );
        mutations.markAccepted(
          token,
          optimisticValue: false,
          agrees: (value) => value == false,
        );
        final overlay = mutations.overlayFor(scope);
        final provider = accountScoped
            ? accountNotificationsProvider(account)
            : notificationsProvider;
        container.listen(provider, (_, _) {});
        final first = container.read(provider.future);
        await repository.started.future;
        container
            .read(notificationFilterProvider.notifier)
            .select(NotificationCategory.follow);
        final current = await container.read(provider.future);
        expect(current.items.map(_socialRkey), ['new']);
        await first;
        oldPage.complete(NotificationPage(items: [_follow('stale')]));
        await Future<void>.delayed(Duration.zero);
        expect(
          mutations.overlayFor(scope),
          same(overlay),
          reason: 'discarded builds must not reconcile current account state',
        );
      },
    );
  }

  test(
    'disposed notification work preserves the request cause and stack',
    () async {
      final account = AccountKey('did:plc:alice');
      final registry = auth_model.SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: account.did.value,
        handle: 'alice.test',
      );
      final page = Completer<NotificationPage>();
      final repository = _FakeNotificationRepository()
        ..responses.add(page.future);
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          accountNotificationRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final original = Exception('synthetic request failure');
      final originalStack = StackTrace.fromString(
        'original notification request frame',
      );
      final pending = container.read(
        accountNotificationsProvider(account).future,
      );
      await repository.started.future;
      final inspected = pending.then<void>(
        (_) => fail('expected request failure'),
        onError: (Object error, StackTrace stack) {
          expect(error, same(original));
          expect(stack.toString(), originalStack.toString());
        },
      );
      container.dispose();
      page.completeError(original, originalStack);
      await inspected;
    },
  );

  for (final accountScoped in [false, true]) {
    test('rebuilt notification pagination cannot reconcile old overlays '
        '(account=$accountScoped)', () async {
      final account = AccountKey('did:plc:alice');
      final registry = auth_model.SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: account.did.value,
        handle: 'alice.test',
      );
      final oldPage = Completer<NotificationPage>();
      final repository = _FakeNotificationRepository()
        ..responses.addAll([
          Future.value(
            NotificationPage(
              items: [_follow('first', actorDid: 'did:plc:bob')],
              cursor: 'next',
            ),
          ),
          oldPage.future,
          Future.value(
            NotificationPage(items: [_follow('new', actorDid: 'did:plc:bob')]),
          ),
        ]);
      final mutations = PdsRecordOperationController(schedule: (_, _) {});
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          notificationRepositoryProvider.overrideWithValue(repository),
          accountNotificationRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
          pdsRecordOperationControllerProvider.overrideWithValue(mutations),
        ],
      );
      addTearDown(mutations.reset);
      await container.read(sessionRegistryProvider.future);
      final scope = blockProfileMutationScope(
        container.read,
        Did.parse('did:plc:alice'),
      );
      final token = mutations.begin(
        scope: scope,
        operationKey: 'fixture',
        endpoint: '/fixture',
        immutableBody: '{}',
      );
      mutations.markAccepted(
        token,
        optimisticValue: false,
        agrees: (value) => value == false,
      );
      final overlay = mutations.overlayFor(scope);
      final provider = accountScoped
          ? accountNotificationsProvider(account)
          : notificationsProvider;
      container.listen(provider, (_, _) {});
      await container.read(provider.future);
      final operation = accountScoped
          ? container
                .read(accountNotificationsProvider(account).notifier)
                .loadMore()
          : container.read(notificationsProvider.notifier).loadMore();
      await Future<void>.delayed(Duration.zero);
      expect(repository.calls, hasLength(2));
      container
          .read(notificationFilterProvider.notifier)
          .select(NotificationCategory.follow);
      final current = await container.read(provider.future);
      expect(current.items.map(_socialRkey), ['new']);
      oldPage.complete(NotificationPage(items: [_follow('stale')]));
      await operation;
      expect(
        mutations.overlayFor(scope),
        same(overlay),
        reason: 'discarded pagination must not reconcile current account state',
      );
    });
  }

  test(
    'category changes reset pagination and discard pending old pages',
    () async {
      final pending = Completer<NotificationPage>();
      final repo = _FakeNotificationRepository()
        ..responses.addAll([
          Future.value(
            NotificationPage(items: [_follow('all')], cursor: 'all-next'),
          ),
          pending.future,
          Future.value(
            NotificationPage(
              items: [_follow('filtered')],
              cursor: 'filtered-next',
            ),
          ),
          Future.value(NotificationPage(items: [_follow('filtered-more')])),
        ]);
      final container = ProviderContainer.test(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
      )..listen(notificationsProvider, (_, _) {});
      await container.read(notificationsProvider.future);
      final oldLoad = container.read(notificationsProvider.notifier).loadMore();
      container
          .read(notificationFilterProvider.notifier)
          .select(NotificationCategory.follow);
      await container.read(notificationsProvider.future);
      pending.complete(NotificationPage(items: [_follow('stale')]));
      await oldLoad;
      expect(
        container
            .read(notificationsProvider)
            .requireValue
            .items
            .map(_socialRkey),
        ['filtered'],
      );
      await container.read(notificationsProvider.notifier).loadMore();
      expect(repo.calls.map((call) => call.category), [
        null,
        null,
        NotificationCategory.follow,
        NotificationCategory.follow,
      ]);
      expect(repo.calls.map((call) => call.cursor), [
        null,
        'all-next',
        null,
        'filtered-next',
      ]);
      expect(
        container
            .read(notificationsProvider)
            .requireValue
            .items
            .map(_socialRkey),
        ['filtered', 'filtered-more'],
      );
    },
  );

  test('category selection resets when the active account changes', () async {
    final registry = auth_model.SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    expect(container.read(notificationFilterProvider), isNull);
    container
        .read(notificationFilterProvider.notifier)
        .select(NotificationCategory.mention);
    expect(
      container.read(notificationFilterProvider),
      NotificationCategory.mention,
    );
    await container
        .read(sessionRegistryProvider.notifier)
        .activate(registry.leaseFor(AccountKey('did:plc:alice'))!);
    expect(container.read(notificationFilterProvider), isNull);
  });

  test('initial load retries after failure', () async {
    final repo = _FakeNotificationRepository()
      ..responses.add(Future<NotificationPage>.error(Exception('nope')))
      ..responses.add(Future.value(NotificationPage(items: [_follow('one')])));
    final container = ProviderContainer.test(
      overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
    );
    final sub = container.listen(
      notificationsProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(sub.close);

    await Future<void>.delayed(Duration.zero);
    expect(container.read(notificationsProvider).hasError, isTrue);
    container.invalidate(notificationsProvider);
    final state = await container.read(notificationsProvider.future);

    expect(state.items.map(_socialRkey), ['one']);
    expect(repo.calls, hasLength(2));
  });

  test('appends next page and marks terminal cursor', () async {
    final repo = _FakeNotificationRepository()
      ..responses.add(
        Future.value(NotificationPage(items: [_follow('one')], cursor: 'next')),
      )
      ..responses.add(Future.value(NotificationPage(items: [_follow('two')])));
    final container = ProviderContainer.test(
      overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
    );

    await container.read(notificationsProvider.future);
    await container.read(notificationsProvider.notifier).loadMore();

    final state = container.read(notificationsProvider).value!;
    expect(state.items.map(_socialRkey), ['one', 'two']);
    expect(state.hasMore, isFalse);
    expect(repo.calls.last.cursor, 'next');
  });

  test(
    'preserves rows and cursor on load-more failure and guards concurrency',
    () async {
      final gate = Completer<NotificationPage>();
      final repo = _FakeNotificationRepository()
        ..responses.add(
          Future.value(
            NotificationPage(items: [_follow('one')], cursor: 'next'),
          ),
        )
        ..responses.add(gate.future);
      final container = ProviderContainer.test(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
      );

      await container.read(notificationsProvider.future);
      final firstLoadMore = container
          .read(notificationsProvider.notifier)
          .loadMore();
      await container.read(notificationsProvider.notifier).loadMore();
      gate.completeError(Exception('boom'));
      await firstLoadMore;

      final state = container.read(notificationsProvider);
      expect(repo.calls, hasLength(2));
      expect(state.value!.items.map(_socialRkey), ['one']);
      expect(state.value!.cursor, 'next');
      expect(state.hasError, isTrue);
    },
  );

  test(
    'AT-006 suppressActor removes loaded rows and returns the count',
    () async {
      final repo = _FakeNotificationRepository()
        ..responses.add(
          Future.value(
            NotificationPage(
              items: [
                _follow('alice-one'),
                _follow('bob-one', actorDid: 'did:plc:bob'),
                _follow('alice-two'),
                _instagramMatch(),
              ],
            ),
          ),
        );
      final container = ProviderContainer.test(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
      );

      await container.read(notificationsProvider.future);
      final removed = container
          .read(notificationsProvider.notifier)
          .suppressActor('did:plc:alice');
      final retained = container.read(notificationsProvider).requireValue.items;
      final retainedSocial = retained.whereType<SocialNotification>().single;

      expect(removed, 2);
      expect(retained, hasLength(2));
      expect(retainedSocial.actor.did, 'did:plc:bob');
      expect(retained.whereType<InstagramMatchNotification>(), hasLength(1));
    },
  );
}

String _socialRkey(CraftskyNotification notification) =>
    (notification as SocialNotification).rkey.toString();

InstagramMatchNotification _instagramMatch() => CraftskyNotification.fromMap({
  'id': 'instagram-match',
  'type': 'instagramMatch',
  'actor': {
    'available': true,
    'did': 'did:plc:instagram-match',
    'handle': 'instagram-match.test',
    'viewerIsFollowing': true,
  },
  'createdAt': '2026-05-28T13:00:00Z',
  'indexedAt': '2026-05-28T13:00:01Z',
}) as InstagramMatchNotification;

FollowNotification _follow(
  String rkey, {
  String actorDid = 'did:plc:alice',
}) => CraftskyNotification.fromMap({
  'id': 'notification-$rkey',
  'uri': 'at://did:plc:alice/app.bsky.graph.follow/$rkey',
  'cid': 'bafy$rkey',
  'rkey': rkey,
  'type': 'follow',
  'actor': {'did': actorDid, 'handle': 'actor.craftsky.social'},
  'createdAt': '2026-05-28T13:00:00Z',
  'indexedAt': '2026-05-28T13:00:01Z',
}) as FollowNotification;

class _FakeNotificationRepository implements NotificationRepository {
  final responses = <Future<NotificationPage>>[];
  final calls = <_Call>[];
  final started = Completer<void>();

  @override
  Future<NotificationPage> list({
    String? cursor,
    int? limit,
    NotificationCategory? category,
  }) {
    calls.add(_Call(cursor: cursor, limit: limit, category: category));
    if (!started.isCompleted) started.complete();
    return responses.removeAt(0);
  }
}

final class _Call {
  const new({this.cursor, this.limit, this.category});
  final NotificationCategory? category;
  final String? cursor;
  final int? limit;
}

final class _RegistryStorage implements SessionRegistryStorage {
  new(this.registry);

  auth_model.SessionRegistry registry;

  @override
  Future<auth_model.SessionRegistry> read() async => registry;

  @override
  Future<void> write(auth_model.SessionRegistry registry) async {
    this.registry = registry;
  }
}
