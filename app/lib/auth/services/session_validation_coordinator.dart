import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum SessionValidationResult {
  valid,
  transientFailure,
  unauthorized,
  identityMismatch,
}

class SessionValidationCoordinator {
  SessionValidationCoordinator({
    required this.validate,
    required this.applyResult,
    this.inactiveConcurrency = 2,
  }) : assert(inactiveConcurrency > 0, 'Concurrency must be positive');

  final Future<SessionValidationResult> Function(AccountSessionLease lease)
  validate;
  final Future<void> Function(
    AccountSessionLease lease,
    SessionValidationResult result,
  )
  applyResult;
  final int inactiveConcurrency;

  Future<void> run({
    required AccountSessionLease active,
    required List<AccountSessionLease> inactive,
  }) async {
    await _validateAndApply(active);

    var nextIndex = 0;
    Future<void> worker() async {
      while (nextIndex < inactive.length) {
        final target = inactive[nextIndex++];
        await _validateAndApply(target);
      }
    }

    await Future.wait([
      for (
        var index = 0;
        index < inactiveConcurrency && index < inactive.length;
        index++
      )
        worker(),
    ]);
  }

  Future<void> _validateAndApply(AccountSessionLease lease) async {
    final result = await validate(lease);
    await applyResult(lease, result);
  }
}

typedef SessionValidationLauncher =
    Future<void> Function(
      SessionRegistry snapshot,
    );

class AppSessionValidationLauncher {
  AppSessionValidationLauncher(this.ref);

  final Ref ref;

  Future<void> run(SessionRegistry snapshot) async {
    final active = snapshot.activeLease?.session;
    if (active == null) return;
    final inactive = snapshot.orderedSessions
        .where((session) => session.did != snapshot.activeDid)
        .map((session) => snapshot.leaseFor(AccountKey(session.did.value))!)
        .toList(growable: false);
    final coordinator = SessionValidationCoordinator(
      validate: _validate,
      applyResult: _apply,
    );
    await coordinator.run(active: active, inactive: inactive);
  }

  Future<SessionValidationResult> _validate(AccountSessionLease lease) async {
    try {
      final dio = await ref.read(accountDioProvider(lease.account).future);
      final who = await AuthApiClient(dio).whoami();
      return who.did == lease.account.did
          ? SessionValidationResult.valid
          : SessionValidationResult.identityMismatch;
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

  Future<void> _apply(
    AccountSessionLease lease,
    SessionValidationResult result,
  ) async {
    if (result
        case SessionValidationResult.unauthorized ||
            SessionValidationResult.identityMismatch) {
      await ref.read(accountSessionInvalidatorProvider)(lease);
    }
  }
}

final sessionValidationLauncherProvider = Provider<SessionValidationLauncher>(
  (ref) => AppSessionValidationLauncher(ref).run,
);

/// Prevents active-account MRU changes from relaunching the startup validation
/// sweep. A new or refreshed session has a new generation and remains eligible
/// for validation, while activation and cached-identity writes do not.
final Provider<SessionValidationLaunchGuard>
sessionValidationLaunchGuardProvider = Provider(
  (ref) => SessionValidationLaunchGuard(),
);

final class SessionValidationLaunchGuard {
  Map<AccountKey, int>? _lastOwnership;

  bool shouldLaunch(SessionRegistry registry) {
    final ownership = {
      for (final session in registry.sessions.values)
        AccountKey(session.did.value): session.sessionGeneration,
    };
    final previous = _lastOwnership;
    if (previous != null && _sameOwnership(previous, ownership)) return false;
    _lastOwnership = Map.unmodifiable(ownership);
    return true;
  }

  bool _sameOwnership(
    Map<AccountKey, int> left,
    Map<AccountKey, int> right,
  ) {
    if (left.length != right.length) return false;
    for (final entry in left.entries) {
      if (right[entry.key] != entry.value) return false;
    }
    return true;
  }

  @override
  String toString() => 'SessionValidationLaunchGuard(<redacted>)';
}
