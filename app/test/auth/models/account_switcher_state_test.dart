import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('switcher projects only retained ordinary accounts', () {
    final sessions = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );

    final state = AccountSwitcherState.fromRegistry(sessions);

    expect(state.rows, hasLength(2));
    expect(state.rows.where((row) => row.isCurrent).single.handle, 'bob.test');
    expect(state.canAddAccount, isTrue);
  });
}
