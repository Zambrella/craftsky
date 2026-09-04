enum VideoServiceOutcome {
  processing,
  completed,
  emailUnverified,
  quotaExhausted,
  providerUnsupported,
  validationFailed,
  processingFailed,
}

final class VideoServiceBlob {
  const VideoServiceBlob({
    required this.cid,
    required this.mimeType,
    required this.size,
  });

  factory VideoServiceBlob.fromJson(Map<String, Object?> json) {
    final ref = json['ref'];
    if (json[r'$type'] != 'blob' || ref is! Map<String, Object?>) {
      throw const FormatException('Invalid video blob');
    }
    final cid = ref[r'$link'];
    final mimeType = json['mimeType'];
    final size = json['size'];
    if (cid is! String ||
        cid.isEmpty ||
        mimeType is! String ||
        mimeType != 'video/mp4' ||
        size is! int ||
        size <= 0 ||
        size > 300000000) {
      throw const FormatException('Invalid video blob');
    }
    return VideoServiceBlob(cid: cid, mimeType: mimeType, size: size);
  }

  final String cid;
  final String mimeType;
  final int size;

  Map<String, Object> toJson() => {
    r'$type': 'blob',
    'ref': {r'$link': cid},
    'mimeType': mimeType,
    'size': size,
  };
}

final class VideoServiceResult {
  const VideoServiceResult({
    required this.outcome,
    required this.jobId,
    this.blob,
    this.progress,
    this.retryAfter,
  });

  factory VideoServiceResult.fromJson(Map<String, Object?> json) {
    final state = json['state'];
    final error = json['error'];
    final jobId = json['jobId'];
    final progress = json['progress'];
    final blob = _parseBlob(json['blob']);

    final outcome = switch ((state, error, blob)) {
      ('JOB_STATE_COMPLETED', null, final VideoServiceBlob _) =>
        VideoServiceOutcome.completed,
      (_, 'already_exists', final VideoServiceBlob _) =>
        VideoServiceOutcome.completed,
      (_, 'email_unverified' || 'unconfirmed_email', _) =>
        VideoServiceOutcome.emailUnverified,
      (_, 'quota_exhausted', _) => VideoServiceOutcome.quotaExhausted,
      (_, 'provider_unsupported', _) => VideoServiceOutcome.providerUnsupported,
      (_, 'video_too_long' || 'video_too_large' || 'invalid_video', _) =>
        VideoServiceOutcome.validationFailed,
      ('JOB_STATE_FAILED' || 'JOB_STATE_COMPLETED', _, _) =>
        VideoServiceOutcome.processingFailed,
      (_, final String _, _) => VideoServiceOutcome.processingFailed,
      _ => VideoServiceOutcome.processing,
    };

    return VideoServiceResult(
      outcome: outcome,
      jobId: jobId is String ? jobId : '',
      blob: outcome == VideoServiceOutcome.completed ? blob : null,
      progress: progress is int && progress >= 0 && progress <= 100
          ? progress
          : null,
    );
  }

  final VideoServiceOutcome outcome;
  final String jobId;
  final VideoServiceBlob? blob;
  final int? progress;
  final Duration? retryAfter;

  VideoServiceResult withRetryAfter(Duration? value) => VideoServiceResult(
    outcome: outcome,
    jobId: jobId,
    blob: blob,
    progress: progress,
    retryAfter: value,
  );

  static VideoServiceBlob? _parseBlob(Object? value) {
    if (value is! Map<String, Object?>) return null;
    try {
      return VideoServiceBlob.fromJson(value);
    } on FormatException {
      return null;
    }
  }

  @override
  String toString() =>
      'VideoServiceResult(outcome: $outcome, hasBlob: ${blob != null}, '
      'hasProgress: ${progress != null})';
}
