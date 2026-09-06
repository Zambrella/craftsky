import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

/// Minimized local authority retained only until a deletion intent is
/// cancelled or durably accepted. The reauthentication proof is deliberately
/// absent and remains single-use server authority carried by the callback.
@immutable
final class PendingAccountDeletion {
  const PendingAccountDeletion({
    required this.jobId,
    required this.lease,
    required this.confirmationDid,
    required this.expiresAt,
  });

  factory PendingAccountDeletion.capture({
    required String jobId,
    required ActiveAccountLease lease,
    required String confirmationDid,
    required DateTime expiresAt,
  }) {
    final normalizedJobId = jobId.trim();
    final parsedConfirmationDid = Did.parse(confirmationDid);
    if (normalizedJobId.isEmpty ||
        parsedConfirmationDid != lease.session.account.did) {
      throw const FormatException('Invalid pending account deletion');
    }
    return PendingAccountDeletion(
      jobId: normalizedJobId,
      lease: lease,
      confirmationDid: parsedConfirmationDid.value,
      expiresAt: expiresAt.toUtc(),
    );
  }

  factory PendingAccountDeletion.fromMap(Map<String, Object?> map) {
    final jobId = _requiredString(map, 'jobId');
    final did = _requiredString(map, 'did');
    final confirmationDid = Did.parse(_requiredString(map, 'confirmationDid'));
    final sessionGeneration = _requiredPositiveInt(map, 'sessionGeneration');
    final activationGeneration = _requiredNonNegativeInt(
      map,
      'activationGeneration',
    );
    final expiresAt = DateTime.parse(
      _requiredString(map, 'expiresAt'),
    ).toUtc();
    if (jobId.trim().isEmpty || confirmationDid != Did.parse(did)) {
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
      confirmationDid: confirmationDid.value,
      expiresAt: expiresAt,
    );
  }

  final String jobId;
  final ActiveAccountLease lease;
  final String confirmationDid;
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
    'confirmationDid': confirmationDid,
    'expiresAt': expiresAt.toIso8601String(),
  };

  bool sameAs(PendingAccountDeletion other) =>
      jobId == other.jobId &&
      lease == other.lease &&
      confirmationDid == other.confirmationDid &&
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
