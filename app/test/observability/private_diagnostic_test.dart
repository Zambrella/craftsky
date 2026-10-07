import 'dart:convert';

import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test('AT-006 private failure owner requires a bounded DID value', () {
    for (final value in [
      'opaque private account',
      'PRIVATE_ACCOUNT@example.invalid',
      'did:plc:${'x' * 2048}',
    ]) {
      final fields = PrivateOperationalFailureContext(
        accountDid: value,
      ).selectedFields;
      expect(fields.containsKey('operationAccountDid'), isFalse);
      expect(jsonEncode(fields), isNot(contains(value)));
    }
    expect(
      const PrivateOperationalFailureContext(
        accountDid: 'did:plc:owner',
      ).selectedFields['operationAccountDid'],
      'did:plc:owner',
    );
  });
  test(
    'AT-006 private failure retains initiating account and '
    'workflow but no success history',
    () {
      final error = StateError('opaque private schedule canary');
      const workflow = PrivateOperationalFailureContext(
        accountDid: 'did:plc:alice',
        workflowRef: '00000000-0000-4000-8000-000000000401',
      );
      ReportContext context(Object? cause) => ReportContext(
        feature: 'Schedule',
        operation: 'schedule.create',
        classification: 'schedule.failed',
        workflow: workflow,
        cause: cause,
        safeDiagnostics: const {'failureStage': 'storage_write'},
      );
      final failed = selectDiagnosticRecord(
        LogRecord(
          Level.SEVERE,
          'Operation failed',
          'Schedule',
          error,
          null,
          null,
          DiagnosticMessage('Operation failed', context: context(error)),
        ),
      );
      final exported = SentryErrorReporter.attributesFor(context(error));
      for (final output in [
        jsonEncode(failed),
        exported
            .map((key, attribute) => MapEntry(key, attribute.value))
            .toString(),
      ]) {
        expect(output, contains('did:plc:alice'));
        expect(output, contains('00000000-0000-4000-8000-000000000401'));
        expect(output, isNot(contains('opaque private schedule')));
      }
      final success = selectDiagnosticRecord(
        LogRecord(
          Level.INFO,
          'Operation failed',
          'Schedule',
          null,
          null,
          null,
          DiagnosticMessage('Operation failed', context: context(null)),
        ),
      );
      expect(jsonEncode(success), isNot(contains('did:plc:alice')));
      expect(
        SentryErrorReporter.attributesFor(context(null)).toString(),
        isNot(contains('did:plc:alice')),
      );
    },
  );
}
