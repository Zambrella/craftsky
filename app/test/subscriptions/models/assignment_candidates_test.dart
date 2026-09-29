import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_rules.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-007 candidates include retained owner and eligible non-owner', () {
    final registry = _registry();
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;

    expect(
      assignmentCandidates(
        registry: registry,
        retainedLeases: [alice, bob],
        state: _state(),
      ).map((value) => value.account.did),
      [Did.parse('did:plc:alice'), Did.parse('did:plc:bob')],
    );
  });

  test('UT-007 assigned, missing, and stale targets are excluded', () {
    final registry = _registry();
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;
    final missing = AccountSessionLease(
      account: AccountKey('did:plc:missing'),
      sessionGeneration: 1,
    );
    final staleBob = AccountSessionLease(
      account: bob.account,
      sessionGeneration: bob.sessionGeneration + 1,
    );

    expect(
      assignmentCandidates(
        registry: registry,
        retainedLeases: [alice, bob, missing, staleBob],
        state: _state(assignedDid: 'did:plc:bob'),
      ),
      [alice],
    );
  });

  test(
    'IT-008 removing dormant assignment enables replacement license target',
    () {
      final registry = _registry();
      final leases = [
        registry.leaseFor(AccountKey('did:plc:alice'))!,
        registry.leaseFor(AccountKey('did:plc:bob'))!,
      ];

      final before = assignmentCandidates(
        registry: registry,
        retainedLeases: leases,
        state: _replacementState(dormantAssignedDid: 'did:plc:bob'),
      );
      final after = assignmentCandidates(
        registry: registry,
        retainedLeases: leases,
        state: _replacementState(),
      );

      expect(before.map((lease) => lease.account.did), [
        Did.parse('did:plc:alice'),
      ]);
      expect(after.map((lease) => lease.account.did), [
        Did.parse('did:plc:alice'),
        Did.parse('did:plc:bob'),
      ]);
    },
  );
}

SessionRegistry _registry() => SessionRegistry.empty()
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

BillingState _state({String? assignedDid}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  subscriptions: const [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000001',
      productId: 'craftsky_business',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    ),
  ],
  licenses: [
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000001',
      subscriptionId: '30000000-0000-4000-8000-000000000001',
      tier: SubscriptionTier.business,
      assignedDid: assignedDid == null ? null : Did.parse(assignedDid),
      assignable: true,
      anomaly: 'none',
    ),
  ],
);

BillingState _replacementState({String? dormantAssignedDid}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 2,
  reconciledGeneration: 2,
  reconciliationStale: false,
  subscriptions: const [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000001',
      productId: 'craftsky_business_old',
      store: 'app_store',
      status: 'expired',
      givesAccess: false,
      pendingPayment: false,
      autoRenewalStatus: 'will_not_renew',
      anomaly: 'none',
    ),
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000002',
      productId: 'craftsky_business_new',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    ),
  ],
  licenses: [
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000001',
      subscriptionId: '30000000-0000-4000-8000-000000000001',
      tier: SubscriptionTier.business,
      assignedDid: dormantAssignedDid == null
          ? null
          : Did.parse(dormantAssignedDid),
      assignable: false,
      anomaly: 'none',
    ),
    const BillingLicense(
      id: '40000000-0000-4000-8000-000000000002',
      subscriptionId: '30000000-0000-4000-8000-000000000002',
      tier: SubscriptionTier.business,
      assignable: true,
      anomaly: 'none',
    ),
  ],
);
