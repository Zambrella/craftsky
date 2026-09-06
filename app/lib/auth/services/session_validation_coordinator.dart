import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum SessionValidationResult {
  valid,
  transientFailure,
  unauthorized,
  identityMismatch,
}

typedef SessionValidationLauncher =
    Future<void> Function(
      AccountSessionLease active,
    );

/// Validates only the account currently being used. Retained inactive
/// accounts are validated lazily when activation makes them current.
class AppSessionValidationLauncher {
  AppSessionValidationLauncher(this.ref);

  final Ref ref;

  Future<void> run(AccountSessionLease active) async {
    final result = await _validate(active);
    if (result
        case SessionValidationResult.unauthorized ||
            SessionValidationResult.identityMismatch) {
      await ref.read(accountSessionInvalidatorProvider)(active);
    }
  }

  Future<SessionValidationResult> _validate(
    AccountSessionLease lease,
  ) async {
    try {
      final dio = await ref.read(accountDioProvider(lease.account).future);
      final who = await AuthApiClient(dio).whoami();
      if (who.did != lease.account.did) {
        return SessionValidationResult.identityMismatch;
      }
      await ref
          .read(sessionRegistryProvider.notifier)
          .updateHandle(lease, who.handle);
      return SessionValidationResult.valid;
    } on ApiUnauthorized {
      return SessionValidationResult.unauthorized;
    } on ApiNetworkError {
      return SessionValidationResult.transientFailure;
    } on ApiServerError {
      return SessionValidationResult.transientFailure;
    } on ApiCanceled {
      return SessionValidationResult.transientFailure;
    }
  }
}

final sessionValidationLauncherProvider = Provider<SessionValidationLauncher>(
  (ref) => AppSessionValidationLauncher(ref).run,
);
