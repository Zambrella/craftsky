import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'each active DID loads its own customer and access across switches',
    () async {
      final registry = _registry();
      final alice = _Api('did:plc:alice', SubscriptionTier.plus);
      final bob = _Api(
        'did:plc:bob',
        SubscriptionTier.business,
        revenueCatAppUserId: '20000000-0000-4000-8000-000000000002',
      );
      final revenueCat = _RevenueCat();
      final storage = _Storage(registry);
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith(
            (ref, account) async =>
                account.did == Did.parse('did:plc:alice') ? alice : bob,
          ),
        ],
      );
      addTearDown(container.dispose);
      container.listen(subscriptionPageModelProvider, (_, _) {});
      final first = await container.read(subscriptionPageModelProvider.future);
      expect(first.access.effectiveTier, SubscriptionTier.plus);
      expect(
        first.billingState?.revenueCatAppUserId,
        alice.revenueCatAppUserId,
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(registry.leaseFor(AccountKey('did:plc:bob'))!);
      final second = await container.read(subscriptionPageModelProvider.future);
      expect(second.access.effectiveTier, SubscriptionTier.business);
      expect(second.billingState?.revenueCatAppUserId, bob.revenueCatAppUserId);
      expect(revenueCat.appUserId, bob.revenueCatAppUserId);
      expect(storage.value.billingCustomers, hasLength(2));
      expect(bob.ownerReads, 1);
    },
  );
  test(
    'removing the former payer does not prevent another DID subscribing',
    () async {
      final registry = _registry().remove('did:plc:alice');
      final bob = _Api(
        'did:plc:bob',
        SubscriptionTier.free,
        revenueCatAppUserId: '20000000-0000-4000-8000-000000000002',
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(_RevenueCat()),
          subscriptionRepositoryProvider.overrideWith(
            (ref, account) async => bob,
          ),
        ],
      );
      addTearDown(container.dispose);
      container.listen(subscriptionPageModelProvider, (_, _) {});
      final model = await container.read(subscriptionPageModelProvider.future);
      expect(model.billingState?.revenueCatAppUserId, bob.revenueCatAppUserId);
      expect(model.access.effectiveTier, SubscriptionTier.free);
    },
  );
}

SessionRegistry _registry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
    )
    .upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    )
    .setBillingCustomer(
      'did:plc:alice',
      '20000000-0000-4000-8000-000000000001',
    );

final class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Api implements SubscriptionApi {
  new(
    this.did,
    this.tier, {
    this.revenueCatAppUserId = '20000000-0000-4000-8000-000000000001',
  });
  final String did;
  final SubscriptionTier tier;
  final String revenueCatAppUserId;
  int ownerReads = 0;

  @override
  Future<SubscriptionAccess> getAccess() async => SubscriptionAccess(
    did: Did.parse(did),
    effectiveTier: tier,
    givesAccess: tier != SubscriptionTier.free,
  );

  @override
  Future<BillingState> getBillingAccount() async {
    ownerReads++;
    return BillingState(
      billingAccountId: '10000000-0000-4000-8000-000000000001',
      revenueCatAppUserId: revenueCatAppUserId,
      requestedGeneration: 1,
      reconciledGeneration: 1,
      reconciliationStale: false,
      subscriptions: [],
      licenses: [],
    );
  }

  @override
  Future<BillingState> ensureBillingAccount() => getBillingAccount();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();
}

final class _RevenueCat implements RevenueCatService {
  new();
  String appUserId = '20000000-0000-4000-8000-000000000001';
  int identityCalls = 0;
  int identifyCalls = 0;
  final List<String> offeringCalls = [];

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    identityCalls++;
    return RevenueCatIdentity.identified(appUserId);
  }

  @override
  Future<void> identify(String appViewUuid) async {
    appUserId = appViewUuid;
    identifyCalls++;
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async {
    offeringCalls.add(identifier);
    return _Offering(identifier, null);
  }

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) => throw UnimplementedError();

  @override
  Future<DirectPaywallResult> presentPaywall(RevenueCatOffering offering) =>
      throw UnimplementedError();

  @override
  Future<void> restorePurchases() => throw UnimplementedError();
}

final class _Offering implements RevenueCatOffering {
  const new(this.identifier, this.price);

  @override
  final String identifier;
  final String? price;

  @override
  bool hasPackage(String identifier) => identifier == r'$rc_monthly';

  @override
  String? packagePrice(String identifier) =>
      hasPackage(identifier) ? price : null;
}
