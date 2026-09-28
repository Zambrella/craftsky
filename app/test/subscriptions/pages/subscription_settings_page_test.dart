import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/pages/subscription_page.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-004 beneficiary sees self access and no owner details', (
    tester,
  ) async {
    var switched = false;
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.beneficiary,
        access: _access(
          SubscriptionTier.free,
          assignedTier: SubscriptionTier.plus,
        ),
        ownerRetained: true,
        billingAvailability: BillingAvailability.available,
        onSwitchToOwner: () => switched = true,
      ),
    );

    expect(find.textContaining('Plus is assigned'), findsOneWidget);
    expect(find.text('Restore purchases'), findsNothing);
    expect(find.text('Manage subscription'), findsNothing);
    expect(find.textContaining('20000000-'), findsNothing);
    await tester.tap(find.text('Switch to billing owner'));
    expect(switched, isTrue);
  });

  testWidgets('AT-013 dormant assignment is visible for every page role', (
    tester,
  ) async {
    final access = _access(
      SubscriptionTier.free,
      assignedTier: SubscriptionTier.plus,
    );
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.neverReserved,
        access: access,
        billingAvailability: BillingAvailability.available,
      ),
    );

    expect(find.text('Current access: Free'), findsOneWidget);
    expect(find.textContaining('Plus is assigned'), findsOneWidget);

    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: access,
        billingAvailability: BillingAvailability.available,
        ownerRecoveryLocked: true,
      ),
    );

    expect(find.text('Current access: Free'), findsOneWidget);
    expect(find.textContaining('Plus is assigned'), findsOneWidget);
  });

  testWidgets('AT-004 active assigned access is not described as dormant', (
    tester,
  ) async {
    for (final tier in [SubscriptionTier.plus, SubscriptionTier.business]) {
      await _pump(
        tester,
        SubscriptionPageModel(
          role: SubscriptionPageRole.neverReserved,
          access: _access(tier, assignedTier: tier),
          billingAvailability: BillingAvailability.available,
        ),
      );

      expect(find.text('Current access: ${_tierName(tier)}'), findsOneWidget);
      expect(
        find.textContaining('is not currently providing access'),
        findsNothing,
      );
    }
  });

  testWidgets('AT-013 owner joins shuffled billing rows by subscription ID', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.plus),
        billingAvailability: BillingAvailability.available,
        billingState: _ownerState(unknownBusinessStatus: true),
        assignedAccountLabels: {
          Did.parse('did:plc:bob'): '@bob.test',
        },
        onPurchase: (_) {},
        onRestore: () {},
        onManage: () {},
        onRefresh: () {},
        onAssign: (_) {},
        onUnassign: (_) {},
      ),
    );

    expect(find.text('Plus'), findsWidgets);
    expect(find.textContaining('Renews on'), findsOneWidget);
    expect(find.text('Assigned to @bob.test'), findsOneWidget);
    expect(find.text('Business'), findsWidgets);
    expect(find.text('Status unavailable'), findsOneWidget);
    expect(find.textContaining('future_status'), findsNothing);
    expect(find.text('Restore purchases'), findsOneWidget);
    expect(find.text('Manage subscription'), findsOneWidget);
    expect(find.textContaining('Product:'), findsNothing);
    expect(find.textContaining('Oct 1'), findsOneWidget);
    expect(find.textContaining('Current period:'), findsNothing);
    expect(find.textContaining('License can'), findsNothing);
    expect(find.text('Not assigned'), findsNothing);
    expect(find.textContaining('Reconciliation requested:'), findsNothing);
    expect(find.textContaining('Last reconciled:'), findsNothing);
    expect(find.text('Change assignment'), findsOneWidget);
  });

  testWidgets('AT-014 pending reconciliation offers refresh, not checkout', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _emptyState(),
        operationNotice: SubscriptionOperationNotice.pending,
        onPurchase: (_) {},
        onRestore: () {},
        onManage: () {},
        onRefresh: () {},
        onAssign: (_) {},
        onUnassign: (_) {},
      ),
    );

    expect(find.textContaining('waiting for confirmed'), findsOneWidget);
    expect(find.text('Refresh status'), findsOneWidget);
    expect(_tierDetails(SubscriptionTier.plus), findsNothing);
    expect(_tierDetails(SubscriptionTier.business), findsNothing);
  });

  testWidgets('AT-014 reconciled anomaly requires support', (tester) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _emptyState(),
        operationNotice: SubscriptionOperationNotice.supportRequired,
        onRefresh: () {},
      ),
    );

    expect(find.text('Subscription state needs support'), findsOneWidget);
    expect(find.textContaining('waiting for confirmed'), findsNothing);
  });

  testWidgets('AT-014 unresolved owner relationship requires support', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _orphanSubscriptionState(),
        onPurchase: (_) {},
      ),
    );

    expect(find.text('Subscription state needs support'), findsNWidgets(2));
    expect(_tierDetails(SubscriptionTier.plus), findsOneWidget);
    expect(_tierDetails(SubscriptionTier.business), findsOneWidget);
  });

  testWidgets('AT-005 unsafe dormant tier hides repurchase action', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.plus),
        billingAvailability: BillingAvailability.available,
        billingState: _ownerState(
          unknownBusinessStatus: false,
          businessRenewal: 'will_renew',
        ),
        onPurchase: (_) {},
      ),
    );

    expect(_tierDetails(SubscriptionTier.business), findsOneWidget);
  });

  testWidgets('AT-005 canonical dormant assignment can be removed', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.plus),
        billingAvailability: BillingAvailability.available,
        billingState: _ownerState(
          unknownBusinessStatus: false,
          businessAssigned: 'did:plc:alice',
        ),
        assignedAccountLabels: {
          Did.parse('did:plc:alice'): '@alice.test',
          Did.parse('did:plc:bob'): '@bob.test',
        },
        onPurchase: (_) {},
        onAssign: (_) {},
        onUnassign: (_) {},
      ),
    );

    expect(_tierDetails(SubscriptionTier.business), findsOneWidget);
    expect(find.text('Change assignment'), findsOneWidget);
    expect(find.text('Remove assignment'), findsNWidgets(2));
  });

  testWidgets(
    'AT-013 replacement license and dormant assignment expose both actions',
    (tester) async {
      String? assignedLicenseId;
      String? unassignedLicenseId;
      await _pump(
        tester,
        SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: _access(SubscriptionTier.free),
          billingAvailability: BillingAvailability.available,
          billingState: _replacementState(),
          assignedAccountLabels: {
            Did.parse('did:plc:bob'): '@bob.test',
          },
          onAssign: (license) => assignedLicenseId = license.id,
          onUnassign: (license) => unassignedLicenseId = license.id,
        ),
      );

      expect(find.text('Assign license'), findsOneWidget);
      expect(find.text('Remove assignment'), findsOneWidget);
      expect(find.text('Active'), findsOneWidget);
      expect(find.text('Not currently providing access'), findsNothing);
      expect(find.textContaining('Product:'), findsNothing);
      expect(find.textContaining('Access ends:'), findsNothing);
      expect(find.text('License cannot currently be assigned'), findsNothing);

      await _tapVisible(tester, find.text('Assign license'));
      await _tapVisible(tester, find.text('Remove assignment'));

      expect(assignedLicenseId, '40000000-0000-4000-8000-000000000004');
      expect(unassignedLicenseId, '40000000-0000-4000-8000-000000000003');
    },
  );

  testWidgets('AT-013 every secondary owned license is rendered', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _replacementState(
          oldGivesAccess: true,
          oldAssigned: false,
          oldAssignable: true,
        ),
        onAssign: (_) {},
      ),
    );

    expect(find.textContaining('Product:'), findsNothing);
    expect(find.text('Active'), findsOneWidget);
    expect(find.textContaining('Access until'), findsOneWidget);
    expect(find.text('Assign license'), findsNWidgets(2));
  });

  testWidgets('AT-014 anomalous tier fail-closes secondary assignment', (
    tester,
  ) async {
    final unassignedLicenseIds = <String>[];
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _replacementState(
          oldGivesAccess: true,
          oldAnomaly: 'duplicate_tier',
          newAssigned: true,
        ),
        assignedAccountLabels: {
          Did.parse('did:plc:bob'): '@bob.test',
        },
        onAssign: (_) {},
        onUnassign: (license) => unassignedLicenseIds.add(license.id),
      ),
    );

    expect(find.text('Subscription state needs support'), findsOneWidget);
    expect(find.text('Assign license'), findsNothing);
    expect(find.text('Change assignment'), findsNothing);
    expect(find.text('Remove assignment'), findsNWidgets(2));

    await _tapVisible(tester, find.text('Remove assignment').first);
    await _tapVisible(tester, find.text('Remove assignment').last);

    expect(unassignedLicenseIds, [
      '40000000-0000-4000-8000-000000000003',
      '40000000-0000-4000-8000-000000000004',
    ]);
  });

  testWidgets('AT-013 renders dormant and anomalous secondary licenses', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _replacementState(
          oldAssigned: false,
          includeAnomalousSecondary: true,
        ),
        onAssign: (_) {},
        onUnassign: (_) {},
      ),
    );

    expect(find.textContaining('Product:'), findsNothing);
    expect(find.text('Subscription state needs support'), findsNWidgets(2));
    expect(find.text('Assign license'), findsNothing);
    expect(find.text('Change assignment'), findsNothing);
    expect(find.text('Remove assignment'), findsNothing);
  });

  testWidgets('UT-010 dormant end-only timing stays hidden', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _ownerState(
          unknownBusinessStatus: false,
          businessStore: 'future_private_store',
          businessEndsAt: DateTime.utc(2026, 11, 2, 12),
        ),
      ),
    );

    expect(find.textContaining('future_private_store'), findsNothing);
    expect(find.textContaining('Product:'), findsNothing);
    expect(find.textContaining('Access ends:'), findsNothing);
    expect(find.textContaining('Access until'), findsNothing);
    expect(find.textContaining('Nov 2'), findsNothing);
  });

  testWidgets('owner tier cards show localized RevenueCat prices', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _emptyState(),
        tierPrices: const {
          SubscriptionTier.plus: r'$4.99',
          SubscriptionTier.business: '£8.99',
        },
      ),
    );

    expect(find.text(r'$4.99 / month'), findsOneWidget);
    expect(find.text('£8.99 / month'), findsOneWidget);
    expect(find.text('Price unavailable'), findsNothing);

    final businessPrice = tester.widget<Text>(find.text('£8.99 / month'));
    final accent = businessPrice.style!.color!;
    final surface = Theme.of(
      tester.element(find.text('£8.99 / month')),
    ).colorScheme.surface;
    final cardBackground = Color.alphaBlend(
      accent.withValues(alpha: 0.08),
      surface,
    );
    expect(_contrastRatio(accent, cardBackground), greaterThanOrEqualTo(4.5));
  });

  testWidgets(
    'AT-012 unavailable billing keeps self access and hides native actions',
    (
      tester,
    ) async {
      await _pump(
        tester,
        SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: _access(SubscriptionTier.plus),
          billingAvailability: BillingAvailability.unavailable,
          billingState: _emptyState(),
          onPurchase: (_) {},
          onRestore: () {},
          onManage: () {},
        ),
      );

      expect(find.text('Current access: Plus'), findsOneWidget);
      expect(find.textContaining('fully usable'), findsOneWidget);
      expect(_tierDetails(SubscriptionTier.plus), findsNothing);
      expect(_tierDetails(SubscriptionTier.business), findsNothing);
      expect(find.text('Restore purchases'), findsNothing);
      expect(find.text('Manage subscription'), findsNothing);
    },
  );

  testWidgets('AT-011 locked owner recovery preserves self access', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.plus),
        billingAvailability: BillingAvailability.available,
        ownerRecoveryLocked: true,
      ),
    );

    expect(find.text('Current access: Plus'), findsOneWidget);
    final error = find.textContaining(
      "We couldn't confirm this account's billing details",
    );
    expect(error, findsOneWidget);
    expect(
      tester.widget<Text>(error).style?.color,
      Theme.of(tester.element(error)).colorScheme.error,
    );
    expect(
      tester
          .widget<CraftskyCard>(
            find.ancestor(
              of: error,
              matching: find.byType(CraftskyCard),
            ),
          )
          .clipBehavior,
      Clip.none,
    );
    expect(find.text('Retry setup'), findsNothing);
    expect(_tierDetails(SubscriptionTier.plus), findsNothing);
  });

  testWidgets('expired sign-in prompts login without claiming billing loss', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.plus),
        billingAvailability: BillingAvailability.available,
        ownerSignInRequired: true,
      ),
    );

    final message = find.textContaining('Your sign-in has expired');
    expect(message, findsOneWidget);
    expect(find.text('Current access: Plus'), findsOneWidget);
    expect(
      tester.widget<Text>(message).style?.color,
      Theme.of(tester.element(message)).colorScheme.error,
    );
    expect(find.textContaining('contact support'), findsNothing);
    expect(find.text('Refresh status'), findsNothing);
    expect(find.text('Retry setup'), findsNothing);
  });

  testWidgets('owner choice explains reconnecting on a new device', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.neverReserved,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
      ),
    );

    expect(find.textContaining("we'll reconnect it here"), findsOneWidget);
    expect(
      find.textContaining("Billing ownership can't be moved"),
      findsOneWidget,
    );
  });

  testWidgets('AT-011 missing owner exposes reauthentication guidance only', (
    tester,
  ) async {
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.inactiveOwner,
        access: _access(SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
      ),
    );

    expect(find.text('Current access: Free'), findsOneWidget);
    expect(
      find.text(
        'The billing owner must sign in again before subscriptions can be '
        'managed.',
      ),
      findsOneWidget,
    );
    expect(find.text('Switch to billing owner'), findsNothing);
    expect(find.text('Restore purchases'), findsNothing);
    expect(find.text('Manage subscription'), findsNothing);
    expect(find.textContaining('Product:'), findsNothing);
  });

  testWidgets('AT-014 transient owner failure preserves access and refresh', (
    tester,
  ) async {
    var refreshed = false;
    await _pump(
      tester,
      SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access(SubscriptionTier.plus),
        billingAvailability: BillingAvailability.available,
        operationNotice: SubscriptionOperationNotice.providerFailure,
        onRefresh: () => refreshed = true,
      ),
    );

    expect(find.text('Current access: Plus'), findsOneWidget);
    expect(find.textContaining('provider could not complete'), findsOneWidget);
    await tester.tap(find.text('Refresh status'));
    expect(refreshed, isTrue);
  });

  testWidgets('AT-010 app resume refreshes owner state', (tester) async {
    var reads = 0;
    final model = SubscriptionPageModel(
      role: SubscriptionPageRole.owner,
      access: _access(SubscriptionTier.free),
      billingAvailability: BillingAvailability.available,
      billingState: _emptyState(),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          subscriptionPageModelProvider.overrideWith((_) async {
            reads++;
            return model;
          }),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const SubscriptionPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
    await tester.pump();
    expect(reads, 1);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();

    expect(reads, 2);
  });

  testWidgets('AT-008 assignment failures retain distinct safe messages', (
    tester,
  ) async {
    const cases = {
      SubscriptionOperationNotice.assignmentConflict:
          'That license is already assigned.',
      SubscriptionOperationNotice.assignmentCooldown:
          'That license cannot be reassigned yet.',
      SubscriptionOperationNotice.licenseNotFound:
          'That license is no longer available.',
      SubscriptionOperationNotice.targetIneligible:
          'That account cannot receive this license.',
      SubscriptionOperationNotice.unauthorized:
          'Your session can no longer make that billing change.',
      SubscriptionOperationNotice.actionFailed:
          'That subscription change could not be completed.',
    };

    for (final MapEntry(key: notice, value: message) in cases.entries) {
      await _pump(
        tester,
        SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: _access(SubscriptionTier.free),
          billingAvailability: BillingAvailability.available,
          billingState: _emptyState(),
          operationNotice: notice,
        ),
      );
      expect(find.textContaining(message), findsOneWidget);
    }
  });
}

