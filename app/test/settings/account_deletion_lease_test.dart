import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('captured Alice deletion flow becomes stale when Bob activates', () {
    var registry = SessionRegistry.empty()
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
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    registry = registry.activate(alice);
    final fence = AccountDeletionLeaseFence.capture(registry);

    expect(fence.isCurrent(registry), isTrue);

    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;
    registry = registry.activate(bob);

    expect(fence.isCurrent(registry), isFalse);
    expect(fence.account, alice.account);
  });
}
