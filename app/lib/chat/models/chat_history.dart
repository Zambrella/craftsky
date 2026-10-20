import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:flutter/foundation.dart';

@immutable
final class ChatHistory {
  const new empty() : messages = const [], cursor = null;

  new _(List<ChatMessage> messages, this.cursor)
    : messages = List.unmodifiable(messages);

  final List<ChatMessage> messages;
  final ChatCursor? cursor;

  bool get hasMore => cursor != null;

  ChatHistory merge(ChatMessagePage page) {
    final byId = {for (final message in messages) message.id: message};
    for (final message in page.items) {
      final previous = byId[message.id];
      // Message deletion is permanent. A stale overlapping page must never
      // restore a body removed by a newer tombstone.
      if (previous?.kind == ChatMessageKind.deleted) continue;
      if (previous != null &&
          message.kind != ChatMessageKind.deleted &&
          message.revision.compareTo(previous.revision) < 0) {
        continue;
      }
      byId[message.id] = message;
    }
    final ordered = byId.values.toList()
      ..sort((a, b) {
        final time = a.sentAt.compareTo(b.sentAt);
        return time == 0 ? a.id.value.compareTo(b.id.value) : time;
      });
    return ChatHistory._(ordered, page.cursor);
  }

  @override
  String toString() => 'ChatHistory(<redacted>)';
}
