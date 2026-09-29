import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/customer_center_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-009 management callback reconciles and refreshes owner state',
    () async {
      final api = _ScriptedApi([_state(7, 7), _state(7, 7), _state(8, 8)]);
      final service = _CustomerCenterService(
        event: CustomerCenterEvent.managementOptionSelected,
      );

      final result = await _coordinator(api, service).present();

      expect(result.outcome, CustomerCenterOutcome.completed);
      expect(result.state?.reconciledGeneration, 8);
      expect(api.reconciliationCalls, 1);
    },
  );

  test('AT-010 plain dismissal refreshes without reconciliation', () async {
    final api = _ScriptedApi([_state(7, 7), _state(7, 7)]);

    final result = await _coordinator(
      api,
      _CustomerCenterService(),
    ).present();

    expect(result.outcome, CustomerCenterOutcome.refreshed);
    expect(api.reconciliationCalls, 0);
  });

  test('AT-014 reconciled Customer Center anomaly requires support', () async {
    final api = _ScriptedApi([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, anomaly: true),
    ]);
    final service = _CustomerCenterService(
      event: CustomerCenterEvent.managementOptionSelected,
    );

    final result = await _coordinator(api, service).present();

    expect(result.outcome.name, 'anomaly');
    expect(result.state, isNotNull);
    expect(result.retry, isNull);
  });

  test('IT-009 mismatched identity never presents Customer Center', () async {
    final service = _CustomerCenterService(
      appUserId: '20000000-0000-4000-8000-000000000099',
    );
    final result = await _coordinator(
      _ScriptedApi([_state(7, 7)]),
      service,
    ).present();

    expect(result.outcome, CustomerCenterOutcome.providerError);
    expect(service.presentations, 0);
  });

  test('IT-014 AppView UUID mismatch never presents Customer Center', () async {
    final service = _CustomerCenterService();

    await expectLater(
      _coordinator(
        _ScriptedApi([
          _state(
            7,
            7,
            revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
          ),
        ]),
        service,
      ).present(),
      throwsA(isA<BillingOwnerGuardException>()),
    );
    expect(service.presentations, 0);
  });

  test('IT-009 stale Customer Center mutation remains pending', () async {
    var current = true;
    final service = _CustomerCenterService(
      event: CustomerCenterEvent.managementOptionSelected,
      onPresent: () => current = false,
    );

    final result = await _coordinator(
      _ScriptedApi([_state(7, 7)]),
      service,
      isCurrent: () => current,
    ).present();

    expect(result.outcome, CustomerCenterOutcome.pending);
  });

  test('IT-010 stale pre-Customer Center owner read is cancelled', () async {
    var registry = _registry();
    final api = _ScriptedApi(
      [_state(7, 7)],
      onRead: (read) {
        if (read == 1) registry = _beneficiaryActiveRegistry();
      },
    );
    final service = _CustomerCenterService();

    final result = await _coordinator(
      api,
      service,
      readRegistry: () => registry,
    ).present();

    expect(result.outcome, CustomerCenterOutcome.cancelled);
    expect(service.presentations, 0);
  });

  test('IT-010 stale post-Customer Center owner read is cancelled', () async {
    var registry = _registry();
    final api = _ScriptedApi(
      [_state(7, 7), _state(7, 7)],
      onRead: (read) {
        if (read == 2) registry = _beneficiaryActiveRegistry();
      },
    );
    final service = _CustomerCenterService();

    final result = await _coordinator(
      api,
      service,
      readRegistry: () => registry,
    ).present();

    expect(result.outcome, CustomerCenterOutcome.cancelled);
    expect(service.presentations, 1);
  });

  test('IT-009 management return reconciles after foreground resume', () async {
    var foreground = true;
    final api = _ScriptedApi([_state(7, 7), _state(7, 7), _state(8, 8)]);
    final service = _CustomerCenterService(
      event: CustomerCenterEvent.managementOptionSelected,
      onPresent: () => foreground = false,
    );

    final result = await _coordinator(
      api,
      service,
      isForeground: () => foreground,
    ).present();
    expect(result.outcome, CustomerCenterOutcome.pending);
    expect(result.retry, isNotNull);
    expect(api.reconciliationCalls, 0);

    foreground = true;
    final resumed = await result.retry!();
    expect(resumed.outcome, CustomerCenterOutcome.completed);
    expect(api.reconciliationCalls, 1);
    expect(service.presentations, 1);
  });
}

