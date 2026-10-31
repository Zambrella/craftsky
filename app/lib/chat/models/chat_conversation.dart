import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

enum ChatBucket { accepted, request }

@immutable
final class ChatActor {
  const new _({required this.did, this.handle, this.displayName, this.avatar});

  factory fromMap(Map<String, dynamic> map) {
    final rawDid = map['did'];
    if (rawDid is! String) {
      throw const FormatException('Invalid messaging actor');
    }
    String? bounded(String key, int max) {
      final value = map[key];
      return value is String && value.isNotEmpty && value.length <= max
          ? value
          : null;
    }

    return ChatActor._(
      did: Did.parse(rawDid),
      handle: bounded('handle', 253),
      displayName: bounded('displayName', 640),
      avatar: bounded('avatar', 2048),
    );
  }

  final Did did;
  final String? handle;
  final String? displayName;
  final String? avatar;

  String get label => displayName ?? handle ?? 'Someone';

  @override
  String toString() => 'ChatActor(<redacted>)';
}

@immutable
final class ChatConversation {
  const new _({
    required this.id,
    required this.revision,
    required this.bucket,
    required this.muted,
    required this.unreadCount,
    required this.canSend,
    required this.peer,
    this.lastMessage,
  });

  factory fromMap(Map<String, dynamic> map) {
    final id = map['id'];
    final revision = map['revision'];
    final status = map['status'];
    final muted = map['muted'];
    final unread = map['unreadCount'];
    final canSend = map['canSend'];
    final peer = map['peer'];
    if (id is! String ||
        revision is! String ||
        revision.length > 512 ||
        (status != 'accepted' && status != 'request') ||
        muted is! bool ||
        unread is! int ||
        unread < 0 ||
        canSend is! bool ||
        peer is! Map<String, dynamic>) {
      throw const FormatException('Invalid messaging conversation');
    }
    final last = map['lastMessage'];
    ChatMessage? lastMessage;
    if (last is Map<String, dynamic>) {
      try {
        lastMessage = ChatMessage.fromMap(last);
      } on FormatException {
        // Unknown upstream unions may have no usable identity/date. Keep the
        // conversation accessible without inventing a message or
        // retaining raw data.
        if (last['kind'] != 'unsupported') rethrow;
      }
    }
    return ChatConversation._(
      id: ChatConversationId.parse(id),
      revision: revision,
      bucket: status == 'accepted' ? ChatBucket.accepted : ChatBucket.request,
      muted: muted,
      unreadCount: unread,
      canSend: canSend,
      peer: ChatActor.fromMap(peer),
      lastMessage: lastMessage,
    );
  }

  final ChatConversationId id;
  final String revision;
  final ChatBucket bucket;
  final bool muted;
  final int unreadCount;
  final bool canSend;
  final ChatActor peer;
  final ChatMessage? lastMessage;

  @override
  String toString() => 'ChatConversation(<redacted>)';
}

@immutable
final class ChatConversationPage {
  new({required List<ChatConversation> items, this.cursor})
    : items = List.unmodifiable(items);

  factory fromMap(Map<String, dynamic> map) {
    final items = map['items'];
    final cursor = map['cursor'];
    if (items is! List ||
        items.length > 100 ||
        (cursor != null && cursor is! String)) {
      throw const FormatException('Invalid messaging page');
    }
    return ChatConversationPage(
      items: [
        for (final item in items)
          if (item is Map<String, dynamic>)
            ChatConversation.fromMap(item)
          else
            throw const FormatException('Invalid messaging page'),
      ],
      cursor: cursor == null ? null : ChatCursor.parse(cursor as String),
    );
  }

  final List<ChatConversation> items;
  final ChatCursor? cursor;

  @override
  String toString() => 'ChatConversationPage(<redacted>)';
}
