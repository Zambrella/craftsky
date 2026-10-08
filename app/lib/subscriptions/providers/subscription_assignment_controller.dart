import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';

enum AssignmentMutationOutcome {
  success,
  targetIneligible,
  conflict,
  cooldown,
  licenseNotFound,
  unauthorized,
  failed,
  cancelled,
  inProgress,
}

final class AssignmentMutationResult {
  const AssignmentMutationResult({
    required this.outcome,
    this.ownerState,
    this.targetAccess,
  });

  final AssignmentMutationOutcome outcome;
  final BillingState? ownerState;
  final SubscriptionAccess? targetAccess;
}

final class SubscriptionAssignmentController {
  SubscriptionAssignmentController({
    required this.ownerGuard,
    required this.readRegistry,
    required this.api,
    required this.readTargetAccess,
    this.isCurrent = _alwaysCurrent,
  });

  final BillingOwnerGuard ownerGuard;
  final SessionRegistry Function() readRegistry;
  final SubscriptionApi api;
  final Future<SubscriptionAccess> Function(AccountSessionLease target)
  readTargetAccess;
  final bool Function() isCurrent;
  bool _mutationInProgress = false;

  Future<AssignmentMutationResult> assign({
    required String licenseId,
    required AccountSessionLease target,
  }) async {
    if (_mutationInProgress) {
      return const AssignmentMutationResult(
        outcome: AssignmentMutationOutcome.inProgress,
      );
    }
    _mutationInProgress = true;
    try {
      final owner = ownerGuard.capture();
      final beforeMutation = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
      if (!isCurrent()) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      }
      final license = beforeMutation.licenses
          .where((value) => value.id == licenseId)
          .firstOrNull;
      if (license == null) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.licenseNotFound,
        );
      }
      if (!_targetIsCurrent(target)) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.targetIneligible,
        );
      }

      var outcome = AssignmentMutationOutcome.success;
      try {
        await ownerGuard.dispatch(
          owner,
          () => api.assign(licenseId, target.account.did.value),
        );
      } on ApiUnauthorized {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.unauthorized,
        );
      } on ApiBadRequest catch (error) {
        outcome = _mapError(error.code);
      } on BillingOwnerGuardException {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      } on Object {
        outcome = AssignmentMutationOutcome.failed;
      }

      if (!isCurrent()) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      }

      final ownerState = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
      if (outcome != AssignmentMutationOutcome.success) {
        return AssignmentMutationResult(
          outcome: outcome,
          ownerState: ownerState,
        );
      }
      if (!_targetIsCurrent(target)) {
        return AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.targetIneligible,
          ownerState: ownerState,
        );
      }
      final targetAccess = await readTargetAccess(target);
      if (!ownerGuard.isCurrent(owner) || !_targetIsCurrent(target)) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      }
      if (targetAccess.did != target.account.did ||
          !targetAccess.givesAccess ||
          targetAccess.effectiveTier != license.tier) {
        return AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.failed,
          ownerState: ownerState,
          targetAccess: targetAccess,
        );
      }
      return AssignmentMutationResult(
        outcome: AssignmentMutationOutcome.success,
        ownerState: ownerState,
        targetAccess: targetAccess,
      );
    } on BillingOwnerGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const AssignmentMutationResult(
        outcome: AssignmentMutationOutcome.cancelled,
      );
    } finally {
      _mutationInProgress = false;
    }
  }

  Future<AssignmentMutationResult> unassign({required String licenseId}) async {
    if (_mutationInProgress) {
      return const AssignmentMutationResult(
        outcome: AssignmentMutationOutcome.inProgress,
      );
    }
    _mutationInProgress = true;
    try {
      final owner = ownerGuard.capture();
      await ownerGuard.dispatchBillingState(owner, api.getBillingAccount);
      if (!isCurrent()) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      }
      var outcome = AssignmentMutationOutcome.success;
      try {
        await ownerGuard.dispatch(owner, () => api.unassign(licenseId));
      } on ApiUnauthorized {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.unauthorized,
        );
      } on ApiBadRequest catch (error) {
        outcome = _mapError(error.code);
      } on BillingOwnerGuardException {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      } on Object {
        outcome = AssignmentMutationOutcome.failed;
      }
      if (!isCurrent()) {
        return const AssignmentMutationResult(
          outcome: AssignmentMutationOutcome.cancelled,
        );
      }
      final ownerState = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
      return AssignmentMutationResult(
        outcome: outcome,
        ownerState: ownerState,
      );
    } on BillingOwnerGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const AssignmentMutationResult(
        outcome: AssignmentMutationOutcome.cancelled,
      );
    } finally {
      _mutationInProgress = false;
    }
  }

  bool _targetIsCurrent(AccountSessionLease target) =>
      readRegistry().leaseFor(target.account) == target;

  AssignmentMutationOutcome _mapError(String? code) => switch (code) {
    'assignment_target_ineligible' =>
      AssignmentMutationOutcome.targetIneligible,
    'assignment_conflict' => AssignmentMutationOutcome.conflict,
    'assignment_cooldown' => AssignmentMutationOutcome.cooldown,
    'billing_license_not_found' => AssignmentMutationOutcome.licenseNotFound,
    _ => AssignmentMutationOutcome.failed,
  };
}

bool _alwaysCurrent() => true;
