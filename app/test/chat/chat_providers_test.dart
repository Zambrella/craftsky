import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_conversation_details.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_draft_store_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/providers/chat_session_runtime_provider.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

// IT-015 / IT-019 / NFR-001: account dependency and stale acquisition.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  for (final removeAccount in [false, true]) {
    test('cold-start DM open survives a delayed conversation lookup '
        'removed=$removeAccount', () async {
      final account = AccountKey('did:plc:alice');
      final base = SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic',
        did: account.did.value,
        handle: 'alice.test',
      );
      final lease = base.activeLease!.session;
      final repository = _Repository();
      final response = Completer<ChatConversationId>();
      repository.notificationTarget = response.future;
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(base.saveRoutingBinding(lease, 'binding')),
          ),
          accountChatRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
          onboardingStatusProvider(lease).overrideWithBuild(
            (ref, _) async => const OnboardingCompletion(completed: true),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final effects = <NotificationEffect>[];
      final subscription = container
          .read(notificationEffectStreamProvider)
          .listen(effects.add);
      addTearDown(subscription.cancel);
      final runtime = container.read(notificationRuntimeProvider);
      await runtime.start();
      await runtime.receiveOpen(
        NotificationOpenAttempt.fromProviderData({
          'payloadVersion': '1',
          'type': 'chat',
          'accountSubscriptionId': 'binding',
          'openReference': '00000000-0000-4000-8000-000000000123',
        }, source: NotificationOpenSource.initialOpen),
      );
      final opening = runtime.updateReadiness(
        did: account.did,
        onboarded: true,
      );
      await repository.notificationEntered.future;
      // Let a frame elapse while the network response is pending.
      await container.pump();
      expect(effects, isEmpty);
      if (removeAccount) {
        await container
            .read(sessionRegistryProvider.notifier)
            .removeConfirmed(lease);
      }
      response.complete(ChatConversationId.parse('conversation'));
      await opening;
      await container.pump();
      expect(effects.whereType<NotificationUnavailableEffect>(), isEmpty);
      final navigation = effects.whereType<NotificationNavigationEffect>();
      if (removeAccount) {
        expect(navigation, isEmpty);
      } else {
        expect(
          navigation.single.outcome.destination,
          isA<ChatConversationDestination>(),
        );
      }
      expect(
        container.exists(
          chatNotificationOpenProvider(
            account,
            ChatOpenReference.parse('00000000-0000-4000-8000-000000000123'),
          ),
        ),
        isFalse,
      );
    });
  }
  test('foreground push refreshes counts, lists and visible messages '
      'without replacing the draft', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:alice');
    final repository = _Repository()..page = _privatePage();
    final container = _sessionContainer(repository)
      ..listen(chatSessionRuntimeProvider, (_, _) {});
    await container.read(sessionRegistryProvider.future);
    final sync = await container.read(
      chatSyncCoordinatorProvider(account).future,
    );
    final family = chatConversationControllerProvider(
      account,
      ChatConversationId.parse('conversation'),
    );
    container.listen(family, (_, _) {});
    final controller = await container.read(family.future);
    controller.edit('Keep this draft');
    sync
      ..setView(ChatView.conversation, conversationId: controller.id)
      ..setForeground(active: true);
    await container.pump();
    final counts = repository.countCalls;
    final lists = repository.listCalls;
    final history = repository.historyCalls;
    container.read(Provider<void>((ref) => refreshChatAfterPush(ref, account)));
    await container.pump();
    expect(repository.countCalls, greaterThan(counts));
    expect(repository.listCalls, greaterThan(lists));
    expect(repository.historyCalls, greaterThan(history));
    expect(container.read(family).value, same(controller));
    expect(controller.draft.text, 'Keep this draft');
    sync.setForeground(active: false);
  });
  for (final bucket in ChatBucket.values) {
    test(
      'optimistic unread $bucket reconciles once and rolls back on failure',
      () async {
        final account = AccountKey('did:plc:alice');
        final repository = _Repository()
          ..unreadAccepted = 2
          ..unreadRequests = 2;
        final container = _sessionContainer(repository);
        await container.read(sessionRegistryProvider.future);
        await container.read(chatStatusProvider(account).future);
        await container.read(chatUnreadCountsProvider(account).future);
        final display = displayedChatUnreadCountsProvider(account);
        container.listen(display, (_, _) {});
        int count() {
          final counts = container.read(display)!;
          return bucket == ChatBucket.accepted
              ? counts.accepted
              : counts.requests;
        }

        final conversation = ChatConversation.fromMap({
          'id': 'conversation',
          'revision': 'r',
          'status': bucket == ChatBucket.accepted ? 'accepted' : 'request',
          'muted': false,
          'unreadCount': 1,
          'canSend': true,
          'peer': const {'did': 'did:plc:peer'},
        });
        expect(count(), 2);
        final pending = container.read(
          chatPendingReadsProvider(account).notifier,
        );
        final finish = pending.begin(conversation);
        expect(count(), 1);
        final duplicate = pending.begin(conversation);
        duplicate();
        expect(count(), 1);
        // A count refresh before the read lands must not undo the optimism.
        container.invalidate(chatUnreadCountsProvider(account));
        await container.read(chatUnreadCountsProvider(account).future);
        expect(count(), 1);
        repository
          ..unreadAccepted = 1
          ..unreadRequests = 1;
        container.invalidate(chatUnreadCountsProvider(account));
        await container.read(chatUnreadCountsProvider(account).future);
        expect(count(), 1); // No second subtraction after reconciliation.
        finish();
        expect(count(), 1);
        final rollback = pending.begin(conversation);
        expect(count(), 0);
        rollback();
        expect(count(), 1);
      },
    );
  }
  for (final signedIn in [false, true]) {
    test(
      'UT-006 disabled or signed-out runtime avoids private reads '
      'signedIn=$signedIn',
      () async {
        final account = AccountKey('did:plc:alice');
        final repository = _Repository()..enabled = false;
        final registry = signedIn
            ? SessionRegistry.empty().upsertAndActivate(
                token: 'synthetic',
                did: account.did.value,
                handle: 'alice.test',
              )
            : SessionRegistry.empty();
        final container = ProviderContainer.test(
          retry: (_, _) => null,
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            accountChatRepositoryProvider.overrideWith(
              (ref, account) async => repository,
            ),
          ],
        )..listen(chatSessionRuntimeProvider, (_, _) {});
        await container.read(sessionRegistryProvider.future);
        await container.pump();
        if (signedIn) await container.read(chatStatusProvider(account).future);
        await container.pump();
        expect(repository.statusCalls, signedIn ? 1 : 0);
        expect(repository.countCalls, 0);
        expect(repository.heartbeatCalls, 0);
        expect(repository.historyCalls, 0);
        expect(repository.listCalls, 0);
        expect(container.exists(chatSyncCoordinatorProvider(account)), false);
      },
    );
  }
  testWidgets(
    'UT-010 idle cache evicts oldest history and preserves active history',
    (tester) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:alice');
      final repository = _Repository()..page = _privatePage();
      final container = _sessionContainer(repository)
        ..listen(chatSessionRuntimeProvider, (_, _) {});
      await container.read(sessionRegistryProvider.future);
      await container.read(chatStatusProvider(account).future);
      await tester.pump();
      final activeFamily = chatConversationControllerProvider(
        account,
        ChatConversationId.parse('active'),
      );
      container.listen(activeFamily, (_, _) {});
      final active = await container.read(activeFamily.future);
      final loaded = [active];
      for (var index = 0; index < 4; index++) {
        final family = chatConversationControllerProvider(
          account,
          ChatConversationId.parse('idle-$index'),
        );
        final sub = container.listen(family, (_, _) {});
        loaded.add(await container.read(family.future));
        sub.close();
        await tester.pump();
      }
      await tester.pump(const Duration(milliseconds: 1));
      expect(loaded.map((item) => item.history.messages.length).toList(), [
        1,
        0,
        1,
        1,
        1,
      ]);
      for (final controller in [active, ...loaded.skip(2)]) {
        expect(controller.history.messages.single.text, 'private-cache-canary');
      }
      expect(repository.historyCalls, 5);
      expect(repository.heartbeatCalls, 0);
      container.dispose();
      await tester.pump();
    },
  );
  for (final boundary in [
    'owner switch',
    'reactivation',
    'logout',
    'disable',
    'permission loss',
    'generation',
    'authorization failure',
  ]) {
    testWidgets('IT-017 idle private cache clears on $boundary', (
      tester,
    ) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:alice');
      final repository = _Repository()..page = _privatePage();
      final container = _sessionContainer(repository)
        ..listen(chatSessionRuntimeProvider, (_, _) {});
      await container.read(sessionRegistryProvider.future);
      await container.read(chatStatusProvider(account).future);
      await tester.pump();
      final inboxFamily = chatInboxControllerProvider(
        account,
        ChatBucket.accepted,
      );
      final inboxSub = container.listen(inboxFamily, (_, _) {});
      final inbox = await container.read(inboxFamily.future);
      final conversationFamily = chatConversationControllerProvider(
        account,
        ChatConversationId.parse('cached'),
      );
      final conversationSub = container.listen(conversationFamily, (_, _) {});
      final conversation = await container.read(conversationFamily.future);
      inboxSub.close();
      conversationSub.close();
      await tester.pump();
      expect(inbox.items, isNotEmpty);
      expect(conversation.history.messages, isNotEmpty);
      switch (boundary) {
        case 'owner switch':
          await container
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'other',
                did: 'did:plc:other',
                handle: 'other.test',
              );
        case 'reactivation':
          await container
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'replacement',
                did: account.did.value,
                handle: 'alice.test',
              );
        case 'logout':
          final lease = container
              .read(sessionRegistryProvider)
              .requireValue
              .activeLease!;
          await container
              .read(sessionRegistryProvider.notifier)
              .removeConfirmed(lease.session);
        case 'authorization failure':
          repository.statusFailure = const ApiUnauthorized();
          container.invalidate(chatStatusProvider(account));
          await expectLater(
            container.read(chatStatusProvider(account).future),
            throwsA(isA<ApiUnauthorized>()),
          );
        case _:
          container
              .read(chatStatusProvider(account).notifier)
              .publishConfirmedStatus(
                ChatStatus.fromMap({
                  'enabled': boundary != 'disable',
                  'pushEnabled': true,
                  'generation': boundary == 'generation' ? 3 : 2,
                  'state': boundary == 'permission loss'
                      ? 'authorizationNeeded'
                      : boundary == 'disable'
                      ? 'disabled'
                      : 'ready',
                  'capabilities': {
                    'canRead': boundary != 'permission loss',
                    'canSend': true,
                    'canReport': true,
                    'canSetPolicy': true,
                  },
                }),
              );
      }
      await tester.pump();
      await tester.pump();
      expect(inbox.items, isEmpty);
      expect(conversation.history.messages, isEmpty);
      expect(repository.listCalls, 1);
      expect(repository.historyCalls, 1);
      expect(repository.heartbeatCalls, 0);
      container.dispose();
      await tester.pump();
    });
  }
  testWidgets(
    'UT-010 idle conversation expires but active reuse stays loaded',
    (tester) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:alice');
      final repository = _Repository()..page = _privatePage();
      final container = _sessionContainer(repository)
        ..listen(chatSessionRuntimeProvider, (_, _) {});
      await container.read(sessionRegistryProvider.future);
      await container.read(chatStatusProvider(account).future);
      await tester.pump();
      final family = chatConversationControllerProvider(
        account,
        ChatConversationId.parse('cached'),
      );
      final first = container.listen(family, (_, _) {});
      final loaded = await container.read(family.future);
      loaded.edit('still-unsent');
      await loaded.flush();
      first.close();
      await tester.pump(const Duration(minutes: 1));
      expect(loaded.history.messages.single.text, 'private-cache-canary');
      final active = container.listen(family, (_, _) {});
      expect(await container.read(family.future), same(loaded));
      await tester.pump(const Duration(minutes: 3));
      expect(loaded.history.messages.single.text, 'private-cache-canary');
      active.close();
      await tester.pump(const Duration(minutes: 2, milliseconds: 1));
      await tester.pump();
      expect(loaded.history.messages, isEmpty);
      final store = container.read(chatDraftStoreProvider);
      expect(
        (await store.load(
          store.lease(account),
          ChatConversationId.parse('cached'),
        )).text,
        'still-unsent',
      );
      final reopened = container.listen(family, (_, _) {});
      expect(await container.read(family.future), isNot(same(loaded)));
      expect(repository.historyCalls, 2);
      reopened.close();
      container.dispose();
      await tester.pump();
    },
  );
  for (final bucket in ChatBucket.values) {
    test('UT-010 session retains the loaded $bucket list', () async {
      final account = AccountKey('did:plc:alice');
      final repository = _Repository();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'synthetic',
                did: account.did.value,
                handle: 'alice.test',
              ),
            ),
          ),
          accountChatRepositoryProvider(account)
              .overrideWith((ref) async => repository),
        ],
      )..listen(chatSessionRuntimeProvider, (_, _) {});
      await container.read(sessionRegistryProvider.future);
      await container.read(chatStatusProvider(account).future);
      await container.pump();
      final family = chatInboxControllerProvider(account, bucket);
      final first = container.listen(family, (_, _) {});
      final loaded = await container.read(family.future);
      expect(
        repository.listBuckets.where((value) => value == bucket),
        hasLength(1),
      );
      first.close();
      await container.pump();
      final reopened = container.listen(family, (_, _) {});
      expect(await container.read(family.future), same(loaded));
      expect(
        repository.listBuckets.where((value) => value == bucket),
        hasLength(1),
      );
      reopened.close();
    });
  }
  test('UT-010 search and report forms stay page scoped', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _Repository();
    final dio = Dio();
    addTearDown(dio.close);
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'synthetic',
              did: account.did.value,
              handle: 'alice.test',
            ),
          ),
        ),
        accountChatRepositoryProvider.overrideWith(
          (ref, account) async => repository,
        ),
        messagingChatDioProvider(account).overrideWith((ref) async => dio),
      ],
    )..listen(chatSessionRuntimeProvider, (_, _) {});
    await container.read(sessionRegistryProvider.future);
    await container.read(chatStatusProvider(account).future);
    await container.read(chatSyncCoordinatorProvider(account).future);
    await container.read(
      chatInboxControllerProvider(account, ChatBucket.accepted).future,
    );
    final search = chatRecipientControllerProvider(account);
    final report = chatReportControllerProvider(
      account,
      ChatConversationId.parse('conversation'),
      ChatMessageId.parse('message'),
    );
    final searchSub = container.listen(search, (_, _) {});
    final reportSub = container.listen(report, (_, _) {});
    await container.read(search.future);
    await container.read(report.future);
    searchSub.close();
    reportSub.close();
    await container.pump();
    expect(container.exists(search), false);
    expect(container.exists(report), false);
    expect(container.exists(chatSyncCoordinatorProvider(account)), true);
    expect(repository.heartbeatCalls, 0);
    expect(repository.historyCalls, 0);
    expect(repository.listCalls, 1);
  });
  test(
    'UT-006 runtime warms inbox state; foreground scope starts polling',
    () async {
      final account = AccountKey('did:plc:alice');
      final repository = _Repository();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'synthetic',
                did: account.did.value,
                handle: 'alice.test',
              ),
            ),
          ),
          accountChatRepositoryProvider(account)
              .overrideWith((ref) async => repository),
        ],
      );
      final owner = container.listen(chatSessionRuntimeProvider, (_, _) {});
      await container.read(sessionRegistryProvider.future);
      await container.pump();
      expect(container.exists(chatStatusProvider(account)), true);
      await container.read(chatStatusProvider(account).future);
      await container.pump();
      expect(container.exists(chatUnreadCountsProvider(account)), true);
      await container.read(chatUnreadCountsProvider(account).future);
      await container.read(chatSyncCoordinatorProvider(account).future);
      expect(repository.statusCalls, 1);
      expect(repository.countCalls, 1);
      expect(container.exists(chatSyncCoordinatorProvider(account)), true);
      expect(repository.heartbeatCalls, 0);
      owner.close();
      await container.pump();
      expect(container.exists(chatSyncCoordinatorProvider(account)), false);
    },
  );
  test('UT-006 offline status retains only the exact '
      'previously authorized activation', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final a = AccountKey('did:plc:alice');
    final id = ChatConversationId.parse('conversation');
    final repository = _Repository()
      ..page = ChatMessagePage.fromMap(const {
        'items': [
          {
            'id': 'message',
            'revision': 'r',
            'kind': 'text',
            'text': 'private-loaded',
            'senderDid': 'did:plc:peer',
            'sentAt': '2026-10-19T12:00:00Z',
          },
        ],
      });
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'a-token',
              did: a.did.value,
              handle: 'alice.test',
            ),
          ),
        ),
        accountChatRepositoryProvider(a)
            .overrideWith((ref) async => repository),
        chatDraftStoreProvider.overrideWithValue(
          ChatDraftStore(const FlutterSecureStorage()),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final family = chatConversationControllerProvider(a, id);
    final subscription = container.listen(family, (_, _) {});
    addTearDown(subscription.close);
    final controller = await container.read(family.future);
    controller.edit('private-draft');
    repository.statusFailure = const ApiNetworkError('offline');
    container.invalidate(chatStatusProvider(a));
    final unavailable = await container.read(chatStatusProvider(a).future);
    expect(unavailable.state, ChatConnectionState.unavailable);
    expect(unavailable.generation, 2);
    expect(await container.read(family.future), same(controller));
    expect(controller.history.messages.single.text, 'private-loaded');
    expect(controller.draft.text, 'private-draft');
    repository.statusFailure = null;
    container.invalidate(chatStatusProvider(a));
    await container.read(chatStatusProvider(a).future);
    expect(await container.read(family.future), same(controller));
    repository.statusFailure = const ApiUnauthorized();
    container.invalidate(chatStatusProvider(a));
    await expectLater(
      container.read(chatStatusProvider(a).future),
      throwsA(isA<ApiUnauthorized>()),
    );
    await expectLater(container.read(family.future), throwsA(anything));
    expect(controller.history.messages, isEmpty);
    repository.statusFailure = const ApiNetworkError('offline');
    container.invalidate(chatStatusProvider(a));
    await expectLater(
      container.read(chatStatusProvider(a).future),
      throwsA(isA<ApiNetworkError>()),
    );
  });

  test('IT-017 confirmed disconnect evicts content even '
      'when status refresh is offline', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final a = AccountKey('did:plc:alice');
    final id = ChatConversationId.parse('conversation');
    final repository = _Repository()
      ..page = ChatMessagePage.fromMap(const {
        'items': [
          {
            'id': 'message',
            'revision': 'r',
            'kind': 'text',
            'text': 'private-loaded',
            'senderDid': 'did:plc:peer',
            'sentAt': '2026-10-19T12:00:00Z',
          },
        ],
      });
    final store = ChatDraftStore(const FlutterSecureStorage());
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'a-token',
              did: a.did.value,
              handle: 'alice.test',
            ),
          ),
        ),
        accountChatRepositoryProvider(a)
            .overrideWith((ref) async => repository),
        chatDraftStoreProvider.overrideWithValue(store),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final family = chatConversationControllerProvider(a, id);
    final subscription = container.listen(family, (_, _) {});
    addTearDown(subscription.close);
    final controller = await container.read(family.future);
    controller.edit('retained-local-draft');
    await controller.flush();
    final settingsFamily = chatSettingsControllerProvider(a);
    final settingsSubscription = container.listen(settingsFamily, (_, _) {});
    addTearDown(settingsSubscription.close);
    final settings = await container.read(settingsFamily.future);
    repository.statusFailure = const ApiNetworkError('offline');
    await settings.disconnect();
    final status = await container.read(chatStatusProvider(a).future);
    expect(status.enabled, false);
    expect(status.generation, 3);
    await expectLater(container.read(family.future), throwsA(anything));
    expect(controller.history.messages, isEmpty);
    expect((await store.load(store.lease(a), id)).text, 'retained-local-draft');
  });

  for (final scenario in [
    (code: 'chat_disconnected', snapshot: false),
    (code: 'chat_authorization_required', snapshot: false),
    (code: 'chat_restricted', snapshot: false),
    (code: 'chat_connection_changed', snapshot: false),
    (code: 'authorizationNeeded', snapshot: true),
    (code: 'restricted', snapshot: true),
  ]) {
    test('UT-006 remote ${scenario.code} evicts content even '
        'when status refresh is offline', () async {
      FlutterSecureStorage.setMockInitialValues({});
      final a = AccountKey('did:plc:alice');
      final id = ChatConversationId.parse('conversation');
      final repository = _Repository()
        ..page = ChatMessagePage.fromMap(const {
          'items': [
            {
              'id': 'message',
              'revision': 'r',
              'kind': 'text',
              'text': 'private-loaded',
              'senderDid': 'did:plc:peer',
              'sentAt': '2026-10-19T12:00:00Z',
            },
          ],
        });
      final store = ChatDraftStore(const FlutterSecureStorage());
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'a-token',
                did: a.did.value,
                handle: 'alice.test',
              ),
            ),
          ),
          accountChatRepositoryProvider(a)
              .overrideWith((ref) async => repository),
          chatDraftStoreProvider.overrideWithValue(store),
        ],
      );
      final owner = (await container.read(sessionRegistryProvider.future))
          .activeLease;
      final family = chatConversationControllerProvider(a, id);
      final subscription = container.listen(family, (_, _) {});
      addTearDown(subscription.close);
      final controller = await container.read(family.future);
      controller.edit('retained-local-draft');
      await controller.flush();
      final syncFamily = chatSyncCoordinatorProvider(a);
      final syncSubscription = container.listen(syncFamily, (_, _) {});
      addTearDown(syncSubscription.close);
      final sync = await container.read(syncFamily.future);
      repository
        ..heartbeatFailure = scenario.snapshot
            ? null
            : ApiBadRequest(
                scenario.code,
                details: const ApiFailureDetails(statusCode: 409),
              )
        ..heartbeatState = scenario.snapshot ? scenario.code : 'ready'
        ..statusFailure = const ApiNetworkError('offline');
      sync
        ..setView(ChatView.conversation, conversationId: id)
        ..setForeground(active: true);
      await Future<void>.delayed(Duration.zero);
      await container.pump();
      final status = await container.read(chatStatusProvider(a).future);
      expect(status.enabled, scenario.code != 'chat_disconnected');
      expect(status.generation, 2);
      expect(status.canRead, false);
      expect(
        (await container.read(sessionRegistryProvider.future)).activeLease,
        owner,
      );
      await expectLater(container.read(family.future), throwsA(anything));
      expect(controller.history.messages, isEmpty);
      expect(
        (await store.load(store.lease(a), id)).text,
        'retained-local-draft',
      );
    });
  }

  test('IT-017 messaging generation cancels content HTTP '
      'while shared status transport survives', () async {
    final a = AccountKey('did:plc:alice');
    final adapter = _HeldChatAdapter();
    var generation = 1;
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'))
      ..httpClientAdapter = adapter;
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'a-token',
              did: a.did.value,
              handle: 'alice.test',
            ),
          ),
        ),
        accountDioProvider(a).overrideWith((ref) async => dio),
        chatStatusProvider(a).overrideWithBuild(
          (ref, _) async => ChatStatus.fromMap({
            'generation': generation,
            'enabled': true,
            'pushEnabled': true,
            'state': 'ready',
            'capabilities': const {
              'canRead': true,
              'canSend': true,
              'canReport': true,
              'canSetPolicy': true,
            },
          }),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final family = messagingChatRepositoryProvider(a);
    final subscription = container.listen(family, (_, _) {});
    addTearDown(subscription.close);
    final repository = await container.read(family.future);
    Object? caught;
    final pending = repository
        .conversation(ChatConversationId.parse('conversation'))
        .then<void>(
          (_) {},
          onError: (Object error, StackTrace stack) {
            caught = error;
          },
        );
    await adapter.entered.future;
    generation++;
    container.invalidate(chatStatusProvider(a));
    await container.read(chatStatusProvider(a).future);
    await container.pump();
    await Future<void>.delayed(Duration.zero);
    adapter.release.complete(
      ResponseBody.fromString(
        '{"id":"conversation","revision":"r",'
        '"status":"accepted","muted":false,"unreadCount":0,'
        '"canSend":true,"peer":{"did":"did:plc:peer",'
        '"handle":"peer.test"}}',
        200,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      ),
    );
    await pending;
    expect(adapter.cancelled, true);
    expect(caught, isNotNull);
    expect(adapter.closed, false);
  });

  test('IT-017 switching owner cancels private account '
      'HTTP without closing shared Dio', () async {
    final a = AccountKey('did:plc:alice');
    final adapter = _HeldChatAdapter();
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'))
      ..httpClientAdapter = adapter;
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'a-token',
              did: a.did.value,
              handle: 'alice.test',
            ),
          ),
        ),
        accountDioProvider(a).overrideWith((ref) async => dio),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final subscription = container.listen(
      accountChatRepositoryProvider(a),
      (_, _) {},
    );
    addTearDown(subscription.close);
    final repository = await container.read(
      accountChatRepositoryProvider(a).future,
    );
    Object? caught;
    final pending = repository.status().then<void>(
      (_) {},
      onError: (Object error, StackTrace stack) {
        caught = error;
      },
    );
    await adapter.entered.future;
    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'b-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    await container.pump();
    await Future<void>.delayed(Duration.zero);
    adapter.release.complete(
      ResponseBody.fromString(
        '{"generation":1,"enabled":true,"pushEnabled":true,'
        '"state":"ready","capabilities":{"canRead":true,'
        '"canSend":true,"canReport":true,'
        '"canSetPolicy":true}}',
        200,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      ),
    );
    await pending;
    expect(adapter.cancelled, true);
    expect(caught, isNotNull);
    expect(adapter.closed, false);
  });

  test(
    'IT-015 missing active account cannot dispatch a messaging read',
    () async {
      final a = AccountKey('did:plc:alice');
      final repository = _Repository();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(SessionRegistry.empty()),
          ),
          accountChatRepositoryProvider(a)
              .overrideWith((ref) async => repository),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final family = chatStatusProvider(a);
      final subscription = container.listen(family, (_, _) {});
      addTearDown(subscription.close);
      await expectLater(
        container.read(family.future),
        throwsA(isA<AccountOperationDiscarded>()),
      );
      expect(repository.statusCalls, 0);
    },
  );
  test(
    'IT-017 disconnect evicts private controller but retains local draft',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final a = AccountKey('did:plc:alice');
      final id = ChatConversationId.parse('conversation-1');
      final initial = SessionRegistry.empty().upsertAndActivate(
        token: 'a-token',
        did: a.did.value,
        handle: 'alice.test',
      );
      final repository = _Repository()
        ..page = ChatMessagePage.fromMap(const {
          'items': [
            {
              'id': 'message-1',
              'revision': 'rev-1',
              'kind': 'text',
              'text': 'private-body',
              'senderDid': 'did:plc:peer',
              'sentAt': '2026-10-19T12:00:00Z',
            },
          ],
        });
      final store = ChatDraftStore(const FlutterSecureStorage());
      var enabled = true;
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(initial),
          ),
          accountChatRepositoryProvider(a)
              .overrideWith((ref) async => repository),
          chatDraftStoreProvider.overrideWithValue(store),
          chatStatusProvider(a).overrideWithBuild(
            (ref, _) async => ChatStatus.fromMap({
              'enabled': enabled,
              'state': enabled ? 'ready' : 'disabled',
              'generation': 2,
              'pushEnabled': true,
              'capabilities': {
                'canRead': enabled,
                'canSend': enabled,
                'canReport': false,
                'canSetPolicy': false,
              },
            }),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final family = chatConversationControllerProvider(a, id);
      final subscription = container.listen(family, (_, _) {});
      addTearDown(subscription.close);
      final controller = await container.read(family.future);
      controller.edit('private-unsent-edit');
      await controller.flush();
      expect(controller.history.messages, hasLength(1));
      enabled = false;
      container.invalidate(chatStatusProvider(a));
      await container.read(chatStatusProvider(a).future);
      await expectLater(container.read(family.future), throwsA(anything));
      expect(controller.history.messages, isEmpty);
      expect(controller.draft.text, isEmpty);
      expect(
        (await store.load(store.lease(a), id)).text,
        'private-unsent-edit',
      );
    },
  );
  test('UT-006 unchanged status retains the open private controller', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final a = AccountKey('did:plc:alice');
    final id = ChatConversationId.parse('conversation-1');
    final initial = SessionRegistry.empty().upsertAndActivate(
      token: 'a-token',
      did: a.did.value,
      handle: 'alice.test',
    );
    final repository = _Repository();
    var generation = 2;
    var connectionState = 'ready';
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(initial),
        ),
        accountChatRepositoryProvider(a)
            .overrideWith((ref) async => repository),
        chatDraftStoreProvider.overrideWithValue(
          ChatDraftStore(const FlutterSecureStorage()),
        ),
        chatStatusProvider(a).overrideWithBuild(
          (ref, _) async => ChatStatus.fromMap({
            'enabled': true,
            'state': connectionState,
            'generation': generation,
            'pushEnabled': true,
            'capabilities': const {
              'canRead': true,
              'canSend': true,
              'canReport': true,
              'canSetPolicy': true,
            },
          }),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final family = chatConversationControllerProvider(a, id);
    final subscription = container.listen(family, (_, _) {});
    addTearDown(subscription.close);
    final controller = await container.read(family.future);
    controller.edit('private-unsent-edit');
    container.invalidate(chatStatusProvider(a));
    await container.read(chatStatusProvider(a).future);
    expect(await container.read(family.future), same(controller));
    expect(controller.draft.text, 'private-unsent-edit');
    connectionState = 'unavailable';
    container.invalidate(chatStatusProvider(a));
    await container.read(chatStatusProvider(a).future);
    expect(await container.read(family.future), same(controller));
    expect(controller.draft.text, 'private-unsent-edit');
    connectionState = 'ready';
    container.invalidate(chatStatusProvider(a));
    await container.read(chatStatusProvider(a).future);
    expect(await container.read(family.future), same(controller));
    generation++;
    container.invalidate(chatStatusProvider(a));
    await container.read(chatStatusProvider(a).future);
    final reconnected = await container.read(family.future);
    expect(reconnected, isNot(same(controller)));
    expect(controller.draft.text, isEmpty);
    expect(reconnected.draft.text, 'private-unsent-edit');
  });
  test(
    'IT-007 account controller refreshes the watched native aggregate',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final a = AccountKey('did:plc:alice');
      final id = ChatConversationId.parse('conversation-1');
      final initial = SessionRegistry.empty().upsertAndActivate(
        token: 'a-token',
        did: a.did.value,
        handle: 'alice.test',
      );
      final repository = _Repository();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(initial),
          ),
          accountChatRepositoryProvider(a)
              .overrideWith((ref) async => repository),
          chatDraftStoreProvider.overrideWithValue(
            ChatDraftStore(const FlutterSecureStorage()),
          ),
          chatStatusProvider(a).overrideWithBuild(
            (ref, _) async => ChatStatus.fromMap(const {
              'enabled': true,
              'state': 'ready',
              'generation': 2,
              'pushEnabled': true,
              'capabilities': {
                'canRead': true,
                'canSend': true,
                'canReport': true,
                'canSetPolicy': true,
              },
            }),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final counts = chatUnreadCountsProvider(a);
      final watchedCounts = container.listen(counts, (_, _) {});
      addTearDown(watchedCounts.close);
      expect((await container.read(counts.future)).requests, 5);
      final family = chatConversationControllerProvider(a, id);
      final watchedConversation = container.listen(family, (_, _) {});
      addTearDown(watchedConversation.close);
      final controller = await container.read(family.future);
      await controller.accept();
      expect((await container.read(counts.future)).requests, 2);
      expect(repository.countCalls, 2);
      expect(controller.conversation!.bucket, ChatBucket.accepted);
    },
  );
  test(
    'account repository cannot acquire under a switched activation',
    () async {
      final a = AccountKey('did:plc:alice');
      final b = AccountKey('did:plc:bob');
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'b-token',
            did: b.did.value,
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'a-token',
            did: a.did.value,
            handle: 'alice.test',
          );
      final delayed = Completer<Dio>();
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'));
      var requests = 0;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests++;
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onGet(
        '/v1/chat/status',
        (server) => server.reply(200, {
          'enabled': true,
          'state': 'ready',
          'generation': 2,
          'pushEnabled': true,
          'capabilities': {
            'canRead': true,
            'canSend': true,
            'canReport': true,
            'canSetPolicy': true,
          },
        }),
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountDioProvider(a).overrideWith((ref) => delayed.future),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final status = chatStatusProvider(a);
      final subscription = container.listen(status, (_, _) {});
      addTearDown(subscription.close);
      final pending = container.read(status.future);
      final observed = expectLater(
        pending,
        throwsA(isA<AccountOperationDiscarded>()),
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(registry.leaseFor(b)!);
      delayed.complete(dio);
      await observed;
      expect(requests, 0);
      dio.close(force: true);
    },
  );
}

