import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:flutter_test/flutter_test.dart';

// UT-003 / IT-002 / FR-003: stable IDs, chronological merging and permanent
// tombstones; identical text is never a delivery or deduplication identity.
void main() {
  test('overlapping history replaces deleted text without resurrection', () {
    Map<String, dynamic> message(
      String id,
      int minute, {
      String kind = 'text',
      String revision = '0001',
    }) => {
      'id': id,
      'kind': kind,
      'revision': revision,
      'senderDid': 'did:plc:history-peer',
      'text': 'private-history-canary',
      'sentAt': '2026-10-19T12:0$minute:00Z',
    };
    final first = ChatMessagePage.fromMap({
      'items': [message('message-3', 3), message('message-2', 2)],
      'cursor': '00000000-0000-4000-8000-000000000123',
    });
    var history = const ChatHistory.empty().merge(first);
    history = history.merge(
      ChatMessagePage.fromMap({
        'items': [
          message('message-2', 2, kind: 'deleted', revision: '0004'),
          message('message-1', 1),
        ],
        'cursor': '00000000-0000-4000-8000-000000000456',
      }),
    );
    history = history.merge(
      ChatMessagePage.fromMap({
        'items': [
          message('message-2', 2, revision: '0002'),
          message('message-4', 4, kind: 'future.message'),
        ],
      }),
    );
    expect(history.messages.map((item) => item.id.value), [
      'message-1',
      'message-2',
      'message-3',
      'message-4',
    ]);
    expect(history.messages[1].kind, ChatMessageKind.deleted);
    expect(history.messages[1].text, isNull);
    expect(history.messages[2].text, 'private-history-canary');
    expect(history.messages[3].kind, ChatMessageKind.unsupported);
    expect(history.messages[3].text, isNull);
    expect(history.hasMore, isFalse);
    history = history.merge(
      ChatMessagePage.fromMap(const {
        'items': <Map<String, dynamic>>[],
        'cursor': '00000000-0000-4000-8000-000000000789',
      }),
    );
    expect(history.hasMore, isTrue);
    expect(history.messages, hasLength(4));
    for (final value in [history, ...history.messages, first.cursor]) {
      expect('$value', isNot(contains('private-history-canary')));
      expect('$value', isNot(contains('history-peer')));
      expect('$value', isNot(contains('message-')));
    }
  });
}
