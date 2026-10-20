import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

enum ChatMessageKind { text, deleted, unsupported }

@immutable
final class ChatMessage {
  const new _({
    required this.id,
    required this.kind,
    required this.revision,
    required this.sentAt,
    this.senderDid,
    this.text,
  });

  factory fromMap(Map<String, dynamic> map) {
    final rawId = map['id'];
    final revision = map['revision'];
    final rawDate = map['sentAt'];
    if (rawId is! String || revision is! String || rawDate is! String) {
      throw const FormatException('Invalid messaging response');
    }
    final date = DateTime.tryParse(rawDate);
    if (date == null || !date.isUtc || revision.length > 512) {
      throw const FormatException('Invalid messaging response');
    }
    final id = ChatMessageId.parse(rawId);
    var kind = ChatMessageKind.unsupported;
    String? text;
    Did? sender;
    if (map['kind'] == 'deleted') {
      kind = ChatMessageKind.deleted;
    } else if (map['kind'] == 'text') {
      final rawText = map['text'];
      final rawSender = map['senderDid'];
      if (rawText is String &&
          rawSender is String &&
          validateMessageText(rawText) == null) {
        sender = Did.parse(rawSender);
        text = rawText;
        kind = ChatMessageKind.text;
      }
    }
    return ChatMessage._(
      id: id,
      kind: kind,
      revision: revision,
      sentAt: date.toUtc(),
      senderDid: sender,
      text: text,
    );
  }

  final ChatMessageId id;
  final ChatMessageKind kind;
  final String revision;
  final DateTime sentAt;
  final Did? senderDid;
  final String? text;

  @override
  String toString() => 'ChatMessage(<redacted>)';
}

@immutable
final class ChatMessagePage {
  new({required List<ChatMessage> items, this.cursor})
    : items = List.unmodifiable(items);

  factory fromMap(Map<String, dynamic> map) {
    final rawItems = map['items'];
    final rawCursor = map['cursor'];
    if (rawItems is! List ||
        rawItems.length > 100 ||
        (rawCursor != null && rawCursor is! String)) {
      throw const FormatException('Invalid messaging response');
    }
    return ChatMessagePage(
      items: [
        for (final item in rawItems)
          if (item is Map<String, dynamic>)
            ChatMessage.fromMap(item)
          else
            throw const FormatException('Invalid messaging response'),
      ],
      cursor: rawCursor == null ? null : ChatCursor.parse(rawCursor as String),
    );
  }

  final List<ChatMessage> items;
  final ChatCursor? cursor;

  @override
  String toString() => 'ChatMessagePage(<redacted>)';
}
