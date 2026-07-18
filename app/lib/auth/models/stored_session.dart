import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'stored_session.mapper.dart';

/// Single JSON blob we persist to `flutter_secure_storage` under the
/// key `craftsky_session`. `did` and `handle` are cached so cold start
/// can render an optimistic `SignedIn(did, handle)` without waiting
/// for a `/whoami` round-trip — background validation reconciles them.
@MappableClass(includeCustomMappers: [DidMapper(), HandleMapper()])
class StoredSession with StoredSessionMappable {
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