CustomerCenterCoordinator _coordinator(
  _ScriptedApi api,
  _CustomerCenterService service, {
  bool Function()? isCurrent,
  bool Function()? isForeground,
  SessionRegistry Function()? readRegistry,
}) => CustomerCenterCoordinator(
  ownerGuard: BillingOwnerGuard(readRegistry ?? _registry),
  api: api,
  revenueCat: service,
  wait: (_) async {},
  isCurrent: isCurrent ?? () => true,
  isForeground: isForeground ?? () => true,
  maxReconciliationAttempts: 3,
);

SessionRegistry _registry() => SessionRegistry.empty()
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

SessionRegistry _beneficiaryActiveRegistry() => _registry().upsertAndActivate(
  token: 'bob-token',
  did: 'did:plc:bob',
  handle: 'bob.test',
);

BillingState _state(
  int requested,
  int reconciled, {
  String revenueCatAppUserId = '20000000-0000-4000-8000-000000000001',
  bool anomaly = false,
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: revenueCatAppUserId,
  requestedGeneration: requested,
  reconciledGeneration: reconciled,
  reconciliationStale: requested > reconciled,
  subscriptions: anomaly
      ? const [
          BillingSubscription(
            id: '30000000-0000-4000-8000-000000000001',
            productId: 'craftsky_plus',
            store: 'app_store',
            status: 'active',
            givesAccess: true,
            pendingPayment: false,
            anomaly: 'duplicate_tier',
          ),
        ]
      : const [],
  licenses: anomaly
      ? const [
          BillingLicense(
            id: '40000000-0000-4000-8000-000000000001',
            subscriptionId: '30000000-0000-4000-8000-000000000001',
            tier: SubscriptionTier.plus,
            assignable: true,
            anomaly: 'duplicate_tier',
          ),
        ]
      : const [],
);

final class _CustomerCenterService implements RevenueCatService {
  _CustomerCenterService({
    this.event,
    this.appUserId = '20000000-0000-4000-8000-000000000001',
    this.onPresent,
  });

  final CustomerCenterEvent? event;
  final String appUserId;
  final void Function()? onPresent;
  int presentations = 0;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async =>
      RevenueCatIdentity.identified(appUserId);

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) async {
    presentations++;
    onPresent?.call();
    if (event case final event?) onEvent(event);
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) =>
      throw UnimplementedError();

  @override
  Future<void> identify(String appViewUuid) => throw UnimplementedError();

  @override
  Future<DirectPaywallResult> presentPaywall(RevenueCatOffering offering) =>
      throw UnimplementedError();

  @override
  Future<void> restorePurchases() => throw UnimplementedError();
}

final class _ScriptedApi implements SubscriptionApi {
  _ScriptedApi(this.states, {this.onRead});

  final List<BillingState> states;
  final void Function(int read)? onRead;
  int billingReads = 0;
  int reconciliationCalls = 0;

  @override
  Future<BillingState> getBillingAccount() async {
    billingReads++;
    onRead?.call(billingReads);
    return states.removeAt(0);
  }

  @override
  Future<void> requestReconciliation() async => reconciliationCalls++;

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      throw UnimplementedError();

  @override
  Future<BillingState> ensureBillingAccount() => throw UnimplementedError();

  @override
  Future<SubscriptionAccess> getAccess() => throw UnimplementedError();

  @override
  Future<void> unassign(String licenseId) => throw UnimplementedError();
}
