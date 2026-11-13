import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/account_eligibility/models/account_eligibility_status.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/widgets/active_account_initialization_gate.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_identity_sync_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test('signed-out and disabled sessions create no billing identity', () async {
    final guest = _Harness();
    addTearDown(guest.dispose);
    guest.start();
    await guest.settle();
    expect(guest.api.calls, isEmpty);
    expect(guest.service.identified, isEmpty);

    final disabled = _Harness(initial: _registry('alice'), enabled: false);
    addTearDown(disabled.dispose);
    disabled.start();
    await disabled.settle();
    expect(disabled.api.calls, isEmpty);
    expect(disabled.service.identified, isEmpty);
  });

  test(
    'sign-in identifies after account readiness and persists its UUID',
    () async {
      final harness = _Harness();
      addTearDown(harness.dispose);
      harness.start();
      await harness.settle();
      await harness.signIn('alice');
      await harness.settle();
      expect(harness.api.calls, [_did('alice')]);
      expect(harness.service.identified, [_uuid('alice')]);
      expect(
        harness.registry.billingCustomers[_key('alice').did],
        _uuid('alice'),
      );
      expect(harness.service.persistedBeforeIdentify, isTrue);
    },
  );

  test('restored session identifies once without opening billing', () async {
    final harness = _Harness(initial: _registry('alice'));
    addTearDown(harness.dispose);
    harness.start();
    await harness.settle();
    expect(harness.service.identified, [_uuid('alice')]);
    harness.container.invalidate(activeAccountInitializationProvider);
    await harness.settle();
    expect(harness.api.calls, [_did('alice')]);
    expect(harness.service.identified, [_uuid('alice')]);
  });

  test(
    'switching retained accounts uses their distinct permanent UUIDs',
    () async {
      final harness = _Harness(initial: _registry('alice'));
      addTearDown(harness.dispose);
      harness.start();
      await harness.settle();
      await harness.signIn('bob');
      await harness.settle();
      await harness.switchTo('alice');
      await harness.settle();
      expect(harness.service.identified, [
        _uuid('alice'),
        _uuid('bob'),
        _uuid('alice'),
      ]);
      expect(harness.registry.billingCustomers, hasLength(2));
    },
  );

  test(
    'incomplete and restricted accounts wait until eligible readiness',
    () async {
      final harness = _Harness(initial: _registry('alice'));
      addTearDown(harness.dispose);
      harness.readiness.onboarded = false;
      harness.start();
      await harness.settle();
      expect(harness.api.calls, isEmpty);
      harness.readiness
        ..onboarded = true
        ..restricted = true;
      harness.container.invalidate(activeAccountInitializationProvider);
      await harness.settle();
      expect(harness.api.calls, isEmpty);
      harness.readiness.restricted = false;
      harness.container.invalidate(activeAccountInitializationProvider);
      await harness.settle();
      expect(harness.service.identified, [_uuid('alice')]);
    },
  );

  test('late ensure response cannot update the newly active account', () async {
    final harness = _Harness(initial: _registry('alice'));
    addTearDown(harness.dispose);
    final delayed = Completer<BillingState>();
    harness.api.delayed[_did('alice')] = delayed.future;
    harness.start();
    await harness.settle();
    expect(harness.api.calls, [_did('alice')]);
    await harness.signIn('bob');
    await harness.settle();
    delayed.complete(_state('alice'));
    await harness.settle();
    expect(harness.service.identified, [_uuid('bob')]);
    expect(harness.registry.billingCustomers.keys, [_key('bob').did]);
  });

  test('standalone setup retains its real repository and Dio '
      'until lease retirement', () async {
    final harness = _Harness(initial: _registry('alice'), useDio: true);
    addTearDown(harness.dispose);
    final delayed = Completer<ResponseBody>();
    harness.api.responses[_did('alice')] = delayed.future;
    harness.start();
    await harness.settle();
    final adapter = harness.api.adapters[_did('alice')]!;
    expect(harness.api.calls, [_did('alice')]);
    expect(adapter.closed, isFalse);
    await harness.settle();
    expect(adapter.closed, isFalse);
    await harness.signIn('bob');
    await harness.settle();
    expect(adapter.closed, isTrue);
    delayed.complete(_response('alice'));
    await harness.settle();
    expect(harness.service.identified, [_uuid('bob')]);
  });

  test(
    'leaving the identity scope cancels a delayed setup before SDK work',
    () async {
      final harness = _Harness(initial: _registry('alice'));
      addTearDown(harness.dispose);
      final delayed = Completer<BillingState>();
      harness.api.delayed[_did('alice')] = delayed.future;
      harness.start();
      await harness.settle();
      harness.listener!.close();
      harness.listener = null;
      await harness.settle();
      delayed.complete(_state('alice'));
      await harness.settle();
      expect(harness.service.identified, isEmpty);
      expect(harness.registry.billingCustomers, isEmpty);
    },
  );

  test(
    'store conflict retains server identity without switching the SDK',
    () async {
      final harness = _Harness(initial: _registry('alice'));
      addTearDown(harness.dispose);
      harness.service
        ..identity = _uuid('bob')
        ..conflicts.add(_uuid('alice'));
      harness.start();
      await harness.settle();
      expect(
        harness.registry.billingCustomers[_key('alice').did],
        _uuid('alice'),
      );
      expect(harness.service.identity, _uuid('bob'));
      expect(harness.service.identified, isEmpty);
      expect(
        harness.container.read(activeAccountInitializationProvider).hasValue,
        isTrue,
      );
    },
  );

  test('unavailable SDK still establishes the server-owned customer', () async {
    final harness = _Harness(initial: _registry('alice'));
    addTearDown(harness.dispose);
    harness.service.available = false;
    harness.start();
    await harness.settle();
    expect(harness.api.calls, [_did('alice')]);
    expect(
      harness.registry.billingCustomers[_key('alice').did],
      _uuid('alice'),
    );
    expect(harness.service.ownershipChecks, isEmpty);
  });

  test('switching while ownership loads discards the old SDK login', () async {
    final harness = _Harness(initial: _registry('alice'));
    addTearDown(harness.dispose);
    final delayed = Completer<void>();
    harness.service.ownershipWaits[_uuid('alice')] = delayed.future;
    harness.start();
    await harness.settle();
    expect(harness.service.ownershipChecks, [_uuid('alice')]);
    await harness.signIn('bob');
    await harness.settle();
    expect(
      harness.container.read(activeAccountInitializationProvider).hasValue,
      isTrue,
    );
    expect(harness.service.identified, isEmpty);
    delayed.complete();
    await harness.settle();
    expect(harness.service.identified, [_uuid('bob')]);
  });

  test(
    'SDK identity waits for in-flight billing and skips a superseded account',
    () async {
      final harness = _Harness(initial: _registry('bob'));
      addTearDown(harness.dispose);
      harness.service.identity = _uuid('alice');
      final entered = Completer<void>();
      final release = Completer<void>();
      final purchase = RevenueCatIdentityGuard(harness.service).dispatch(
        _uuid('alice'),
        () async {
          entered.complete();
          await release.future;
        },
      );
      await entered.future;
      harness.start();
      await harness.settle();
      expect(harness.registry.billingCustomers[_key('bob').did], _uuid('bob'));
      expect(harness.service.identified, isEmpty);
      await harness.signIn('charlie');
      await harness.settle();
      release.complete();
      await purchase;
      await harness.settle();
      expect(harness.service.identified, [_uuid('charlie')]);
    },
  );

  test('successful setup refreshes the shared access projection', () async {
    final harness = _Harness(initial: _registry('alice'));
    addTearDown(harness.dispose);
    await harness.container.read(sessionRegistryProvider.future);
    final lease = harness.registry.activeLease!.session;
    final projection = harness.container.listen(
      subscriptionAccessProvider(lease),
      (_, _) {},
    );
    addTearDown(projection.close);
    await harness.container.read(subscriptionAccessProvider(lease).future);
    expect(harness.api.accessReads, 1);
    harness.start();
    await harness.settle();
    expect(harness.api.accessReads, 2);
  });

  testWidgets('identity work and its failures do not block the account gate', (
    tester,
  ) async {
    final harness = _Harness(initial: _registry('alice'));
    addTearDown(harness.dispose);
    final delayed = Completer<BillingState>();
    harness.api.delayed[_did('alice')] = delayed.future;
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: harness.container,
        child: const MaterialApp(
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ActiveAccountInitializationGate(child: Text('Account ready')),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Account ready'), findsOneWidget);
    expect(harness.service.identified, isEmpty);
    delayed.completeError(StateError('test backend unavailable'));
    await tester.pumpAndSettle();
    expect(find.text('Account ready'), findsOneWidget);
    expect(
      harness.container.read(activeAccountInitializationProvider).hasValue,
      isTrue,
    );
    expect(
      harness.container
          .read(subscriptionIdentitySyncProvider(harness.registry.activeLease!))
          .hasError,
      isTrue,
    );
    expect(tester.takeException(), isNull);
  });
}

