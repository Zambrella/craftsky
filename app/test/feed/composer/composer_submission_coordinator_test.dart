import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/composer/composer_ownership.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final error in <Object>[
    StateError('PRIVATE_COMPOSER_INPUT'),
    const ApiNetworkError('offline'),
    const ApiCanceled(),
    const AccountOperationDiscarded(),
    const ComposerSubmissionFailure(),
    ComposerOwnershipChanged(),
    const VideoPublicationException(
      null,
      ineligibilityReason: VideoUploadIneligibilityReason.emailUnverified,
    ),
    const VideoPublicationException(VideoServiceOutcome.emailUnverified),
    const VideoPublicationException(VideoServiceOutcome.quotaExhausted),
    const VideoPublicationException(VideoServiceOutcome.providerUnsupported),
    const VideoPublicationException(VideoServiceOutcome.validationFailed),
    const VideoPublicationException(VideoServiceOutcome.providerSizeLimit),
    const VideoPublicationException(null),
  ]) {
    test('consumed submission $error has one diagnostic owner', () async {
      final reporter = _Reporter();
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake([]),
        reporter: reporter,
      );
      final stack = StackTrace.fromString(
        '#0 submit (package:craftsky_app/composer/submit.dart:12:3)',
      );
      var failed = false;
      await coordinator.run(
        accountDid: 'did:plc:original',
        kind: ComposerSubmissionKind.project,
        presentOverlay: () async {},
        ownershipIsCurrent: () => true,
        saveOriginSnapshot: () async => Error.throwWithStackTrace(error, stack),
        operation: () async {},
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async {},
        onRunningChanged: ({required running}) {},
        onFailure: (_) => failed = true,
      );
      expect(failed, isTrue);
      expect(coordinator.isRunning, isFalse);
      if ((error is StateError && error is! ComposerOwnershipChanged) ||
          (error is VideoPublicationException && !error.expectedRejection)) {
        expect(reporter.errors, [same(error)]);
        expect(reporter.stacks, [same(stack)]);
        final context = reporter.contexts.single;
        expect(context.operation, 'project.submit');
        expect(context.workflow!.selectedFields, {
          'operationAccountDid': 'did:plc:original',
        });
      } else {
        expect(reporter.errors, isEmpty);
      }
    });
  }

  test(
    'submission failure retains the original cause and caught stack',
    () async {
      final original = StateError('PRIVATE_COMPOSER_CONTENT');
      final stack = StackTrace.fromString(
        '#0 snapshot (package:craftsky_app/drafts/snapshot.dart:12:3)',
      );
      Object? failure;
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake([]),
      );
      await coordinator.run(
        presentOverlay: () async {},
        ownershipIsCurrent: () => true,
        saveOriginSnapshot: () async =>
            Error.throwWithStackTrace(original, stack),
        operation: () async {},
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async {},
        onRunningChanged: ({required running}) {},
        onFailure: (error) => failure = error,
      );
      expect(failure, isA<DiagnosticFailureCause>());
      final diagnostic = failure! as DiagnosticFailureCause;
      expect(diagnostic.diagnosticCause, same(original));
      expect(diagnostic.diagnosticStack, same(stack));
      expect(failure.toString(), isNot(contains('PRIVATE_COMPOSER_CONTENT')));
    },
  );

  test('canceling video submission silently keeps the snapshot '
      'and releases the overlay', () async {
    final events = <String>[];
    final coordinator = ComposerSubmissionCoordinator(
      screenAwake: _RecordingScreenAwake(events),
    );
    final token = CancelToken()..cancel();
    await coordinator.run(
      presentOverlay: () async => events.add('present'),
      ownershipIsCurrent: () => true,
      saveOriginSnapshot: () async => events.add('snapshot'),
      operation: () async => throw token.cancelError!,
      didSucceed: () => false,
      deleteOriginAfterSuccess: () async => events.add('delete'),
      onRunningChanged: ({required running}) => events.add('running:$running'),
      onFailure: (_) => events.add('failure'),
    );
    expect(events, [
      'running:true',
      'present',
      'enable',
      'snapshot',
      'disable',
      'running:false',
    ]);
    expect(coordinator.isRunning, isFalse);
  });

  test(
    'orders presentation, snapshot, operation, cleanup, and release',
    () async {
      final events = <String>[];
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake(events),
      );

      await coordinator.run(
        presentOverlay: () async => events.add('present'),
        ownershipIsCurrent: () => true,
        saveOriginSnapshot: () async => events.add('snapshot'),
        operation: () async => events.add('operation'),
        didSucceed: () => true,
        deleteOriginAfterSuccess: () async => events.add('delete'),
        onRunningChanged: ({required running}) =>
            events.add('running:$running'),
        onFailure: (_) => events.add('failure'),
      );

      expect(events, [
        'running:true',
        'present',
        'enable',
        'snapshot',
        'operation',
        'delete',
        'disable',
        'running:false',
      ]);
    },
  );

  test(
    'failure skips cleanup, releases ownership, and stays retryable',
    () async {
      final events = <String>[];
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake(events),
      );

      await coordinator.run(
        presentOverlay: () async => events.add('present'),
        ownershipIsCurrent: () => true,
        saveOriginSnapshot: () async => events.add('snapshot'),
        operation: () async => throw StateError('safe fake failure'),
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async => events.add('delete'),
        onRunningChanged: ({required running}) =>
            events.add('running:$running'),
        onFailure: (_) => events.add('failure'),
      );

      expect(events, [
        'running:true',
        'present',
        'enable',
        'snapshot',
        'disable',
        'failure',
        'running:false',
      ]);
      expect(coordinator.isRunning, isFalse);
    },
  );

  test(
    'stops before submission work when captured ownership is stale',
    () async {
      final events = <String>[];
      var ownershipIsCurrent = true;
      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: _RecordingScreenAwake(events),
      );

      await coordinator.run(
        presentOverlay: () async {
          events.add('present');
          ownershipIsCurrent = false;
        },
        ownershipIsCurrent: () => ownershipIsCurrent,
        saveOriginSnapshot: () async => events.add('snapshot'),
        operation: () async => events.add('operation'),
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async => events.add('delete'),
        onRunningChanged: ({required running}) =>
            events.add('running:$running'),
        onFailure: (_) => events.add('failure'),
      );

      expect(events, ['running:true', 'present', 'failure', 'running:false']);
    },
  );
}

final class _RecordingScreenAwake implements SubmissionScreenAwake {
  new(this.events);

  final List<String> events;

  @override
  Future<void> enable() async => events.add('enable');

  @override
  Future<void> disable() async => events.add('disable');
}

final class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
    return null;
  }
}
