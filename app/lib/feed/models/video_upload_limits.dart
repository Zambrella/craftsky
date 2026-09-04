enum VideoUploadIneligibilityReason {
  emailUnverified,
  quotaExhausted,
  providerUnsupported,
  unknown,
}

final class VideoUploadLimits {
  const VideoUploadLimits({
    required this.canUpload,
    this.remainingDailyVideos,
    this.remainingDailyBytes,
    this.reason,
  });

  factory VideoUploadLimits.fromMap(Map<String, dynamic> map) =>
      VideoUploadLimits(
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
  const VideoUploadAuthorization._(this._token, this.expiresAt);

  factory VideoUploadAuthorization.fromMap(Map<String, dynamic> map) {
    final token = map['token'];
    final expiresAt = map['expiresAt'];
    if (token is! String || token.isEmpty || expiresAt is! String) {
      throw const FormatException('Invalid video authorization');
    }
    return VideoUploadAuthorization._(token, DateTime.parse(expiresAt).toUtc());
  }

  final String _token;
  final DateTime expiresAt;

  String get authorizationHeader => 'Bearer $_token';

  @override
  String toString() => 'VideoUploadAuthorization(<redacted>)';
}
