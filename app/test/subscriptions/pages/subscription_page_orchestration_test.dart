import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/pages/subscription_page.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_pending_operation_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'IT-004 persisted owner UUID recovers identity through GET only',
    (tester) async {
      final registry = _registry();
      final events = <String>[];
      final ownerApi = _OwnerApi(
        [
          _state(generation: 7),
          _state(generation: 7),
          _state(generation: 7),
        ],
        events: events,
      );
      final revenueCat = _RevenueCat(anonymous: true, events: events);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            revenueCatServiceProvider.overrideWithValue(revenueCat),
            subscriptionRepositoryProvider.overrideWith(
              (_, _) async => ownerApi,
            ),
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

      expect(find.text('Retry setup'), findsOneWidget);
      expect(_tierDetails(SubscriptionTier.plus), findsNothing);
      expect(ownerApi.ensureCalls, 0);
      expect(revenueCat.identifyCalls, 0);
      events.clear();

      await tester.tap(find.text('Retry setup'));
      await tester.pumpAndSettle();

      expect(events.take(3), ['billingGet', 'identityRead', 'identify']);
      expect(_tierDetails(SubscriptionTier.plus), findsOneWidget);
      expect(ownerApi.ensureCalls, 0);
      expect(revenueCat.identifyCalls, 1);
    },
  );

  testWidgets(
    'IT-006 purchased license requires explicit tier and target assignment',
    (tester) async {
      final registry = _registry();
      final unassigned = _state(generation: 8);
      final assigned = _state(generation: 8, assignedDid: 'did:plc:bob');
      final ownerApi = _OwnerApi([
        _state(generation: 7),
        _state(generation: 7),
        unassigned,
        unassigned,
        assigned,
        assigned,
        assigned,
        _stateWithBusiness(generation: 9, assignedDid: 'did:plc:bob'),
      ]);
      final targetApi = _TargetApi();
      final revenueCat = _RevenueCat();
      final model = SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: _access('did:plc:alice', SubscriptionTier.free),
        billingAvailability: BillingAvailability.available,
        billingState: _state(generation: 7),
        assignedAccountLabels: {
          Did.parse('did:plc:alice'): '@alice.test',
          Did.parse('did:plc:bob'): '@bob.test',
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            revenueCatServiceProvider.overrideWithValue(revenueCat),
            subscriptionRepositoryProvider.overrideWith((_, account) async {
              return account.did == Did.parse('did:plc:alice')
                  ? ownerApi
                  : targetApi;
            }),
            subscriptionPageModelProvider.overrideWith((ref) async {
              await ref.watch(sessionRegistryProvider.future);
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

      await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
      await tester.pumpAndSettle();
      expect(find.text('Assign license'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.byType(AccountAvatar), findsNWidgets(2));

      await tester.tap(find.text('@bob.test'));
      await tester.pumpAndSettle();
      expect(
        find.textContaining('Assign Plus to @bob.test?'),
        findsOneWidget,
      );

      await tester.tap(find.text('Assign license'));
      await tester.pumpAndSettle();

      expect(ownerApi.assignedTarget, 'did:plc:bob');
      expect(targetApi.accessReads, 1);
      expect(find.textContaining('could not be completed'), findsNothing);

      final pendingPurchase = Completer<DirectPaywallResult>();
      revenueCat.pendingPresentation = pendingPurchase;
      await _tapVisible(tester, _tierDetails(SubscriptionTier.business));
      await tester.pump();
      expect(
        find.ancestor(
          of: find.byKey(const ValueKey('subscription-action-progress')),
          matching: find.byType(ChunkyButton),
        ),
        findsOneWidget,
      );
      expect(
        find.text(
          'Your provider action completed. CraftSky is waiting for confirmed '
          'subscription status.',
        ),
        findsNothing,
      );
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump();
      pendingPurchase.complete(DirectPaywallResult.purchased);
      await tester.pumpAndSettle();
      expect(ownerApi.reconciliationRequests, 1);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();

      expect(ownerApi.reconciliationRequests, 2);
    },
  );

  testWidgets(
    'IT-008 dormant removal enables refreshed replacement assignment',
    (tester) async {
      final registry = _registry();
      final initial = _replacementState(dormantAssigned: true);
      final released = _replacementState(dormantAssigned: false);
      final reassigned = _replacementState(
        dormantAssigned: false,
        replacementAssigned: true,
      );
      var pageState = initial;
      late final _OwnerApi ownerApi;
      ownerApi = _OwnerApi(
        [initial, released, released, reassigned],
        onStateRead: (state) => pageState = state,
      );
      final targetApi = _TargetApi();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            revenueCatServiceProvider.overrideWithValue(_RevenueCat()),
            subscriptionRepositoryProvider.overrideWith((_, account) async {
              return account.did == Did.parse('did:plc:alice')
                  ? ownerApi
                  : targetApi;
            }),
            subscriptionPageModelProvider.overrideWith((ref) async {
              await ref.watch(sessionRegistryProvider.future);
              return SubscriptionPageModel(
                role: SubscriptionPageRole.owner,
                access: _access('did:plc:alice', SubscriptionTier.free),
                billingAvailability: BillingAvailability.available,
                billingState: pageState,
                assignedAccountLabels: {
                  Did.parse('did:plc:alice'): '@alice.test',
                  Did.parse('did:plc:bob'): '@bob.test',
                },
              );
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

      expect(find.text('Assign license'), findsOneWidget);
      expect(find.text('Remove assignment'), findsOneWidget);

      await _tapVisible(tester, find.text('Remove assignment'));
      await tester.pumpAndSettle();
      await tester.tap(
        find.widgetWithText(ChunkyButton, 'Remove assignment').last,
      );
      await tester.pumpAndSettle();

      expect(
        ownerApi.unassignedLicenseId,
        '40000000-0000-4000-8000-000000000003',
      );
      await _tapVisible(tester, find.text('Assign license'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('@bob.test'));
      await tester.pumpAndSettle();
      await tester.tap(
        find.widgetWithText(ChunkyButton, 'Assign license').last,
      );
      await tester.pumpAndSettle();

      expect(
        ownerApi.assignedLicenseId,
        '40000000-0000-4000-8000-000000000004',
      );
      expect(ownerApi.assignedTarget, 'did:plc:bob');
      expect(targetApi.accessReads, 1);
      expect(ownerApi.billingReads, 4);
    },
  );

  for (final unassign in [false, true]) {
    testWidgets(
      'IT-008 ${unassign ? 'unassignment' : 'assignment'} '
      'survives route disposal without duplication',
      (tester) async {
        final registry = _registry();
        final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
        final pendingMutation = Completer<void>();
        final initialState = _state(
          generation: 8,
          assignedDid: unassign ? 'did:plc:bob' : null,
        );
        final completedState = _state(
          generation: 8,
          assignedDid: unassign ? null : 'did:plc:bob',
        );
        final ownerApi = _OwnerApi(
          [initialState, initialState, initialState, completedState],
          pendingAssign: unassign ? null : pendingMutation,
          pendingUnassign: unassign ? pendingMutation : null,
        );
        final targetApi = _TargetApi();
        final container = ProviderContainer(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            revenueCatServiceProvider.overrideWithValue(_RevenueCat()),
            subscriptionRepositoryProvider.overrideWith((_, account) async {
              return account.did == Did.parse('did:plc:alice')
                  ? ownerApi
                  : targetApi;
            }),
            subscriptionPageModelProvider.overrideWith((ref) async {
              await ref.watch(sessionRegistryProvider.future);
              final billingState = await ownerApi.getBillingAccount();
              return SubscriptionPageModel(
                role: SubscriptionPageRole.owner,
                access: _access('did:plc:alice', SubscriptionTier.free),
                billingAvailability: BillingAvailability.available,
                billingState: billingState,
                assignedAccountLabels: {
                  Did.parse('did:plc:alice'): '@alice.test',
                  Did.parse('did:plc:bob'): '@bob.test',
                },
              );
            }),
          ],
        );
        addTearDown(container.dispose);
        final targetAccessSubscription = container.listen(
          subscriptionAccessProvider(bobLease),
          (_, _) {},
          fireImmediately: true,
        );
        addTearDown(targetAccessSubscription.close);
        await container.read(subscriptionAccessProvider(bobLease).future);

        Widget app(Widget home) => UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: home,
          ),
        );

        final action = unassign ? 'Remove assignment' : 'Assign license';
        await tester.pumpWidget(app(const SubscriptionPage()));
        await tester.pumpAndSettle();
        await _tapVisible(tester, find.text(action));
        await tester.pumpAndSettle();
        if (!unassign) {
          await tester.tap(find.text('@bob.test'));
          await tester.pumpAndSettle();
        }
        await tester.tap(find.widgetWithText(ChunkyButton, action).last);
        await tester.pump();

        expect(unassign ? ownerApi.unassignCalls : ownerApi.assignCalls, 1);
        await tester.pumpWidget(app(const SizedBox.shrink()));
        await tester.pump();
        await tester.pumpWidget(app(const SubscriptionPage()));
        await _pumpUntilActionProgress(tester);

        final progressButton = find.ancestor(
          of: _actionProgress(),
          matching: find.byType(unassign ? TextButton : ChunkyButton),
        );
        expect(progressButton, findsOneWidget);
        if (unassign) {
          expect(tester.widget<TextButton>(progressButton).onPressed, isNull);
        } else {
          expect(tester.widget<ChunkyButton>(progressButton).onPressed, isNull);
        }
        expect(unassign ? ownerApi.unassignCalls : ownerApi.assignCalls, 1);

        pendingMutation.complete();
        await tester.pumpAndSettle();
        expect(
          find.text(unassign ? 'Assign license' : 'Remove assignment'),
          findsOneWidget,
        );
        expect(ownerApi.billingReads, 4);
        expect(targetApi.accessReads, 2);
      },
    );
  }

  testWidgets('AT-007 pending purchase refresh never reopens checkout', (
    tester,
  ) async {
    final registry = _registry();
    final ownerApi = _OwnerApi(
      [
        _state(generation: 7),
        _state(generation: 7),
        _state(generation: 8),
      ],
      failReadNumbers: {2},
    );
    final revenueCat = _RevenueCat();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
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

    await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
    await tester.pumpAndSettle();

    expect(find.text('Refresh status'), findsOneWidget);
    expect(_tierDetails(SubscriptionTier.plus), findsNothing);
    expect(ownerApi.reconciliationRequests, 0);

    await _tapVisible(tester, find.text('Refresh status'));
    await tester.pumpAndSettle();

    expect(find.text('Assign license'), findsOneWidget);
    expect(ownerApi.reconciliationRequests, 1);
    expect(revenueCat.presentations, 1);
  });

  testWidgets(
    'IT-010 pending purchase disables native actions and keeps its intent',
    (tester) async {
      final registry = _registry();
      final ownerApi = _OwnerApi(
        [_state(generation: 7), _state(generation: 7)],
        failReadNumbers: {2},
      );
      final revenueCat = _RevenueCat();
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SubscriptionPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
      await tester.pumpAndSettle();

      final restoreButton = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Restore purchases'),
      );
      final manageButton = tester.widget<FilledButton>(
        find.widgetWithText(FilledButton, 'Manage subscription'),
      );
      expect(restoreButton.onPressed, isNull);
      expect(manageButton.onPressed, isNull);
      final pending = container.read(subscriptionPendingOperationProvider)!;
      expect(pending.kind, PendingSubscriptionOperationKind.purchase);
      expect(pending.tier, SubscriptionTier.plus);
      expect(
        container
            .read(subscriptionPendingOperationProvider.notifier)
            .beginRestore(Did.parse('did:plc:alice')),
        isFalse,
      );
      expect(
        container.read(subscriptionPendingOperationProvider),
        same(pending),
      );
      expect(revenueCat.restoreCalls, 0);
    },
  );

  testWidgets(
    'IT-006 in-flight purchase poll remains pending after background resume',
    (tester) async {
      final registry = _registry();
      final pollResult = Completer<BillingState>();
      final ownerApi = _OwnerApi(
        [_state(generation: 7), _state(generation: 7)],
        delayedReads: {3: pollResult},
      );
      final revenueCat = _RevenueCat();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            revenueCatServiceProvider.overrideWithValue(revenueCat),
            subscriptionRepositoryProvider.overrideWith(
              (_, _) async => ownerApi,
            ),
            subscriptionPageModelProvider.overrideWith((ref) async {
              await ref.watch(sessionRegistryProvider.future);
              return SubscriptionPageModel(
                role: SubscriptionPageRole.owner,
                access: _access('did:plc:alice', SubscriptionTier.free),
                billingAvailability: BillingAvailability.available,
                billingState: _state(generation: 7),
              );
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

      await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
      await tester.pump();
      expect(ownerApi.billingReads, 3);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump();
      pollResult.complete(_state(generation: 7));
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      for (var attempt = 0; attempt < 30; attempt++) {
        await tester.pump(const Duration(seconds: 1));
      }
      await tester.pumpAndSettle();

      expect(find.text('Refresh status'), findsOneWidget);
      expect(_tierDetails(SubscriptionTier.plus), findsNothing);
      expect(revenueCat.presentations, 1);
    },
  );

  testWidgets(
    'IT-010 in-flight restore poll remains pending after background resume',
    (tester) async {
      final registry = _registry();
      final pollResult = Completer<BillingState>();
      final ownerApi = _OwnerApi(
        [_state(generation: 7), _state(generation: 7)],
        delayedReads: {3: pollResult},
      );
      final revenueCat = _RevenueCat();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            revenueCatServiceProvider.overrideWithValue(revenueCat),
            subscriptionRepositoryProvider.overrideWith(
              (_, _) async => ownerApi,
            ),
            subscriptionPageModelProvider.overrideWith((ref) async {
              await ref.watch(sessionRegistryProvider.future);
              return SubscriptionPageModel(
                role: SubscriptionPageRole.owner,
                access: _access('did:plc:alice', SubscriptionTier.free),
                billingAvailability: BillingAvailability.available,
                billingState: _state(generation: 7),
              );
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

      await _tapVisible(tester, find.text('Restore purchases'));
      await tester.pump();
      expect(ownerApi.billingReads, 3);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump();
      pollResult.complete(_state(generation: 7));
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      for (var attempt = 0; attempt < 30; attempt++) {
        await tester.pump(const Duration(seconds: 1));
      }
      await tester.pumpAndSettle();

      expect(find.text('Refresh status'), findsOneWidget);
      expect(revenueCat.restoreCalls, 1);
    },
  );

  testWidgets(
    'IT-010 confirmed purchase remains pending after route disposal',
    (tester) async {
      final registry = _registry();
      final ownerApi = _OwnerApi([
        _state(generation: 7),
        _state(generation: 7),
        _state(generation: 8),
      ]);
      final revenueCat = _RevenueCat();
      final pendingPurchase = Completer<DirectPaywallResult>();
      revenueCat.pendingPresentation = pendingPurchase;
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      Widget app(Widget home) => UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: home,
        ),
      );

      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();
      await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
      await tester.pump();

      await tester.pumpWidget(app(const SizedBox.shrink()));
      pendingPurchase.complete(DirectPaywallResult.purchased);
      await tester.pumpAndSettle();
      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();

      expect(find.text('Refresh status'), findsOneWidget);
      expect(_tierDetails(SubscriptionTier.plus), findsNothing);
      expect(revenueCat.presentations, 1);
      expect(ownerApi.reconciliationRequests, 0);

      await _tapVisible(tester, find.text('Refresh status'));
      await tester.pumpAndSettle();

      expect(find.text('Assign license'), findsOneWidget);
      expect(revenueCat.presentations, 1);
      expect(ownerApi.reconciliationRequests, 1);
    },
  );

  testWidgets(
    'IT-010 unresolved purchase cannot be repeated after route disposal',
    (tester) async {
      final registry = _registry();
      final ownerApi = _OwnerApi([
        _state(generation: 7),
        _state(generation: 7),
      ]);
      final revenueCat = _RevenueCat();
      final pendingPurchase = Completer<DirectPaywallResult>();
      revenueCat.pendingPresentation = pendingPurchase;
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      Widget app(Widget home) => UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: home,
        ),
      );

      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();
      await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
      await tester.pump();
      expect(revenueCat.presentations, 1);

      await tester.pumpWidget(app(const SizedBox.shrink()));
      await tester.pump();
      await tester.pumpWidget(app(const SubscriptionPage()));
      await _pumpUntilActionProgress(tester);

      final purchaseButton = tester.widget<ChunkyButton>(
        find.ancestor(
          of: _actionProgress(),
          matching: find.byType(ChunkyButton),
        ),
      );
      expect(purchaseButton.onPressed, isNull);
      expect(find.text('Refresh status'), findsNothing);
      expect(revenueCat.presentations, 1);

      pendingPurchase.complete(DirectPaywallResult.cancelled);
      await tester.pump();
    },
  );

  testWidgets(
    'IT-010 confirmed restore remains pending after route disposal',
    (tester) async {
      final registry = _registry();
      final ownerApi = _OwnerApi([
        _state(generation: 7),
        _state(generation: 7),
        _emptyState(generation: 8),
      ]);
      final revenueCat = _RevenueCat();
      final pendingRestore = Completer<void>();
      revenueCat.pendingRestore = pendingRestore;
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      Widget app(Widget home) => UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: home,
        ),
      );

      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();
      await _tapVisible(tester, find.text('Restore purchases'));
      await tester.pump();

      await tester.pumpWidget(app(const SizedBox.shrink()));
      pendingRestore.complete();
      await tester.pumpAndSettle();
      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();

      expect(find.text('Refresh status'), findsOneWidget);
      expect(revenueCat.restoreCalls, 1);
      expect(ownerApi.reconciliationRequests, 0);

      await _tapVisible(tester, find.text('Refresh status'));
      await tester.pumpAndSettle();

      expect(find.text('Refresh status'), findsNothing);
      expect(revenueCat.restoreCalls, 1);
      expect(ownerApi.reconciliationRequests, 1);
    },
  );

  testWidgets(
    'IT-010 unresolved restore cannot be repeated after route disposal',
    (tester) async {
      final registry = _registry();
      final ownerApi = _OwnerApi([
        _state(generation: 7),
        _state(generation: 7),
      ]);
      final revenueCat = _RevenueCat();
      final pendingRestore = Completer<void>();
      revenueCat.pendingRestore = pendingRestore;
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      Widget app(Widget home) => UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: home,
        ),
      );

      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();
      await _tapVisible(tester, find.text('Restore purchases'));
      await tester.pump();
      expect(revenueCat.restoreCalls, 1);

      await tester.pumpWidget(app(const SizedBox.shrink()));
      await tester.pump();
      await tester.pumpWidget(app(const SubscriptionPage()));
      await _pumpUntilActionProgress(tester);

      final restoreButton = tester.widget<TextButton>(
        find.ancestor(
          of: _actionProgress(),
          matching: find.byType(TextButton),
        ),
      );
      expect(restoreButton.onPressed, isNull);
      expect(find.text('Refresh status'), findsNothing);
      expect(revenueCat.restoreCalls, 1);

      pendingRestore.complete();
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'IT-009 Customer Center mutation survives disposal and owner switch',
    (tester) async {
      final registry = _registry();
      final aliceLease = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
      final ownerApi = _OwnerApi([
        _state(generation: 7),
        _state(generation: 7),
        _emptyState(generation: 8),
      ]);
      final revenueCat = _RevenueCat()
        ..customerCenterEvent = CustomerCenterEvent.managementOptionSelected
        ..pendingCustomerCenter = Completer<void>();
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(generation: 7),
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      Widget app(Widget home) => UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: home,
        ),
      );

      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();
      await _tapVisible(tester, find.text('Manage subscription'));
      await tester.pump();
      expect(revenueCat.customerCenterCalls, 1);
      expect(
        container.read(subscriptionPendingOperationProvider)?.canReconcile,
        isTrue,
      );

      await tester.pumpWidget(app(const SizedBox.shrink()));
      await tester.pump();
      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      await tester.pumpWidget(app(const SubscriptionPage()));
      await tester.pumpAndSettle();

      expect(revenueCat.customerCenterCalls, 1);
      expect(ownerApi.reconciliationRequests, 0);

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceLease);
      await tester.pumpAndSettle();
      final refreshButton = tester.widget<ChunkyButton>(
        find.widgetWithText(ChunkyButton, 'Refresh status'),
      );
      expect(refreshButton.onPressed, isNotNull);

      await _tapVisible(tester, find.text('Refresh status'));
      await tester.pumpAndSettle();

      expect(revenueCat.customerCenterCalls, 1);
      expect(ownerApi.reconciliationRequests, 1);
      expect(container.read(subscriptionPendingOperationProvider), isNull);

      revenueCat.pendingCustomerCenter!.complete();
      await tester.pumpAndSettle();
      expect(ownerApi.reconciliationRequests, 1);
    },
  );

  testWidgets(
    'IT-010 pending purchase survives account switching without checkout',
    (tester) async {
      final registry = _registry();
      final aliceLease = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
      final baselineRead = Completer<BillingState>();
      final ownerApi = _OwnerApi(
        [
          _state(generation: 7),
          _state(generation: 7),
          _state(generation: 8),
        ],
        delayedReads: {2: baselineRead},
      );
      final revenueCat = _RevenueCat();
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            final current = await ref.watch(sessionRegistryProvider.future);
            final ownerActive =
                current.activeLease?.session.account.did ==
                Did.parse('did:plc:alice');
            return SubscriptionPageModel(
              role: ownerActive
                  ? SubscriptionPageRole.owner
                  : SubscriptionPageRole.beneficiary,
              access: _access(
                ownerActive ? 'did:plc:alice' : 'did:plc:bob',
                SubscriptionTier.free,
              ),
              billingAvailability: BillingAvailability.available,
              ownerRetained: true,
              billingState: ownerActive ? _state(generation: 7) : null,
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SubscriptionPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await _tapVisible(tester, _tierDetails(SubscriptionTier.plus));
      await tester.pump();
      expect(
        container.read(subscriptionPendingOperationProvider)?.canReconcile,
        isTrue,
      );

      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      baselineRead.complete(_state(generation: 7));
      await tester.pumpAndSettle();
      expect(find.text('Refresh status'), findsNothing);

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceLease);
      await tester.pumpAndSettle();
      expect(find.text('Refresh status'), findsOneWidget);
      await _tapVisible(tester, find.text('Refresh status'));
      await tester.pumpAndSettle();

      expect(find.text('Assign license'), findsOneWidget);
      expect(revenueCat.presentations, 1);
      expect(ownerApi.reconciliationRequests, 1);
    },
  );

  testWidgets(
    'IT-010 pending restore survives account switching without restore',
    (tester) async {
      final registry = _registry();
      final aliceLease = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
      final baselineRead = Completer<BillingState>();
      final ownerApi = _OwnerApi(
        [
          _state(generation: 7),
          _state(generation: 7),
          _emptyState(generation: 8),
        ],
        delayedReads: {2: baselineRead},
      );
      final revenueCat = _RevenueCat();
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(revenueCat),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            final current = await ref.watch(sessionRegistryProvider.future);
            final ownerActive =
                current.activeLease?.session.account.did ==
                Did.parse('did:plc:alice');
            return SubscriptionPageModel(
              role: ownerActive
                  ? SubscriptionPageRole.owner
                  : SubscriptionPageRole.beneficiary,
              access: _access(
                ownerActive ? 'did:plc:alice' : 'did:plc:bob',
                SubscriptionTier.free,
              ),
              billingAvailability: BillingAvailability.available,
              ownerRetained: true,
              billingState: ownerActive ? _state(generation: 7) : null,
            );
          }),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SubscriptionPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await _tapVisible(tester, find.text('Restore purchases'));
      await tester.pump();
      expect(
        container.read(subscriptionPendingOperationProvider)?.canReconcile,
        isTrue,
      );

      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      baselineRead.complete(_state(generation: 7));
      await tester.pumpAndSettle();
      expect(find.text('Refresh status'), findsNothing);

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceLease);
      await tester.pumpAndSettle();
      expect(find.text('Refresh status'), findsOneWidget);
      await _tapVisible(tester, find.text('Refresh status'));
      await tester.pumpAndSettle();

      expect(find.text('Refresh status'), findsNothing);
      expect(revenueCat.restoreCalls, 1);
      expect(ownerApi.reconciliationRequests, 1);
    },
  );

  testWidgets('AT-008 reassignment confirmation explains cooldown', (
    tester,
  ) async {
    final registry = _registry();
    final ownerApi = _OwnerApi([
      _state(generation: 8, assignedDid: 'did:plc:bob'),
    ]);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          revenueCatServiceProvider.overrideWithValue(_RevenueCat()),
          subscriptionRepositoryProvider.overrideWith((_, _) async => ownerApi),
          subscriptionPageModelProvider.overrideWith((ref) async {
            await ref.watch(sessionRegistryProvider.future);
            return SubscriptionPageModel(
              role: SubscriptionPageRole.owner,
              access: _access('did:plc:alice', SubscriptionTier.free),
              billingAvailability: BillingAvailability.available,
              billingState: _state(
                generation: 8,
                assignedDid: 'did:plc:bob',
              ),
              assignedAccountLabels: {
                Did.parse('did:plc:alice'): '@alice.test',
                Did.parse('did:plc:bob'): '@bob.test',
              },
            );
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

    await _tapVisible(tester, find.text('Change assignment'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('alice.test'));
    await tester.pumpAndSettle();

    expect(
      find.textContaining(
        'cannot be assigned to another account for seven days',
      ),
      findsOneWidget,
    );
  });
}

SessionRegistry _registry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
      cachedDisplayName: 'Bob',
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

BillingState _state({required int generation, String? assignedDid}) {
  if (generation == 7) {
    return const BillingState(
      billingAccountId: '10000000-0000-4000-8000-000000000001',
      revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
      requestedGeneration: 7,
      reconciledGeneration: 7,
      reconciliationStale: false,
      subscriptions: [],
      licenses: [],
    );
  }
  return BillingState(
    billingAccountId: '10000000-0000-4000-8000-000000000001',
    revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
    requestedGeneration: generation,
    reconciledGeneration: generation,
    reconciliationStale: false,
    subscriptions: const [
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
        assignedDid: assignedDid == null ? null : Did.parse(assignedDid),
        assignable: true,
        anomaly: 'none',
      ),
    ],
  );
}

BillingState _emptyState({required int generation}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: generation,
  reconciledGeneration: generation,
  reconciliationStale: false,
  subscriptions: const [],
  licenses: const [],
);

BillingState _replacementState({
  required bool dormantAssigned,
  bool replacementAssigned = false,
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 9,
  reconciledGeneration: 9,
  reconciliationStale: false,
  subscriptions: const [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000003',
      productId: 'plus-old',
      store: 'app_store',
      status: 'expired',
      givesAccess: false,
      pendingPayment: false,
      autoRenewalStatus: 'will_not_renew',
      anomaly: 'none',
    ),
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000004',
      productId: 'plus-new',
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
      id: '40000000-0000-4000-8000-000000000003',
      subscriptionId: '30000000-0000-4000-8000-000000000003',
      tier: SubscriptionTier.plus,
      assignedDid: dormantAssigned ? Did.parse('did:plc:bob') : null,
      assignable: false,
      anomaly: 'none',
    ),
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000004',
      subscriptionId: '30000000-0000-4000-8000-000000000004',
      tier: SubscriptionTier.plus,
      assignedDid: replacementAssigned ? Did.parse('did:plc:bob') : null,
      assignable: true,
      anomaly: 'none',
    ),
  ],
);

