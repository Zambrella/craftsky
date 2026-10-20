import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/api_chat_repository.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/services/chat_conversation_controller.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test(
    'IT-006 uncertain response reconciles before another send can start',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation');
      final operation = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final receipt = Completer<ChatReceipt>();
      final receiptEntered = Completer<void>();
      final remote = _Repository()
        ..sendOverride = (op, text) async {
          return ChatSendResult.fromMap({
            'operationId': op.value,
            'kind': 'send',
            'state': 'uncertain',
          });
        }
        ..receiptReader = (op) {
          receiptEntered.complete();
          return receipt.future;
        };
      var allocated = 0;
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        newOperationId: () {
          allocated++;
          return operation;
        },
      );
      addTearDown(controller.dispose);
      await controller.load();
      controller.edit('private-original');
      final sending = controller.send();
      await receiptEntered.future;
      expect(controller.busy, isTrue);
      controller.edit('private-later-edit');
      await controller.sendAgain(duplicateRiskAcknowledged: true);
      await controller.send();
      expect(allocated, 1);
      expect(remote.sendCalls, 1);
      receipt.complete(
        ChatReceipt.fromMap({
          'operationId': operation.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'original-message',
        }),
      );
      await sending;
      expect(remote.historyCalls, 2);
      expect(controller.draft.pending, isNull);
      expect(controller.draft.text, 'private-later-edit');
      expect(controller.busy, isFalse);
      expect(
        (await store.load(store.lease(account), id)).text,
        'private-later-edit',
      );
    },
  );
  test('UT-009 receipt lookup failure still refreshes history'
      ' without inferring delivery', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final id = ChatConversationId.parse('conversation');
    final original = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    await store.save(
      store.lease(account),
      id,
      const ChatDraft.empty()
          .edit('private-original')
          .beginSend(original)
          .markUncertain(),
    );
    final remote = _Repository();
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: remote,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    await controller.load();
    remote
      ..receiptReader = (_) async {
        throw StateError('private-provider-prose');
      }
      ..page = ChatMessagePage.fromMap(const {
        'items': [
          {
            'id': 'similar-message',
            'revision': 'r',
            'kind': 'text',
            'text': 'private-original',
            'senderDid': 'did:plc:owner',
            'sentAt': '2026-10-19T12:00:00Z',
          },
        ],
      });
    await controller.checkReceipt();
    expect(remote.historyCalls, 2);
    expect(controller.history.messages.single.text, 'private-original');
    expect(controller.draft.pending!.operationId, original);
    expect(controller.draft.pending!.state, ChatDraftSendState.uncertain);
    expect(controller.draft.text, 'private-original');
    expect(remote.sendCalls, 0);
    await controller.sendAgain(duplicateRiskAcknowledged: false);
    expect(remote.receiptCalls, 1);
    expect(remote.sendCalls, 0);
  });
  test(
    'UT-009 confirmed original receipt prevents a fresh send operation',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation');
      final original = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      final next = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000456',
      );
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      await store.save(
        store.lease(account),
        id,
        const ChatDraft.empty()
            .edit('private-original')
            .beginSend(original)
            .markUncertain(),
      );
      final remote = _Repository()
        ..receiptResult = ChatReceipt.fromMap({
          'operationId': original.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'original-message',
        })
        ..sendOverride = (operation, text) async => ChatSendResult.fromMap({
          'operationId': operation.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'duplicate-message',
        });
      var allocated = 0;
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        newOperationId: () {
          allocated++;
          return next;
        },
      );
      addTearDown(controller.dispose);
      await controller.load();
      await controller.sendAgain(duplicateRiskAcknowledged: true);
      expect(remote.sendCalls, 0);
      expect(allocated, 0);
      expect(remote.receiptCalls, 1);
      expect(remote.historyCalls, 2);
      expect(controller.draft.pending, isNull);
      expect(controller.draft.text, isEmpty);
      expect((await store.load(store.lease(account), id)).pending, isNull);
    },
  );
  test('IT-006 consumer failed receipt retains a failed local draft', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final id = ChatConversationId.parse('conversation');
    final operation = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'))
      ..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onPost(
      '/v1/chat/conversations/conversation/messages',
      (server) => server.reply(200, {
        'operationId': operation.value,
        'kind': 'send',
        'state': 'failed',
        'failureCode': 'rejected',
      }),
      data: {'operationId': operation.value, 'text': 'private-rejected-text'},
    );
    final remote = _Repository()
      ..sendOverride = (op, text) => ApiChatRepository(dio).send(id, op, text);
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: remote,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
      newOperationId: () => operation,
    );
    addTearDown(controller.dispose);
    await controller.load();
    controller.edit('private-rejected-text');
    await controller.send();
    expect(controller.draft.pending!.state, ChatDraftSendState.failed);
    expect(controller.draft.pending!.operationId, operation);
    expect(controller.draft.text, 'private-rejected-text');
    expect(controller.history.messages, isEmpty);
    expect(remote.sendCalls, 1);
    expect(
      (await store.load(store.lease(account), id)).pending!.state,
      ChatDraftSendState.failed,
    );
  });
  test(
    'IT-013 access loss hides content and fences held private completions',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation');
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      Map<String, dynamic> message(String id) => {
        'id': id,
        'revision': 'r',
        'kind': 'text',
        'text': 'private-content',
        'senderDid': 'did:plc:peer',
        'sentAt': '2026-10-19T12:00:00Z',
      };
      final remote = _Repository()
        ..page = ChatMessagePage.fromMap({
          'items': [message('loaded')],
          'cursor': '00000000-0000-4000-8000-000000000123',
        })
        ..olderPage = Completer<ChatMessagePage>()
        ..readResult = Completer<ChatConversation>();
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      controller.edit('original-draft');
      final sending = controller.send();
      await remote.entered.future;
      controller.edit('newer-local-draft');
      await controller.flush();
      final older = controller.loadOlder();
      controller.setForeground(value: true);
      final reading = controller.presented([ChatMessageId.parse('loaded')]);
      await Future<void>.delayed(Duration.zero);
      // Same consumer response asserted by the migrated upstream/HTTP test.
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'))
        ..interceptors.add(const ErrorMappingInterceptor());
      DioAdapter(dio: dio).onGet(
        '/v1/chat/conversations/conversation',
        (server) => server.reply(404, {
          'error': 'chat_conversation_unavailable',
          'message': 'conversation or message unavailable',
          'requestId': 'synthetic-request',
        }),
      );
      remote.summaryReader = () => ApiChatRepository(dio).conversation(id);
      await controller.refresh();
      expect(controller.conversation, isNull);
      expect(controller.history.messages, isEmpty);
      expect(controller.draft.text, 'newer-local-draft');
      expect(controller.canSend, false);
      remote.summaryReader = null;
      remote.olderPage!.complete(
        ChatMessagePage.fromMap({
          'items': [message('stale-older')],
        }),
      );
      remote.readResult!.complete(_summary(id));
      remote.result.complete(
        ChatSendResult.fromMap({
          'operationId': controller.draft.pending!.operationId.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'stale-send',
          'message': message('stale-send'),
        }),
      );
      await Future.wait([older, reading, sending]);
      expect(controller.conversation, isNull);
      expect(controller.history.messages, isEmpty);
      expect(controller.draft.text, 'newer-local-draft');
      expect(
        (await store.load(store.lease(account), id)).text,
        'newer-local-draft',
      );
    },
  );

  test(
    'UT-009 failed local resend save preserves the original unresolved receipt',
    () async {
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation-1');
      final first = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      final next = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000456',
      );
      final backend = _FailingBackend();
      final store = ChatDraftStore.withBackend(backend);
      await store.initialize({account});
      await store.save(
        store.lease(account),
        id,
        const ChatDraft.empty()
            .edit('private-original')
            .beginSend(first)
            .markUncertain(),
      );
      final remote = _Repository();
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        newOperationId: () => next,
      );
      addTearDown(controller.dispose);
      await controller.load();
      controller.edit('private-newer');
      backend.fail = true;
      await controller.sendAgain(duplicateRiskAcknowledged: true);
      expect(remote.sendCalls, 0);
      expect(controller.saveFailed, isTrue);
      expect(controller.draft.text, 'private-newer');
      expect(controller.draft.pending!.operationId, first);
      expect(controller.draft.pending!.state, ChatDraftSendState.uncertain);
      backend.fail = false;
    },
  );
  test('UT-009 uncertain resend needs acknowledgment and '
      'a fresh durable operation', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final id = ChatConversationId.parse('conversation-1');
    final first = ChatOperationId.parse('00000000-0000-4000-8000-000000000123');
    final next = ChatOperationId.parse('00000000-0000-4000-8000-000000000456');
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    await store.save(
      store.lease(account),
      id,
      const ChatDraft.empty()
          .edit('private-original')
          .beginSend(first)
          .markUncertain(),
    );
    final remote = _Repository()
      ..sendOverride = (operation, text) async {
        expect(operation, next);
        expect(text, 'private-original');
        expect(
          (await store.load(store.lease(account), id)).pending!.operationId,
          next,
        );
        return ChatSendResult.fromMap({
          'operationId': operation.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'message-2',
        });
      };
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: remote,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
      newOperationId: () => next,
    );
    addTearDown(controller.dispose);
    await controller.load();
    controller.edit('private-newer');
    await controller.sendAgain(duplicateRiskAcknowledged: false);
    expect(remote.sendCalls, 0);
    expect(controller.draft.pending!.operationId, first);
    await controller.sendAgain(duplicateRiskAcknowledged: true);
    expect(remote.sendCalls, 1);
    expect(controller.draft.text, 'private-newer');
    expect(controller.draft.pending, isNull);
  });
  test(
    'IT-007 request waits for acceptance and refreshes native counts',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final id = ChatConversationId.parse('conversation-1');
      final remote = _Repository()..request = true;
      var countRefreshes = 0;
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onSharedStateChanged: () async {
          countRefreshes++;
        },
      );
      addTearDown(controller.dispose);
      await controller.load();
      controller.edit('private-request-draft');
      expect(controller.canSend, isFalse);
      await controller.send();
      expect(remote.sendCalls, 0);
      expect(remote.acceptCalls, 0);
      expect(countRefreshes, 0);
      await controller.accept();
      expect(remote.acceptCalls, 1);
      expect(controller.canSend, isTrue);
      expect(controller.draft.text, 'private-request-draft');
      expect(countRefreshes, 1);
      await controller.setMuted(muted: true);
      expect(controller.conversation!.muted, isTrue);
      expect(countRefreshes, 2);
      expect(await controller.leave(), isTrue);
      expect(remote.leaveCalls, 1);
      expect(controller.left, isTrue);
      expect(controller.canSend, isFalse);
      expect(countRefreshes, 3);
      expect(remote.readIds, isEmpty);
    },
  );
  test(
    'UT-003 delayed older page cannot replace a newer history-gap cursor',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation');
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      ChatMessagePage page(String id, String cursor) =>
          ChatMessagePage.fromMap({
            'items': [
              {
                'id': id,
                'revision': 'r',
                'kind': 'text',
                'text': 'private',
                'senderDid': 'did:plc:peer',
                'sentAt': '2026-10-19T12:00:00Z',
              },
            ],
            'cursor': cursor,
          });
      const old = '00000000-0000-4000-8000-000000000123';
      const gap = '00000000-0000-4000-8000-000000000456';
      const older = '00000000-0000-4000-8000-000000000789';
      final remote = _Repository()
        ..page = page('old-message', old)
        ..olderPage = Completer<ChatMessagePage>();
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      final pending = controller.loadOlder();
      remote.page = page('new-message', gap);
      await controller.refresh();
      remote.olderPage!.complete(page('older-message', older));
      await pending;
      expect(controller.history.cursor?.value, gap);
      expect(
        controller.history.messages.map((message) => message.id.value),
        containsAll(['old-message', 'new-message', 'older-message']),
      );
    },
  );

  test(
    'UT-003 disjoint refresh keeps a continuation through the history gap',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation');
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      ChatMessagePage page(List<String> ids, String? cursor) =>
          ChatMessagePage.fromMap({
            'items': [
              for (final id in ids)
                {
                  'id': id,
                  'revision': 'r',
                  'kind': 'text',
                  'text': 'private',
                  'senderDid': 'did:plc:peer',
                  'sentAt': '2026-10-19T12:00:00Z',
                },
            ],
            'cursor': ?cursor,
          });
      const oldCursor = '00000000-0000-4000-8000-000000000123';
      const gapCursor = '00000000-0000-4000-8000-000000000456';
      const nextGap = '00000000-0000-4000-8000-000000000789';
      final remote = _Repository()..page = page(['old-message'], oldCursor);
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      remote.page = page(['newest-message'], gapCursor);
      await controller.refresh();
      expect(controller.history.cursor?.value, gapCursor);
      remote.page = page(['middle-message'], nextGap);
      await controller.loadOlder();
      expect(remote.historyCursors.last?.value, gapCursor);
      expect(controller.history.cursor?.value, nextGap);
      remote.page = page(['old-message', 'gap-edge'], null);
      await controller.loadOlder();
      expect(controller.history.cursor?.value, oldCursor);
      expect(
        controller.history.messages.map((message) => message.id.value),
        containsAll([
          'old-message',
          'newest-message',
          'middle-message',
          'gap-edge',
        ]),
      );
      expect(remote.readIds, isEmpty);
    },
  );
  test(
    'UT-003 older paging coalesces and newest refresh preserves continuation',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final id = ChatConversationId.parse('conversation-1');
      Map<String, dynamic> item(String id) => {
        'id': id,
        'revision': 'rev-1',
        'kind': 'text',
        'text': 'private-text',
        'senderDid': 'did:plc:peer',
        'sentAt': '2026-10-19T12:00:00Z',
      };
      final firstCursor = ChatCursor.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      final olderCursor = ChatCursor.parse(
        '00000000-0000-4000-8000-000000000456',
      );
      final remote = _Repository()
        ..page = ChatMessagePage.fromMap({
          'items': [item('message-2')],
          'cursor': firstCursor.value,
        })
        ..olderPage = Completer<ChatMessagePage>();
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      controller.edit('private-current-draft');
      final older = controller.loadOlder();
      await controller.loadOlder();
      expect(remote.historyCalls, 2);
      remote.olderPage!.complete(
        ChatMessagePage.fromMap({
          'items': [item('message-1'), item('message-2')],
          'cursor': olderCursor.value,
        }),
      );
      await older;
      remote.page = ChatMessagePage.fromMap({
        'items': [item('message-2'), item('message-3')],
        'cursor': firstCursor.value,
      });
      await controller.refresh();
      expect(controller.history.messages.map((item) => item.id.value), [
        'message-1',
        'message-2',
        'message-3',
      ]);
      expect(controller.history.cursor, olderCursor);
      expect(controller.draft.text, 'private-current-draft');
      expect(remote.readIds, isEmpty);
    },
  );
  test('UT-009 uncertain send checks receipt without replay'
      ' and requires an explicit new operation', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    final id = ChatConversationId.parse('conversation-1');
    final operation = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    final remote = _Repository();
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: remote,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
      newOperationId: () => operation,
    );
    addTearDown(controller.dispose);
    await controller.load();
    controller.edit('private-uncertain');
    final sending = controller.send();
    await remote.entered.future;
    remote.result.completeError(StateError('private-provider-prose'));
    await sending;
    expect(controller.draft.pending!.state, ChatDraftSendState.uncertain);
    expect(controller.draft.text, 'private-uncertain');
    expect(remote.receiptCalls, 1);
    expect(remote.historyCalls, 2);
    await controller.send();
    expect(remote.sendCalls, 1);
    remote.receiptResult = ChatReceipt.fromMap({
      'operationId': operation.value,
      'kind': 'send',
      'state': 'confirmed',
      'messageId': 'message-1',
    });
    await controller.checkReceipt();
    expect(remote.receiptCalls, 2);
    expect(remote.historyCalls, 3);
    expect(remote.sendCalls, 1);
    expect(controller.draft.text, isEmpty);
    expect(controller.draft.pending, isNull);
    controller.edit('private-discard');
    await controller.discardDraft();
    expect((await store.load(store.lease(account), id)).text, isEmpty);
  });
  test(
    'UT-004 read uses last presented message only while foreground',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final id = ChatConversationId.parse('conversation-1');
      final remote = _Repository()
        ..page = ChatMessagePage.fromMap({
          'items': [
            for (var index = 1; index <= 3; index++)
              {
                'id': 'message-$index',
                'revision': 'rev-$index',
                'kind': 'text',
                'text': 'private-text',
                'senderDid': 'did:plc:peer',
                'sentAt': '2026-10-19T12:0$index:00Z',
              },
          ],
        });
      var countRefreshes = 0;
      final controller = ChatConversationController(
        onSharedStateChanged: () async {
          countRefreshes++;
        },
        account: account,
        id: id,
        repository: remote,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      expect(remote.readIds, isEmpty);
      final one = ChatMessageId.parse('message-1');
      final two = ChatMessageId.parse('message-2');
      await controller.presented([one]);
      expect(remote.readIds, isEmpty);
      controller.setForeground(value: true);
      await controller.presented([one]);
      expect(remote.readIds, [one]);
      controller.setForeground(value: false);
      await controller.presented([two]);
      expect(remote.readIds, [one]);
      controller.setForeground(value: true);
      await controller.presented([two, one]);
      await controller.presented([one]);
      await controller.presented([ChatMessageId.parse('not-present')]);
      expect(remote.readIds, [one, two]);
      expect(countRefreshes, 2);
    },
  );
  test('IT-017 stale load cannot publish or erase a retained draft', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    final id = ChatConversationId.parse('conversation-1');
    await store.save(
      store.lease(account),
      id,
      const ChatDraft.empty().edit('private-before'),
    );
    final remote = _Repository()
      ..delayedSummary = Completer<ChatConversation>();
    var current = true;
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: remote,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => current,
      ),
    );
    var publications = 0;
    controller.addListener(() => publications++);
    final loading = controller.load();
    current = false;
    remote.delayedSummary!.complete(_summary(id));
    await loading;
    expect(controller.conversation, isNull);
    expect(publications, 0);
    expect(remote.historyCalls, 0);
    controller.dispose();
    expect((await store.load(store.lease(account), id)).text, 'private-before');
  });
  test('UT-009 pending is durable before dispatch'
      ' and confirmation preserves later edit', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    final id = ChatConversationId.parse('conversation-1');
    final operation = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    var countRefreshes = 0;
    final remote = _Repository();
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: remote,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
      onSharedStateChanged: () async {
        countRefreshes++;
      },
      newOperationId: () => operation,
    );
    addTearDown(controller.dispose);
    await controller.load();
    expect(remote.readCalls, 0);
    controller.edit('private-original');
    remote.beforeSend = () async {
      final restored = await store.load(store.lease(account), id);
      expect(restored.pending!.operationId, operation);
      expect(restored.pending!.state, ChatDraftSendState.uncertain);
    };
    final sending = controller.send();
    await remote.entered.future;
    expect(controller.draft.pending!.state, ChatDraftSendState.pending);
    controller.edit('private-newer');
    remote.result.complete(
      ChatSendResult.fromMap({
        'operationId': operation.value,
        'kind': 'send',
        'state': 'confirmed',
        'messageId': 'message-1',
        'message': {
          'id': 'message-1',
          'revision': 'rev-2',
          'kind': 'text',
          'text': 'private-original',
          'senderDid': account.did.value,
          'sentAt': '2026-10-19T12:00:00Z',
        },
      }),
    );
    await sending;
    expect(controller.draft.text, 'private-newer');
    expect(controller.history.messages.single.id.value, 'message-1');
    expect(controller.draft.pending, isNull);
    expect(remote.sendCalls, 1);
    expect(countRefreshes, 1);
    await controller.flush();
    expect((await store.load(store.lease(account), id)).text, 'private-newer');
  });
}

