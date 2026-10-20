import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/services/chat_inbox_controller.dart';
import 'package:flutter_test/flutter_test.dart';

// IT-001 / FR-002, FR-008: independent shared inbox/request pagination.
void main() {
  test(
    'independent buckets preserve empty continuations and merge stable IDs',
    () async {
      final repository = _Repository();
      final guard = ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      );
      final account = AccountKey('did:plc:owner');
      final inbox = ChatInboxController(
        account: account,
        bucket: ChatBucket.accepted,
        repository: repository,
        operation: guard,
      );
      final requests = ChatInboxController(
        account: account,
        bucket: ChatBucket.request,
        repository: repository,
        operation: guard,
      );
      addTearDown(inbox.dispose);
      addTearDown(requests.dispose);
      await inbox.refresh();
      await requests.refresh();
      expect(inbox.items.single.id.value, 'external-conversation');
      expect(requests.items.single.bucket, ChatBucket.request);
      final more = inbox.loadMore();
      await inbox.loadMore();
      expect(repository.acceptedCalls, 2);
      repository.emptyPage.complete(
        ChatConversationPage.fromMap(const {
          'items': <ChatConversation>[],
          'cursor': '00000000-0000-4000-8000-000000000456',
        }),
      );
      await more;
      expect(inbox.items, hasLength(1));
      expect(inbox.hasMore, isTrue);
      await inbox.loadMore();
      expect(inbox.items.map((item) => item.id.value), [
        'external-conversation',
        'another-conversation',
      ]);
      expect(inbox.items.first.muted, isTrue);
      expect(inbox.hasMore, isFalse);
      expect(requests.items.single.id.value, 'external-request');
      expect(repository.requestCalls, 1);
    },
  );
}

class _Repository implements ChatRepository {
  int acceptedCalls = 0;
  int requestCalls = 0;
  final emptyPage = Completer<ChatConversationPage>();
  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    if (bucket == ChatBucket.request) {
      requestCalls++;
      return ChatConversationPage.fromMap({
        'items': [_item('external-request', bucket)],
      });
    }
    acceptedCalls++;
    if (acceptedCalls == 2) return await emptyPage.future;
    return ChatConversationPage.fromMap({
      'items': [
        _item('external-conversation', bucket, muted: acceptedCalls > 1),
        if (acceptedCalls > 1) _item('another-conversation', bucket),
      ],
      if (acceptedCalls == 1) 'cursor': '00000000-0000-4000-8000-000000000123',
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

Map<String, dynamic> _item(
  String id,
  ChatBucket bucket, {
  bool muted = false,
}) => {
  'id': id,
  'revision': 'rev-1',
  'status': bucket.name,
  'muted': muted,
  'unreadCount': 1,
  'canSend': bucket == ChatBucket.accepted,
  'peer': {'did': 'did:plc:external', 'handle': 'external.test'},
};
