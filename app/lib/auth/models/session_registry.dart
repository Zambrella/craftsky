import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

class AccountLimitReached implements Exception {
  const AccountLimitReached();

  @override
  String toString() => 'AccountLimitReached';
}

/// The complete account state retained by this install.
///
/// The codec is intentionally strict: a malformed snapshot is unusable as a
/// whole, so secure-storage callers fail closed to a signed-out registry.
class SessionRegistry {
  SessionRegistry({
    required this.nextSessionGeneration,
    required this.nextUseOrdinal,
    required this.activationGeneration,
    required String? activeDid,
    required Map<String, StoredSession> sessions,
    Map<String, String> routingBindings = const {},
  }) : activeDid = activeDid == null ? null : Did.parse(activeDid),
       sessions = Map.unmodifiable({
         for (final MapEntry(key: did, value: session) in sessions.entries)
           Did.parse(did): session,
       }),
       routingBindings = Map.unmodifiable({
         for (final MapEntry(key: did, value: binding)
             in routingBindings.entries)
           Did.parse(did): binding,
       });

  factory SessionRegistry.empty() => SessionRegistry(
    nextSessionGeneration: 1,
    nextUseOrdinal: 1,
    activationGeneration: 0,
    activeDid: null,
    sessions: const {},
  );

  factory SessionRegistry.fromJson(String source) {
    final decoded = jsonDecode(source);
    if (decoded is! Map<String, Object?>) {
      throw const FormatException('Invalid session registry');
    }
    if (decoded['schemaVersion'] != currentSchemaVersion) {
      throw const FormatException('Unsupported session registry version');
    }

    final rawSessions = decoded['sessions'];
    if (rawSessions is! Map<String, Object?>) {
      throw const FormatException('Invalid session registry sessions');
    }
    final sessions = <String, StoredSession>{};
    for (final MapEntry(key: did, :value) in rawSessions.entries) {
      if (value is! Map<String, Object?> || value['did'] != did) {
        throw const FormatException('Invalid session registry entry');
      }
      sessions[did] = StoredSession(
        token: _requiredString(value, 'token'),
        did: did,
        handle: _requiredString(value, 'handle'),
        sessionGeneration: _requiredPositiveInt(value, 'sessionGeneration'),
        lastUsedOrdinal: _requiredPositiveInt(value, 'lastUsedOrdinal'),
        cachedDisplayName: _optionalString(value, 'cachedDisplayName'),
        cachedAvatarUrl: _optionalString(value, 'cachedAvatarUrl'),
      );
    }

    final routingBindings = <String, String>{};
    final rawRoutingBindings = decoded['routingBindings'];
    if (rawRoutingBindings != null) {
      if (rawRoutingBindings is! Map<String, Object?>) {
        throw const FormatException('Invalid routing bindings');
      }
      for (final MapEntry(key: did, :value) in rawRoutingBindings.entries) {
        Did.parse(did);
        if (value is! String) {
          throw const FormatException('Invalid routing binding');
        }
        routingBindings[did] = value;
      }
    }

    final rawActiveDid = decoded['activeDid'];
    final String? activeDid;
    if (rawActiveDid == null && sessions.isEmpty) {
      activeDid = null;
    } else if (rawActiveDid is String && sessions.containsKey(rawActiveDid)) {
      activeDid = rawActiveDid;
    } else {
      throw const FormatException('Invalid active session');
    }

    final nextSessionGeneration = _requiredPositiveInt(
      decoded,
      'nextSessionGeneration',
    );
    final nextUseOrdinal = _requiredPositiveInt(decoded, 'nextUseOrdinal');
    for (final session in sessions.values) {
      if (session.sessionGeneration >= nextSessionGeneration ||
          session.lastUsedOrdinal >= nextUseOrdinal) {
        throw const FormatException('Invalid registry counters');
      }
    }

    return SessionRegistry(
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: _requiredNonNegativeInt(
        decoded,
        'activationGeneration',
      ),
      activeDid: activeDid,
      sessions: sessions,
      routingBindings: routingBindings,
    );
  }

  static const currentSchemaVersion = 1;
  static const maxRetainedAccounts = 5;
  static const _unchanged = Object();

