import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// A server-issued session that remains unusable until the client confirms
/// that this complete receipt has been durably stored.
final class PendingHandoff {
  PendingHandoff({
    required this.token,
    required String did,
    required String handle,
    required this.receiptId,
    required this.confirmBy,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle) {
    if (token.isEmpty || receiptId.isEmpty) {
      throw const FormatException('Invalid pending handoff');
    }
  }

  factory PendingHandoff.fromMap(Map<String, dynamic> map) {
    final confirmBy = DateTime.tryParse(_requiredString(map, 'confirmBy'));
    if (confirmBy == null) {
      throw const FormatException('Invalid confirmBy');
    }
    return PendingHandoff(
      token: _requiredString(map, 'token'),
      did: _requiredString(map, 'did'),
      handle: _requiredString(map, 'handle'),
      receiptId: _requiredString(map, 'receiptId'),
      confirmBy: confirmBy.toUtc(),
    );
  }

  final String token;
  final Did did;
  final Handle handle;
  final String receiptId;
  final DateTime confirmBy;

  Map<String, Object?> toMap() => {
    'token': token,
    'did': did.value,
    'handle': handle.value,
    'receiptId': receiptId,
    'confirmBy': confirmBy.toUtc().toIso8601String(),
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