BillingState _stateWithBusiness({
  required int generation,
  required String assignedDid,
}) {
  final plus = _state(generation: generation, assignedDid: assignedDid);
  return BillingState(
    billingAccountId: plus.billingAccountId,
    revenueCatAppUserId: plus.revenueCatAppUserId,
    requestedGeneration: generation,
    reconciledGeneration: generation,
    reconciliationStale: false,
    subscriptions: [
      ...plus.subscriptions,
      const BillingSubscription(
        id: '30000000-0000-4000-8000-000000000002',
        productId: 'business',
        store: 'app_store',
        status: 'active',
        givesAccess: true,
        pendingPayment: false,
        autoRenewalStatus: 'will_renew',
        anomaly: 'none',
      ),
    ],
    licenses: [
      ...plus.licenses,
      const BillingLicense(
        id: '40000000-0000-4000-8000-000000000002',
        subscriptionId: '30000000-0000-4000-8000-000000000002',
        tier: SubscriptionTier.business,
        assignable: true,
        anomaly: 'none',
      ),
    ],
  );
}

SubscriptionAccess _access(String did, SubscriptionTier tier) =>
    SubscriptionAccess(
      did: Did.parse(did),
      effectiveTier: tier,
      givesAccess: tier != SubscriptionTier.free,
    );

