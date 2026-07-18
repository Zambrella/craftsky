import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_session_cleanup.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

class AccountLimitReached implements Exception {
  const AccountLimitReached();

  @override
  String toString() => 'AccountLimitReached';
}

/// A complete, versioned snapshot of the accounts retained by this install.
///
/// Persistence deliberately uses an explicit codec so future versions can
/// reject unsupported top-level shapes while recovering valid entries from a
/// supported snapshot independently.
class SessionRegistry {
  SessionRegistry({
    required this.revision,
    required this.nextSessionGeneration,
    required this.nextUseOrdinal,
    required this.activationGeneration,
    required String? activeDid,
    required Map<String, StoredSession> sessions,
    Map<String, String> routingBindings = const {},
    List<PendingSessionCleanup> pendingCleanups = const [],
    this.schemaVersion = currentSchemaVersion,
  }) : activeDid = activeDid == null ? null : Did.parse(activeDid),
       sessions = Map.unmodifiable({
         for (final MapEntry(key: did, value: session) in sessions.entries)
           Did.parse(did): session,
       }),
       routingBindings = Map.unmodifiable({
         for (final MapEntry(key: did, value: binding)
             in routingBindings.entries)
           Did.parse(did): binding,
       }),
       pendingCleanups = List.unmodifiable(pendingCleanups);

  /// Recovers the newest supported top-level snapshot from the two journal
  /// slots. Slot A wins revision ties. If neither slot is usable, callers get
  /// an empty signed-out registry rather than data from a malformed snapshot.
  factory SessionRegistry.recover({String? slotA, String? slotB}) {
    SessionRegistry? decode(String? source) {
      if (source == null) return null;
      try {
        return SessionRegistry.fromJson(source);
      } on Object {
        return null;
      }
    }

    final decodedA = decode(slotA);
    final decodedB = decode(slotB);
    if (decodedA != null &&
        (decodedB == null || decodedA.revision >= decodedB.revision)) {
      return decodedA;
    }
    if (decodedB != null) return decodedB;
    return SessionRegistry.empty();
  }

  factory SessionRegistry.empty() => SessionRegistry(
    revision: 0,
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
      try {
        if (value is! Map<String, Object?>) {
          throw const FormatException('Invalid session registry entry');
        }
        final entryDid = value['did'];
        if (entryDid != did) {
          throw const FormatException('Mismatched session registry DID');
        }
        sessions[did] = StoredSession(
          token: _requiredString(value, 'token'),
          did: did,
          handle: _requiredString(value, 'handle'),
          sessionGeneration: _requiredPositiveInt(
            value,
            'sessionGeneration',
          ),
          lastUsedOrdinal: _requiredPositiveInt(value, 'lastUsedOrdinal'),
          cachedDisplayName: value['cachedDisplayName'] as String?,
          cachedAvatarUrl: value['cachedAvatarUrl'] as String?,
        );
      } on Object {
        // Entries are independent recovery units. A malformed account must not
        // discard other usable sessions in the same verified journal slot.
      }
    }

    final routingBindings = <String, String>{};
    final rawRoutingBindings = decoded['routingBindings'];
    if (rawRoutingBindings != null) {
      if (rawRoutingBindings is! Map<String, Object?>) {
        throw const FormatException('Invalid routing bindings');
      }
      for (final MapEntry(key: did, :value) in rawRoutingBindings.entries) {
        try {
          Did.parse(did);
          if (value is! String) throw const FormatException();
          routingBindings[did] = value;
        } on Object {
          // Bindings are independent recovery units just like sessions.
        }
      }
    }

    final pendingCleanups = <PendingSessionCleanup>[];
    final rawPendingCleanups = decoded['pendingCleanups'];
    if (rawPendingCleanups != null) {
      if (rawPendingCleanups is! List<Object?>) {
        throw const FormatException('Invalid pending cleanups');
      }
      for (final value in rawPendingCleanups) {
        try {
          if (value is! Map<String, Object?>) throw const FormatException();
          pendingCleanups.add(
            PendingSessionCleanup(
              account: AccountKey(_requiredString(value, 'did')),
              sessionGeneration: _requiredPositiveInt(
                value,
                'sessionGeneration',
              ),
              token: _requiredString(value, 'token'),
            ),
          );
        } on Object {
          // Cleanup entries are independent recovery units.
        }
      }
    }

    final rawActiveDid = decoded['activeDid'];
    final requestedActiveDid = rawActiveDid is String ? rawActiveDid : null;
    final repairedActiveDid = sessions.containsKey(requestedActiveDid)
        ? requestedActiveDid
        : _mostRecentlyUsedDid(sessions);

