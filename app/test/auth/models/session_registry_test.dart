import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_session_cleanup.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('round-trips a redacted two-account v1 registry', () {
    final registry = SessionRegistry(
      revision: 7,
      nextSessionGeneration: 12,
      nextUseOrdinal: 20,
      activationGeneration: 4,
      activeDid: 'did:plc:bob',
      sessions: {
        'did:plc:alice': StoredSession(
          token: 'secret-token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
          sessionGeneration: 10,
          lastUsedOrdinal: 18,
        ),
        'did:plc:bob': StoredSession(
          token: 'secret-token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
          sessionGeneration: 11,
          lastUsedOrdinal: 19,
        ),
      },
      routingBindings: const {
        'did:plc:alice': 'alice_binding',
        'did:plc:bob': 'bob_binding',
      },
      pendingCleanups: [
        PendingSessionCleanup(
          account: AccountKey('did:plc:removed'),
          sessionGeneration: 9,
          token: 'cleanup-secret-removed',
        ),
      ],
    );

    final restored = SessionRegistry.fromJson(registry.toJson());

    expect(restored.schemaVersion, 1);
    expect(restored.revision, 7);
    expect(restored.activeDid, 'did:plc:bob');
    expect(restored.sessions.keys, {
      'did:plc:alice',
      'did:plc:bob',
    });
    expect(restored.sessions['did:plc:alice']?.token, 'secret-token-alice');
    expect(restored.sessions['did:plc:bob']?.sessionGeneration, 11);
    expect(restored.routingBindings, {
      'did:plc:alice': 'alice_binding',
      'did:plc:bob': 'bob_binding',
    });
    expect(restored.pendingCleanups, hasLength(1));
    expect(restored.pendingCleanups.single.token, 'cleanup-secret-removed');

    final diagnostic =
        '$restored ${restored.sessions.values.join(' ')} '
        '${restored.pendingCleanups.join(' ')}';
    expect(diagnostic, isNot(contains('secret-token-alice')));
    expect(diagnostic, isNot(contains('secret-token-bob')));
    expect(diagnostic, isNot(contains('did:plc:alice')));
    expect(diagnostic, isNot(contains('did:plc:bob')));
    expect(diagnostic, isNot(contains('alice.test')));
    expect(diagnostic, isNot(contains('bob.test')));
    expect(diagnostic, isNot(contains('alice_binding')));
    expect(diagnostic, isNot(contains('bob_binding')));
    expect(diagnostic, isNot(contains('cleanup-secret-removed')));
    expect(diagnostic, isNot(contains('did:plc:removed')));
  });

  test('drops a corrupt entry and repairs the active DID by MRU', () {
    final restored = SessionRegistry.fromJson(
      jsonEncode({
        'schemaVersion': 1,
        'revision': 8,
        'nextSessionGeneration': 13,
        'nextUseOrdinal': 21,
        'activationGeneration': 5,
        'activeDid': 'did:plc:corrupt',
        'sessions': {
          'did:plc:alice': {
            'token': 'secret-token-alice',
            'did': 'did:plc:alice',
            'handle': 'alice.test',
            'sessionGeneration': 10,
            'lastUsedOrdinal': 18,
          },
          'did:plc:bob': {
            'token': 'secret-token-bob',
            'did': 'did:plc:bob',
            'handle': 'bob.test',
            'sessionGeneration': 11,
            'lastUsedOrdinal': 20,
          },
          'did:plc:corrupt': {
            'token': 42,
            'did': 'did:plc:corrupt',
            'handle': 'corrupt.test',
            'sessionGeneration': 12,
            'lastUsedOrdinal': 19,
          },
        },
      }),
    );

    expect(restored.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
    expect(restored.activeDid, 'did:plc:bob');
  });

  test(
    'selects the newest supported slot without resurrecting older entries',
    () {
      final older = SessionRegistry(
        revision: 4,
        nextSessionGeneration: 8,
        nextUseOrdinal: 8,
        activationGeneration: 2,
        activeDid: 'did:plc:alice',
        sessions: {
          'did:plc:alice': StoredSession(
            token: 'secret-token-alice',
            did: 'did:plc:alice',
            handle: 'alice.test',
            sessionGeneration: 7,
            lastUsedOrdinal: 7,
          ),
        },
      ).toJson();
      final unsupportedNewer = jsonEncode({
        'schemaVersion': 2,
        'revision': 5,
        'nextSessionGeneration': 9,
        'nextUseOrdinal': 9,
        'activationGeneration': 3,
        'activeDid': null,
        'sessions': <String, Object?>{},
      });

      final supportedFallback = SessionRegistry.recover(
        slotA: older,
        slotB: unsupportedNewer,
      );
      expect(supportedFallback.revision, 4);
      expect(supportedFallback.sessions.keys, {'did:plc:alice'});

      final newestWithCorruptOnly = jsonEncode({
        'schemaVersion': 1,
        'revision': 6,
        'nextSessionGeneration': 10,
        'nextUseOrdinal': 10,
        'activationGeneration': 4,
        'activeDid': 'did:plc:alice',
        'sessions': {
          'did:plc:alice': {'token': 42},
        },
      });

      final noResurrection = SessionRegistry.recover(
        slotA: older,
        slotB: newestWithCorruptOnly,
      );
      expect(noResurrection.revision, 6);
      expect(noResurrection.sessions, isEmpty);
      expect(noResurrection.activeDid, isNull);
    },
  );

  test(
    'UT-001 repairs malformed active metadata and monotonic counters '
    'in newest slot',
    () {
      final older = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'old-token',
            did: 'did:plc:old',
            handle: 'old.test',
          )
          .toJson();
      final newest = jsonEncode({
        'schemaVersion': 1,
        'revision': 5,
        'nextSessionGeneration': 2,
        'nextUseOrdinal': 3,
        'activationGeneration': 4,
        'activeDid': 42,
        'sessions': {
          'did:plc:alice': {
            'token': 'alice-token',
            'did': 'did:plc:alice',
            'handle': 'alice.test',
            'sessionGeneration': 7,
            'lastUsedOrdinal': 10,
          },
        },
        'pendingCleanups': [
          {
            'did': 'did:plc:removed',
            'sessionGeneration': 11,
            'token': 'cleanup-token',
          },
        ],
      });

      final recovered = SessionRegistry.recover(
        slotA: older,
        slotB: newest,
      );

      expect(recovered.revision, 5);
      expect(recovered.activeDid?.value, 'did:plc:alice');
      expect(recovered.sessions.keys.map((did) => did.value), [
        'did:plc:alice',
      ]);
      expect(recovered.nextSessionGeneration, 12);
      expect(recovered.nextUseOrdinal, 11);

      final added = recovered.upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      expect(
        added.sessions[AccountKey('did:plc:bob').did]?.sessionGeneration,
        12,
      );
      expect(
        added.sessions[AccountKey('did:plc:bob').did]?.lastUsedOrdinal,
        11,
      );
    },
  );

  test('additively upserts and activates a distinct account', () {
    final registry = SessionRegistry(
      revision: 2,
      nextSessionGeneration: 8,
      nextUseOrdinal: 12,
      activationGeneration: 3,
      activeDid: 'did:plc:alice',
      sessions: {
        'did:plc:alice': StoredSession(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
          sessionGeneration: 7,
          lastUsedOrdinal: 11,
        ),
      },
    );

    final updated = registry.upsertAndActivate(
      token: 'token-bob',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );

    expect(updated.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
    expect(updated.activeDid, 'did:plc:bob');
    expect(updated.sessions['did:plc:bob']?.sessionGeneration, 8);
    expect(updated.sessions['did:plc:bob']?.lastUsedOrdinal, 12);
    expect(updated.nextSessionGeneration, 9);
    expect(updated.nextUseOrdinal, 13);
    expect(updated.activationGeneration, 4);
    expect(updated.revision, 3);
  });

  test('replaces a retained DID at five and rejects a sixth atomically', () {
    final registry = SessionRegistry(
      revision: 9,
      nextSessionGeneration: 10,
      nextUseOrdinal: 10,
      activationGeneration: 5,
      activeDid: 'did:plc:a0',
      sessions: {
        for (var index = 0; index < 5; index++)
          'did:plc:a$index': StoredSession(
            token: 'token-$index',
            did: 'did:plc:a$index',
            handle: 'a$index.test',
            sessionGeneration: index + 1,
            lastUsedOrdinal: index + 1,
          ),
      },
    );

    final refreshed = registry.upsertAndActivate(
      token: 'replacement-token',
      did: 'did:plc:a0',
      handle: 'refreshed.test',
    );
    expect(refreshed.sessions, hasLength(5));
    expect(refreshed.sessions['did:plc:a0']?.token, 'replacement-token');
    expect(refreshed.sessions['did:plc:a0']?.sessionGeneration, 10);

    expect(
      () => registry.upsertAndActivate(
        token: 'sixth-token',
        did: 'did:plc:sixth',
        handle: 'sixth.test',
      ),
      throwsA(isA<AccountLimitReached>()),
    );
    expect(registry.sessions, hasLength(5));
    expect(registry.revision, 9);
    expect(registry.activeDid, 'did:plc:a0');
  });

  test('orders active first and chooses a deterministic MRU fallback', () {
    final registry = SessionRegistry(
      revision: 5,
      nextSessionGeneration: 10,
      nextUseOrdinal: 20,
      activationGeneration: 7,
      activeDid: 'did:plc:alice',
      sessions: {
        'did:plc:alice': StoredSession(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
          sessionGeneration: 1,
          lastUsedOrdinal: 12,
        ),
        'did:plc:bob': StoredSession(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
          sessionGeneration: 2,
          lastUsedOrdinal: 18,
        ),
        'did:plc:carol': StoredSession(
          token: 'token-carol',
          did: 'did:plc:carol',
          handle: 'carol.test',
          sessionGeneration: 3,
          lastUsedOrdinal: 18,
        ),
      },
    );

    expect(
      registry.orderedSessions.map((session) => session.did),
      ['did:plc:alice', 'did:plc:bob', 'did:plc:carol'],
    );

    final removed = registry.remove('did:plc:alice');
    expect(removed.activeDid, 'did:plc:bob');
    expect(
      removed.orderedSessions.map((session) => session.did),
      ['did:plc:bob', 'did:plc:carol'],
    );
    expect(removed.nextSessionGeneration, 10);
    expect(removed.activationGeneration, 8);
  });

  test('UT-016 caches switcher identity only for the matching lease', () {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token-alice',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = registry.activeLease!.session;

    final updated = registry.updateCachedIdentity(
      lease,
      displayName: 'Alice',
      avatarUrl: 'https://example.test/alice.jpg',
    );

    expect(updated.sessions[lease.account.did]?.cachedDisplayName, 'Alice');
    expect(
      updated.sessions[lease.account.did]?.cachedAvatarUrl,
      'https://example.test/alice.jpg',
    );
    expect(
      updated.updateCachedIdentity(
        AccountSessionLease(
          account: lease.account,
          sessionGeneration: lease.sessionGeneration - 1,
        ),
        displayName: 'Stale',
        avatarUrl: null,
      ),
      same(updated),
    );
  });
}