class _Repository implements ChatRepository {
  Future<ChatConversation> Function()? summaryReader;
  Completer<ChatConversation>? readResult;
  Future<ChatSendResult> Function(ChatOperationId, String)? sendOverride;
  bool request = false;
  bool muted = false;
  int acceptCalls = 0;
  int leaveCalls = 0;
  Completer<ChatMessagePage>? olderPage;
  ChatReceipt? receiptResult;
  Future<ChatReceipt> Function(ChatOperationId)? receiptReader;
  int receiptCalls = 0;
  ChatMessagePage? page;
  final readIds = <ChatMessageId>[];
  Completer<ChatConversation>? delayedSummary;
  int historyCalls = 0;
  final historyCursors = <ChatCursor?>[];
  final entered = Completer<void>();
  final result = Completer<ChatSendResult>();
  Future<void> Function()? beforeSend;
  int sendCalls = 0;
  int readCalls = 0;
  @override
  Future<ChatConversation> conversation(ChatConversationId id) async {
    if (summaryReader case final read?) return await read();
    return delayedSummary == null
        ? _summary(id, request: request, muted: muted)
        : await delayedSummary!.future;
  }

  @override
  Future<ChatConversation> accept(ChatConversationId id) async {
    acceptCalls++;
    request = false;
    return _summary(id);
  }