ProviderContainer _sessionContainer(_Repository repository) {
  final account = AccountKey('did:plc:alice');
  return ProviderContainer.test(
    retry: (_, _) => null,
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(
        _Storage(
          SessionRegistry.empty().upsertAndActivate(
            token: 'synthetic',
            did: account.did.value,
            handle: 'alice.test',
          ),
        ),
      ),
      accountChatRepositoryProvider.overrideWith(
        (ref, account) async => repository,
      ),
      chatDraftStoreProvider.overrideWithValue(
        ChatDraftStore(const FlutterSecureStorage()),
      ),
    ],
  );
}

ChatMessagePage _privatePage() => ChatMessagePage.fromMap(const {
  'items': [
    {
      'id': 'message',
      'revision': 'r',
      'kind': 'text',
      'text': 'private-cache-canary',
      'senderDid': 'did:plc:peer',
      'sentAt': '2026-10-19T12:00:00Z',
    },
  ],
});

class _Repository implements ChatRepository {
  final notificationEntered = Completer<void>();
  Future<ChatConversationId>? notificationTarget;
  @override
  Future<ChatConversationId> notificationOpen(
    ChatOpenReference reference,
  ) async {
    notificationEntered.complete();
    return await notificationTarget!;
  }

  bool enabled = true;
  int listCalls = 0;
  final listBuckets = <ChatBucket>[];
  int historyCalls = 0;
  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    listCalls++;
    listBuckets.add(bucket);
    return ChatConversationPage(
      items: [await conversation(ChatConversationId.parse('conversation'))],
    );
  }

  int heartbeatCalls = 0;
  String heartbeatState = 'ready';
  ApiException? heartbeatFailure;
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    heartbeatCalls++;
    if (heartbeatFailure case final error?) throw error;
    return ChatSyncSnapshot.fromMap({
      'state': heartbeatState,
      'revision': 1,
    });
  }

  int statusCalls = 0;
  Object? statusFailure;
  @override
  Future<ChatStatus> status() async {
    statusCalls++;
    if (statusFailure case final error?) {
      Error.throwWithStackTrace(error, StackTrace.current);
    }
    return ChatStatus.fromMap({
      'enabled': enabled,
      'state': enabled ? 'ready' : 'disabled',
      'generation': 2,
      'pushEnabled': true,
      'capabilities': {
        'canRead': enabled,
        'canSend': enabled,
        'canReport': enabled,
        'canSetPolicy': enabled,
      },
    });
  }

  @override
  Future<bool> pushPreference() async => true;
  @override
  Future<ChatPolicySetting> incomingPolicy() async => ChatPolicySetting.fromMap(
    const {'allowIncoming': 'everyone', 'persisted': true},
  );
  @override
  Future<ChatStatus> setConnection({required bool enabled}) async =>
      ChatStatus.fromMap(const {
        'generation': 3,
        'enabled': false,
        'pushEnabled': true,
        'state': 'disabled',
        'capabilities': {
          'canRead': false,
          'canSend': false,
          'canReport': false,
          'canSetPolicy': false,
        },
      });
  ChatMessagePage? page;
  bool accepted = false;
  int countCalls = 0;
  int unreadAccepted = 1;
  int? unreadRequests;
  @override
  Future<ChatUnreadCounts> unreadCounts() async {
    countCalls++;
    return ChatUnreadCounts.fromMap({
      'accepted': unreadAccepted,
      'requests': unreadRequests ?? (accepted ? 2 : 5),
    });
  }

  @override
  Future<ChatConversationDetails> details(ChatConversationId id) async =>
      ChatConversationDetails(
        conversation: await conversation(id),
        messages: await history(id),
      );

  @override
  Future<ChatConversation> conversation(ChatConversationId id) async =>
      ChatConversation.fromMap({
        'id': id.value,
        'revision': 'rev-1',
        'status': accepted ? 'accepted' : 'request',
        'muted': false,
        'unreadCount': 0,
        'canSend': accepted,
        'peer': const {'did': 'did:plc:peer'},
      });
  @override
  Future<ChatMessagePage> history(
    ChatConversationId id, {
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    historyCalls++;
    return page ?? ChatMessagePage(items: const []);
  }

  @override
  Future<ChatConversation> accept(ChatConversationId id) async {
    accepted = true;
    return await conversation(id);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry registry) async {
    value = registry;
  }
}

class _HeldChatAdapter implements HttpClientAdapter {
  final entered = Completer<void>();
  final release = Completer<ResponseBody>();
  bool cancelled = false;
  bool closed = false;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (!entered.isCompleted) entered.complete();
    unawaited(
      cancelFuture?.then((_) {
        cancelled = true;
      }),
    );
    return await release.future;
  }

  @override
  void close({bool force = false}) {
    closed = true;
  }
}
