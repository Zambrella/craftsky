import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'pending_auth.mapper.dart';

/// Records that a sign-in flow is in progress.
///
/// `startedAt` is a UI/diagnostic hint only. The server clock is authoritative
/// for exchange-code expiry.
@MappableClass(includeCustomMappers: [HandleMapper()])
class PendingAuth with PendingAuthMappable {
  PendingAuth({required String handle, required this.startedAt})
    : handle = Handle.parse(handle);

  final Handle handle;
  final DateTime startedAt;
}