  @override
  Future<ChatConversation> setMuted(
    ChatConversationId id, {
    required bool muted,
  }) async {
    this.muted = muted;
    return _summary(id, muted: muted);
  }

  @override
  Future<void> leave(ChatConversationId id) async {
    leaveCalls++;
  }

  @override
  Future<ChatMessagePage> history(
    ChatConversationId id, {
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    historyCalls++;
    historyCursors.add(cursor);
    if (cursor != null && olderPage != null) return await olderPage!.future;
    return page ?? ChatMessagePage(items: const []);
  }

  @override
  Future<ChatConversation> updateRead(
    ChatConversationId id,
    ChatMessageId message,
  ) async {
    readIds.add(message);
    return readResult == null ? _summary(id) : await readResult!.future;
  }

  @override
  Future<ChatSendResult> send(
    ChatConversationId id,
    ChatOperationId operation,
    String text,
  ) async {
    sendCalls++;
    if (sendOverride != null) return await sendOverride!(operation, text);
    await beforeSend?.call();
    entered.complete();
    return await result.future;
  }

  @override
  Future<ChatReceipt> receipt(ChatOperationId operation) async {
    receiptCalls++;
    if (receiptReader case final read?) return await read(operation);
    return receiptResult ??
        ChatReceipt.fromMap({
          'operationId': operation.value,
          'kind': 'send',
          'state': 'uncertain',
        });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

ChatConversation _summary(
  ChatConversationId id, {
  bool request = false,
  bool muted = false,
}) => ChatConversation.fromMap({
  'id': id.value,
  'revision': 'rev-1',
  'status': request ? 'request' : 'accepted',
  'muted': muted,
  'unreadCount': 1,
  'canSend': !request,
  'peer': const {'did': 'did:plc:peer'},
});

class _FailingBackend implements ChatDraftStorageBackend {
  final values = <String, String>{};
  bool fail = false;
  @override
  Future<String?> read(String key) async => values[key];
  @override
  Future<void> write(String key, String value) async {
    if (fail) throw StateError('private-storage-prose');
    values[key] = value;
  }

  @override
  Future<void> delete(String key) async {
    values.remove(key);
  }
}
