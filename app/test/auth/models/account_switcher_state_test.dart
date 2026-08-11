import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('deleting rows are disabled and expose only coarse status state', () {
    final sessions = SessionRegistry.empty().upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );
    final deleting = DeletionStatusRegistry.empty().upsert(
      DeletionStatusEntry.pending(
        jobId: '10000000-0000-0000-0000-000000000001',
        did: 'did:plc:alice',
        handle: 'alice.test',
        statusToken: 'status-token',
      ).withStatus(
        status: AccountDeletionStatus.needsAttention,
        phase: AccountDeletionPhase.waitingForCraftsky,
        canRetry: true,
      ),
    );

    final state = AccountSwitcherState.fromRegistries(
      sessions: sessions,
      deletions: deleting,
    );

    expect(state.rows, hasLength(1));
    expect(state.deletingRows, hasLength(1));
    final row = state.deletingRows.single;
    expect(row.canActivate, isFalse);
    expect(row.canRetry, isTrue);
    expect(row.phase, AccountDeletionPhase.waitingForCraftsky);
    expect(row.toString(), isNot(contains('did:plc:alice')));
    expect(row.toString(), isNot(contains('status-token')));
  });

  test('terminal deletion rows are omitted', () {
    final deleting = DeletionStatusRegistry.empty().upsert(
      DeletionStatusEntry.pending(
        jobId: '10000000-0000-0000-0000-000000000001',
        did: 'did:plc:alice',
        handle: 'alice.test',
        statusToken: 'status-token',
      ).withStatus(
        status: AccountDeletionStatus.deleted,
        phase: AccountDeletionPhase.deleted,
      ),
    );

    final state = AccountSwitcherState.fromRegistries(
      sessions: SessionRegistry.empty(),
      deletions: deleting,
    );

    expect(state.deletingRows, isEmpty);
  });
}
