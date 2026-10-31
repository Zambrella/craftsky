import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:flutter/foundation.dart';

/// Ephemeral presentation of a local attempt, optionally linked to its receipt.
/// Never inserted into authoritative history or persisted as a message cache.
@immutable
final class ChatOutgoingMessage {
  const new({required this.attempt, this.messageId});

  final ChatDraftAttempt attempt;
  final ChatMessageId? messageId;

  @override
  String toString() => 'ChatOutgoingMessage(<redacted>)';
}
