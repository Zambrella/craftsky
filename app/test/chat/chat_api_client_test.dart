import 'package:craftsky_app/chat/data/api_chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

// IT-019 / FR-002 / NFR-003: fixed consumer route, bounded opaque paging.
void main() {
  test('IT-001 future last-message envelopes do not hide the conversation', () {
    final page = ChatConversationPage.fromMap(const {
      'items': [
        {
          'id': 'conversation',
          'revision': 'r',
          'status': 'accepted',
          'muted': false,
          'unreadCount': 1,
          'canSend': true,
          'peer': {'did': 'did:plc:peer'},
          'lastMessage': {
            'kind': 'unsupported',
            'id': '',
            'revision': '',
            'sentAt': '',
          },
        },
      ],
    });
    expect(page.items.single.id.value, 'conversation');
    expect(page.items.single.lastMessage, isNull);
  });

  test('IT-019 confirmed send requires an authoritative message ID', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'));
    final adapter = DioAdapter(dio: dio);
    final repository = ApiChatRepository(dio);
    final operation = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    adapter.onGet(
      '/v1/chat/operations/${operation.value}',
      (server) => server.reply(200, {
        'operationId': operation.value,
        'kind': 'send',
        'state': 'confirmed',
      }),
    );
    await expectLater(
      repository.receipt(operation),
      throwsA(isA<FormatException>()),
    );
  });

  test(
    'status preferences heartbeat report and opaque open use fixed contracts',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'));
      final adapter = DioAdapter(dio: dio);
      final repository = ApiChatRepository(dio);
      final status = <String, dynamic>{
        'enabled': true,
        'pushEnabled': false,
        'state': 'ready',
        'generation': 2,
        'syncState': 'rateLimited',
        'recoveryCode': 'rateLimited',
        'lastSuccessAt': '2026-10-19T12:00:00Z',
        'capabilities': {
          'canRead': true,
          'canSend': true,
          'canReport': false,
          'canSetPolicy': true,
        },
      };
      adapter.onGet('/v1/chat/status', (server) => server.reply(200, status));
      final connection = await repository.status();
      expect(connection.canReport, isFalse);
      expect(connection.syncState, 'rateLimited');
      expect(connection.recoveryCode, 'rateLimited');
      expect(connection.lastSuccessAt, DateTime.utc(2026, 10, 19, 12));
      expect(connection.degraded().lastSuccessAt, connection.lastSuccessAt);
      adapter.onPut(
        '/v1/chat/connection',
        (server) => server.reply(200, {
          ...status,
          'enabled': false,
          'state': 'disabled',
        }),
        data: {'enabled': false},
      );
      expect((await repository.setConnection(enabled: false)).enabled, isFalse);
      adapter.onGet(
        '/v1/chat/unread-counts',
        (server) => server.reply(200, {'accepted': 100, 'requests': 4}),
      );
      final counts = await repository.unreadCounts();
      expect(counts.acceptedLabel, '99+');
      expect(counts.requestsLabel, '4');
      adapter.onGet(
        '/v1/chat/push-preferences',
        (server) => server.reply(200, {'pushEnabled': false}),
      );
      expect(await repository.pushPreference(), isFalse);
      adapter.onPut(
        '/v1/chat/push-preferences',
        (server) => server.reply(200, {'pushEnabled': true}),
        data: {'pushEnabled': true},
      );
      expect(await repository.setPushPreference(enabled: true), isTrue);
      adapter.onGet(
        '/v1/chat/incoming-policy',
        (server) => server.reply(200, {
          'allowIncoming': 'following',
          'persisted': true,
        }),
      );
      expect(
        (await repository.incomingPolicy()).value,
        ChatIncomingPolicy.following,
      );
      adapter.onPut(
        '/v1/chat/incoming-policy',
        (server) =>
            server.reply(200, {'allowIncoming': 'nobody', 'persisted': true}),
        data: {'allowIncoming': 'nobody'},
      );
      expect(
        (await repository.setIncomingPolicy(ChatIncomingPolicy.nobody))
            .persisted,
        isTrue,
      );
      adapter.onPost(
        '/v1/chat/sync',
        (server) => server.reply(200, {
          'revision': 9,
          'state': 'ready',
          'lastSuccessAt': '2026-10-19T10:00:00Z',
        }),
        data: {'view': 'inbox', 'knownRevision': 8},
      );
      expect(
        (await repository.heartbeat(ChatView.inbox, knownRevision: 8)).revision,
        9,
      );
      final reference = ChatOpenReference.parse(
        '00000000-0000-4000-8000-000000000789',
      );
      final id = ChatConversationId.parse('private-conversation');
      adapter.onGet(
        '/v1/chat/notification-opens/${reference.value}',
        (server) => server.reply(200, {
          'type': 'conversation',
          'conversationId': id.value,
        }),
      );
      expect(await repository.notificationOpen(reference), id);
      final operation = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      final message = ChatMessageId.parse('message-1');
      const reason = 'com.atproto.moderation.defs#reasonSpam';
      adapter.onPost(
        '/v1/chat/reports',
        (server) => server.reply(200, {
          'operationId': operation.value,
          'kind': 'report',
          'state': 'confirmed',
        }),
        data: {
          'operationId': operation.value,
          'conversationId': id.value,
          'messageId': message.value,
          'reasonType': reason,
          'reason': 'private-explanation',
        },
      );
      expect(
        (await repository.report(
          operation,
          id,
          message,
          reason,
          'private-explanation',
        )).state,
        ChatOperationState.confirmed,
      );
      adapter.onPost(
        '/v1/chat/authorization',
        (server) => server.reply(200, {
          'authUrl': 'https://pds.test/oauth/authorize?state=private-state',
        }),
        data: {'handoffMode': 'verified_link', 'pushEnabled': false},
      );
      expect(
        (await repository.authorize(
          handoffMode: 'verified_link',
          pushEnabled: false,
        )).scheme,
        'https',
      );
      dio.close(force: true);
    },
  );

  test(
    'fixed conversation operations preserve receipts and explicit watermarks',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'));
      final adapter = DioAdapter(dio: dio);
      final repository = ApiChatRepository(dio);
      final id = ChatConversationId.parse('private-conversation');
      final message = ChatMessageId.parse('message-1');
      final operation = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000321',
      );
      const root = '/v1/chat/conversations/private-conversation';
      final conversation = <String, dynamic>{
        'id': id.value,
        'revision': 'rev-1',
        'status': 'accepted',
        'muted': false,
        'unreadCount': 0,
        'canSend': true,
        'peer': {'did': 'did:plc:external-peer'},
      };
      adapter.onGet(root, (server) => server.reply(200, conversation));
      expect((await repository.conversation(id)).id, id);
      adapter.onGet(
        '$root/messages',
        (server) => server.reply(200, {'items': <Object>[]}),
        queryParameters: {'limit': 50},
      );
      expect((await repository.history(id)).items, isEmpty);
      adapter.onPost(
        '$root/messages',
        (server) => server.reply(202, {
          'operationId': operation.value,
          'kind': 'send',
          'state': 'uncertain',
        }),
        data: {'operationId': operation.value, 'text': 'private-text'},
      );
      expect(
        (await repository.send(id, operation, 'private-text')).receipt.state,
        ChatOperationState.uncertain,
      );
      adapter.onGet(
        '/v1/chat/operations/${operation.value}',
        (server) => server.reply(200, {
          'operationId': operation.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': message.value,
        }),
      );
      expect((await repository.receipt(operation)).messageId, message);
      adapter.onPut(
        '$root/read',
        (server) => server.reply(200, conversation),
        data: {'messageId': message.value},
      );
      await repository.updateRead(id, message);
      adapter.onPost(
        '$root/accept',
        (server) => server.reply(200, conversation),
      );
      await repository.accept(id);
      adapter.onPut('$root/mute', (server) => server.reply(200, conversation));
      await repository.setMuted(id, muted: true);
      adapter.onDelete(
        '$root/mute',
        (server) => server.reply(200, conversation),
      );
      await repository.setMuted(id, muted: false);
      adapter.onPost(
        '$root/leave',
        (server) => server.reply(200, {'left': true}),
      );
      await repository.leave(id);
      final peer = Did.parse('did:plc:external-peer');
      adapter.onGet(
        '/v1/chat/availability',
        (server) => server.reply(200, {'canChat': false}),
        queryParameters: {'recipientDid': peer.value},
      );
      expect((await repository.availability(peer)).canChat, isFalse);
      adapter.onPost(
        '/v1/chat/conversations',
        (server) => server.reply(200, conversation),
        data: {'recipientDid': peer.value},
      );
      expect((await repository.start(peer)).id, id);
      dio.close(force: true);
    },
  );

  test(
    'inbox uses fixed bucket and retains opaque empty-page continuation',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'));
      final adapter = DioAdapter(dio: dio);
      final repository = ApiChatRepository(dio);
      const cursor = '00000000-0000-4000-8000-000000000123';
      adapter.onGet(
        '/v1/chat/conversations',
        (server) => server.reply(200, {
          'items': <Object>[],
          'cursor': cursor,
        }),
        queryParameters: {'status': 'request', 'limit': 50},
      );
      final page = await repository.conversations(bucket: ChatBucket.request);
      expect(page.items, isEmpty);
      expect(page.cursor, ChatCursor.parse(cursor));
      adapter.onGet(
        '/v1/chat/conversations',
        (server) => server.reply(200, {
          'items': [
            {
              'id': 'private-conversation',
              'revision': 'rev-1',
              'status': 'request',
              'muted': true,
              'unreadCount': 7,
              'canSend': false,
              'peer': {
                'did': 'did:plc:external-peer',
                'handle': 'external.test',
              },
            },
          ],
        }),
        queryParameters: {'status': 'request', 'limit': 50, 'cursor': cursor},
      );
      final next = await repository.conversations(
        bucket: ChatBucket.request,
        cursor: page.cursor,
      );
      expect(next.items.single.id.value, 'private-conversation');
      expect(next.items.single.peer.did.value, 'did:plc:external-peer');
      expect(next.items.single.muted, isTrue);
      expect(next.items.single.unreadCount, 7);
      expect(next.cursor, isNull);
      expect('${next.items.single}', isNot(contains('private-conversation')));
      await expectLater(
        repository.conversations(bucket: ChatBucket.accepted, limit: 101),
        throwsArgumentError,
      );
      dio.close(force: true);
    },
  );
}
