import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/notifications/models/notification_badge.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-016 builds active-first rows with identity and capped badges', () {
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

    final state = AccountSwitcherState.fromRegistry(
      registry,
      notificationCounts: {
        AccountKey('did:plc:alice'): 0,
        AccountKey('did:plc:bob'): 123,
        AccountKey('did:plc:carol'): 4,
      },
    );

    expect(state.rows.map((row) => row.handle), [
      'alice.test',
      'bob.test',
      'carol.test',
    ]);
    expect(state.rows.first.isCurrent, isTrue);
    expect(state.rows.first.displayLabel, 'alice.test');
    expect(state.rows[1].avatarUrl, 'https://example.test/bob.jpg');
    expect(state.rows[1].badge.label, '99+');
    expect(state.rows[2].displayLabel, 'Carol');
    expect(state.rows[2].badge.label, NotificationBadge.fromCount(4).label);
    expect(state.canAddAccount, isTrue);
    expect(state.addAccountHelper, isNull);
    expect(state.actions, {
      AccountSwitcherAction.select,
      AccountSwitcherAction.add,
    });
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
    expect(state.addAccountHelper, 'Maximum of 5 accounts');
    expect(state.actions, {AccountSwitcherAction.select});
    expect('$state ${state.rows.join(' ')}', isNot(contains('did:plc:')));
  });
}