Future<void> _pumpUntilActionProgress(WidgetTester tester) async {
  final progress = _actionProgress();
  for (var attempt = 0; attempt < 20; attempt++) {
    await tester.pump(const Duration(milliseconds: 10));
    if (progress.evaluate().isNotEmpty) return;
  }
  fail('Subscription action progress did not appear');
}

Finder _actionProgress() => find.byKey(
  const ValueKey('subscription-action-progress'),
  skipOffstage: false,
);

Finder _tierDetails(SubscriptionTier tier) =>
    find.byKey(ValueKey('subscription-view-details-${tier.name}'));

Future<void> _tapVisible(WidgetTester tester, Finder finder) async {
  await tester.ensureVisible(finder);
  await tester.pump();
  await tester.tap(finder);
}

final class _Storage implements SessionRegistryStorage {
  _Storage(this.registry);

  final SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {}
}

final class _OwnerApi implements SubscriptionApi {
  _OwnerApi(
    this.states, {
    this.failReadNumbers = const {},
    this.delayedReads = const {},
    this.events,
    this.onStateRead,
    this.pendingAssign,
    this.pendingUnassign,
  }) : _lastState = states.last;

  final List<BillingState> states;
  final Set<int> failReadNumbers;
  final Map<int, Completer<BillingState>> delayedReads;
  final List<String>? events;
  final void Function(BillingState state)? onStateRead;
  final Completer<void>? pendingAssign;
  final Completer<void>? pendingUnassign;
  BillingState _lastState;
  String? assignedLicenseId;
  String? assignedTarget;
  String? unassignedLicenseId;
  int assignCalls = 0;
  int unassignCalls = 0;
  int reconciliationRequests = 0;
  int billingReads = 0;
  int ensureCalls = 0;