    final revision = _requiredNonNegativeInt(decoded, 'revision');
    final storedNextSessionGeneration = _requiredPositiveInt(
      decoded,
      'nextSessionGeneration',
    );
    var highestSessionGeneration = 0;
    for (final session in sessions.values) {
      if (session.sessionGeneration > highestSessionGeneration) {
        highestSessionGeneration = session.sessionGeneration;
      }
    }
    for (final cleanup in pendingCleanups) {
      if (cleanup.sessionGeneration > highestSessionGeneration) {
        highestSessionGeneration = cleanup.sessionGeneration;
      }
    }
    final nextSessionGeneration =
        storedNextSessionGeneration > highestSessionGeneration
        ? storedNextSessionGeneration
        : highestSessionGeneration + 1;
    final storedNextUseOrdinal = _requiredPositiveInt(
      decoded,
      'nextUseOrdinal',
    );
    var highestUseOrdinal = 0;
    for (final session in sessions.values) {
      if (session.lastUsedOrdinal > highestUseOrdinal) {
        highestUseOrdinal = session.lastUsedOrdinal;
      }
    }
    final nextUseOrdinal = storedNextUseOrdinal > highestUseOrdinal
        ? storedNextUseOrdinal
        : highestUseOrdinal + 1;
    final activationGeneration = _requiredNonNegativeInt(
      decoded,
      'activationGeneration',
    );

