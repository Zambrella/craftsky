import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderException;
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

void main() {
  for (final error in <Exception>[
    const AccountOperationDiscarded(),
    const InstagramOperationDiscarded(),
  ]) {
    test(
      '${error.runtimeType} remains cancellation through Riverpod',
      () async {
        final container = ProviderContainer.test(retry: (_, _) => null);
        final source = Provider<void>((ref) => throw error);
        final dependent = Provider<void>((ref) => ref.watch(source));
        Object? wrapped;
        try {
          container.read(dependent);
        } on Object catch (failure) {
          wrapped = failure;
        }
        expect(wrapped, isA<ProviderException>());
        const context = ReportContext(
          feature: 'account',
          operation: 'read',
          classification: 'account.read',
          outcome: DiagnosticOutcome.terminal,
        );
        expect(isCanceledDiagnostic(wrapped!), isTrue);
        expect(isExpectedDiagnostic(wrapped, context), isTrue);
        expect(AppErrorMapper.map(error).reportable, isFalse);
        final options = SentryOptions();
        configureDiagnosticOptions(options);
        final event = SentryEvent(
          throwable: ThrowableMechanism(
            Mechanism(type: 'FlutterError'),
            wrapped,
          ),
        );
        expect(await options.beforeSend!(event, Hint()), isNull);
      },
    );
  }

  test(
    'invalid login input is expected but auth infrastructure is actionable',
    () {
      const context = ReportContext(
        feature: 'auth',
        operation: 'signIn',
        classification: 'auth.signIn',
      );
      expect(isExpectedDiagnostic(const InvalidHandle(), context), isTrue);
      expect(isExpectedDiagnostic(const HandleRequired(), context), isTrue);
      expect(isExpectedDiagnostic(const SignInTimedOut(), context), isTrue);
      expect(
        isExpectedDiagnostic(const StorageFailure('PRIVATE_STORAGE'), context),
        isFalse,
      );
      expect(
        isExpectedDiagnostic(const BrowserLaunchFailed(), context),
        isFalse,
      );
      expect(isExpectedDiagnostic(const ServerUnavailable(), context), isFalse);
    },
  );

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