Future<void> _pump(WidgetTester tester, SubscriptionPageModel model) =>
    tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: SubscriptionPageView(model: model),
      ),
    );

Finder _tierDetails(SubscriptionTier tier) =>
    find.byKey(ValueKey('subscription-view-details-${tier.name}'));

Future<void> _tapVisible(WidgetTester tester, Finder finder) async {
  await tester.ensureVisible(finder);
  await tester.pump();
  await tester.tap(finder);
}

double _contrastRatio(Color first, Color second) {
  final firstLuminance = first.computeLuminance();
  final secondLuminance = second.computeLuminance();
  final lighter = firstLuminance > secondLuminance
      ? firstLuminance
      : secondLuminance;
  final darker = firstLuminance > secondLuminance
      ? secondLuminance
      : firstLuminance;
  return (lighter + 0.05) / (darker + 0.05);
}

SubscriptionAccess _access(
  SubscriptionTier tier, {
  SubscriptionTier? assignedTier,
}) => SubscriptionAccess(
  did: Did.parse('did:plc:alice'),
  effectiveTier: tier,
  givesAccess: tier != SubscriptionTier.free,
  assignedTier: assignedTier,
);

String _tierName(SubscriptionTier tier) => switch (tier) {
  SubscriptionTier.free => 'Free',
  SubscriptionTier.plus => 'Plus',
  SubscriptionTier.business => 'Business',
};

