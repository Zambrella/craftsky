import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/pages/subscription_page.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-016 owner actions remain reachable at 320px and 2x text', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(320, 568);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            textScaler: const TextScaler.linear(2),
          ),
          child: child!,
        ),
        home: SubscriptionPageView(
          model: SubscriptionPageModel(
            role: SubscriptionPageRole.owner,
            access: SubscriptionAccess(
              did: Did.parse('did:plc:alice'),
              effectiveTier: SubscriptionTier.plus,
              givesAccess: true,
            ),
            billingAvailability: BillingAvailability.available,
            billingState: _state(),
            onPurchase: (_) {},
            onRestore: () {},
            onManage: () {},
            onRefresh: () {},
            onAssign: (_) {},
            onUnassign: (_) {},
          ),
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(
      find.bySemanticsLabel(RegExp(r'Plus: Active\. View details')),
      findsOneWidget,
    );
    for (final finder in [
      find.byKey(const ValueKey('subscription-view-details-plus')),
      find.text('Assign license'),
      find.byKey(const ValueKey('subscription-view-details-business')),
      find.text('Restore purchases'),
      find.text('Manage subscription'),
    ]) {
      await tester.ensureVisible(finder);
      await tester.pump();
      expect(finder, findsOneWidget);
      expect(tester.getSize(finder).height, greaterThan(0));
    }
    expect(tester.takeException(), isNull);
  });
}

BillingState _state() => const BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  subscriptions: [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000001',
      productId: 'plus',
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
      tier: SubscriptionTier.plus,
      assignable: true,
      anomaly: 'none',
    ),
  ],
);
