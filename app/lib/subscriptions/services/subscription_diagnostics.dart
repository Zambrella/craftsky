import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/observability/consumed_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_setup_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';

/// A consuming boundary captures this before awaiting an account operation.
/// Payment, license, customer and recipient identifiers stay private.
final class SubscriptionDiagnostics {
  const new({this.reporter = const NoopErrorReporter(), this.accountDid});

  final ErrorReporter reporter;
  final String? accountDid;

  void report(Object error, StackTrace stack, {required String operation}) {
    final expected =
        (error is BillingAccountGuardException && error.ownerSessionChanged) ||
        (error is BillingAccountSetupException &&
            error.code == 'account_session_changed') ||
        error is RevenueCatUnavailableException ||
        error is StoreAccountConflictException ||
        error is StoreSubscriptionExistsException ||
        error is NativePlanChangeUnavailableException ||
        (error is RevenueCatIdentityException &&
            (error.code == 'billing_unavailable' ||
                error.code == 'account_session_changed')) ||
        (error is RevenueCatServiceException && error.expectedFailure);
    reportConsumedFailure(
      error,
      stack,
      reporter: reporter,
      feature: 'Subscriptions',
      operation: operation,
      accountDid: accountDid,
      fallbackKind: AppErrorKind.unexpected,
      outcome: expected
          ? DiagnosticOutcome.expected
          : DiagnosticOutcome.automatic,
    );
  }
}
