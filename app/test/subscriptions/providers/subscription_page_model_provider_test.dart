import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/recording_error_reporter.dart';

void main() {
  testWidgets('billing refreshes at a grant expiry before paid access ends', (
    tester,
  ) async {
    var now = DateTime.utc(2026, 11, 16);
    final api = _Api('did:plc:alice', SubscriptionTier.business)
      ..promotionalGrants = [
        PromotionalGrant(
          id: '50000000-0000-4000-8000-000000000001',
          tier: SubscriptionTier.plus,
          givesAccess: true,
          expiresAt: now.add(const Duration(seconds: 30)),
        ),
      ];
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(_registry()),
        ),
        revenueCatServiceProvider.overrideWithValue(
          const UnavailableRevenueCatService(),
        ),
        subscriptionAccessClockProvider.overrideWithValue(() => now),
        subscriptionRepositoryProvider.overrideWith((_, _) async => api),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final listener = container.listen(subscriptionPageModelProvider, (_, _) {});
    await tester.pump();
    final initial = await container.read(subscriptionPageModelProvider.future);
    expect(initial.billingState?.promotionalGrants, hasLength(1));
    expect(api.ownerReads, 1);

    api.promotionalGrants = [];
    now = now.add(const Duration(seconds: 30));
    await tester.pump(const Duration(seconds: 30));
    await tester.pump();
    final refreshed = await container.read(
      subscriptionPageModelProvider.future,
    );

    expect(api.ownerReads, 2);
    expect(refreshed.billingState?.promotionalGrants, isEmpty);
    expect(refreshed.access.allowsBusiness, isTrue);
    listener.close();
    container.dispose();
  });
  test('consumed price failure reports its original cause once', () async {
    final error = Exception('PRIVATE_BILLING_PAYLOAD');
    final reporter = RecordingErrorReporter();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(_registry()),
        ),
        errorReporterProvider.overrideWithValue(reporter),
        revenueCatServiceProvider.overrideWithValue(
          _RevenueCat()..offeringError = error,
        ),
        subscriptionRepositoryProvider.overrideWith(
          (_, _) async => _Api(
            'did:plc:alice',
            SubscriptionTier.plus,
          ),
        ),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final subscription = container.listen(
      subscriptionPageModelProvider,
      (_, _) {},
    );
    addTearDown(subscription.close);

    final model = await container.read(subscriptionPageModelProvider.future);

    expect(model.tierPrices, isEmpty);
    expect(reporter.errors, [same(error)]);
    expect(
      reporter.contexts.single.workflow,
      isA<PrivateOperationalFailureContext>(),
    );
  });

  for (final phase in ['registry', 'access', 'billing']) {
    test(
      'closed subscription page cancels pending $phase work '
      'without reading a disposed ref',
      () async {
        final gate = Completer<void>();
        final registry = _registry();
        final storage = _Storage(
          registry,
          readGate: phase == 'registry' ? gate.future : null,
        );
        final api = _Api(
          'did:plc:alice',
          SubscriptionTier.plus,
          accessGate: phase == 'access' ? gate.future : null,
          ownerGate: phase == 'billing' ? gate.future : null,
        );
        final revenueCat = _RevenueCat();
        final container = ProviderContainer.test(
          retry: (_, _) => null,
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(storage),
            revenueCatServiceProvider.overrideWithValue(revenueCat),
            subscriptionRepositoryProvider.overrideWith(
              (ref, account) async => api,
            ),
          ],
        );
        final subscription = container.listen(
          subscriptionPageModelProvider,
          (_, _) {},
        );
        final pending = container.read(subscriptionPageModelProvider.future);
        await switch (phase) {
          'registry' => storage.readStarted.future,
          'access' => api.accessStarted.future,
          _ => api.ownerStarted.future,
        };
        final identityCalls = revenueCat.identityCalls;
        final ownerReads = api.ownerReads;
        final cancelled = expectLater(
          pending,
          throwsA(
            anyOf(
              isA<AccountOperationCancellation>(),
              isA<StateError>().having(
                (error) => error.message,
                'Riverpod disposed pending future',
                contains('disposed during loading state'),
              ),
            ),
          ),
        );
        subscription.close();
        await container.pump();
        gate.complete();
        await cancelled;
        await container.pump();
        expect(revenueCat.identityCalls, identityCalls);
        expect(api.ownerReads, ownerReads);
      },
    );
  }

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
  test('a conflicting local store does not hide AppView paid access', () async {
    final registry = _registry();
    final api = _Api('did:plc:alice', SubscriptionTier.plus);
    final revenueCat = _ConflictingRevenueCat();
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        revenueCatServiceProvider.overrideWithValue(revenueCat),
        subscriptionRepositoryProvider.overrideWith(
          (ref, account) async => api,
        ),
      ],
    );
    addTearDown(container.dispose);
    container.listen(subscriptionPageModelProvider, (_, _) {});
    final model = await container.read(subscriptionPageModelProvider.future);
    expect(model.access.allowsPlus, isTrue);
    expect(model.billingState?.revenueCatAppUserId, api.revenueCatAppUserId);
    expect(revenueCat.identityCalls, 0);
    expect(revenueCat.identifyCalls, 0);
    expect(revenueCat.offeringCalls, isEmpty);
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
    .setBillingCustomer(
      'did:plc:alice',
      '20000000-0000-4000-8000-000000000001',
    );

