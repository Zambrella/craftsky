import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

enum AccountDeletionStatus { intent, active, retrying, needsAttention, deleted }

enum AccountDeletionPhase {
  preparing,
  removingPrivateData,
  removingCraftskyRecords,
  waitingForCraftsky,
  finalizing,
  deleted,
}

@immutable
final class DeletionStatusEntry {
  DeletionStatusEntry({
    required this.jobId,
    required String did,
    required String handle,
    required this.statusToken,
    required this.status,
    required this.phase,
    required this.canRetry,
    required this.needsReauthentication,
    this.displayName,
    this.avatarUrl,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle) {
    if (jobId.isEmpty || statusToken.isEmpty) {
      throw const FormatException('Invalid deletion status entry');
    }
  }

  factory DeletionStatusEntry.pending({
    required String jobId,
    required String did,
    required String handle,
    required String statusToken,
    String? displayName,
    String? avatarUrl,
  }) => DeletionStatusEntry(
    jobId: jobId,
    did: did,
    handle: handle,
    statusToken: statusToken,
    displayName: displayName,
    avatarUrl: avatarUrl,
    status: AccountDeletionStatus.intent,
    phase: AccountDeletionPhase.preparing,
    canRetry: false,
    needsReauthentication: false,
  );

  factory DeletionStatusEntry.fromMap(Map<String, Object?> map) =>
      DeletionStatusEntry(
        jobId: _requiredString(map, 'jobId'),
        did: _requiredString(map, 'did'),
        handle: _requiredString(map, 'handle'),
        statusToken: _requiredString(map, 'statusToken'),
        displayName: _optionalString(map, 'displayName'),
        avatarUrl: _optionalString(map, 'avatarUrl'),
        status: AccountDeletionStatus.values.byName(
          _requiredString(map, 'status'),
        ),
        phase: AccountDeletionPhase.values.byName(
          _requiredString(map, 'phase'),
        ),
        canRetry: _requiredBool(map, 'canRetry'),
        needsReauthentication: _requiredBool(
          map,
          'needsReauthentication',
        ),
      );

  final String jobId;
  final Did did;
  final Handle handle;
  final String statusToken;
  final String? displayName;
  final String? avatarUrl;
  final AccountDeletionStatus status;
  final AccountDeletionPhase phase;
  final bool canRetry;
  final bool needsReauthentication;

  String get displayLabel {
    final candidate = displayName?.trim();
    return candidate == null || candidate.isEmpty ? handle.value : candidate;
  }

  bool get isTerminal => status == AccountDeletionStatus.deleted;

  DeletionStatusEntry withStatus({
    required AccountDeletionStatus status,
    required AccountDeletionPhase phase,
    bool canRetry = false,
    bool needsReauthentication = false,
  }) => DeletionStatusEntry(
    jobId: jobId,
    did: did.value,
    handle: handle.value,
    statusToken: statusToken,
    displayName: displayName,
    avatarUrl: avatarUrl,
    status: status,
    phase: phase,
    canRetry: canRetry,
    needsReauthentication: needsReauthentication,
  );

  Map<String, Object?> toMap() => {
    'jobId': jobId,
    'did': did.value,
    'handle': handle.value,
    'statusToken': statusToken,
    'displayName': displayName,
    'avatarUrl': avatarUrl,
    'status': status.name,
    'phase': phase.name,
    'canRetry': canRetry,
    'needsReauthentication': needsReauthentication,
  };

  @override
  bool operator ==(Object other) =>
      other is DeletionStatusEntry &&
      other.jobId == jobId &&
      other.did == did &&
      other.handle == handle &&
      other.statusToken == statusToken &&
      other.displayName == displayName &&
      other.avatarUrl == avatarUrl &&
      other.status == status &&
      other.phase == phase &&
      other.canRetry == canRetry &&
      other.needsReauthentication == needsReauthentication;

  @override
  int get hashCode => Object.hash(
    jobId,
    did,
    handle,
    statusToken,
    displayName,
    avatarUrl,
    status,
    phase,
    canRetry,
    needsReauthentication,
  );

  @override
  String toString() => 'DeletionStatusEntry(<redacted>)';
}

@immutable
final class DeletionStatusRegistry {
  DeletionStatusRegistry._(Map<String, DeletionStatusEntry> entries)
    : _byJobId = Map.unmodifiable(entries);

