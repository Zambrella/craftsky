import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('ended account work is cancellation even at a terminal owner', () {
    final error = AccountOperationCanceled();
    const context = ReportContext(
      feature: 'account',
      operation: 'read',
      classification: 'account.read',
      outcome: DiagnosticOutcome.terminal,
    );
    expect(isExpectedDiagnostic(error, context), isTrue);
    final mapped = AppErrorMapper.map(error);
    expect(mapped.reportable, isFalse);
    expect(mapped.diagnosticCause, same(error));
    expect(isExpectedDiagnostic(mapped, context), isTrue);
  });

  test('ordinary state invariants remain reportable', () {
    const context = ReportContext(
      feature: 'account',
      operation: 'read',
      classification: 'account.read',
    );
    expect(
      isExpectedDiagnostic(
        DiagnosticStateError('Account session unavailable'),
        context,
      ),
      isFalse,
    );
  });
}
