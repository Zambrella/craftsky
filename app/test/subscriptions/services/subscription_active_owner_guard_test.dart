import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const operations = [
    'owner-get',
    'offering',
    'identity',
    'paywall',
    'restore',
    'customer-center',
    'reconcile-post',
    'reconcile-poll',
    'assign',
    'unassign',
  ];

  test('UT-013 exact active identified owner reaches every dispatch', () async {
    final registry = _ownerRegistry();
    final guard = BillingOwnerGuard(() => registry);
    final lease = guard.capture();
    final dispatched = <String>[];

    for (final operation in operations) {
      await guard.dispatch(lease, () async => dispatched.add(operation));
    }

    expect(dispatched, operations);
  });

  test('AT-017 beneficiary cannot dispatch any owner operation', () async {
    final registry = _ownerRegistry().upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );
    final guard = BillingOwnerGuard(() => registry);
    var dispatches = 0;

    for (final operation in operations) {
      await expectLater(
        () async {
          final lease = guard.capture();
          await guard.dispatch(lease, () async => dispatches++);
        }(),
        throwsA(isA<BillingOwnerGuardException>()),
        reason: operation,
      );
    }

    expect(dispatches, 0);
  });

  test('IT-014 stale owner confirmation is fenced before dispatch', () async {
    var registry = _ownerRegistry();
    final guard = BillingOwnerGuard(() => registry);
    final lease = guard.capture();
    registry = registry.upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );
    var dispatches = 0;

    await expectLater(
      guard.dispatch(lease, () async => dispatches++),
      throwsA(isA<BillingOwnerGuardException>()),
    );

    expect(dispatches, 0);
  });

  test('IT-014 AppView UUID mismatch fences owner state', () async {
    const guard = BillingOwnerGuard(_ownerRegistry);
    final lease = guard.capture();

    await expectLater(
      guard.dispatchBillingState(
        lease,
        () async => const BillingState(
          billingAccountId: '10000000-0000-4000-8000-000000000001',
          revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
          requestedGeneration: 1,
          reconciledGeneration: 1,
          reconciliationStale: false,
          subscriptions: [],
          licenses: [],
        ),
      ),
      throwsA(
        isA<BillingOwnerGuardException>().having(
          (error) => error.code,
          'code',
          'billing_identity_mismatch',
        ),
      ),
    );
  });

  test(
    'IT-014 owner change during identity read fences native dispatch',
    () async {
      var registry = _ownerRegistry();
      final guard = BillingOwnerGuard(() => registry);
      final lease = guard.capture();
      final identity = Completer<RevenueCatIdentity>();
      var dispatches = 0;
      final operation = guard.dispatchRevenueCat(
        lease,
        RevenueCatIdentityGuard(_PendingIdentityService(identity.future)),
        () async => dispatches++,
      );

      registry = registry.upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      identity.complete(
        const RevenueCatIdentity.identified(
          '20000000-0000-4000-8000-000000000001',
        ),
      );

      await expectLater(
        operation,
        throwsA(isA<BillingOwnerGuardException>()),
      );
      expect(dispatches, 0);
    },
  );
}

SessionRegistry _ownerRegistry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    )
    .reserveBillingOwner('did:plc:alice')
    .completeBillingOwner(
      'did:plc:alice',
      '20000000-0000-4000-8000-000000000001',
    );

final class _PendingIdentityService implements RevenueCatIdentityService {
  const _PendingIdentityService(this.identity);

  final Future<RevenueCatIdentity> identity;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() => identity;

  @override
  Future<void> identify(String appViewUuid) async {}
}
