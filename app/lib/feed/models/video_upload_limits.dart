enum VideoUploadIneligibilityReason {
  emailUnverified,
  quotaExhausted,
  providerUnsupported,
  unknown,
}

final class VideoUploadLimits {
  const new({
    required this.canUpload,
    this.remainingDailyVideos,
    this.remainingDailyBytes,
    this.reason,
  });

  factory fromMap(Map<String, dynamic> map) => VideoUploadLimits(
    canUpload: map['canUpload'] as bool,
    remainingDailyVideos: map['remainingDailyVideos'] as int?,
    remainingDailyBytes: map['remainingDailyBytes'] as int?,
    reason: switch (map['reason']) {
      'email_unverified' => VideoUploadIneligibilityReason.emailUnverified,
      'quota_exhausted' => VideoUploadIneligibilityReason.quotaExhausted,
      'provider_unsupported' =>
        VideoUploadIneligibilityReason.providerUnsupported,
      'unknown' => VideoUploadIneligibilityReason.unknown,
      _ => null,
    },
  );

  final bool canUpload;
  final int? remainingDailyVideos;
  final int? remainingDailyBytes;
  final VideoUploadIneligibilityReason? reason;

  bool get shouldShowQuota =>
      !canUpload ||
      (remainingDailyVideos != null && remainingDailyVideos! <= 1) ||
      (remainingDailyBytes != null && remainingDailyBytes! < 300000000);
}

final class VideoUploadAuthorization {
  const new _(this._token, this.expiresAt, this.bunny);

  factory fromMap(Map<String, dynamic> map) {
    final token = map['token'];
    final expiresAt = map['expiresAt'];
    if (token is! String || token.isEmpty || expiresAt is! String) {
      throw const FormatException('Invalid video authorization');
    }
    return VideoUploadAuthorization._(
      token,
      DateTime.parse(expiresAt).toUtc(),
      map['provider'] == 'bunny' ? BunnyVideoUpload.fromMap(map) : null,
    );
  }

  final BunnyVideoUpload? bunny;
  final String _token;
  final DateTime expiresAt;

  String get authorizationHeader => 'Bearer $_token';

  @override
  String toString() => 'VideoUploadAuthorization(<redacted>)';
}

/// One upload capability; kept only in the active publication attempt.
final class BunnyVideoUpload {
  const new({
    required this.jobId,
    required this.videoId,
    required this.libraryId,
    required this.title,
    required this.collectionId,
    required this.signature,
    required this.expiresAt,
  });
  factory fromMap(Map<String, dynamic> map) {
    final job = map['jobId'];
    final video = map['videoId'];
    final library = map['libraryId'];
    final signature = map['token'];
    final title = map['title'];
    final collection = map['collectionId'];
    final uuid = RegExp(
      r'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$',
    );
    if (title is! String ||
        title.isEmpty ||
        title.length > 200 ||
        collection is! String ||
        !uuid.hasMatch(collection) ||
        job is! String ||
        !uuid.hasMatch(job) ||
        video is! String ||
        !uuid.hasMatch(video) ||
        library is! String ||
        !RegExp(r'^[1-9][0-9]*$').hasMatch(library) ||
        signature is! String ||
        !RegExp(r'^[0-9a-f]{64}$').hasMatch(signature)) {
      throw const FormatException('Invalid video upload authorization');
    }
    return BunnyVideoUpload(
      jobId: job,
      videoId: video,
      libraryId: library,
      title: title,
      collectionId: collection,
      signature: signature,
      expiresAt: DateTime.parse(map['expiresAt'] as String).toUtc(),
    );
  }
  final String jobId;
  final String videoId;
  final String libraryId;
  final String title;
  final String collectionId;
  final String signature;
  final DateTime expiresAt;
  @override
  String toString() => 'BunnyVideoUpload(<redacted>)';
}