BillingState _emptyState() => const BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  subscriptions: [],
  licenses: [],
);

BillingState _orphanSubscriptionState() => const BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  subscriptions: [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000006',
      productId: 'orphan-product',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    ),
  ],
  licenses: [],
);

BillingState _ownerState({
  required bool unknownBusinessStatus,
  String businessRenewal = 'will_not_renew',
  String? businessAssigned,
  String businessStore = 'app_store',
  DateTime? businessEndsAt,
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  reconciliationRequestedAt: DateTime.utc(2026, 9, 10, 12),
  reconciledAt: DateTime.utc(2026, 9, 11, 12),
  subscriptions: [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000002',
      productId: 'business',
      store: businessStore,
      status: unknownBusinessStatus ? 'future_status' : 'active',
      givesAccess: false,
      pendingPayment: false,
      autoRenewalStatus: businessRenewal,
      endsAt: businessEndsAt,
      anomaly: 'none',
    ),
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000001',
      productId: 'plus',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      currentPeriodStartsAt: DateTime.utc(2026, 9, 1, 12),
      currentPeriodEndsAt: DateTime.utc(2026, 10, 1, 12),
      anomaly: 'none',
    ),
  ],
  licenses: [
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000001',
      subscriptionId: '30000000-0000-4000-8000-000000000001',
      tier: SubscriptionTier.plus,
      assignedDid: Did.parse('did:plc:bob'),
      assignable: true,
      anomaly: 'none',
    ),
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000002',
      subscriptionId: '30000000-0000-4000-8000-000000000002',
      tier: SubscriptionTier.business,
      assignedDid: businessAssigned == null
          ? null
          : Did.parse(businessAssigned),
      assignable: businessAssigned == null,
      anomaly: 'none',
    ),
  ],
);

