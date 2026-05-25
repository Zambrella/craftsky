import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'pending_auth.mapper.dart';

/// Records that a sign-in flow is in progress. `startedAt` is used by
/// `AuthController.completeFromDeepLink` to reject stale deep links
/// (older than 10 minutes).
@MappableClass(includeCustomMappers: [HandleMapper()])
class PendingAuth with PendingAuthMappable {
  PendingAuth({required String handle, required this.startedAt})
    : handle = Handle.parse(handle);

  final Handle handle;
  final DateTime startedAt;
}
