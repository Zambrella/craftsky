import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';

String videoPublicationFailureMessage(
  AppLocalizations l10n,
  Object? failure,
) => switch (failure) {
  VideoPublicationException(:final outcome, :final ineligibilityReason) =>
    _failureMessage(l10n, outcome, ineligibilityReason),
  _ => l10n.postVideoRetryableFailure,
};

String videoLimitsMessage(AppLocalizations l10n, VideoUploadLimits limits) => [
  _failureMessage(l10n, null, limits.reason),
  ?videoQuotaMessage(l10n, limits),
].join(' ');

String? videoQuotaMessage(
  AppLocalizations l10n,
  VideoUploadLimits limits,
) {
  if (!limits.shouldShowQuota) return null;
  return switch ((limits.remainingDailyVideos, limits.remainingDailyBytes)) {
    (final videos?, final bytes?) => l10n.postVideoRemainingQuota(
      videos,
      bytes,
    ),
    (final videos?, null) => l10n.postVideoRemainingVideos(videos),
    (null, final bytes?) => l10n.postVideoRemainingBytes(bytes),
    (null, null) => null,
  };
}

String _failureMessage(
  AppLocalizations l10n,
  VideoServiceOutcome? outcome,
  VideoUploadIneligibilityReason? reason,
) => switch ((outcome, reason)) {
  (_, VideoUploadIneligibilityReason.emailUnverified) ||
  (VideoServiceOutcome.emailUnverified, _) => l10n.postVideoEmailUnverified,
  (_, VideoUploadIneligibilityReason.quotaExhausted) ||
  (VideoServiceOutcome.quotaExhausted, _) => l10n.postVideoQuotaExhausted,
  (_, VideoUploadIneligibilityReason.providerUnsupported) ||
  (
    VideoServiceOutcome.providerUnsupported,
    _,
  ) => l10n.postVideoProviderUnsupported,
  (VideoServiceOutcome.validationFailed, _) => l10n.postVideoValidationFailed,
  (VideoServiceOutcome.processingFailed, _) => l10n.postVideoProcessingFailed,
  _ => l10n.postVideoRetryableFailure,
};