  final int nextSessionGeneration;
  final int nextUseOrdinal;
  final int activationGeneration;
  final Did? activeDid;
  final Map<Did, StoredSession> sessions;
  final Map<Did, String> routingBindings;

  List<StoredSession> get orderedSessions {
    final ordered = sessions.values.toList()
      ..sort((left, right) {
        final leftIsActive = left.did == activeDid;
        final rightIsActive = right.did == activeDid;
        if (leftIsActive != rightIsActive) return leftIsActive ? -1 : 1;
        final ordinalComparison = right.lastUsedOrdinal.compareTo(
          left.lastUsedOrdinal,
        );
        return ordinalComparison != 0
            ? ordinalComparison
            : left.did.value.compareTo(right.did.value);
      });
    return List.unmodifiable(ordered);
  }

  ActiveAccountLease? get activeLease {
    final did = activeDid;
    if (did == null) return null;
    final lease = leaseFor(AccountKey(did.value));
    return lease == null
        ? null
        : ActiveAccountLease(
            session: lease,
            activationGeneration: activationGeneration,
          );
  }

  AccountSessionLease? leaseFor(AccountKey account) {
    final session = sessions[account.did];
    return session == null
        ? null
        : AccountSessionLease(
            account: account,
            sessionGeneration: session.sessionGeneration,
          );
  }

  bool isCurrent(ActiveAccountLease? lease) =>
      lease != null && lease == activeLease;

  SessionRegistry upsertAndActivate({
    required String token,
    required String did,
    required String handle,
    String? cachedDisplayName,
    String? cachedAvatarUrl,
  }) {
    final parsedDid = Did.parse(did);
    if (!sessions.containsKey(parsedDid) &&
        sessions.length >= maxRetainedAccounts) {
      throw const AccountLimitReached();
    }
    return _copyWith(
      nextSessionGeneration: nextSessionGeneration + 1,
      nextUseOrdinal: nextUseOrdinal + 1,
      activationGeneration: activationGeneration + 1,
      activeDid: parsedDid,
      sessions: {
        ...sessions,
        parsedDid: StoredSession(
          token: token,
          did: did,
          handle: handle,
          sessionGeneration: nextSessionGeneration,
          lastUsedOrdinal: nextUseOrdinal,
          cachedDisplayName: cachedDisplayName,
          cachedAvatarUrl: cachedAvatarUrl,
        ),
      },
    );
  }

  SessionRegistry activate(AccountSessionLease target) {
    final current = sessions[target.account.did];
    if (current == null ||
        current.sessionGeneration != target.sessionGeneration) {
      throw StateError('Account session unavailable');
    }
    if (activeDid == target.account.did) return this;

    return _copyWith(
      nextUseOrdinal: nextUseOrdinal + 1,
      activationGeneration: activationGeneration + 1,
      activeDid: target.account.did,
      sessions: {
        ...sessions,
        target.account.did: StoredSession(
          token: current.token,
          did: current.did.value,
          handle: current.handle.value,
          sessionGeneration: current.sessionGeneration,
          lastUsedOrdinal: nextUseOrdinal,
          cachedDisplayName: current.cachedDisplayName,
          cachedAvatarUrl: current.cachedAvatarUrl,
        ),
      },
    );
  }

  SessionRegistry remove(String did) {
    final parsedDid = Did.parse(did);
    if (!sessions.containsKey(parsedDid)) return this;
    final remaining = {...sessions}..remove(parsedDid);
    final removedActive = parsedDid == activeDid;
    final bindings = {...routingBindings}..remove(parsedDid);
    return _copyWith(
      activationGeneration: removedActive
          ? activationGeneration + 1
          : activationGeneration,
      activeDid: removedActive ? _mostRecentlyUsedDid(remaining) : activeDid,
      sessions: remaining,
      routingBindings: bindings,
    );
  }

  SessionRegistry saveRoutingBinding(
    AccountSessionLease lease,
    String binding,
  ) {
    if (leaseFor(lease.account) != lease) {
      throw StateError('Account session unavailable');
    }
    return _copyWith(
      routingBindings: {...routingBindings, lease.account.did: binding},
    );
  }

