import 'dart:ui' show SemanticsAction;

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
    'pending checkout cards explain the block and offer safe cancellation',
    (tester) async {
      var canceled = 0;
      await _pump(
        tester,
        _model(pendingCheckout: true, onCancelCheckout: () => canceled++),
      );
      expect(
        find.textContaining('A purchase is still pending'),
        findsOneWidget,
      );
      final card = find.byKey(const ValueKey('subscription-tier-plus'));
      await tester.ensureVisible(card);
      await tester.tapAt(tester.getTopLeft(card) + const Offset(24, 24));
      await tester.pumpAndSettle();
      final dialog = find.byType(AlertDialog);
      expect(dialog, findsOneWidget);
      await tester.tap(
        find.descendant(
          of: dialog,
          matching: find.text('Cancel pending purchase'),
        ),
      );
      await tester.pumpAndSettle();
      expect(canceled, 1);
    },
  );

  testWidgets(
    'entire tier cards activate once and expose one accessible action',
    (tester) async {
      final selected = <SubscriptionTier>[];
      await _pump(tester, _model(onPurchase: selected.add));
      final semantics = tester.ensureSemantics();

      for (final tier in [SubscriptionTier.plus, SubscriptionTier.business]) {
        final card = find.byKey(ValueKey('subscription-tier-${tier.name}'));
        await tester.ensureVisible(card);
        await tester.tapAt(tester.getTopLeft(card) + const Offset(24, 24));
        expect(selected.last, tier);
        final label = tier == SubscriptionTier.plus ? 'Plus' : 'Business';
        final node = tester.getSemantics(
          find.bySemanticsLabel(RegExp('^$label:')),
        );
        expect(node.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);
      }
      expect(selected, [SubscriptionTier.plus, SubscriptionTier.business]);
      semantics.dispose();
    },
  );

  testWidgets('disabled tier cards do not dispatch from their body', (
    tester,
  ) async {
    final selected = <SubscriptionTier>[];
    await _pump(
      tester,
      _model(onPurchase: selected.add, actionsEnabled: false),
    );
    for (final tier in [SubscriptionTier.plus, SubscriptionTier.business]) {
      final card = find.byKey(ValueKey('subscription-tier-${tier.name}'));
      await tester.ensureVisible(card);
      expect(tester.widget<InkWell>(card).onTap, isNull);
      await tester.tapAt(tester.getTopLeft(card) + const Offset(24, 24));
    }
    expect(selected, isEmpty);
  });

  testWidgets(
    'complimentary Plus shows its expiry and both paid subscribe choices',
    (tester) async {
      var selected = SubscriptionTier.free;
      await _pump(
        tester,
        _model(promo: true, onPurchase: (tier) => selected = tier),
      );
      expect(find.text('Current access: Plus'), findsOneWidget);
      expect(find.text('Complimentary access'), findsOneWidget);
      expect(find.textContaining('Plus until'), findsOneWidget);
      expect(find.textContaining('paid billing immediately'), findsOneWidget);
      expect(find.text('Manage subscription'), findsNothing);
      expect(find.text('Subscribe to Plus'), findsOneWidget);
      await tester.ensureVisible(find.text('Subscribe to Business'));
      await tester.tap(find.text('Subscribe to Business'));
      expect(selected, SubscriptionTier.business);
      await tester.ensureVisible(find.text('Subscribe to Plus'));
      await tester.tap(find.text('Subscribe to Plus'));
      expect(selected, SubscriptionTier.plus);
    },
  );

  testWidgets(
    'paid plus complimentary access keeps management without purchase timing',
    (tester) async {
      await _pump(tester, _model(store: 'app_store', promo: true));
      expect(find.text('Complimentary access'), findsOneWidget);
      expect(find.text('Manage subscription'), findsOneWidget);
      expect(find.textContaining('paid billing immediately'), findsNothing);
      expect(find.text('Subscribe to Plus'), findsNothing);
    },
  );
  testWidgets('complimentary lifetime access has a clear no-expiry label', (
    tester,
  ) async {
    await _pump(tester, _model(promo: true, lifetimePromo: true));
    expect(find.text('Plus with no expiry date'), findsOneWidget);
    expect(find.textContaining('Plus until'), findsNothing);
    expect(find.text('Subscribe to Plus'), findsOneWidget);
  });
  testWidgets(
    'free account chooses a tier without a payer or assignment step',
    (tester) async {
      var selected = SubscriptionTier.free;
      await _pump(tester, _model(onPurchase: (tier) => selected = tier));
      expect(
        find.textContaining('Business includes everything in Plus'),
        findsOneWidget,
      );
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
  bool promo = false,
  bool lifetimePromo = false,
  bool actionsEnabled = true,
  bool pendingCheckout = false,
  VoidCallback? onCancelCheckout,
  ValueChanged<SubscriptionTier>? onPurchase,
  SubscriptionOperationNotice? notice,
}) => SubscriptionPageModel(
  access: SubscriptionAccess(
    did: Did.parse('did:plc:alice'),
    effectiveTier: store == null && !promo
        ? SubscriptionTier.free
        : SubscriptionTier.plus,
    givesAccess: store != null || promo,
  ),
  billingAvailability: BillingAvailability.available,
  nativeStore: 'app_store',
  billingState: BillingState(
    billingAccountId: '10000000-0000-4000-8000-000000000001',
    revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
    requestedGeneration: 1,
    reconciledGeneration: 1,
    reconciliationStale: false,
    webCheckoutPending: pendingCheckout,
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
    promotionalGrants: promo
        ? [
            PromotionalGrant(
              id: '50000000-0000-4000-8000-000000000001',
              tier: SubscriptionTier.plus,
              givesAccess: true,
              expiresAt: lifetimePromo ? null : DateTime.utc(2027, 5, 13),
            ),
          ]
        : const [],
  ),
  operationNotice: notice,
  onPurchase: actionsEnabled ? onPurchase ?? (_) {} : null,
  onRestore: () {},
  onManage: () {},
  onRefresh: () {},
  onCancelCheckout: onCancelCheckout,
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
