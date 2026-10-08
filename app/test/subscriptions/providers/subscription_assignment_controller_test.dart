import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_assignment_controller.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-008 success refreshes owner truth then target access', () async {
    final registry = _registry();
    final calls = <String>[];
    final api = _AssignmentApi(calls: calls);
    final controller = _controller(
      registry,
      api,
      readTargetAccess: (_) async {
        calls.add('target');
        return _access(SubscriptionTier.plus);
      },
    );

    final result = await controller.assign(
      licenseId: _licenseId,
      target: registry.leaseFor(AccountKey('did:plc:bob'))!,
    );

    expect(result.outcome, AssignmentMutationOutcome.success);
    expect(calls, ['owner', 'owner', 'target']);
    expect(result.targetAccess?.effectiveTier, SubscriptionTier.plus);
  });

  test(
    'AT-008 exact assignment error refreshes unchanged server truth',
    () async {
      final registry = _registry();
      final api = _AssignmentApi(
        error: const ApiBadRequest('assignment_cooldown'),
      );

      final result = await _controller(registry, api).assign(
        licenseId: _licenseId,
        target: registry.leaseFor(AccountKey('did:plc:bob'))!,
      );

      expect(result.outcome, AssignmentMutationOutcome.cooldown);
      expect(result.ownerState?.licenses.single.assignedDid, isNull);
      expect(api.ownerReads, 2);
    },
  );

  test('AT-008 duplicate taps issue one mutation', () async {
    final registry = _registry();
    final pending = Completer<void>();
    final api = _AssignmentApi(pending: pending);
    final controller = _controller(registry, api);
    final target = registry.leaseFor(AccountKey('did:plc:bob'))!;

    final first = controller.assign(licenseId: _licenseId, target: target);
    final duplicate = await controller.assign(
      licenseId: _licenseId,
      target: target,
    );
    pending.complete();
    await first;

    expect(duplicate.outcome, AssignmentMutationOutcome.inProgress);
    expect(api.assignmentCalls, 1);
  });

  test('IT-008 stale target is ineligible before mutation', () async {
    final registry = _registry();
    final api = _AssignmentApi();
    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;

    final result = await _controller(registry, api).assign(
      licenseId: _licenseId,
      target: AccountSessionLease(
        account: bob.account,
        sessionGeneration: bob.sessionGeneration + 1,
      ),
    );

    expect(result.outcome, AssignmentMutationOutcome.targetIneligible);
    expect(api.assignmentCalls, 0);
  });

  test(
    'AT-008 unassignment refreshes success and exact missing-license error',
    () async {
      final registry = _registry();
      final successApi = _AssignmentApi();
      final success = await _controller(
        registry,
        successApi,
      ).unassign(licenseId: _licenseId);
      final missingApi = _AssignmentApi(
        unassignError: const ApiBadRequest('billing_license_not_found'),
      );
      final missing = await _controller(
        registry,
        missingApi,
      ).unassign(licenseId: _licenseId);

      expect(success.outcome, AssignmentMutationOutcome.success);
      expect(missing.outcome, AssignmentMutationOutcome.licenseNotFound);
      expect(successApi.ownerReads, 2);
      expect(missingApi.ownerReads, 2);
    },
  );

  test('IT-014 UUID mismatch prevents assignment mutation', () async {
    final registry = _registry();
    final api = _AssignmentApi(
      ownerState: _state(
        revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
      ),
    );

    await expectLater(
      _controller(registry, api).assign(
        licenseId: _licenseId,
        target: registry.leaseFor(AccountKey('did:plc:bob'))!,
      ),
      throwsA(isA<BillingOwnerGuardException>()),
    );
    expect(api.assignmentCalls, 0);
  });

  test(
    'IT-006 assignment does not complete without matching self-access',
    () async {
      final registry = _registry();
      final result =
          await _controller(
            registry,
            _AssignmentApi(),
            readTargetAccess: (_) async => _access(SubscriptionTier.free),
          ).assign(
            licenseId: _licenseId,
            target: registry.leaseFor(AccountKey('did:plc:bob'))!,
          );

      expect(result.outcome, AssignmentMutationOutcome.failed);
    },
  );

  test('IT-008 assignment preserves every exact server outcome', () async {
    final registry = _registry();
    const cases = {
      'assignment_target_ineligible':
          AssignmentMutationOutcome.targetIneligible,
      'assignment_conflict': AssignmentMutationOutcome.conflict,
      'assignment_cooldown': AssignmentMutationOutcome.cooldown,
      'billing_license_not_found': AssignmentMutationOutcome.licenseNotFound,
    };

    for (final MapEntry(key: code, value: outcome) in cases.entries) {
      final result =
          await _controller(
            registry,
            _AssignmentApi(error: ApiBadRequest(code)),
          ).assign(
            licenseId: _licenseId,
            target: registry.leaseFor(AccountKey('did:plc:bob'))!,
          );
      expect(result.outcome, outcome, reason: code);
    }

    final unauthorized =
        await _controller(
          registry,
          _AssignmentApi(error: const ApiUnauthorized()),
        ).assign(
          licenseId: _licenseId,
          target: registry.leaseFor(AccountKey('did:plc:bob'))!,
        );
    final failed =
        await _controller(
          registry,
          _AssignmentApi(error: const ApiServerError('redacted')),
        ).assign(
          licenseId: _licenseId,
          target: registry.leaseFor(AccountKey('did:plc:bob'))!,
        );

    expect(unauthorized.outcome, AssignmentMutationOutcome.unauthorized);
    expect(failed.outcome, AssignmentMutationOutcome.failed);
  });

  test('IT-014 UUID mismatch prevents unassignment mutation', () async {
    final api = _AssignmentApi(
      ownerState: _state(
        revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
      ),
    );

    await expectLater(
      _controller(_registry(), api).unassign(licenseId: _licenseId),
      throwsA(isA<BillingOwnerGuardException>()),
    );
    expect(api.unassignmentCalls, 0);
  });

  test('IT-014 stale assignment owner reads return cancellation', () async {
    for (final staleRead in [1, 2]) {
      var registry = _registry();
      final api = _AssignmentApi(
        onOwnerRead: (read) {
          if (read == staleRead) {
            registry = registry.upsertAndActivate(
              token: 'replacement-bob-token',
              did: 'did:plc:bob',
              handle: 'bob.test',
            );
          }
        },
      );

      final result =
          await _controller(
            registry,
            api,
            readRegistry: () => registry,
          ).assign(
            licenseId: _licenseId,
            target: registry.leaseFor(AccountKey('did:plc:bob'))!,
          );

      expect(result.outcome, AssignmentMutationOutcome.cancelled);
      expect(api.assignmentCalls, staleRead == 1 ? 0 : 1);
    }
  });

  test('IT-014 stale unassignment owner reads return cancellation', () async {
    var registry = _registry();
    final api = _AssignmentApi(
      onOwnerRead: (read) {
        if (read == 2) {
          registry = registry.upsertAndActivate(
            token: 'replacement-bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
        }
      },
    );

    final result = await _controller(
      registry,
      api,
      readRegistry: () => registry,
    ).unassign(licenseId: _licenseId);

    expect(result.outcome, AssignmentMutationOutcome.cancelled);
    expect(api.unassignmentCalls, 1);
  });
}

