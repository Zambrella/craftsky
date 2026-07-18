import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_session_cleanup.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';

enum NotificationCleanupResult { complete, alreadyComplete, retryable }

/// Coordinates the security-sensitive ordering for an offline sign-out.
///
/// The local session is quarantined durably before provider state is touched.
/// Registration remains paused until every retained cleanup credential reaches
/// a terminal result and is deleted from secure storage.
final class NotificationSignOutRecovery {
  NotificationSignOutRecovery({
    required this.readRegistry,
    required this.quarantineAndRemove,
    required this.deleteCleanupCredential,
    required this.deleteProviderToken,
    required this.logoutCleanup,
    required this.resumeRegistration,
  });

  final SessionRegistry Function() readRegistry;
  final Future<void> Function(AccountSessionLease lease) quarantineAndRemove;
  final Future<void> Function(PendingSessionCleanup cleanup)
  deleteCleanupCredential;
  final Future<void> Function() deleteProviderToken;
  final Future<NotificationCleanupResult> Function(
    PendingSessionCleanup cleanup,
  )
  logoutCleanup;
  final Future<void> Function() resumeRegistration;

  Future<void> _pendingRetry = Future.value();

  Future<void> begin(AccountSessionLease lease) async {
    await quarantineAndRemove(lease);
    try {
      await deleteProviderToken();
    } on Object {
      // The durable queue keeps registration blocked. Startup/resume retries
      // provider-token invalidation before attempting server cleanup.
    }
  }

  Future<void> retry() {
    final operation = _pendingRetry.then((_) => _retryOnce());
    _pendingRetry = operation.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    return operation;
  }

  Future<void> _retryOnce() async {
    final snapshot = List<PendingSessionCleanup>.of(
      readRegistry().pendingCleanups,
    );
    if (snapshot.isEmpty) {
      await resumeRegistration();
      return;
    }
    try {
      await deleteProviderToken();
    } on Object {
      return;
    }
    for (final cleanup in snapshot) {
      final result = await logoutCleanup(cleanup);
      if (result == NotificationCleanupResult.retryable) return;
      await deleteCleanupCredential(cleanup);
    }
    if (readRegistry().pendingCleanups.isEmpty) await resumeRegistration();
  }
}
