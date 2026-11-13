import 'dart:async';

import 'package:craftsky_app/feed/composer/composer_ownership.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderException;
import 'package:logging/logging.dart';

typedef SubmissionStep = Future<void> Function();
typedef SubmissionRunningChanged = void Function({required bool running});
typedef SubmissionOwnershipCheck = bool Function();

enum ComposerSubmissionKind { post, project }

final _log = Logger('ComposerSubmission');

/// Content-free error passed across the UI failure boundary.
final class ComposerSubmissionFailure
    implements Exception, DiagnosticFailureCause {
  const new({this.diagnosticCause, this.diagnosticStack});

  @override
  final Object? diagnosticCause;
  @override
  final StackTrace? diagnosticStack;

  @override
  String toString() => 'ComposerSubmissionFailure';
}

/// Owns the lifecycle shared by standard and project composer submissions.
final class ComposerSubmissionCoordinator {
  new({
    required SubmissionScreenAwake screenAwake,
    this._reporter = const NoopErrorReporter(),
  }) : _lifecycle = ComposerSubmissionLifecycle(screenAwake);

  final ComposerSubmissionLifecycle _lifecycle;
  final ErrorReporter _reporter;
  bool _running = false;

  bool get isRunning => _running;

  Future<void> run({
    required SubmissionStep presentOverlay,
    required SubmissionOwnershipCheck ownershipIsCurrent,
    required SubmissionStep saveOriginSnapshot,
    required SubmissionStep operation,
    required bool Function() didSucceed,
    required SubmissionStep deleteOriginAfterSuccess,
    required SubmissionRunningChanged onRunningChanged,
    required void Function(Object error) onFailure,
    String? accountDid,
    ComposerSubmissionKind kind = ComposerSubmissionKind.post,
  }) async {
    if (_running) return;
    _running = true;
    onRunningChanged(running: true);
    try {
      await presentOverlay();
      requireComposerOwnership(ownershipIsCurrent);
      await _lifecycle.run(() async {
        await saveOriginSnapshot();
        requireComposerOwnership(ownershipIsCurrent);
        await operation();
        requireComposerOwnership(ownershipIsCurrent);
        if (didSucceed()) await deleteOriginAfterSuccess();
      });
    } on Object catch (error, stackTrace) {
      if (error is DioException && CancelToken.isCancel(error)) return;
      final failure = error is ComposerSubmissionFailure
          ? error
          : ComposerSubmissionFailure(
              diagnosticCause: error,
              diagnosticStack: stackTrace,
            );
      // A no-cause signal is already owned by the published-edit provider.
      // Dependency acquisition errors likewise belong to Riverpod.
      final cause = failure.diagnosticCause;
      if (cause != null && cause is! ProviderException) {
        unawaited(_reportFailure(failure, accountDid, kind));
      }
      onFailure(failure);
    } finally {
      _running = false;
      onRunningChanged(running: false);
    }
  }

  Future<void> dispose() => _lifecycle.dispose();

  Future<void> _reportFailure(
    ComposerSubmissionFailure failure,
    String? accountDid,
    ComposerSubmissionKind kind,
  ) async {
    final error = failure.diagnosticCause!;
    final context = ReportContext(
      feature: 'ComposerSubmission',
      operation: kind == ComposerSubmissionKind.post
          ? 'post.submit'
          : 'project.submit',
      classification: 'composer.submit.failed',
      outcome: error is VideoPublicationException && error.expectedRejection
          ? DiagnosticOutcome.expected
          : DiagnosticOutcome.automatic,
      workflow: accountDid == null
          ? null
          : PrivateOperationalFailureContext(accountDid: accountDid),
    );
    final message = DiagnosticMessage('Submission failed', context: context);
    if (isExpectedDiagnostic(error, context)) {
      _log.warning(message, error, failure.diagnosticStack);
      return;
    }
    _log.severe(message, error, failure.diagnosticStack);
    await GuardedErrorReporter(_reporter).captureException(
      error,
      stackTrace: failure.diagnosticStack,
      context: context,
    );
  }
}
