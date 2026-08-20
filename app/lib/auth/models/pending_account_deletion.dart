import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:flutter/foundation.dart';

/// Minimized local authority retained only until a deletion intent is
/// cancelled or durably accepted. The reauthentication proof is deliberately
/// absent and remains single-use server authority carried by the callback.
@immutable
final class PendingAccountDeletion {
  const PendingAccountDeletion({
    required this.jobId,
    required this.lease,
    required this.requiredHandle,
    required this.expiresAt,
  });

  factory PendingAccountDeletion.capture({
    required String jobId,
    required ActiveAccountLease lease,
    required String handle,
    required DateTime expiresAt,
  }) {
    final normalizedJobId = jobId.trim();
    final normalizedHandle = handle.trim().replaceFirst(RegExp('^@'), '');
    if (normalizedJobId.isEmpty || normalizedHandle.isEmpty) {
      throw const FormatException('Invalid pending account deletion');
    }
    return PendingAccountDeletion(
      jobId: normalizedJobId,
      lease: lease,
      requiredHandle: '@$normalizedHandle',
      expiresAt: expiresAt.toUtc(),
    );
  }

  factory PendingAccountDeletion.fromMap(Map<String, Object?> map) {
    final jobId = _requiredString(map, 'jobId');
    final did = _requiredString(map, 'did');
    final requiredHandle = _requiredString(map, 'requiredHandle');
    final sessionGeneration = _requiredPositiveInt(map, 'sessionGeneration');
    final activationGeneration = _requiredNonNegativeInt(
      map,
      'activationGeneration',
    );
    final expiresAt = DateTime.parse(
      _requiredString(map, 'expiresAt'),
    ).toUtc();
    if (jobId.trim().isEmpty ||
        requiredHandle.length < 2 ||
        !requiredHandle.startsWith('@')) {
      throw const FormatException('Invalid pending account deletion');
    }
    return PendingAccountDeletion(
      jobId: jobId,
      lease: ActiveAccountLease(
        session: AccountSessionLease(
          account: AccountKey(did),
          sessionGeneration: sessionGeneration,
        ),
        activationGeneration: activationGeneration,
      ),
      requiredHandle: requiredHandle,
      expiresAt: expiresAt,
    );
  }

  final String jobId;
  final ActiveAccountLease lease;
  final String requiredHandle;
  final DateTime expiresAt;

  bool isCurrent(ActiveAccountLease? current, {DateTime? now}) =>
      current == lease && (now ?? DateTime.now().toUtc()).isBefore(expiresAt);

  bool protects(AccountSessionLease candidate, {DateTime? now}) =>
      lease.session == candidate &&
      (now ?? DateTime.now().toUtc()).isBefore(expiresAt);

  Map<String, Object?> toMap() => {
    'jobId': jobId,
    'did': lease.session.account.did.value,
    'sessionGeneration': lease.session.sessionGeneration,
    'activationGeneration': lease.activationGeneration,
    'requiredHandle': requiredHandle,
    'expiresAt': expiresAt.toIso8601String(),
  };

  bool sameAs(PendingAccountDeletion other) =>
      jobId == other.jobId &&
      lease == other.lease &&
      requiredHandle == other.requiredHandle &&
      expiresAt == other.expiresAt;

  @override
  String toString() => 'PendingAccountDeletion(<redacted>)';
}

String _requiredString(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is! String) throw FormatException('Invalid $key');
  return value;
}

int _requiredPositiveInt(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is! int || value < 1) throw FormatException('Invalid $key');
  return value;
}

int _requiredNonNegativeInt(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is! int || value < 0) throw FormatException('Invalid $key');
  return value;
}
