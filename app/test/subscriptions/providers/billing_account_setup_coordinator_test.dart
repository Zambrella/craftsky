import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_setup_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('customer ID is persisted before SDK identity changes', () async {
    var registry = _aliceRegistry();
    final events = <String>[];
    await BillingAccountSetupCoordinator(
      readRegistry: () => registry,
      setCustomer: (did, id) async {
        registry = registry.setBillingCustomer(did, id);
        events.add('persist');
      },
      api: _FakeSubscriptionApi(
        ensure: () async {
          events.add('ensure');
          return _billingState();
        },
      ),
      revenueCat: _FakeRevenueCat(events),
    ).setup(registry.activeLease!);
    expect(events.take(4), ['ensure', 'persist', 'identity-read', 'identify']);
    expect(registry.billingCustomers.values.single, _revenueCatAppUserId);
  });
  test(
    'account switch during ensure cannot bind the result to the next DID',
    () async {
      var registry = _aliceRegistry();
      final response = Completer<BillingState>();
      var persists = 0;
      final events = <String>[];
      final operation = BillingAccountSetupCoordinator(
        readRegistry: () => registry,
        setCustomer: (_, _) async {
          persists++;
        },
        api: _FakeSubscriptionApi(ensure: () => response.future),
        revenueCat: _FakeRevenueCat(events),
      ).setup(registry.activeLease!);
      registry = registry.upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      response.complete(_billingState());
      await expectLater(
        operation,
        throwsA(isA<BillingAccountSetupException>()),
      );
      expect(persists, 0);
      expect(events, isEmpty);
    },
  );
  test('failed persistence prevents SDK identification', () async {
    final registry = _aliceRegistry();
    final events = <String>[];
    await expectLater(
      BillingAccountSetupCoordinator(
        readRegistry: () => registry,
        setCustomer: (_, _) async => throw StateError('storage unavailable'),
        api: _FakeSubscriptionApi(ensure: () async => _billingState()),
        revenueCat: _FakeRevenueCat(events),
      ).setup(registry.activeLease!),
      throwsStateError,
    );
    expect(events, isEmpty);
  });
}

SessionRegistry _aliceRegistry() => SessionRegistry.empty().upsertAndActivate(
  token: 'alice-token',
  did: 'did:plc:alice',
  handle: 'alice.test',
);

const _revenueCatAppUserId = '20000000-0000-4000-8000-000000000001';

BillingState _billingState() => const BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: _revenueCatAppUserId,
  requestedGeneration: 0,
  reconciledGeneration: 0,
  reconciliationStale: false,
  subscriptions: [],
  licenses: [],
);

final class _FakeRevenueCat implements RevenueCatIdentityService {
  new(this.events);

  final List<String> events;
  String? appUserId;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    events.add('identity-read');
    return appUserId == null
        ? const RevenueCatIdentity.anonymous()
        : RevenueCatIdentity.identified(appUserId!);
  }

  @override
  Future<void> identify(String appViewUuid) async {
    events.add('identify');
    appUserId = appViewUuid;
  }
}

final class _FakeSubscriptionApi implements SubscriptionApi {
  new({this.ensure});

  final Future<BillingState> Function()? ensure;

  @override
  Future<BillingState> ensureBillingAccount() => ensure!();

  @override
  Future<BillingState> getBillingAccount() => ensure!();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();

  @override
  Future<SubscriptionAccess> getAccess() => throw UnimplementedError();
}