  @override
  Future<BillingState> getBillingAccount() async {
    events?.add('billingGet');
    billingReads++;
    if (failReadNumbers.contains(billingReads)) throw StateError('unavailable');
    final delayed = delayedReads[billingReads];
    if (delayed != null) {
      _lastState = await delayed.future;
      onStateRead?.call(_lastState);
      return _lastState;
    }
    if (states.isNotEmpty) _lastState = states.removeAt(0);
    onStateRead?.call(_lastState);
    return _lastState;
  }

  @override
  Future<void> requestReconciliation() async {
    reconciliationRequests++;
  }

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) async {
    assignCalls++;
    assignedLicenseId = licenseId;
    assignedTarget = targetDid;
    await pendingAssign?.future;
    return BillingAssignment(
      licenseId: licenseId,
      targetDid: Did.parse(targetDid),
      assignedAt: DateTime.utc(2026, 9, 11),
    );
  }

  @override
  Future<void> unassign(String licenseId) async {
    unassignCalls++;
    unassignedLicenseId = licenseId;
    await pendingUnassign?.future;
  }

  @override
  Future<BillingState> ensureBillingAccount() async {
    ensureCalls++;
    return _lastState;
  }

  @override
  Future<SubscriptionAccess> getAccess() async =>
      _access('did:plc:alice', SubscriptionTier.free);
}