SubscriptionAssignmentController _controller(
  SessionRegistry registry,
  _AssignmentApi api, {
  Future<SubscriptionAccess> Function(AccountSessionLease)? readTargetAccess,
  SessionRegistry Function()? readRegistry,
}) => SubscriptionAssignmentController(
  ownerGuard: BillingOwnerGuard(readRegistry ?? () => registry),
  readRegistry: readRegistry ?? () => registry,
  api: api,
  readTargetAccess:
      readTargetAccess ?? (_) async => _access(SubscriptionTier.free),
);

SessionRegistry _registry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
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

const _licenseId = '40000000-0000-4000-8000-000000000001';

BillingState _state({
  String revenueCatAppUserId = '20000000-0000-4000-8000-000000000001',
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: revenueCatAppUserId,
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  subscriptions: const [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000001',
      productId: 'craftsky_plus',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    ),
  ],
  licenses: const [
    BillingLicense(
      id: _licenseId,
      subscriptionId: '30000000-0000-4000-8000-000000000001',
      tier: SubscriptionTier.plus,
      assignable: true,
      anomaly: 'none',
    ),
  ],
);

SubscriptionAccess _access(SubscriptionTier tier) => SubscriptionAccess(
  did: Did.parse('did:plc:bob'),
  effectiveTier: tier,
  givesAccess: tier != SubscriptionTier.free,
);

final class _AssignmentApi implements SubscriptionApi {
  _AssignmentApi({
    this.error,
    this.unassignError,
    this.pending,
    this.calls,
    this.ownerState,
    this.onOwnerRead,
  });

  final ApiException? error;
  final ApiException? unassignError;
  final Completer<void>? pending;
  final List<String>? calls;
  final BillingState? ownerState;
  final void Function(int read)? onOwnerRead;
  int assignmentCalls = 0;
  int unassignmentCalls = 0;
  int ownerReads = 0;

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) async {
    assignmentCalls++;
    await pending?.future;
    if (error case final error?) throw error;
    return BillingAssignment(
      licenseId: licenseId,
      targetDid: Did.parse(targetDid),
      assignedAt: DateTime.utc(2026),
    );
  }

  @override
  Future<BillingState> getBillingAccount() async {
    ownerReads++;
    onOwnerRead?.call(ownerReads);
    calls?.add('owner');
    return ownerState ?? _state();
  }

  @override
  Future<void> unassign(String licenseId) async {
    unassignmentCalls++;
    if (unassignError case final error?) throw error;
  }

  @override
  Future<BillingState> ensureBillingAccount() => throw UnimplementedError();

  @override
  Future<SubscriptionAccess> getAccess() => throw UnimplementedError();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();
}
