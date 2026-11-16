import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/pages/subscription_page.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_pending_operation_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_feature_lock.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../test_support/recording_error_reporter.dart';

void main() {
  testWidgets('pending Stripe payment opens portal instead of native billing', (
    tester,
  ) async {
    final harness = _Harness(initialTier: SubscriptionTier.business);
    harness.api
      ..store = 'stripe'
      ..paidAccess = false
      ..pendingPayment = true;
    addTearDown(harness.dispose);
    await harness.pump(tester);
    await _tap(tester, 'manage');
    await tester.pumpAndSettle();

    expect(harness.managementURLs.single.host, 'billing.stripe.com');
    expect(harness.service.effectCalls, 0);
    expect(harness.pending, isNull);
  });
  testWidgets(
    'Business paywall can restore Plus without a pending Business purchase',
    (tester) async {
      final harness = _Harness();
      addTearDown(harness.dispose);
      final presentation = Completer<DirectPaywallResult>();
      harness.service.paywall = () => presentation.future;
      await harness.pump(tester);
      final business = find.byKey(
        const ValueKey('subscription-view-details-business'),
      );
      await tester.ensureVisible(business);
      await tester.tap(business);
      await tester.pump();
      expect(harness.service.effectCalls, 1);

      harness.api.tier = SubscriptionTier.plus;
      presentation.complete(DirectPaywallResult.restored);
      await tester.pumpAndSettle();

      expect(harness.pending, isNull);
      expect(harness.access.effectiveTier, SubscriptionTier.plus);
      expect(
        find.byKey(const ValueKey('subscription-status-plus')),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('subscription-status-business')),
        findsNothing,
      );
      expect(harness.service.effectCalls, 1);
    },
  );
  testWidgets('Stripe Manage opens the owner portal without native billing', (
    tester,
  ) async {
    final harness = _Harness(initialTier: SubscriptionTier.plus);
    harness.api.store = 'stripe';
    addTearDown(harness.dispose);
    await harness.pump(tester);
    await _tap(tester, 'manage');
    await tester.pumpAndSettle();
    expect(harness.managementURLs.single.host, 'billing.stripe.com');
    expect(harness.service.effectCalls, 0);
    expect(harness.pending, isNull);
  });
  testWidgets('leaving before a Stripe portal response prevents navigation', (
    tester,
  ) async {
    final harness = _Harness(initialTier: SubscriptionTier.plus);
    harness.api.store = 'stripe';
    harness.api.managementPending = Completer<Uri>();
    addTearDown(harness.dispose);
    await harness.pump(tester);
    await _tap(tester, 'manage');
    await tester.pump();
    harness.showPage.value = false;
    await tester.pump();
    harness.api.managementPending!.complete(
      Uri.parse('https://billing.stripe.com/p/session/test_owner'),
    );
    await tester.pumpAndSettle();
    expect(harness.managementURLs, isEmpty);
    expect(harness.pending, isNull);
  });

  for (final result in [
    DirectPaywallResult.cancelled,
    DirectPaywallResult.purchased,
  ]) {
    testWidgets(
      'paid feature directly opens Plus and returns to origin '
      'on ${result.name}',
      (tester) async {
        final harness = _Harness();
        addTearDown(harness.dispose);
        final presented = Completer<DirectPaywallResult>();
        harness.service.paywall = () => presented.future;
        await harness.pump(tester, featureEntry: true);
        await tester.tap(find.text('Paid feature'));
        for (
          var frame = 0;
          frame < 30 && harness.service.effectCalls == 0;
          frame++
        ) {
          await tester.pump(const Duration(milliseconds: 50));
        }
        expect(harness.service.effectCalls, 1);
        expect(find.text('Learn about subscriptions'), findsNothing);
        expect(find.byType(SubscriptionPageView), findsNothing);
        if (result == DirectPaywallResult.purchased) {
          harness.api.tier = SubscriptionTier.plus;
        }
        presented.complete(result);
        await tester.pumpAndSettle();
        expect(find.byType(SubscriptionPage), findsNothing);
        expect(find.text('Paid feature'), findsOneWidget);
        expect(harness.pending, isNull);
        expect(
          harness.access.effectiveTier,
          result == DirectPaywallResult.purchased
              ? SubscriptionTier.plus
              : SubscriptionTier.free,
        );
        expect(tester.takeException(), isNull);
      },
      skip: !subscriptionsEnabled,
    );
  }
  testWidgets(
    'cancelling feature entry while its offering loads prevents checkout',
    (tester) async {
      final harness = _Harness();
      addTearDown(harness.dispose);
      final offering = Completer<RevenueCatOffering?>();
      var reads = 0;
      harness.service.offering = (identifier) async {
        reads++;
        return reads <= 2 ? _Offering(identifier) : await offering.future;
      };
      await harness.pump(tester, featureEntry: true);
      await tester.tap(find.text('Paid feature'));
      for (var frame = 0; frame < 30 && reads < 3; frame++) {
        await tester.pump(const Duration(milliseconds: 50));
      }
      expect(reads, 3);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      offering.complete(const _Offering('plus'));
      await tester.pumpAndSettle();
      expect(harness.service.effectCalls, 0);
      expect(harness.pending, isNull);
      expect(find.text('Paid feature'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
    skip: !subscriptionsEnabled,
  );
  testWidgets(
    'feature entry rechecks promotional access before presenting a paywall',
    (tester) async {
      final harness = _Harness();
      harness.api.promotionalTier = SubscriptionTier.plus;
      addTearDown(harness.dispose);
      await harness.pump(tester, featureEntry: true);
      await tester.tap(find.text('Paid feature'));
      await tester.pumpAndSettle();
      expect(harness.service.effectCalls, 0);
      expect(find.byType(SubscriptionPage), findsNothing);
      expect(harness.access.allowsPlus, isTrue);
      expect(tester.takeException(), isNull);
    },
    skip: !subscriptionsEnabled,
  );
  testWidgets('consumed paywall failure reports its original cause once', (
    tester,
  ) async {
    final harness = _Harness();
    addTearDown(harness.dispose);
    await harness.pump(tester);
    final cause = StateError('PRIVATE_BILLING_PAYLOAD');
    final presentation = Completer<DirectPaywallResult>();
    harness.service.paywall = () => presentation.future;
    await _tap(tester, 'purchase');
    await tester.pump();
    presentation.completeError(cause, StackTrace.current);
    await tester.pumpAndSettle();

    expect(harness.reporter.errors, [same(cause)]);
    expect(
      (harness.reporter.contexts.single.workflow!
              as PrivateOperationalFailureContext)
          .accountDid,
      'did:plc:alice',
    );
    expect(harness.pending, isNull);
    expect(find.textContaining('PRIVATE_BILLING_PAYLOAD'), findsNothing);
    expect(tester.takeException(), isNull);
  });

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
      expect(find.text('Manage'), findsNothing);
      await tester.ensureVisible(find.text('Subscribe to Plus'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Subscribe to Plus'));
      await tester.pumpAndSettle();
      expect(harness.service.effectCalls, 1);
      expect(harness.pending, isNull);
      expect(harness.api.tier, SubscriptionTier.plus);
      expect(harness.access.effectiveTier, SubscriptionTier.business);
      expect(find.text('Subscribe to Plus'), findsNothing);
      expect(find.text('Manage'), findsOneWidget);
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
      await tester.ensureVisible(
        find.byKey(const ValueKey('subscription-view-details-business')),
      );
      await tester.pumpAndSettle();
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
              ? 'Current access: Standard'
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
    _ => find.text('Manage'),
  };
  await tester.ensureVisible(finder);
  await tester.pumpAndSettle();
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
        subscriptionManagementLauncherProvider.overrideWithValue((uri) async {
          managementURLs.add(uri);
          return true;
        }),
        errorReporterProvider.overrideWithValue(reporter),
      ],
    );
    // AppShell keeps this provider alive while the settings route is open.
    accessListener = container.listen(
      subscriptionAccessProvider(registry.activeLease!.session),
      (_, _) {},
    );
  }

  final api = _Api();
  final reporter = RecordingErrorReporter();
  final managementURLs = <Uri>[];
  final ValueNotifier<bool> showPage = ValueNotifier(true);
  late final _Service service;
  late final SessionRegistry registry;
  late final ProviderContainer container;
  late final ProviderSubscription<AsyncValue<SubscriptionAccess>>
  accessListener;
  bool _disposed = false;

  SubscriptionAccess get access => container
      .read(subscriptionAccessProvider(registry.activeLease!.session))
      .requireValue;
  PendingSubscriptionOperation? get pending =>
      container.read(subscriptionPendingOperationProvider);

  Future<void> pump(WidgetTester tester, {bool featureEntry = false}) async {
    await tester.pumpWidget(
      _HarnessLifetime(
        onDispose: dispose,
        child: UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ValueListenableBuilder<bool>(
              valueListenable: showPage,
              builder: (_, visible, _) => !visible
                  ? const SizedBox()
                  : featureEntry
                  ? Builder(
                      builder: (context) => Scaffold(
                        body: TextButton(
                          onPressed: () =>
                              showPlusFeaturePrompt(context, 'Folders'),
                          child: const Text('Paid feature'),
                        ),
                      ),
                    )
                  : const SubscriptionPage(),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  void dispose() {
    if (_disposed) return;
    _disposed = true;
    accessListener.close();
    container.dispose();
    showPage.dispose();
  }
}

class _HarnessLifetime extends StatefulWidget {
  const new({required this.onDispose, required this.child});
  final VoidCallback onDispose;
  final Widget child;
  @override
  State<_HarnessLifetime> createState() => _HarnessLifetimeState();
}

class _HarnessLifetimeState extends State<_HarnessLifetime> {
  @override
  Widget build(BuildContext context) => widget.child;
  @override
  void dispose() {
    widget.onDispose();
    super.dispose();
  }
}

const _customer = '20000000-0000-4000-8000-000000000001';
const _subscription = '30000000-0000-4000-8000-000000000001';

final class _Api implements SubscriptionApi, SubscriptionManagementApi {
  String store = 'app_store';
  Completer<Uri>? managementPending;
  @override
  Future<Uri> managementURL() async => managementPending == null
      ? Uri.parse('https://billing.stripe.com/p/session/test_owner#capability')
      : await managementPending!.future;
  SubscriptionTier tier = SubscriptionTier.free;
  bool paidAccess = true;
  bool pendingPayment = false;
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
            BillingSubscription(
              id: _subscription,
              productId: 'plus',
              store: store,
              status: 'active',
              givesAccess: paidAccess,
              pendingPayment: pendingPayment,
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
    effectiveTier: !paidAccess
        ? promotionalTier ?? SubscriptionTier.free
        : promotionalTier != null && promotionalTier!.index > tier.index
        ? promotionalTier!
        : tier,
    givesAccess:
        (tier != SubscriptionTier.free && paidAccess) ||
        promotionalTier != null,
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
