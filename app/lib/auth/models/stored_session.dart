import 'package:dart_mappable/dart_mappable.dart';

part 'stored_session.mapper.dart';

/// Single JSON blob we persist to `flutter_secure_storage` under the
/// key `craftsky_session`. `did` and `handle` are cached so cold start
/// can render an optimistic `SignedIn(did, handle)` without waiting
/// for a `/whoami` round-trip — background validation reconciles them.
@MappableClass()
class StoredSession with StoredSessionMappable {
  const StoredSession({
    required this.token,
    required this.did,
    required this.handle,
  });

  final String token;
  final String did;
  final String handle;
}
