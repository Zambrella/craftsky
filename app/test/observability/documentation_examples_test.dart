import 'dart:io';

import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test(
    'IT-014 contributor examples retain selected causes '
    'and exclude private input',
    () async {
      final guide = File(
        '../docs/development/logging-error-reporting.md',
      ).readAsStringSync();
      expect(guide, contains('PrivateOperationalFailureContext'));
      expect(guide, contains('beforeSend'));
      final records = <String>[];
      final subscription = configureRootLogForwarding(
        platformSink: records.add,
      );
      addTearDown(subscription.cancel);
      Logger.root.level = Level.ALL;
      final log = Logger('Post');
      const reporter = NoopErrorReporter();
      const error = FormatException('PRIVATE_BODY');
      final stack = StackTrace.fromString(
        '#0 loadPost (package:craftsky_app/post.dart:12:3)',
      );
      const context = ReportContext(
        feature: 'Post',
        operation: 'read',
        classification: 'post.read',
        workflow: PublicRecordContext(
          actorDid: 'did:plc:actor',
          targetDid: 'did:plc:target',
        ),
        safeDiagnostics: {
          'appViewRequestId': '40000000-0000-4000-8000-000000000001',
        },
      );
      log.severe(
        const DiagnosticMessage(
          'Operation failed',
          context: context,
        ),
        error,
        stack,
      );
      await reporter.captureException(
        error,
        stackTrace: stack,
        context: context,
      );
      const privateContext = ReportContext(
        feature: 'Schedule',
        operation: 'publish',
        classification: 'schedule.failed',
        outcome: DiagnosticOutcome.retry,
        workflow: PrivateOperationalFailureContext(
          accountDid: 'did:plc:actor',
          workflowRef: '40000000-0000-4000-8000-000000000002',
        ),
      );
      log
        ..warning(
          const DiagnosticMessage('Operation failed', context: privateContext),
          error,
          stack,
        )
        ..severe('Post load failed token=PRIVATE_TOKEN', error, stack);
      await Future<void>.delayed(Duration.zero);
      expect(records, hasLength(3));
      for (final field in [
        'FormatException',
        'loadPost',
        'did:plc:target',
        'operationAccountDid',
        '40000000-0000-4000-8000-000000000001',
      ]) {
        expect(records.join(), contains(field));
      }
      expect(records.join(), isNot(contains('PRIVATE_')));
    },
  );
}
