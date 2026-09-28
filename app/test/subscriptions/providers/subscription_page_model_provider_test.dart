import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'AT-003 switching accounts uses only each AppView self-access',
    () async {
      final registry = _registry();
      final alice = _Api('did:plc:alice', SubscriptionTier.plus);
      final bob = _Api('did:plc:bob', SubscriptionTier.business);
      final revenueCat = _RevenueCat();
      final storage = _Storage(registry);
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((ref, account) async {
            return account.did == Did.parse('did:plc:alice') ? alice : bob;
          }),
        ],
      );
      addTearDown(container.dispose);

      final aliceModel = await container.read(
        subscriptionPageModelProvider.future,
      );
      expect(aliceModel.role, SubscriptionPageRole.owner);
      expect(aliceModel.access.effectiveTier, SubscriptionTier.plus);
      final ownerIdentityCalls = revenueCat.identityCalls;

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(
            registry.leaseFor(AccountKey('did:plc:bob'))!,
          );
      final bobModel = await container.read(
        subscriptionPageModelProvider.future,
      );

      expect(bobModel.role, SubscriptionPageRole.beneficiary);
      expect(bobModel.access.effectiveTier, SubscriptionTier.business);
      expect(bob.ownerReads, 0);
      expect(revenueCat.identityCalls, ownerIdentityCalls);
      expect(revenueCat.identifyCalls, 0);
    },
  );

  test('AT-011 missing retained owner requires reauthentication', () async {
    final registry = _registry().remove('did:plc:alice');
    final bob = _Api('did:plc:bob', SubscriptionTier.free);
    final revenueCat = _RevenueCat();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        revenueCatServiceProvider.overrideWithValue(revenueCat),
        subscriptionRepositoryProvider.overrideWith((_, _) async => bob),
      ],
    );
    addTearDown(container.dispose);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.role, SubscriptionPageRole.inactiveOwner);
    expect(model.ownerRetained, isFalse);
    expect(model.access.effectiveTier, SubscriptionTier.free);
    expect(model.billingState, isNull);
    expect(bob.ownerReads, 0);
    expect(revenueCat.identityCalls, 0);
    expect(revenueCat.identifyCalls, 0);
  });

  test(
    'IT-003 owner page locks billing but preserves access on mismatch',
    () async {
      final registry = _registry();
      final revenueCat = _RevenueCat();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith(
            (_, _) async => _Api(
              'did:plc:alice',
              SubscriptionTier.plus,
              revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
            ),
          ),
        ],
      );
      addTearDown(container.dispose);

      final model = await container.read(subscriptionPageModelProvider.future);

      expect(model.access.effectiveTier, SubscriptionTier.plus);
      expect(model.role, SubscriptionPageRole.owner);
      expect(model.ownerRecoveryLocked, isTrue);
      expect(model.billingState, isNull);
      expect(model.operationNotice, isNull);
      expect(revenueCat.identityCalls, 0);
      expect(revenueCat.identifyCalls, 0);
    },
  );

  test('IT-014 missing persisted owner account never identifies', () async {
    final registry = _registry();
    final revenueCat = _RevenueCat();
    final api = _Api(
      'did:plc:alice',
      SubscriptionTier.plus,
      ownerError: const ApiBadRequest(
        'billing_account_not_found',
        details: ApiFailureDetails(statusCode: 404),
      ),
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        revenueCatServiceProvider.overrideWithValue(revenueCat),
        subscriptionRepositoryProvider.overrideWith((_, _) async => api),
      ],
    );
    addTearDown(container.dispose);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.ownerRecoveryLocked, isTrue);
    expect(api.ownerReads, 1);
    expect(revenueCat.identityCalls, 0);
    expect(revenueCat.identifyCalls, 0);
  });

  test('expired owner sign-in is not treated as a billing mismatch', () async {
    final revenueCat = _RevenueCat();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(_registry()),
        ),
        revenueCatServiceProvider.overrideWithValue(revenueCat),
        subscriptionRepositoryProvider.overrideWith(
          (_, _) async => _Api(
            'did:plc:alice',
            SubscriptionTier.plus,
            ownerError: const ApiUnauthorized(),
          ),
        ),
      ],
    );
    addTearDown(container.dispose);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.access.effectiveTier, SubscriptionTier.plus);
    expect(model.ownerSignInRequired, isTrue);
    expect(model.ownerRecoveryLocked, isFalse);
    expect(model.billingState, isNull);
    expect(revenueCat.identityCalls, 0);
  });

  test('IT-014 mismatched RevenueCat owner never reidentifies', () async {
    final registry = _registry();
    final revenueCat = _RevenueCat(
      appUserId: '20000000-0000-4000-8000-000000000099',
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        revenueCatServiceProvider.overrideWithValue(revenueCat),
        subscriptionRepositoryProvider.overrideWith(
          (_, _) async => _Api('did:plc:alice', SubscriptionTier.plus),
        ),
      ],
    );
    addTearDown(container.dispose);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.ownerRecoveryLocked, isTrue);
    expect(model.billingState, isNull);
    expect(revenueCat.identityCalls, 1);
    expect(revenueCat.identifyCalls, 0);
  });

  test('IT-004 RevenueCat identity failure preserves retry state', () async {
    final registry = _registry();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        revenueCatServiceProvider.overrideWithValue(
          _RevenueCat(identityError: Exception('provider unavailable')),
        ),
        subscriptionRepositoryProvider.overrideWith(
          (_, _) async => _Api('did:plc:alice', SubscriptionTier.plus),
        ),
      ],
    );
    addTearDown(container.dispose);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.ownerRecoveryLocked, isFalse);
    expect(model.billingState, isNull);
    expect(
      model.operationNotice,
      SubscriptionOperationNotice.providerFailure,
    );
  });

  test(
    'IT-004 owner provider failure preserves access with retry state',
    () async {
      final registry = _registry();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(_RevenueCat()),
          subscriptionRepositoryProvider.overrideWith(
            (_, _) async => _Api(
              'did:plc:alice',
              SubscriptionTier.plus,
              ownerError: const ApiServerError('server_error'),
            ),
          ),
        ],
      );
      addTearDown(container.dispose);

      final model = await container.read(subscriptionPageModelProvider.future);

      expect(model.access.effectiveTier, SubscriptionTier.plus);
      expect(model.role, SubscriptionPageRole.owner);
      expect(model.ownerRecoveryLocked, isFalse);
      expect(model.billingState, isNull);
      expect(
        model.operationNotice,
        SubscriptionOperationNotice.providerFailure,
      );
    },
  );

  test(
    'owner model keeps prices from offerings that load successfully',
    () async {
      final registry = _registry();
      final revenueCat = _RevenueCat(
        prices: const {
          SubscriptionTier.plus: r'$4.99',
          SubscriptionTier.business: '£8.99',
        },
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith(
            (_, _) async => _Api('did:plc:alice', SubscriptionTier.free),
          ),
        ],
      );
      addTearDown(container.dispose);

      final model = await container.read(subscriptionPageModelProvider.future);

      expect(model.tierPrices, {
        SubscriptionTier.plus: r'$4.99',
        SubscriptionTier.business: '£8.99',
      });
      expect(revenueCat.offeringCalls, ['plus', 'business']);
    },
  );

  test('one failed offering does not discard another tier price', () async {
    final registry = _registry();
    final revenueCat = _RevenueCat(
      prices: const {SubscriptionTier.business: '€9.99'},
      failedOfferings: const {'plus'},
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        revenueCatServiceProvider.overrideWithValue(revenueCat),
        subscriptionRepositoryProvider.overrideWith(
          (_, _) async => _Api('did:plc:alice', SubscriptionTier.free),
        ),
      ],
    );
    addTearDown(container.dispose);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.tierPrices, {
      SubscriptionTier.business: '€9.99',
    });
    expect(revenueCat.offeringCalls, ['plus', 'business']);
  });

  test('IT-014 account switch fences owner GET before dispatch', () async {
    final registry = _registry();
    final storage = _Storage(registry);
    final alice = _Api('did:plc:alice', SubscriptionTier.plus);
    final repository = Completer<SubscriptionApi>();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        revenueCatServiceProvider.overrideWithValue(_RevenueCat()),
        subscriptionRepositoryProvider.overrideWith(
          (_, _) => repository.future,
        ),
      ],
    );
    addTearDown(container.dispose);

    final pending = container.read(subscriptionPageModelProvider.future);
    await Future<void>.delayed(Duration.zero);
    await container
        .read(sessionRegistryProvider.notifier)
        .activate(registry.leaseFor(AccountKey('did:plc:bob'))!);
    repository.complete(alice);

    await expectLater(pending, throwsA(anything));
    expect(alice.ownerReads, 0);
  });
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
    .reserveBillingOwner('did:plc:alice')
    .completeBillingOwner(
      'did:plc:alice',
      '20000000-0000-4000-8000-000000000001',
    );

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Api implements SubscriptionApi {
  _Api(
    this.did,
    this.tier, {
    this.revenueCatAppUserId = '20000000-0000-4000-8000-000000000001',
    this.ownerError,
  });
  final String did;
  final SubscriptionTier tier;
  final String revenueCatAppUserId;
  final Exception? ownerError;
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
    if (ownerError case final error?) throw error;
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
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      throw UnimplementedError();

  @override
  Future<BillingState> ensureBillingAccount() => throw UnimplementedError();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();

  @override
  Future<void> unassign(String licenseId) => throw UnimplementedError();
}

final class _RevenueCat implements RevenueCatService {
  _RevenueCat({
    this.appUserId = '20000000-0000-4000-8000-000000000001',
    this.identityError,
    this.prices = const {},
    this.failedOfferings = const {},
  });

  final String appUserId;
  final Exception? identityError;
  final Map<SubscriptionTier, String> prices;
  final Set<String> failedOfferings;
  int identityCalls = 0;
  int identifyCalls = 0;
  final List<String> offeringCalls = [];

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    identityCalls++;
    if (identityError case final error?) throw error;
    return RevenueCatIdentity.identified(appUserId);
  }

  @override
  Future<void> identify(String appViewUuid) async => identifyCalls++;

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async {
    offeringCalls.add(identifier);
    if (failedOfferings.contains(identifier)) {
      throw StateError('offering unavailable');
    }
    final tier = SubscriptionTier.values.firstWhere(
      (value) => value.name == identifier,
    );
    return _Offering(identifier, prices[tier]);
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
  const _Offering(this.identifier, this.price);

  @override
  final String identifier;
  final String? price;

  @override
  bool hasPackage(String identifier) => identifier == r'$rc_monthly';

  @override
  String? packagePrice(String identifier) =>
      hasPackage(identifier) ? price : null;
}
