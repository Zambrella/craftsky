import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_setup_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-001 persists DID and UUID before ensure and identity', () async {
    var registry = _aliceRegistry();
    final events = <String>[];
    final api = _FakeSubscriptionApi(
      ensure: () async {
        events.add('ensure');
        return _billingState();
      },
    );
    final revenueCat = _FakeRevenueCat(events);
    final coordinator = BillingOwnerSetupCoordinator(
      readRegistry: () => registry,
      reserveOwner: (did) async {
        registry = registry.reserveBillingOwner(did);
        events.add('persist-did');
      },
      completeOwner: (did, appUserId) async {
        registry = registry.completeBillingOwner(did, appUserId);
        events.add('persist-uuid');
      },
      api: api,
      revenueCat: revenueCat,
    );

    expect(events, isEmpty);
    await coordinator.setup(registry.activeLease!);

    expect(events, [
      'persist-did',
      'ensure',
      'persist-uuid',
      'identity-read',
      'identify',
    ]);
    expect(
      registry.billingOwner?.revenueCatAppUserId,
      _revenueCatAppUserId,
    );
  });

  test('AT-011 owner removal fences a pending GET completion', () async {
    var registry = _completedAliceRegistry();
    final getStarted = Completer<void>();
    final getResult = Completer<BillingState>();
    final events = <String>[];
    final api = _FakeSubscriptionApi(
      get: () {
        getStarted.complete();
        return getResult.future;
      },
    );
    final revenueCat = _FakeRevenueCat(events);
    final coordinator = _coordinator(
      registry: () => registry,
      replaceRegistry: (next) => registry = next,
      api: api,
      revenueCat: revenueCat,
    );

    final operation = coordinator.setup(registry.activeLease!);
    await getStarted.future;
    registry = registry.remove('did:plc:alice');
    getResult.complete(_billingState());

    await expectLater(
      operation,
      throwsA(
        isA<BillingOwnerSetupException>().having(
          (error) => error.code,
          'code',
          'owner_session_changed',
        ),
      ),
    );
    expect(registry.billingOwner?.did.value, 'did:plc:alice');
    expect(events, isEmpty);
  });

  test(
    'AT-011 another active DID cannot replace or ensure the owner',
    () async {
      var registry = _completedAliceRegistry().upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      var apiCalls = 0;
      final coordinator = _coordinator(
        registry: () => registry,
        replaceRegistry: (next) => registry = next,
        api: _FakeSubscriptionApi(
          get: () async {
            apiCalls++;
            return _billingState();
          },
        ),
        revenueCat: _FakeRevenueCat(<String>[]),
      );

      await expectLater(
        coordinator.setup(registry.activeLease!),
        throwsA(isA<BillingOwnerSetupException>()),
      );

      expect(apiCalls, 0);
      expect(registry.billingOwner?.did.value, 'did:plc:alice');
    },
  );

  test('AT-001 failed durable writes stop later setup calls', () async {
    final registry = _aliceRegistry();
    var ensureCalls = 0;
    final events = <String>[];
    final api = _FakeSubscriptionApi(
      ensure: () async {
        ensureCalls++;
        return _billingState();
      },
    );
    final didWriteFailure = BillingOwnerSetupCoordinator(
      readRegistry: () => registry,
      reserveOwner: (_) async => throw StateError('storage unavailable'),
      completeOwner: (_, _) async {},
      api: api,
      revenueCat: _FakeRevenueCat(events),
    );

    await expectLater(
      didWriteFailure.setup(registry.activeLease!),
      throwsStateError,
    );
    expect(ensureCalls, 0);

    final reserved = registry.reserveBillingOwner('did:plc:alice');
    final uuidWriteFailure = BillingOwnerSetupCoordinator(
      readRegistry: () => reserved,
      reserveOwner: (_) async {},
      completeOwner: (_, _) async => throw StateError('storage unavailable'),
      api: api,
      revenueCat: _FakeRevenueCat(events),
    );
    await expectLater(
      uuidWriteFailure.setup(reserved.activeLease!),
      throwsStateError,
    );
    expect(ensureCalls, 1);
    expect(reserved.billingOwner?.revenueCatAppUserId, isNull);
    expect(events, isEmpty);
  });
}

BillingOwnerSetupCoordinator _coordinator({
  required SessionRegistry Function() registry,
  required void Function(SessionRegistry value) replaceRegistry,
  required SubscriptionApi api,
  required RevenueCatIdentityService revenueCat,
}) => BillingOwnerSetupCoordinator(
  readRegistry: registry,
  reserveOwner: (did) async {
    replaceRegistry(registry().reserveBillingOwner(did));
  },
  completeOwner: (did, appUserId) async {
    replaceRegistry(registry().completeBillingOwner(did, appUserId));
  },
  api: api,
  revenueCat: revenueCat,
);

SessionRegistry _aliceRegistry() => SessionRegistry.empty().upsertAndActivate(
  token: 'alice-token',
  did: 'did:plc:alice',
  handle: 'alice.test',
);

SessionRegistry _completedAliceRegistry() => _aliceRegistry()
    .reserveBillingOwner('did:plc:alice')
    .completeBillingOwner('did:plc:alice', _revenueCatAppUserId);

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
  _FakeRevenueCat(this.events);

  final List<String> events;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    events.add('identity-read');
    return const RevenueCatIdentity.anonymous();
  }

  @override
  Future<void> identify(String appViewUuid) async {
    events.add('identify');
  }
}

final class _FakeSubscriptionApi implements SubscriptionApi {
  _FakeSubscriptionApi({this.ensure, this.get});

  final Future<BillingState> Function()? ensure;
  final Future<BillingState> Function()? get;

  @override
  Future<BillingState> ensureBillingAccount() => ensure!();

  @override
  Future<BillingState> getBillingAccount() => get!();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      throw UnimplementedError();

  @override
  Future<void> unassign(String licenseId) => throw UnimplementedError();

  @override
  Future<SubscriptionAccess> getAccess() => throw UnimplementedError();
}