  SessionRegistry removeRoutingBinding(AccountSessionLease lease) {
    if (leaseFor(lease.account) != lease ||
        !routingBindings.containsKey(lease.account.did)) {
      return this;
    }
    return _copyWith(
      routingBindings: {...routingBindings}..remove(lease.account.did),
    );
  }

  SessionRegistry updateCachedIdentity(
    AccountSessionLease lease, {
    required String? displayName,
    required String? avatarUrl,
  }) {
    final stored = sessions[lease.account.did];
    if (stored == null || stored.sessionGeneration != lease.sessionGeneration) {
      return this;
    }
    if (stored.cachedDisplayName == displayName &&
        stored.cachedAvatarUrl == avatarUrl) {
      return this;
    }
    return _copyWith(
      sessions: {
        ...sessions,
        lease.account.did: StoredSession(
          token: stored.token,
          did: stored.did.value,
          handle: stored.handle.value,
          sessionGeneration: stored.sessionGeneration,
          lastUsedOrdinal: stored.lastUsedOrdinal,
          cachedDisplayName: displayName,
          cachedAvatarUrl: avatarUrl,
        ),
      },
    );
  }

  String toJson() => jsonEncode({
    'schemaVersion': currentSchemaVersion,
    'nextSessionGeneration': nextSessionGeneration,
    'nextUseOrdinal': nextUseOrdinal,
    'activationGeneration': activationGeneration,
    'activeDid': activeDid,
    'routingBindings': {
      for (final MapEntry(key: did, value: binding) in routingBindings.entries)
        did: binding,
    },
    'sessions': {
      for (final MapEntry(key: did, value: session) in sessions.entries)
        did: {
          'token': session.token,
          'did': session.did,
          'handle': session.handle,
          'sessionGeneration': session.sessionGeneration,
          'lastUsedOrdinal': session.lastUsedOrdinal,
          'cachedDisplayName': session.cachedDisplayName,
          'cachedAvatarUrl': session.cachedAvatarUrl,
        },
    },
  });

  SessionRegistry _copyWith({
    int? nextSessionGeneration,
    int? nextUseOrdinal,
    int? activationGeneration,
    Object? activeDid = _unchanged,
    Map<Did, StoredSession>? sessions,
    Map<Did, String>? routingBindings,
  }) {
    final resolvedActiveDid = identical(activeDid, _unchanged)
        ? this.activeDid
        : activeDid as Did?;
    final resolvedSessions = sessions ?? this.sessions;
    final resolvedBindings = routingBindings ?? this.routingBindings;
    return SessionRegistry(
      nextSessionGeneration:
          nextSessionGeneration ?? this.nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal ?? this.nextUseOrdinal,
      activationGeneration: activationGeneration ?? this.activationGeneration,
      activeDid: resolvedActiveDid?.value,
      sessions: {
        for (final entry in resolvedSessions.entries)
          entry.key.value: entry.value,
      },
      routingBindings: {
        for (final entry in resolvedBindings.entries)
          entry.key.value: entry.value,
      },
    );
  }

  @override
  String toString() => 'SessionRegistry(<redacted>)';

  static String _requiredString(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value is! String) throw FormatException('Invalid $key');
    return value;
  }

  static String? _optionalString(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value != null && value is! String) {
      throw FormatException('Invalid $key');
    }
    return value as String?;
  }

  static int _requiredInt(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value is! int) throw FormatException('Invalid $key');
    return value;
  }

  static int _requiredNonNegativeInt(Map<String, Object?> map, String key) {
    final value = _requiredInt(map, key);
    if (value < 0) throw FormatException('Invalid $key');
    return value;
  }

  static int _requiredPositiveInt(Map<String, Object?> map, String key) {
    final value = _requiredInt(map, key);
    if (value < 1) throw FormatException('Invalid $key');
    return value;
  }

  static Did? _mostRecentlyUsedDid(Map<Did, StoredSession> sessions) {
    if (sessions.isEmpty) return null;
    final entries = sessions.entries.toList()
      ..sort((left, right) {
        final ordinalComparison = right.value.lastUsedOrdinal.compareTo(
          left.value.lastUsedOrdinal,
        );
        return ordinalComparison != 0
            ? ordinalComparison
            : left.key.value.compareTo(right.key.value);
      });
    return entries.first.key;
  }
}
