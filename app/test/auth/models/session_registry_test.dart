import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_account_deletion.dart';
import 'package:craftsky_app/auth/models/pending_handoff.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'REG-019 pending deletion fence survives reconstruction and account '
    'switching stays stale',
    () {
      var registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final pending = PendingAccountDeletion.capture(
        jobId: '10000000-0000-4000-8000-000000000001',
        lease: registry.activeLease!,
        handle: 'alice.test',
        expiresAt: DateTime.utc(2027),
      );

      registry = SessionRegistry.fromJson(
        registry.stageAccountDeletion(pending).toJson(),
      );

      expect(registry.pendingAccountDeletion?.jobId, pending.jobId);
      expect(registry.pendingAccountDeletion?.requiredHandle, '@alice.test');
      expect(
        registry.pendingAccountDeletion?.isCurrent(
          registry.activeLease,
          now: DateTime.utc(2026, 8, 20),
        ),
        isTrue,
      );
      expect(
        registry.pendingAccountDeletion?.protects(
          registry.activeLease!.session,
          now: DateTime.utc(2027),
        ),
        isFalse,
      );
      expect(
        '$registry ${registry.pendingAccountDeletion}',
        isNot(contains('alice.test')),
      );

      registry = registry.upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      expect(
        registry.pendingAccountDeletion?.isCurrent(
          registry.activeLease,
          now: DateTime.utc(2026, 8, 20),
        ),
        isFalse,
      );
    },
  );

  test('durably stages a pending handoff before making it active', () {
    final pending = PendingHandoff(
      token: 'pending-secret-token',
      did: 'did:plc:pending',
      handle: 'pending.test',
      receiptId: '00000000-0000-4000-8000-000000000811',
      confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
    );

    final staged = SessionRegistry.empty().stageHandoff(pending);
    final restored = SessionRegistry.fromJson(staged.toJson());

    expect(restored.sessions, isEmpty);
    expect(restored.activeDid, isNull);
    expect(restored.pendingHandoff?.token, 'pending-secret-token');
    expect(restored.pendingHandoff?.receiptId, pending.receiptId);
    expect(restored.toString(), isNot(contains('pending-secret-token')));
    expect(
      restored.pendingHandoff.toString(),
      isNot(contains('pending-secret-token')),
    );

    final confirmed = restored.confirmHandoff(pending.receiptId);
    expect(confirmed.pendingHandoff, isNull);
    expect(confirmed.activeDid, 'did:plc:pending');
    expect(
      confirmed.sessions['did:plc:pending']?.token,
      'pending-secret-token',
    );
  });

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
          cachedCustomisation: const ProfileCustomisation(
            colour: 'teal',
            border: 'thick',
          ),
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
    expect(
      restored.sessions[AccountKey('did:plc:bob').did]?.cachedCustomisation,
      const ProfileCustomisation(colour: 'teal', border: 'thick'),
    );
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

  test('older session snapshots default absent cached customisation', () {
    final restored = SessionRegistry.fromJson(
      jsonEncode({
        'schemaVersion': 1,
        'nextSessionGeneration': 2,
        'nextUseOrdinal': 2,
        'activationGeneration': 1,
        'activeDid': 'did:plc:alice',
        'sessions': {
          'did:plc:alice': {
            'token': 'secret-token-alice',
            'did': 'did:plc:alice',
            'handle': 'alice.test',
            'sessionGeneration': 1,
            'lastUsedOrdinal': 1,
          },
        },
      }),
    );

    expect(
      restored.orderedSessions.single.cachedCustomisation,
      ProfileCustomisation.defaults,
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
      customisation: const ProfileCustomisation(colour: 'teal'),
    );

    expect(updated.sessions[lease.account.did]?.cachedDisplayName, 'Alice');
    expect(
      updated.sessions[lease.account.did]?.cachedCustomisation.colour,
      'teal',
    );
    expect(
      updated.updateCachedIdentity(
        AccountSessionLease(
          account: lease.account,
          sessionGeneration: lease.sessionGeneration - 1,
        ),
        displayName: 'Stale',
        avatarUrl: null,
        customisation: ProfileCustomisation.defaults,
      ),
      same(updated),
    );

    final recoloured = updated.updateCachedCustomisation(
      lease,
      const ProfileCustomisation(colour: 'rose'),
    );
    expect(
      recoloured.sessions[lease.account.did]?.cachedCustomisation.colour,
      'rose',
    );
  });
}
