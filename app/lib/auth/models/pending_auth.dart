import 'package:dart_mappable/dart_mappable.dart';

part 'pending_auth.mapper.dart';

/// Records that a sign-in flow is in progress. `startedAt` is used by
/// `AuthController.completeFromDeepLink` to reject stale deep links
/// (older than 10 minutes).
@MappableClass()
class PendingAuth with PendingAuthMappable {
  const PendingAuth({required this.handle, required this.startedAt});

  final String handle;
  final DateTime startedAt;
}
