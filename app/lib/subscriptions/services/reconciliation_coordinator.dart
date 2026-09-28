import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';

enum ReconciliationIntent { purchase, restore }

enum ReconciliationDecision { pending, completed, failed }

enum ReconciliationOutcome { completed, failed, timedOut, cancelled }

final class ReconciliationEvaluation {
  const ReconciliationEvaluation(this.decision, this.state);

  final ReconciliationDecision decision;
  final BillingState state;
}

final class ReconciliationResult {
  const ReconciliationResult({
    required this.outcome,
    required this.targetGeneration,
    this.state,
  });

  final ReconciliationOutcome outcome;
  final int? targetGeneration;
  final BillingState? state;
}

final class ReconciliationTracker {
  ReconciliationTracker({
    required this.baseline,
    required this.intent,
    BillingState? comparisonBaseline,
    this.expectedTier,
  }) : comparisonBaseline = comparisonBaseline ?? baseline;

  final BillingState baseline;
  final BillingState comparisonBaseline;
  final ReconciliationIntent intent;
  final SubscriptionTier? expectedTier;
  int? targetGeneration;

  ReconciliationEvaluation evaluate(BillingState state) {
    targetGeneration ??=
        state.requestedGeneration > baseline.requestedGeneration
        ? state.requestedGeneration
        : null;
    final target = targetGeneration;
    if (target == null || state.reconciledGeneration < target) {
      return ReconciliationEvaluation(ReconciliationDecision.pending, state);
    }
    if (_hasAnomaly(state)) {
      return ReconciliationEvaluation(ReconciliationDecision.failed, state);
    }
    if (intent == ReconciliationIntent.purchase &&
        _hasInvalidNewPurchaseLicense(comparisonBaseline, state)) {
      return ReconciliationEvaluation(ReconciliationDecision.failed, state);
    }
    if (intent == ReconciliationIntent.restore ||
        _hasRecognizedPurchaseChange(comparisonBaseline, state)) {
      return ReconciliationEvaluation(ReconciliationDecision.completed, state);
    }
    return ReconciliationEvaluation(ReconciliationDecision.pending, state);
  }

  bool _hasAnomaly(BillingState state) {
    if (!billingRelationshipsResolved(state)) return true;
    if (expectedTier == null) {
      return state.subscriptions.any(
            (value) => isBillingAnomaly(value.anomaly),
          ) ||
          state.licenses.any((value) => isBillingAnomaly(value.anomaly));
    }
    final subscriptions = {
      for (final value in state.subscriptions) value.id: value,
    };
    return state.licenses
        .where((value) => value.tier == expectedTier)
        .any(
          (license) =>
              isBillingAnomaly(license.anomaly) ||
              isBillingAnomaly(
                subscriptions[license.subscriptionId]?.anomaly ?? '',
              ),
        );
  }

  bool _hasRecognizedPurchaseChange(
    BillingState baseline,
    BillingState current,
  ) {
    final baselineLicenseIds = baseline.licenses
        .map((value) => value.id)
        .toSet();
    final tierLicenses = current.licenses.where(
      (value) => expectedTier == null || value.tier == expectedTier,
    );
    final subscriptions = {
      for (final value in current.subscriptions) value.id: value,
    };
    if (tierLicenses.any(
      (value) =>
          !baselineLicenseIds.contains(value.id) &&
          (subscriptions[value.subscriptionId]?.givesAccess ?? false),
    )) {
      return true;
    }
    final baselineSubscriptions = {
      for (final value in baseline.subscriptions) value.id: value,
    };
    final tierSubscriptionIds = tierLicenses
        .map((value) => value.subscriptionId)
        .toSet();
    return current.subscriptions.any((value) {
      if (!tierSubscriptionIds.contains(value.id)) return false;
      final previous = baselineSubscriptions[value.id];
      return previous != null && !previous.givesAccess && value.givesAccess;
    });
  }

  bool _hasInvalidNewPurchaseLicense(
    BillingState baseline,
    BillingState current,
  ) {
    final baselineLicenseIds = baseline.licenses
        .map((value) => value.id)
        .toSet();
    final subscriptions = {
      for (final value in current.subscriptions) value.id: value,
    };
    return current.licenses.any(
      (license) =>
          (expectedTier == null || license.tier == expectedTier) &&
          !baselineLicenseIds.contains(license.id) &&
          !(subscriptions[license.subscriptionId]?.givesAccess ?? false),
    );
  }
}

typedef ReconciliationWait = Future<void> Function(Duration duration);

final class ReconciliationCoordinator {
  ReconciliationCoordinator({
    required this.requestReconciliation,
    required this.readBillingState,
    ReconciliationWait? wait,
    bool Function()? isCurrent,
    this.maxAttempts = 30,
  }) : wait = wait ?? _defaultWait,
       isCurrent = isCurrent ?? _alwaysCurrent {
    if (maxAttempts < 1) throw ArgumentError.value(maxAttempts, 'maxAttempts');
  }

  final Future<void> Function() requestReconciliation;
  final Future<BillingState> Function() readBillingState;
  final ReconciliationWait wait;
  final bool Function() isCurrent;
  final int maxAttempts;

  Future<ReconciliationResult> reconcile({
    required BillingState baseline,
    required ReconciliationIntent intent,
    BillingState? comparisonBaseline,
    SubscriptionTier? expectedTier,
  }) async {
    if (!isCurrent()) return _result(ReconciliationOutcome.cancelled, null);
    try {
      await requestReconciliation();
    } on Object {
      return _result(
        isCurrent()
            ? ReconciliationOutcome.failed
            : ReconciliationOutcome.cancelled,
        null,
      );
    }

    final tracker = ReconciliationTracker(
      baseline: baseline,
      intent: intent,
      comparisonBaseline: comparisonBaseline,
      expectedTier: expectedTier,
    );
    for (var attempt = 0; attempt < maxAttempts; attempt++) {
      if (!isCurrent()) {
        return _result(
          ReconciliationOutcome.cancelled,
          tracker.targetGeneration,
        );
      }
      try {
        final evaluation = tracker.evaluate(await readBillingState());
        if (!isCurrent()) {
          return _result(
            ReconciliationOutcome.cancelled,
            tracker.targetGeneration,
          );
        }
        if (evaluation.decision != ReconciliationDecision.pending) {
          return ReconciliationResult(
            outcome: evaluation.decision == ReconciliationDecision.completed
                ? ReconciliationOutcome.completed
                : ReconciliationOutcome.failed,
            targetGeneration: tracker.targetGeneration,
            state: evaluation.state,
          );
        }
      } on Object {
        if (!isCurrent()) {
          return _result(
            ReconciliationOutcome.cancelled,
            tracker.targetGeneration,
          );
        }
      }
      if (attempt + 1 < maxAttempts) {
        await wait(const Duration(seconds: 1));
      }
    }
    return _result(ReconciliationOutcome.timedOut, tracker.targetGeneration);
  }

  ReconciliationResult _result(
    ReconciliationOutcome outcome,
    int? targetGeneration,
  ) => ReconciliationResult(
    outcome: outcome,
    targetGeneration: targetGeneration,
  );

  static Future<void> _defaultWait(Duration duration) =>
      Future.delayed(duration);
  static bool _alwaysCurrent() => true;
}
