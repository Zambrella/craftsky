import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/pages/subscription_page.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_pending_operation_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'complimentary Business can buy Plus without repeated payment or pending',
    (tester) async {
      final harness = _Harness();
      harness.api.promotionalTier = SubscriptionTier.business;
      addTearDown(harness.dispose);
      await harness.pump(tester);
      expect(harness.access.effectiveTier, SubscriptionTier.business);
      expect(find.text('Subscribe to Plus'), findsOneWidget);
      expect(find.text('Subscribe to Business'), findsOneWidget);
      expect(find.text('Manage subscription'), findsNothing);
      await tester.ensureVisible(find.text('Subscribe to Plus'));
      await tester.tap(find.text('Subscribe to Plus'));
      await tester.pumpAndSettle();
      expect(harness.service.effectCalls, 1);
      expect(harness.pending, isNull);
      expect(harness.api.tier, SubscriptionTier.plus);
      expect(harness.access.effectiveTier, SubscriptionTier.business);
      expect(find.text('Subscribe to Plus'), findsNothing);
      expect(find.text('Manage subscription'), findsOneWidget);
      expect(find.textContaining('paid billing immediately'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets(
    'restart with AppView Free and owned store plan '
    'blocks a second purchase',
    (tester) async {
      // A fresh container has no pending operation, as after process death.
      final harness = _Harness();
      harness.service.storeHasActiveSubscription = true;
      addTearDown(harness.dispose);
      await harness.pump(tester);
      expect(harness.pending, isNull);
      expect(harness.access.effectiveTier, SubscriptionTier.free);
      await tester.tap(
        find.byKey(const ValueKey('subscription-view-details-business')),
      );
      await tester.pumpAndSettle();
      expect(harness.service.effectCalls, 0);
      expect(harness.pending, isNull);
      expect(harness.api.generation, 2);
      expect(
        find.textContaining('This store already has a subscription'),
        findsOneWidget,
      );
      expect(
        find.textContaining('Your provider action completed'),
        findsNothing,
      );
      expect(find.text('Refresh status'), findsOneWidget);
      await _tap(tester, 'restore');
      await tester.pumpAndSettle();
      expect(harness.service.effectCalls, 1);
      expect(harness.access.effectiveTier, SubscriptionTier.plus);
      expect(harness.pending, isNull);
    },
  );

  for (final operation in ['purchase', 'restore', 'manage']) {
    testWidgets('$operation refreshes the shared access projection', (
      tester,
    ) async {
      final harness = _Harness(
        initialTier: operation == 'manage'
            ? SubscriptionTier.plus
            : SubscriptionTier.free,
      );
      addTearDown(harness.dispose);
      await harness.pump(tester);
      expect(
        harness.access.effectiveTier,
        operation == 'manage' ? SubscriptionTier.plus : SubscriptionTier.free,
      );
      await _tap(tester, operation);
      await tester.pumpAndSettle();
      expect(harness.pending, isNull);
      expect(
        harness.access.effectiveTier,
        operation == 'manage' ? SubscriptionTier.free : SubscriptionTier.plus,
      );
      expect(
        find.text(
          operation == 'manage'
              ? 'Current access: Free'
              : 'Current access: Plus',
        ),
        findsOneWidget,
      );
    });

    testWidgets(
      'leaving during $operation preparation releases the operation',
      (tester) async {
        final harness = _Harness(
          initialTier: operation == 'manage'
              ? SubscriptionTier.plus
              : SubscriptionTier.free,
        );
        addTearDown(harness.dispose);
        await harness.pump(tester);
        final response = Completer<BillingState>();
        harness.api.billingRead = () => response.future;
        await _tap(tester, operation);
        await tester.pump();
        expect(
          harness.pending?.phase,
          PendingSubscriptionOperationPhase.nativeInFlight,
        );
        harness.showPage.value = false;
        await tester.pump();
        harness.api.billingRead = null;
        response.complete(harness.api.state);
        await tester.pumpAndSettle();
        expect(harness.pending, isNull);
        harness.showPage.value = true;
        await tester.pumpAndSettle();
        await _tap(tester, operation);
        await tester.pumpAndSettle();
        expect(harness.service.effectCalls, 1);
        expect(harness.pending, isNull);
        expect(tester.takeException(), isNull);
      },
    );

    testWidgets(
      'leaving during final $operation ownership lookup stops effects',
      (tester) async {
        final harness = _Harness(
          initialTier: operation == 'manage'
              ? SubscriptionTier.plus
              : SubscriptionTier.free,
        );
        addTearDown(harness.dispose);
        await harness.pump(tester);
        final entered = Completer<void>();
        final release = Completer<void>();
        final initialChecks = harness.service.ownershipCalls;
        harness.service.ownership = () async {
          if (harness.service.ownershipCalls == initialChecks + 2) {
            entered.complete();
            await release.future;
          }
        };
        await _tap(tester, operation);
        await tester.pump();
        expect(entered.isCompleted, isTrue);
        harness.showPage.value = false;
        await tester.pump();
        release.complete();
        await tester.pumpAndSettle();
        expect(harness.service.effectCalls, 0);
        expect(harness.pending, isNull);
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('leaving while the purchase offering loads stops checkout', (
    tester,
  ) async {
    final harness = _Harness();
    addTearDown(harness.dispose);
    await harness.pump(tester);
    final response = Completer<RevenueCatOffering?>();
    harness.service.offering = (_) => response.future;
    await _tap(tester, 'purchase');
    await tester.pump();
    harness.showPage.value = false;
    await tester.pump();
    response.complete(const _Offering('plus'));
    await tester.pumpAndSettle();
    expect(harness.service.effectCalls, 0);
    expect(harness.pending, isNull);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'provider-confirmed work survives leaving and reconciles on return',
    (tester) async {
      final harness = _Harness();
      addTearDown(harness.dispose);
      await harness.pump(tester);
      final response = Completer<DirectPaywallResult>();
      harness.service.paywall = () => response.future;
      await _tap(tester, 'purchase');
      await tester.pump();
      await tester.pump();
      expect(harness.service.effectCalls, 1);
      harness.showPage.value = false;
      await tester.pump();
      harness.api.tier = SubscriptionTier.plus;
      response.complete(DirectPaywallResult.purchased);
      await tester.pumpAndSettle();
      expect(harness.pending?.canReconcile, isTrue);
      harness.showPage.value = true;
      await tester.pumpAndSettle();
      await tester.tap(find.text('Refresh status'));
      await tester.pumpAndSettle();
      expect(harness.pending, isNull);
      expect(harness.service.effectCalls, 1);
      expect(harness.access.effectiveTier, SubscriptionTier.plus);
      expect(tester.takeException(), isNull);
    },
  );
}

Future<void> _tap(WidgetTester tester, String operation) async {
  final finder = switch (operation) {
    'purchase' => find.byKey(
      const ValueKey('subscription-view-details-plus'),
    ),
    'restore' => find.text('Restore purchases'),
    _ => find.text('Manage subscription'),
  };
  await tester.ensureVisible(finder);
  await tester.tap(finder);
}

final class _Harness {
  new({SubscriptionTier initialTier = SubscriptionTier.free}) {
    api.tier = initialTier;
    service = _Service(api);
    registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .setBillingCustomer('did:plc:alice', _customer);
    container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        subscriptionRepositoryProvider.overrideWith(
          (ref, account) async => api,
        ),
        revenueCatServiceProvider.overrideWithValue(service),
      ],
    );
    // AppShell keeps this provider alive while the settings route is open.
    accessListener = container.listen(
      subscriptionAccessProvider(registry.activeLease!.session),
      (_, _) {},
    );
  }

  final api = _Api();
  final ValueNotifier<bool> showPage = ValueNotifier(true);
  late final _Service service;
  late final SessionRegistry registry;
  late final ProviderContainer container;
  late final ProviderSubscription<AsyncValue<SubscriptionAccess>>
  accessListener;

  SubscriptionAccess get access => container
      .read(subscriptionAccessProvider(registry.activeLease!.session))
      .requireValue;
  PendingSubscriptionOperation? get pending =>
      container.read(subscriptionPendingOperationProvider);

  Future<void> pump(WidgetTester tester) async {
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ValueListenableBuilder<bool>(
            valueListenable: showPage,
            builder: (_, visible, _) =>
                visible ? const SubscriptionPage() : const SizedBox(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  void dispose() {
    accessListener.close();
    container.dispose();
    showPage.dispose();
  }
}

const _customer = '20000000-0000-4000-8000-000000000001';
const _subscription = '30000000-0000-4000-8000-000000000001';

final class _Api implements SubscriptionApi {
  SubscriptionTier tier = SubscriptionTier.free;
  SubscriptionTier? promotionalTier;
  int generation = 1;
  Future<BillingState> Function()? billingRead;

  BillingState get state => BillingState(
    billingAccountId: '10000000-0000-4000-8000-000000000001',
    revenueCatAppUserId: _customer,
    requestedGeneration: generation,
    reconciledGeneration: generation,
    reconciliationStale: false,
    subscriptions: tier == SubscriptionTier.free
        ? []
        : [
            const BillingSubscription(
              id: _subscription,
              productId: 'plus',
              store: 'app_store',
              status: 'active',
              givesAccess: true,
              pendingPayment: false,
              anomaly: 'none',
              autoRenewalStatus: 'will_renew',
            ),
          ],
    licenses: tier == SubscriptionTier.free
        ? []
        : [
            BillingLicense(
              id: '40000000-0000-4000-8000-000000000001',
              subscriptionId: _subscription,
              tier: tier,
              assignable: false,
              anomaly: 'none',
              assignedDid: Did.parse('did:plc:alice'),
            ),
          ],
    promotionalGrants: promotionalTier == null
        ? const []
        : [
            PromotionalGrant(
              id: '50000000-0000-4000-8000-000000000001',
              tier: promotionalTier!,
              givesAccess: true,
              expiresAt: DateTime.utc(2027, 5, 13),
            ),
          ],
  );

  @override
  Future<BillingState> ensureBillingAccount() async => state;
  @override
  Future<BillingState> getBillingAccount() async =>
      billingRead == null ? state : await billingRead!();
  @override
  Future<void> requestReconciliation() async => generation++;
  @override
  Future<SubscriptionAccess> getAccess() async => SubscriptionAccess(
    did: Did.parse('did:plc:alice'),
    effectiveTier:
        promotionalTier != null && promotionalTier!.index > tier.index
        ? promotionalTier!
        : tier,
    givesAccess: tier != SubscriptionTier.free || promotionalTier != null,
    assignedTier: promotionalTier != null && promotionalTier!.index > tier.index
        ? promotionalTier
        : tier == SubscriptionTier.free
        ? null
        : tier,
  );
}

final class _Service implements RevenueCatService, NativeStoreOwnershipService {
  new(this.api);
  final _Api api;
  int effectCalls = 0;
  int ownershipCalls = 0;
  bool storeHasActiveSubscription = false;
  Future<void> Function()? ownership;
  Future<RevenueCatOffering?> Function(String identifier)? offering;
  Future<DirectPaywallResult> Function()? paywall;

  @override
  BillingAvailability get availability => BillingAvailability.available;
  @override
  String get store => 'app_store';
  @override
  bool get supportsPlanChanges => false;
  @override
  Future<void> verifyStoreOwnership(
    String appUserId, {
    bool requireNoActiveSubscription = false,
  }) async {
    ownershipCalls++;
    await ownership?.call();
    if (requireNoActiveSubscription && storeHasActiveSubscription) {
      throw const StoreSubscriptionExistsException();
    }
  }

  @override
  Future<RevenueCatIdentity> currentIdentity() async =>
      const RevenueCatIdentity.identified(_customer);
  @override
  Future<void> identify(String appViewUuid) async {}
  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async =>
      offering == null ? _Offering(identifier) : await offering!(identifier);
  @override
  Future<DirectPaywallResult> presentPaywall(
    RevenueCatOffering offering,
  ) async {
    effectCalls++;
    if (paywall != null) return await paywall!();
    api.tier = SubscriptionTier.plus;
    return DirectPaywallResult.purchased;
  }

  @override
  Future<void> restorePurchases() async {
    effectCalls++;
    api.tier = SubscriptionTier.plus;
  }

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) async {
    effectCalls++;
    api.tier = SubscriptionTier.free;
    onEvent(CustomerCenterEvent.managementOptionSelected);
  }
}

final class _Offering implements RevenueCatOffering {
  const new(this.identifier);
  @override
  final String identifier;
  @override
  bool hasPackage(String identifier) => true;
  @override
  String? packagePrice(String identifier) => '£2.49';
}

final class _Storage implements SessionRegistryStorage {
  new(this.registry);
  SessionRegistry registry;
  @override
  Future<SessionRegistry> read() async => registry;
  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}