final class _TargetApi implements SubscriptionApi {
  int accessReads = 0;

  @override
  Future<SubscriptionAccess> getAccess() async {
    accessReads++;
    return _access('did:plc:bob', SubscriptionTier.plus);
  }

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      throw UnimplementedError();

  @override
  Future<BillingState> ensureBillingAccount() => throw UnimplementedError();

  @override
  Future<BillingState> getBillingAccount() => throw UnimplementedError();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();

  @override
  Future<void> unassign(String licenseId) => throw UnimplementedError();
}

final class _RevenueCatOffering implements RevenueCatOffering {
  @override
  String get identifier => 'plus';

  @override
  bool hasPackage(String identifier) => identifier == r'$rc_monthly';

  @override
  String? packagePrice(String identifier) =>
      identifier == r'$rc_monthly' ? r'$4.99' : null;
}

final class _RevenueCat implements RevenueCatService {
  _RevenueCat({this.anonymous = false, this.events});

  bool anonymous;
  final List<String>? events;
  Completer<DirectPaywallResult>? pendingPresentation;
  Completer<void>? pendingRestore;
  Completer<void>? pendingCustomerCenter;
  CustomerCenterEvent? customerCenterEvent;
  int presentations = 0;
  int restoreCalls = 0;
  int customerCenterCalls = 0;
  int identifyCalls = 0;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    events?.add('identityRead');
    return anonymous
        ? const RevenueCatIdentity.anonymous()
        : const RevenueCatIdentity.identified(
            '20000000-0000-4000-8000-000000000001',
          );
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async =>
      _RevenueCatOffering();

  @override
  Future<DirectPaywallResult> presentPaywall(
    RevenueCatOffering offering,
  ) async {
    presentations++;
    return pendingPresentation?.future ?? DirectPaywallResult.purchased;
  }

  @override
  Future<void> identify(String appViewUuid) async {
    events?.add('identify');
    identifyCalls++;
    anonymous = false;
  }

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) async {
    customerCenterCalls++;
    final event = customerCenterEvent;
    if (event != null) onEvent(event);
    await pendingCustomerCenter?.future;
  }

  @override
  Future<void> restorePurchases() async {
    restoreCalls++;
    await pendingRestore?.future;
  }
}
