import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// One retained account entry inside the secure session-registry snapshot.
class StoredSession {
  StoredSession({
    required this.token,
    required String did,
    required String handle,
    this.sessionGeneration = 0,
    this.lastUsedOrdinal = 0,
    this.cachedDisplayName,
    this.cachedAvatarUrl,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final String token;
  final Did did;
  final Handle handle;
  final int sessionGeneration;
  final int lastUsedOrdinal;
  final String? cachedDisplayName;
  final String? cachedAvatarUrl;

  /// Never include the token in string form — the default mappable
  /// `toString` prints every field, which would land bearer tokens in
  /// any accidental `_log.fine('$session')` call site.
  @override
  String toString() => 'StoredSession(<redacted>)';
}