String _did(String name) => 'did:plc:$name';
AccountKey _key(String name) => AccountKey(_did(name));
String _uuid(String name) => switch (name) {
  'alice' => '20000000-0000-4000-8000-000000000001',
  'bob' => '20000000-0000-4000-8000-000000000002',
  _ => '20000000-0000-4000-8000-000000000003',
};
SessionRegistry _registry(String name) =>
    SessionRegistry.empty().upsertAndActivate(
      token: '$name-token',
      did: _did(name),
      handle: '$name.test',
    );
BillingState _state(String name) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: _uuid(name),
  requestedGeneration: 1,
  reconciledGeneration: 0,
  reconciliationStale: true,
  subscriptions: const [],
  licenses: const [],
);

final class _Readiness {
  bool onboarded = true;
  bool restricted = false;
}

final class _Harness {
  new({SessionRegistry? initial, bool enabled = true, bool useDio = false}) {
    container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(initial ?? SessionRegistry.empty()),
        ),
        subscriptionIdentitySyncEnabledProvider.overrideWithValue(enabled),
        activeAccountInitializationProvider.overrideWith((ref) {
          final lease = ref.watch(sessionRegistryProvider).value?.activeLease;
          return lease == null
              ? null
              : ActiveAccountInitialization(
                  lease: lease,
                  languagePreferences: const LanguagePreferences(
                    primaryLanguage: 'en',
                    contentLanguages: ['en'],
                  ),
                  onboardingComplete: readiness.onboarded,
                  accountEligibility: AccountEligibilityStatus(
                    state: readiness.restricted
                        ? AccountEligibilityState.restricted
                        : AccountEligibilityState.eligible,
                    appealable: false,
                  ),
                );
        }),
        if (useDio)
          accountDioProvider.overrideWith((ref, account) async {
            final did = account.did.value;
            final adapter = _Adapter(api, did);
            api.adapters[did] = adapter;
            final client = Dio()..httpClientAdapter = adapter;
            ref.onDispose(() => client.close(force: true));
            return client;
          })
        else
          subscriptionRepositoryProvider.overrideWith(
            (ref, account) async => _AccountApi(api, account.did.value),
          ),
        revenueCatServiceProvider.overrideWithValue(service),
      ],
    );
    service.hasCustomer = (id) => registry.billingCustomers.values.contains(id);
  }
  final api = _Api();
  final service = _Service();
  final readiness = _Readiness();
  late final ProviderContainer container;
  ProviderSubscription<void>? listener;
  SessionRegistry get registry =>
      container.read(sessionRegistryProvider).requireValue;
  void start() => listener = container.listen(
    activeAccountSubscriptionIdentitySyncProvider,
    (_, _) {},
  );
  Future<void> settle() async {
    for (var i = 0; i < 8; i++) {
      await Future<void>.delayed(Duration.zero);
    }
  }

  Future<void> signIn(String name) => container
      .read(sessionRegistryProvider.notifier)
      .upsertAndActivate(
        token: '$name-token',
        did: _did(name),
        handle: '$name.test',
      );
  Future<void> switchTo(String name) => container
      .read(sessionRegistryProvider.notifier)
      .activate(registry.leaseFor(_key(name))!);
  void dispose() {
    listener?.close();
    container.dispose();
  }
}

