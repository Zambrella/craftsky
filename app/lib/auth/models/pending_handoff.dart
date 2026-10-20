import 'dart:convert';

import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:crypto/crypto.dart';

enum HandoffPurpose { login, chat }

final class ChatHandoffSourceUnavailable
    implements Exception, AccountOperationCancellation {
  const new();
  @override
  String toString() => 'Messaging consent source session unavailable';
}

/// A server-issued session that remains unusable until the client confirms
/// that this complete receipt has been durably stored.
final class PendingHandoff {
  new({
    required this.token,
    required String did,
    required String handle,
    required this.receiptId,
    required this.confirmBy,
    this.purpose = HandoffPurpose.login,
    this.sourceTokenHash,
    this.sourceSessionGeneration,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle) {
    if (token.isEmpty || receiptId.isEmpty) {
      throw const FormatException('Invalid pending handoff');
    }
    if (purpose == HandoffPurpose.chat &&
        (sourceTokenHash == null ||
            !RegExp(r'^[A-Za-z0-9_-]{43}$').hasMatch(sourceTokenHash!) ||
            base64Url.decode('$sourceTokenHash=').length != 32 ||
            (sourceSessionGeneration != null &&
                sourceSessionGeneration! < 1))) {
      throw const FormatException('Invalid chat handoff binding');
    }
    if (purpose == HandoffPurpose.login &&
        (sourceTokenHash != null || sourceSessionGeneration != null)) {
      throw const FormatException('Unexpected chat handoff binding');
    }
  }

  factory fromMap(Map<String, dynamic> map) {
    final confirmBy = DateTime.tryParse(_requiredString(map, 'confirmBy'));
    if (confirmBy == null) {
      throw const FormatException('Invalid confirmBy');
    }
    final purpose = map['purpose'];
    if (purpose != null && purpose != 'login' && purpose != 'chat') {
      throw const FormatException('Invalid handoff purpose');
    }
    final sourceHash = map['sourceTokenHash'];
    final sourceGeneration = map['sourceSessionGeneration'];
    if ((sourceHash != null && sourceHash is! String) ||
        (sourceGeneration != null && sourceGeneration is! int)) {
      throw const FormatException('Invalid chat handoff binding');
    }
    return PendingHandoff(
      token: _requiredString(map, 'token'),
      did: _requiredString(map, 'did'),
      handle: _requiredString(map, 'handle'),
      receiptId: _requiredString(map, 'receiptId'),
      confirmBy: confirmBy.toUtc(),
      purpose: purpose == 'chat' ? HandoffPurpose.chat : HandoffPurpose.login,
      sourceTokenHash: sourceHash as String?,
      sourceSessionGeneration: sourceGeneration as int?,
    );
  }

  final String token;
  final Did did;
  final Handle handle;
  final String receiptId;
  final DateTime confirmBy;
  final HandoffPurpose purpose;
  final String? sourceTokenHash;
  final int? sourceSessionGeneration;

  bool matchesSourceToken(String token) =>
      sourceTokenHash ==
      base64Url
          .encode(sha256.convert(utf8.encode(token)).bytes)
          .replaceAll('=', '');

  PendingHandoff bindSource(int generation) => PendingHandoff(
    token: token,
    did: did.value,
    handle: handle.value,
    receiptId: receiptId,
    confirmBy: confirmBy,
    purpose: purpose,
    sourceTokenHash: sourceTokenHash,
    sourceSessionGeneration: generation,
  );

  Map<String, Object?> toMap() => {
    'token': token,
    'did': did.value,
    'handle': handle.value,
    'receiptId': receiptId,
    'confirmBy': confirmBy.toUtc().toIso8601String(),
    if (purpose == HandoffPurpose.chat) 'purpose': purpose.name,
    'sourceTokenHash': ?sourceTokenHash,
    'sourceSessionGeneration': ?sourceSessionGeneration,
  };

  @override
  String toString() => 'PendingHandoff(<redacted>)';

  static String _requiredString(Map<String, dynamic> map, String key) {
    final value = map[key];
    if (value is! String || value.isEmpty) {
      throw FormatException('Invalid $key');
    }
    return value;
  }
}