final class _Storage implements SessionRegistryStorage {
  new(this.value, {this.readGate});
  SessionRegistry value;
  final Future<void>? readGate;
  final readStarted = Completer<void>();

  @override
  Future<SessionRegistry> read() async {
    if (!readStarted.isCompleted) readStarted.complete();
    if (readGate != null) await readGate;
    return value;
  }

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Api implements SubscriptionApi {
  new(
    this.did,
    this.tier, {
    this.revenueCatAppUserId = '20000000-0000-4000-8000-000000000001',
    this.accessGate,
    this.ownerGate,
  });
  final String did;
  final SubscriptionTier tier;
  final String revenueCatAppUserId;
  int ownerReads = 0;
  List<PromotionalGrant> promotionalGrants = [];
  final Future<void>? accessGate;
  final Future<void>? ownerGate;
  final accessStarted = Completer<void>();
  final ownerStarted = Completer<void>();

  @override
  Future<SubscriptionAccess> getAccess() async {
    if (!accessStarted.isCompleted) accessStarted.complete();
    if (accessGate != null) await accessGate;
    return SubscriptionAccess(
      did: Did.parse(did),
      effectiveTier: tier,
      givesAccess: tier != SubscriptionTier.free,
    );
  }

  @override
  Future<BillingState> getBillingAccount() async {
    ownerReads++;
    if (!ownerStarted.isCompleted) ownerStarted.complete();
    if (ownerGate != null) await ownerGate;
    return BillingState(
      billingAccountId: '10000000-0000-4000-8000-000000000001',
      revenueCatAppUserId: revenueCatAppUserId,
      requestedGeneration: 1,
      reconciledGeneration: 1,
      reconciliationStale: false,
      subscriptions: [],
      licenses: [],
      promotionalGrants: promotionalGrants,
    );
  }

  @override
  Future<BillingState> ensureBillingAccount() => getBillingAccount();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();
}

class _RevenueCat implements RevenueCatService {
  new();
  String appUserId = '20000000-0000-4000-8000-000000000001';
  Exception? offeringError;
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
    if (identifier == 'plus' && offeringError != null) throw offeringError!;
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

final class _ConflictingRevenueCat extends _RevenueCat
    implements NativeStoreOwnershipService {
  @override
  String get store => 'app_store';
  @override
  bool get supportsPlanChanges => true;
  @override
  Future<void> verifyStoreOwnership(
    String appUserId, {
    bool requireNoActiveSubscription = false,
  }) async => throw const StoreAccountConflictException();
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