BillingState _replacementState({
  bool oldGivesAccess = false,
  bool oldAssigned = true,
  bool oldAssignable = false,
  String oldAnomaly = 'none',
  bool newAssigned = false,
  bool newAssignable = true,
  bool includeAnomalousSecondary = false,
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 2,
  reconciledGeneration: 2,
  reconciliationStale: false,
  subscriptions: [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000003',
      productId: 'plus-old',
      store: 'app_store',
      status: oldGivesAccess ? 'active' : 'expired',
      givesAccess: oldGivesAccess,
      pendingPayment: false,
      autoRenewalStatus: 'will_not_renew',
      endsAt: DateTime.utc(2026, 8, 31, 12),
      anomaly: oldAnomaly,
    ),
    const BillingSubscription(
      id: '30000000-0000-4000-8000-000000000004',
      productId: 'plus-new',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    ),
    if (includeAnomalousSecondary)
      const BillingSubscription(
        id: '30000000-0000-4000-8000-000000000005',
        productId: 'plus-anomaly',
        store: 'app_store',
        status: 'expired',
        givesAccess: false,
        pendingPayment: false,
        autoRenewalStatus: 'will_not_renew',
        anomaly: 'duplicate_tier',
      ),
  ],
  licenses: [
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000003',
      subscriptionId: '30000000-0000-4000-8000-000000000003',
      tier: SubscriptionTier.plus,
      assignedDid: oldAssigned ? Did.parse('did:plc:bob') : null,
      assignable: oldAssignable,
      anomaly: 'none',
    ),
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000004',
      subscriptionId: '30000000-0000-4000-8000-000000000004',
      tier: SubscriptionTier.plus,
      assignedDid: newAssigned ? Did.parse('did:plc:bob') : null,
      assignable: newAssignable,
      anomaly: 'none',
    ),
    if (includeAnomalousSecondary)
      const BillingLicense(
        id: '40000000-0000-4000-8000-000000000005',
        subscriptionId: '30000000-0000-4000-8000-000000000005',
        tier: SubscriptionTier.plus,
        assignable: true,
        anomaly: 'duplicate_tier',
      ),
  ],
);