  factory DeletionStatusRegistry.empty() => DeletionStatusRegistry._(const {});

  factory DeletionStatusRegistry.fromJson(String source) {
    final decoded = jsonDecode(source);
    if (decoded is! Map<String, Object?> || decoded['schemaVersion'] != 1) {
      throw const FormatException('Invalid deletion status registry');
    }
    final rawEntries = decoded['entries'];
    if (rawEntries is! List<Object?>) {
      throw const FormatException('Invalid deletion status entries');
    }
    final entries = <String, DeletionStatusEntry>{};
    for (final raw in rawEntries) {
      if (raw is! Map<String, Object?>) {
        throw const FormatException('Invalid deletion status entry');
      }
      final entry = DeletionStatusEntry.fromMap(raw);
      if (entries.containsKey(entry.jobId)) {
        throw const FormatException('Duplicate deletion status entry');
      }
      entries[entry.jobId] = entry;
    }
    return DeletionStatusRegistry._(entries);
  }

  final Map<String, DeletionStatusEntry> _byJobId;

  List<DeletionStatusEntry> get entries => List.unmodifiable(_byJobId.values);

  DeletionStatusEntry? operator [](String jobId) => _byJobId[jobId];

  DeletionStatusEntry? forAccount(AccountKey account) {
    for (final entry in _byJobId.values) {
      if (entry.did == account.did && !entry.isTerminal) return entry;
    }
    return null;
  }

  DeletionStatusRegistry upsert(DeletionStatusEntry entry) =>
      DeletionStatusRegistry._({..._byJobId, entry.jobId: entry});

  DeletionStatusRegistry remove(String jobId) {
    if (!_byJobId.containsKey(jobId)) return this;
    return DeletionStatusRegistry._({..._byJobId}..remove(jobId));
  }

  String toJson() => jsonEncode({
    'schemaVersion': 1,
    'entries': [for (final entry in entries) entry.toMap()],
  });

  @override
  String toString() => 'DeletionStatusRegistry(<redacted>)';
}

@immutable
final class LocalDeletionTransition {
  const LocalDeletionTransition._({
    required this.sessions,
    required this.statusRegistry,
  });

  factory LocalDeletionTransition.accept({
    required SessionRegistry sessions,
    required DeletionStatusRegistry statusRegistry,
    required AccountSessionLease deletingLease,
    required DeletionStatusEntry entry,
  }) {
    if (sessions.leaseFor(deletingLease.account) != deletingLease ||
        deletingLease.account.did != entry.did) {
      throw StateError('Account deletion lease unavailable');
    }
    return LocalDeletionTransition._(
      sessions: sessions.remove(entry.did.value),
      statusRegistry: statusRegistry.upsert(entry),
    );
  }

  final SessionRegistry sessions;
  final DeletionStatusRegistry statusRegistry;

  bool get statusIsPrimary =>
      sessions.activeDid == null && statusRegistry.entries.isNotEmpty;
}

@immutable
final class AccountDeletionLeaseFence {
  const AccountDeletionLeaseFence._(this.lease);

  factory AccountDeletionLeaseFence.capture(SessionRegistry registry) {
    final active = registry.activeLease;
    if (active == null) throw StateError('No active account');
    return AccountDeletionLeaseFence._(active);
  }

  final ActiveAccountLease lease;

  AccountKey get account => lease.session.account;

  bool isCurrent(SessionRegistry registry) => registry.isCurrent(lease);

  @override
  String toString() => 'AccountDeletionLeaseFence(<redacted>)';
}

enum DeletionLoginDestination {
  ordinaryAuthentication,
  status,
  freshOnboarding,
}

abstract final class DeletionLoginPolicy {
  static DeletionLoginDestination destination({
    required AccountDeletionStatus? status,
  }) => switch (status) {
    null => DeletionLoginDestination.ordinaryAuthentication,
    AccountDeletionStatus.deleted => DeletionLoginDestination.freshOnboarding,
    _ => DeletionLoginDestination.status,
  };
}

String _requiredString(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is! String || value.isEmpty) throw FormatException('Invalid $key');
  return value;
}

String? _optionalString(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value != null && value is! String) throw FormatException('Invalid $key');
  return value as String?;
}

bool _requiredBool(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is! bool) throw FormatException('Invalid $key');
  return value;
}
