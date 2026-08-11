import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final alice = AccountKey('did:plc:alice');

  test('acceptance removes ordinary Alice and activates the MRU fallback', () {
    var sessions = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'carol-token',
          did: 'did:plc:carol',
          handle: 'carol.test',
        );
    sessions = sessions.activate(sessions.leaseFor(alice)!);

    final entry = DeletionStatusEntry.pending(
      jobId: '10000000-0000-0000-0000-000000000001',
      did: alice.did.value,
      handle: 'alice.test',
      displayName: 'Alice',
      avatarUrl: 'https://example.invalid/alice.jpg',
      statusToken: 'status-token',
    );
    final result = LocalDeletionTransition.accept(
      sessions: sessions,
      statusRegistry: DeletionStatusRegistry.empty(),
      deletingLease: sessions.leaseFor(alice)!,
      entry: entry,
    );

    expect(result.sessions.sessions, isNot(contains(alice.did)));
    expect(result.sessions.activeDid?.value, 'did:plc:carol');
    expect(result.statusRegistry.entries.single, entry);
    expect(result.statusIsPrimary, isFalse);
  });

  test('acceptance makes status primary when no ordinary account remains', () {
    final sessions = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final result = LocalDeletionTransition.accept(
      sessions: sessions,
      statusRegistry: DeletionStatusRegistry.empty(),
      deletingLease: sessions.leaseFor(alice)!,
      entry: DeletionStatusEntry.pending(
        jobId: '10000000-0000-0000-0000-000000000001',
        did: alice.did.value,
        handle: 'alice.test',
        statusToken: 'status-token',
      ),
    );

    expect(result.sessions.activeDid, isNull);
    expect(result.statusIsPrimary, isTrue);
  });

  test('terminal observation removes only the matching status entry', () {
    final registry = DeletionStatusRegistry.empty()
        .upsert(
          DeletionStatusEntry.pending(
            jobId: '10000000-0000-0000-0000-000000000001',
            did: 'did:plc:alice',
            handle: 'alice.test',
            statusToken: 'alice-status',
          ),
        )
        .upsert(
          DeletionStatusEntry.pending(
            jobId: '10000000-0000-0000-0000-000000000002',
            did: 'did:plc:bob',
            handle: 'bob.test',
            statusToken: 'bob-status',
          ),
        );

    final next = registry.remove('10000000-0000-0000-0000-000000000001');

    expect(next.entries.map((entry) => entry.handle.value), ['bob.test']);
  });
}
