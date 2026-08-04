import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';

typedef SubmissionStep = Future<void> Function();
typedef SubmissionRunningChanged = void Function({required bool running});
typedef SubmissionOwnershipCheck = bool Function();

/// Content-free error passed across the UI failure boundary.
final class ComposerSubmissionFailure implements Exception {
  const ComposerSubmissionFailure();

  @override
  String toString() => 'ComposerSubmissionFailure';
}

/// Owns the lifecycle shared by standard and project composer submissions.
final class ComposerSubmissionCoordinator {
  ComposerSubmissionCoordinator({required SubmissionScreenAwake screenAwake})
    : _lifecycle = ComposerSubmissionLifecycle(screenAwake);

  final ComposerSubmissionLifecycle _lifecycle;
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
  }) async {
    if (_running) return;
    _running = true;
    onRunningChanged(running: true);
    try {
      await presentOverlay();
      _requireCurrentOwnership(ownershipIsCurrent);
      await _lifecycle.run(() async {
        await saveOriginSnapshot();
        _requireCurrentOwnership(ownershipIsCurrent);
        await operation();
        _requireCurrentOwnership(ownershipIsCurrent);
        if (didSucceed()) await deleteOriginAfterSuccess();
      });
    } on Object {
      onFailure(const ComposerSubmissionFailure());
    } finally {
      _running = false;
      onRunningChanged(running: false);
    }
  }

  Future<void> dispose() => _lifecycle.dispose();
}

void _requireCurrentOwnership(SubmissionOwnershipCheck ownershipIsCurrent) {
  if (!ownershipIsCurrent()) {
    throw StateError('submission ownership changed');
  }
}
