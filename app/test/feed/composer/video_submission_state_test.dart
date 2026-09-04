import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/composer/video_submission_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 cancellation is limited to upload and processing', () {
    for (final stage in VideoPublicationStage.values) {
      expect(
        canCancelVideoPublication(stage),
        stage == VideoPublicationStage.uploading ||
            stage == VideoPublicationStage.processing,
        reason: stage.name,
      );
    }
  });

  test('lifecycle interruption also cancels validation', () {
    for (final stage in VideoPublicationStage.values) {
      expect(
        shouldCancelVideoPublicationOnLifecycleInterruption(stage),
        stage == VideoPublicationStage.validating ||
            stage == VideoPublicationStage.uploading ||
            stage == VideoPublicationStage.processing,
        reason: stage.name,
      );
    }
  });

  test('UT-014 follows publication stages and locks cancellation', () {
    final machine = VideoSubmissionMachine();
    [
      VideoSubmissionStage.validating,
      VideoSubmissionStage.uploading,
      VideoSubmissionStage.processing,
    ].forEach(machine.transitionTo);
    expect(machine.canCancel, isTrue);

    machine.transitionTo(VideoSubmissionStage.publishing);
    expect(machine.canCancel, isFalse);
    machine.transitionTo(VideoSubmissionStage.complete);
    expect(machine.canCancel, isFalse);
  });

  test('UT-014 cancellation/retry preserve deterministic boundaries', () {
    final machine = VideoSubmissionMachine()
      ..transitionTo(VideoSubmissionStage.validating)
      ..transitionTo(VideoSubmissionStage.uploading)
      ..cancel();
    expect(machine.stage, VideoSubmissionStage.canceled);
    machine.retry();
    expect(machine.stage, VideoSubmissionStage.validating);
    expect(
      () => machine.transitionTo(VideoSubmissionStage.complete),
      throwsStateError,
    );
  });
}
