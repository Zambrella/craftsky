import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-016 builds active-first rows with cached identity', () {
    final registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'c-token',
          did: 'did:plc:carol',
          handle: 'carol.test',
          cachedDisplayName: 'Carol',
        )
        .upsertAndActivate(
          token: 'b-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
          cachedAvatarUrl: 'https://example.test/bob.jpg',
        )
        .upsertAndActivate(
          token: 'a-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );

    final state = AccountSwitcherState.fromRegistry(registry);

    expect(state.rows.map((row) => row.handle), [
      'alice.test',
      'bob.test',
      'carol.test',
    ]);
    expect(state.rows.first.isCurrent, isTrue);
    expect(state.rows.first.displayLabel, 'alice.test');
    expect(state.rows[1].avatarUrl, 'https://example.test/bob.jpg');
    expect(state.rows[2].displayLabel, 'Carol');
    expect(state.canAddAccount, isTrue);
  });

  test('UT-016 disables Add at five without exposing removal actions', () {
    var registry = SessionRegistry.empty();
    for (var index = 0; index < SessionRegistry.maxRetainedAccounts; index++) {
      registry = registry.upsertAndActivate(
        token: 'token-$index',
        did: 'did:plc:a$index',
        handle: 'a$index.test',
      );
    }

    final state = AccountSwitcherState.fromRegistry(registry);

    expect(state.canAddAccount, isFalse);
    expect('$state ${state.rows.join(' ')}', isNot(contains('did:plc:')));
  });
}
