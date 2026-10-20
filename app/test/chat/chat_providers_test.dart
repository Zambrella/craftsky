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
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_draft_store_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
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

class _Repository implements ChatRepository {
  String heartbeatState = 'ready';
  ApiException? heartbeatFailure;
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
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
    return ChatStatus.fromMap(const {
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
  @override
  Future<ChatUnreadCounts> unreadCounts() async {
    countCalls++;
    return ChatUnreadCounts.fromMap({
      'accepted': 1,
      'requests': accepted ? 2 : 5,
    });
  }

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
  }) async => page ?? ChatMessagePage(items: const []);
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
