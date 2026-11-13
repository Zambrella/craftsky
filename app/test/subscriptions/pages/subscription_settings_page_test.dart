import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/pages/subscription_page.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'free account chooses a tier without a payer or assignment step',
    (tester) async {
      var selected = SubscriptionTier.free;
      await _pump(tester, _model(onPurchase: (tier) => selected = tier));
      expect(find.textContaining('alternative plans'), findsOneWidget);
      expect(find.textContaining('billing owner'), findsNothing);
      expect(find.textContaining('Assign'), findsNothing);
      expect(find.textContaining('web'), findsNothing);
      await tester.tap(
        find.byKey(const ValueKey('subscription-view-details-business')),
      );
      expect(selected, SubscriptionTier.business);
    },
  );
  testWidgets(
    'native subscriber manages existing plan while replacements are gated',
    (tester) async {
      var selected = SubscriptionTier.free;
      await _pump(
        tester,
        _model(store: 'app_store', onPurchase: (tier) => selected = tier),
      );
      expect(find.text('Manage subscription'), findsOneWidget);
      expect(find.text('Restore purchases'), findsOneWidget);
      expect(find.textContaining('Assigned to'), findsNothing);
      await tester.tap(
        find.byKey(const ValueKey('subscription-view-details-business')),
      );
      expect(selected, SubscriptionTier.free);
    },
  );
  testWidgets(
    'Stripe access hides native checkout and external purchase promotion',
    (tester) async {
      await _pump(tester, _model(store: 'stripe'));
      expect(find.text('Current access: Plus'), findsOneWidget);
      expect(find.text('View details'), findsNothing);
      expect(find.text('Manage subscription'), findsNothing);
      expect(find.textContaining('web'), findsNothing);
      expect(find.textContaining('Stripe'), findsNothing);
    },
  );
  testWidgets(
    'store conflict explains ownership without guessing a handle',
    (tester) async {
      await _pump(
        tester,
        _model(notice: SubscriptionOperationNotice.storeAccountConflict),
      );
      expect(
        find.textContaining('linked to another CraftSky account'),
        findsOneWidget,
      );
      expect(find.textContaining('@'), findsNothing);
      expect(find.textContaining('website'), findsNothing);
    },
  );
}

SubscriptionPageModel _model({
  String? store,
  ValueChanged<SubscriptionTier>? onPurchase,
  SubscriptionOperationNotice? notice,
}) => SubscriptionPageModel(
  access: SubscriptionAccess(
    did: Did.parse('did:plc:alice'),
    effectiveTier: store == null
        ? SubscriptionTier.free
        : SubscriptionTier.plus,
    givesAccess: store != null,
  ),
  billingAvailability: BillingAvailability.available,
  nativeStore: 'app_store',
  billingState: BillingState(
    billingAccountId: '10000000-0000-4000-8000-000000000001',
    revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
    requestedGeneration: 1,
    reconciledGeneration: 1,
    reconciliationStale: false,
    subscriptions: store == null
        ? []
        : [
            BillingSubscription(
              id: '30000000-0000-4000-8000-000000000001',
              productId: 'plus',
              store: store,
              status: 'active',
              givesAccess: true,
              pendingPayment: false,
              anomaly: 'none',
              autoRenewalStatus: 'will_renew',
            ),
          ],
    licenses: store == null
        ? []
        : [
            BillingLicense(
              id: '40000000-0000-4000-8000-000000000001',
              subscriptionId: '30000000-0000-4000-8000-000000000001',
              tier: SubscriptionTier.plus,
              assignedDid: Did.parse('did:plc:alice'),
              assignable: false,
              anomaly: 'none',
            ),
          ],
  ),
  operationNotice: notice,
  onPurchase: onPurchase ?? (_) {},
  onRestore: () {},
  onManage: () {},
  onRefresh: () {},
);

Future<void> _pump(WidgetTester tester, SubscriptionPageModel model) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: appLocalizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: SubscriptionPageView(model: model),
    ),
  );
  await tester.pumpAndSettle();
}
