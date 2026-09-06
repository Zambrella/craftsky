enum VideoSubmissionStage {
  local,
  validating,
  uploading,
  processing,
  publishing,
  complete,
  failed,
  canceled,
}

final class VideoSubmissionMachine {
  VideoSubmissionStage _stage = VideoSubmissionStage.local;

  VideoSubmissionStage get stage => _stage;
  bool get canCancel =>
      _stage == VideoSubmissionStage.uploading ||
      _stage == VideoSubmissionStage.processing;

  void transitionTo(VideoSubmissionStage next) {
    final valid = switch ((_stage, next)) {
      (VideoSubmissionStage.local, VideoSubmissionStage.validating) => true,
      (VideoSubmissionStage.validating, VideoSubmissionStage.uploading) => true,
      (VideoSubmissionStage.uploading, VideoSubmissionStage.processing) => true,
      (VideoSubmissionStage.processing, VideoSubmissionStage.publishing) =>
        true,
      (VideoSubmissionStage.publishing, VideoSubmissionStage.complete) => true,
      (_, VideoSubmissionStage.failed) => true,
      _ => false,
    };
    if (!valid) throw StateError('Invalid video submission transition');
    _stage = next;
  }

  void cancel() {
    if (!canCancel) throw StateError('Video publication cannot be canceled');
    _stage = VideoSubmissionStage.canceled;
  }

  void retry() {
    if (_stage != VideoSubmissionStage.failed &&
        _stage != VideoSubmissionStage.canceled) {
      throw StateError('Video publication cannot be retried');
    }
    _stage = VideoSubmissionStage.validating;
  }
}