final class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Api {
  final calls = <String>[];
  final delayed = <String, Future<BillingState>>{};
  final responses = <String, Future<ResponseBody>>{};
  final adapters = <String, _Adapter>{};
  int accessReads = 0;
}

ResponseBody _response(String name) => ResponseBody.fromString(
  jsonEncode({
    'billingAccountId': '10000000-0000-4000-8000-000000000001',
    'revenueCatAppUserId': _uuid(name),
    'requestedGeneration': 1,
    'reconciledGeneration': 0,
    'reconciliationStale': true,
    'subscriptions': <Object>[],
    'licenses': <Object>[],
  }),
  200,
  headers: {
    Headers.contentTypeHeader: [Headers.jsonContentType],
  },
);

final class _Adapter implements HttpClientAdapter {
  new(this.api, this.did);
  final _Api api;
  final String did;
  bool closed = false;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    api.calls.add(did);
    return await (api.responses[did] ??
        Future.value(_response(did.split(':').last)));
  }

  @override
  void close({bool force = false}) => closed = true;
}

final class _AccountApi implements SubscriptionApi {
  new(this.api, this.did);
  final _Api api;
  final String did;
  String get name => did.split(':').last;
  @override
  Future<BillingState> ensureBillingAccount() async {
    api.calls.add(did);
    return await (api.delayed[did] ?? Future.value(_state(name)));
  }

