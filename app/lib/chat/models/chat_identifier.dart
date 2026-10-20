import 'dart:convert';

import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

@immutable
sealed class ChatIdentifier {
  const new _(this.value);

  final String value;

  @override
  bool operator ==(Object other) =>
      other is ChatIdentifier &&
      other.runtimeType == runtimeType &&
      other.value == value;

  @override
  int get hashCode => Object.hash(runtimeType, value);

  @override
  String toString() => 'ChatIdentifier(<redacted>)';
}

final class ChatMessageId extends ChatIdentifier {
  const new _(super.value) : super._();

  factory parse(String raw) => ChatMessageId._(_opaqueIdentifier(raw));
}

final class ChatConversationId extends ChatIdentifier {
  const new _(super.value) : super._();

  factory parse(String raw) => ChatConversationId._(_opaqueIdentifier(raw));
}

final class ChatCursor extends ChatIdentifier {
  const new _(super.value) : super._();

  factory parse(String raw) {
    if (!RegExp(
          r'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$',
          caseSensitive: false,
        ).hasMatch(raw) ||
        raw == '00000000-0000-0000-0000-000000000000') {
      throw const FormatException('Invalid messaging continuation');
    }
    return ChatCursor._(raw.toLowerCase());
  }
}

final class ChatOperationId extends ChatIdentifier {
  const new _(super.value) : super._();

  factory parse(String raw) => ChatOperationId._(ChatCursor.parse(raw).value);
}

final class ChatOpenReference extends ChatIdentifier {
  const new _(super.value) : super._();

  factory parse(String raw) => ChatOpenReference._(ChatCursor.parse(raw).value);
}

String _opaqueIdentifier(String value) {
  if (value.isEmpty ||
      utf8.encode(value).length > 512 ||
      RegExp(r'[/\\?#%\x00-\x20\x7f]').hasMatch(value)) {
    throw const FormatException('Invalid messaging identifier');
  }
  final units = value.codeUnits;
  for (var i = 0; i < units.length; i++) {
    final unit = units[i];
    if (unit >= 0xd800 && unit <= 0xdbff) {
      if (++i >= units.length || units[i] < 0xdc00 || units[i] > 0xdfff) {
        throw const FormatException('Invalid messaging identifier');
      }
    } else if (unit >= 0xdc00 && unit <= 0xdfff) {
      throw const FormatException('Invalid messaging identifier');
    }
  }
  return value;
}

/// Already parsed public identity used in a private messaging context.
final class ChatRecipientId extends ChatIdentifier {
  new _(this.did) : super._(did.value);
  factory fromDid(Did did) => ChatRecipientId._(did);
  final Did did;
}
