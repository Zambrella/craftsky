import 'dart:convert';

import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';
import 'package:craftsky_app/subscriptions/services/subscription_diagnostics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/serialized_sentry_transport.dart';

const _baseline = BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 7,
  reconciledGeneration: 7,
  reconciliationStale: false,
  subscriptions: [],
  licenses: [],
);

void main() {
  final stack = StackTrace.fromString(
    '#0 reconcileBilling (package:craftsky_app/subscriptions/billing.dart:42:7)',
  );

  test(
    'exhausted polling reports the last original failure only once',
    () async {
      final reporter = _Reporter();
      final error = Exception('PRIVATE_PAYMENT');
      var reads = 0;
      final result = await ReconciliationCoordinator(
        requestReconciliation: () async {},
        readBillingState: () async {
          reads++;
          Error.throwWithStackTrace(error, stack);
        },
        wait: (_) async {},
        maxAttempts: 3,
        diagnostics: SubscriptionDiagnostics(
          reporter: reporter,
          accountDid: 'did:plc:initiating',
        ),
      ).reconcile(baseline: _baseline, intent: ReconciliationIntent.restore);

      expect(result.outcome, ReconciliationOutcome.timedOut);
      expect(reads, 3);
      expect(reporter.errors, [same(error)]);
      expect(reporter.stacks.single.toString(), stack.toString());
      expect(reporter.contexts.single.operation, 'reconciliation.poll');
      expect(
        reporter.contexts.single.workflow,
        isA<PrivateOperationalFailureContext>(),
      );
    },
  );

  test('recovery and cancellation leave polling failures unreported', () async {
    for (final cancel in [false, true]) {
      final reporter = _Reporter();
      var reads = 0;
      var current = true;
      final result = await ReconciliationCoordinator(
        requestReconciliation: () async {},
        readBillingState: () async {
          if (++reads == 1) throw Exception('PRIVATE_PAYMENT');
          return _baseline;
        },
        wait: (_) async {
          if (cancel) current = false;
        },
        isCurrent: () => current,
        maxAttempts: 3,
        diagnostics: SubscriptionDiagnostics(reporter: reporter),
      ).reconcile(baseline: _baseline, intent: ReconciliationIntent.restore);
      expect(
        result.outcome,
        cancel
            ? ReconciliationOutcome.cancelled
            : ReconciliationOutcome.timedOut,
      );
      expect(reporter.errors, isEmpty);
    }
  });

  test('expected offline, cancellation and SDK rejection stay local', () {
    final reporter = _Reporter();
    final diagnostics = SubscriptionDiagnostics(reporter: reporter);
    for (final error in [
      const ApiNetworkError('PRIVATE_NETWORK'),
      const ApiCanceled(),
      const ApiUnauthorized(),
      const BillingOwnerGuardException('owner_session_changed'),
      RevenueCatServiceException(
        'restore_failed',
        diagnosticCause: PlatformException(
          code: '1',
          message: 'PRIVATE_CANCEL',
        ),
      ),
      RevenueCatServiceException(
        'restore_failed',
        diagnosticCause: PlatformException(
          code: '10',
          message: 'PRIVATE_NETWORK',
        ),
      ),
    ]) {
      diagnostics.report(error, stack, operation: 'restore.provider');
    }
    expect(reporter.errors, isEmpty);
  });

  test(
    'reporter failure cannot alter the failed reconciliation result',
    () async {
      final reporter = _Reporter()..fail = true;
      final error = Exception('PRIVATE_PAYMENT');
      final result = await ReconciliationCoordinator(
        requestReconciliation: () async => throw error,
        readBillingState: () async => _baseline,
        maxAttempts: 1,
        diagnostics: SubscriptionDiagnostics(reporter: reporter),
      ).reconcile(baseline: _baseline, intent: ReconciliationIntent.restore);
      expect(result.outcome, ReconciliationOutcome.failed);
      expect(reporter.errors, [same(error)]);
    },
  );

  test(
    'serialized subscription issues preserve evidence without payment data',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options, enableLogs: false);
      });
      addTearDown(Sentry.close);
      const requestId = '12345678-1234-4123-8123-123456789012';
      final error = ApiServerError(
        'PRIVATE_PAYMENT_PROSE',
        details: ApiFailureDetails(
          method: 'POST',
          statusCode: 503,
          appViewError: 'billing_unavailable',
          requestId: requestId,
          cause: StateError('PRIVATE_CUSTOMER_ID'),
          stackTrace: stack,
          fields: const {
            'license': 'PRIVATE_LICENSE',
            'token': 'PRIVATE_TOKEN',
          },
        ),
      );
      const SubscriptionDiagnostics(
        reporter: SentryErrorReporter(),
        accountDid: 'did:plc:initiating',
      ).report(error, stack, operation: 'reconciliation.request');
      await transport.issueReceived.future.timeout(const Duration(seconds: 5));
      await Sentry.close();
      final events = transport.payloads
          .map(jsonDecode)
          .where((value) => (value as Map).containsKey('exception'))
          .toList();
      expect(events, hasLength(1));
      final wire = jsonEncode(events);
      for (final evidence in [
        'ApiServerError',
        'StateError',
        'reconcileBilling',
        'reconciliation.request',
        requestId,
        'billing_unavailable',
        'did:plc:initiating',
      ]) {
        expect(wire, contains(evidence));
      }
      expect(wire, isNot(contains('PRIVATE_')));
      expect(wire, isNot(contains(_baseline.revenueCatAppUserId)));
    },
  );
}

final class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace>[];
  final contexts = <ReportContext>[];
  bool fail = false;

  @override
  bool get enabled => true;

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace!);
    contexts.add(context);
    if (fail) throw StateError('PRIVATE_TELEMETRY');
    return null;
  }
}