  @override
  Future<BillingState> getBillingAccount() async => _state(name);
  @override
  Future<void> requestReconciliation() async {}
  @override
  Future<SubscriptionAccess> getAccess() async {
    api.accessReads++;
    return SubscriptionAccess(
      did: _key(name).did,
      effectiveTier: SubscriptionTier.free,
      givesAccess: false,
    );
  }
}

final class _Service implements RevenueCatService, NativeStoreOwnershipService {
  final identified = <String>[];
  final ownershipChecks = <String>[];
  final conflicts = <String>{};
  final ownershipWaits = <String, Future<void>>{};
  String? identity;
  bool available = true;
  bool persistedBeforeIdentify = true;
  bool Function(String)? hasCustomer;
  @override
  BillingAvailability get availability => available
      ? BillingAvailability.available
      : BillingAvailability.unavailable;
  @override
  String get store => 'app_store';
  @override
  bool get supportsPlanChanges => true;
  @override
  Future<void> verifyStoreOwnership(
    String appUserId, {
    bool requireNoActiveSubscription = false,
  }) async {
    ownershipChecks.add(appUserId);
    await ownershipWaits[appUserId];
    if (conflicts.contains(appUserId)) {
      throw const StoreAccountConflictException();
    }
  }

  @override
  Future<RevenueCatIdentity> currentIdentity() async => identity == null
      ? const RevenueCatIdentity.anonymous()
      : RevenueCatIdentity.identified(identity!);
  @override
  Future<void> identify(String appViewUuid) async {
    persistedBeforeIdentify =
        persistedBeforeIdentify && (hasCustomer?.call(appViewUuid) ?? true);
    identity = appViewUuid;
    identified.add(appViewUuid);
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async => null;
  @override
  Future<DirectPaywallResult> presentPaywall(
    RevenueCatOffering offering,
  ) async => DirectPaywallResult.cancelled;
  @override
  Future<void> restorePurchases() async {}
  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) async {}
}
