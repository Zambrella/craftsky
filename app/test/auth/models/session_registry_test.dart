import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('round-trips a redacted two-account registry', () {
    final registry = SessionRegistry(
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
    );

    final restored = SessionRegistry.fromJson(registry.toJson());

    expect(restored.activeDid, 'did:plc:bob');
    expect(restored.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
    expect(restored.routingBindings, {
      'did:plc:alice': 'alice_binding',
      'did:plc:bob': 'bob_binding',
    });
    final diagnostic = '$restored ${restored.sessions.values.join(' ')}';
    expect(diagnostic, isNot(contains('secret-token')));
    expect(diagnostic, isNot(contains('did:plc:alice')));
    expect(diagnostic, isNot(contains('alice.test')));
    expect(diagnostic, isNot(contains('alice_binding')));
  });

  test('SIM-UT-001 fails closed when any registry entry is corrupt', () {
    expect(
      () => SessionRegistry.fromJson(
        jsonEncode({
          'schemaVersion': 1,
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
            'did:plc:corrupt': {
              'token': 42,
              'did': 'did:plc:corrupt',
              'handle': 'corrupt.test',
              'sessionGeneration': 12,
              'lastUsedOrdinal': 19,
            },
          },
        }),
      ),
      throwsFormatException,
    );
  });

  test('additively upserts and enforces the five-account limit', () {
    var registry = SessionRegistry.empty();
    for (var index = 0; index < 5; index++) {
      registry = registry.upsertAndActivate(
        token: 'token-$index',
        did: 'did:plc:a$index',
        handle: 'a$index.test',
      );
    }

    final refreshed = registry.upsertAndActivate(
      token: 'replacement-token',
      did: 'did:plc:a0',
      handle: 'refreshed.test',
    );
    expect(refreshed.sessions, hasLength(5));
    expect(
      refreshed.sessions[AccountKey('did:plc:a0').did]?.token,
      'replacement-token',
    );
    expect(
      () => registry.upsertAndActivate(
        token: 'sixth-token',
        did: 'did:plc:sixth',
        handle: 'sixth.test',
      ),
      throwsA(isA<AccountLimitReached>()),
    );
    expect(registry.sessions, hasLength(5));
  });

  test('orders active first and chooses a deterministic MRU fallback', () {
    final registry = SessionRegistry(
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

    expect(registry.orderedSessions.map((session) => session.did), [
      'did:plc:alice',
      'did:plc:bob',
      'did:plc:carol',
    ]);
    final removed = registry.remove('did:plc:alice');
    expect(removed.activeDid, 'did:plc:bob');
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