    return SessionRegistry(
      revision: revision,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: activationGeneration,
      activeDid: repairedActiveDid,
      sessions: sessions,
      routingBindings: routingBindings,
      pendingCleanups: pendingCleanups,
    );
  }

  static const currentSchemaVersion = 1;
  static const maxRetainedAccounts = 5;

  final int schemaVersion;
  final int revision;
  final int nextSessionGeneration;
  final int nextUseOrdinal;
  final int activationGeneration;
  final Did? activeDid;
  final Map<Did, StoredSession> sessions;
  final Map<Did, String> routingBindings;
  final List<PendingSessionCleanup> pendingCleanups;

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
    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration + 1,
      nextUseOrdinal: nextUseOrdinal + 1,
      activationGeneration: activationGeneration + 1,
      activeDid: did,
      sessions: {
        for (final entry in sessions.entries) entry.key.value: entry.value,
        parsedDid.value: StoredSession(
          token: token,
          did: did,
          handle: handle,
          sessionGeneration: nextSessionGeneration,
          lastUsedOrdinal: nextUseOrdinal,
          cachedDisplayName: cachedDisplayName,
          cachedAvatarUrl: cachedAvatarUrl,
        ),
      },
      routingBindings: {
        for (final entry in routingBindings.entries)
          entry.key.value: entry.value,
      },
      pendingCleanups: pendingCleanups,
    );
  }

  SessionRegistry activate(AccountSessionLease target) {
    final current = sessions[target.account.did];
    if (current == null ||
        current.sessionGeneration != target.sessionGeneration) {
      throw StateError('Account session unavailable');
    }
    if (activeDid == target.account.did) return this;

    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal + 1,
      activationGeneration: activationGeneration + 1,
      activeDid: target.account.did.value,
      sessions: {
        for (final entry in sessions.entries)
          entry.key.value: entry.key == target.account.did
              ? StoredSession(
                  token: entry.value.token,
                  did: entry.value.did.value,
                  handle: entry.value.handle.value,
                  sessionGeneration: entry.value.sessionGeneration,
                  lastUsedOrdinal: nextUseOrdinal,
                  cachedDisplayName: entry.value.cachedDisplayName,
                  cachedAvatarUrl: entry.value.cachedAvatarUrl,
                )
              : entry.value,
      },
      routingBindings: {
        for (final entry in routingBindings.entries)
          entry.key.value: entry.value,
      },
      pendingCleanups: pendingCleanups,
    );
  }

  SessionRegistry remove(String did) {
    final parsedDid = Did.parse(did);
    if (!sessions.containsKey(parsedDid)) return this;

    final remaining = <String, StoredSession>{
      for (final entry in sessions.entries)
        if (entry.key != parsedDid) entry.key.value: entry.value,
    };
    final removedActive = parsedDid == activeDid;

    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: removedActive
          ? activationGeneration + 1
          : activationGeneration,
      activeDid: removedActive ? _mostRecentlyUsedDid(remaining) : activeDid,
      sessions: remaining,
      routingBindings: {
        for (final entry in routingBindings.entries)
          if (entry.key != parsedDid) entry.key.value: entry.value,
      },
      pendingCleanups: pendingCleanups,
    );
  }

  SessionRegistry saveRoutingBinding(
    AccountSessionLease lease,
    String binding,
  ) {
    if (leaseFor(lease.account) != lease) {
      throw StateError('Account session unavailable');
    }
    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: activationGeneration,
      activeDid: activeDid?.value,
      sessions: {
        for (final entry in sessions.entries) entry.key.value: entry.value,
      },
      routingBindings: {
        for (final entry in routingBindings.entries)
          entry.key.value: entry.value,
        lease.account.did.value: binding,
      },
      pendingCleanups: pendingCleanups,
    );
  }

  SessionRegistry removeRoutingBinding(AccountSessionLease lease) {
    if (leaseFor(lease.account) != lease ||
        !routingBindings.containsKey(lease.account.did)) {
      return this;
    }
    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: activationGeneration,
      activeDid: activeDid?.value,
      sessions: {
        for (final entry in sessions.entries) entry.key.value: entry.value,
      },
      routingBindings: {
        for (final entry in routingBindings.entries)
          if (entry.key != lease.account.did) entry.key.value: entry.value,
      },
      pendingCleanups: pendingCleanups,
    );
  }

  /// Atomically makes [lease] unavailable and retains the credential needed
  /// to finish its server-side cleanup after connectivity returns.
  SessionRegistry quarantineAndRemove(AccountSessionLease lease) {
    final stored = sessions[lease.account.did];
    if (stored == null || stored.sessionGeneration != lease.sessionGeneration) {
      return this;
    }

    final remaining = <String, StoredSession>{
      for (final entry in sessions.entries)
        if (entry.key != lease.account.did) entry.key.value: entry.value,
    };
    final removedActive = activeDid == lease.account.did;
    final cleanup = PendingSessionCleanup(
      account: lease.account,
      sessionGeneration: lease.sessionGeneration,
      token: stored.token,
    );

    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: removedActive
          ? activationGeneration + 1
          : activationGeneration,
      activeDid: removedActive ? _mostRecentlyUsedDid(remaining) : activeDid,
      sessions: remaining,
      routingBindings: {
        for (final entry in routingBindings.entries)
          if (entry.key != lease.account.did) entry.key.value: entry.value,
      },
      pendingCleanups: [
        for (final pending in pendingCleanups)
          if (pending != cleanup) pending,
        cleanup,
      ],
    );
  }

  SessionRegistry removePendingCleanup(PendingSessionCleanup cleanup) {
    if (!pendingCleanups.contains(cleanup)) return this;
    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: activationGeneration,
      activeDid: activeDid?.value,
      sessions: {
        for (final entry in sessions.entries) entry.key.value: entry.value,
      },
      routingBindings: {
        for (final entry in routingBindings.entries)
          entry.key.value: entry.value,
      },
      pendingCleanups: [
        for (final pending in pendingCleanups)
          if (pending != cleanup) pending,
      ],
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
    return SessionRegistry(
      revision: revision + 1,
      nextSessionGeneration: nextSessionGeneration,
      nextUseOrdinal: nextUseOrdinal,
      activationGeneration: activationGeneration,
      activeDid: activeDid?.value,
      sessions: {
        for (final entry in sessions.entries)
          entry.key.value: entry.key == lease.account.did
              ? StoredSession(
                  token: entry.value.token,
                  did: entry.value.did.value,
                  handle: entry.value.handle.value,
                  sessionGeneration: entry.value.sessionGeneration,
                  lastUsedOrdinal: entry.value.lastUsedOrdinal,
                  cachedDisplayName: displayName,
                  cachedAvatarUrl: avatarUrl,
                )
              : entry.value,
      },
      routingBindings: {
        for (final entry in routingBindings.entries)
          entry.key.value: entry.value,
      },
      pendingCleanups: pendingCleanups,
    );
  }

  String toJson() => jsonEncode({
    'schemaVersion': schemaVersion,
    'revision': revision,
    'nextSessionGeneration': nextSessionGeneration,
    'nextUseOrdinal': nextUseOrdinal,
    'activationGeneration': activationGeneration,
    'activeDid': activeDid,
    'routingBindings': {
      for (final MapEntry(key: did, value: binding) in routingBindings.entries)
        did: binding,
    },
    'pendingCleanups': [
      for (final cleanup in pendingCleanups)
        {
          'did': cleanup.account.did,
          'sessionGeneration': cleanup.sessionGeneration,
          'token': cleanup.token,
        },
    ],
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

  @override
  String toString() => 'SessionRegistry(<redacted>)';

  static String _requiredString(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value is! String) throw FormatException('Invalid $key');
    return value;
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

  static String? _mostRecentlyUsedDid(Map<String, StoredSession> sessions) {
    if (sessions.isEmpty) return null;
    final entries = sessions.entries.toList()
      ..sort((left, right) {
        final ordinalComparison = right.value.lastUsedOrdinal.compareTo(
          left.value.lastUsedOrdinal,
        );
        return ordinalComparison != 0
            ? ordinalComparison
            : left.key.compareTo(right.key);
      });
    return entries.first.key;
  }
}
