import 'package:craftsky_app/feed/composer/video_job_poller.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/observability/video_diagnostics.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:logging/logging.dart';

final _log = Logger('VideoPublication');

enum VideoPublicationStage {
  validating,
  uploading,
  processing,
  publishing,
  complete,
  canceled,
  failed,
}

bool canCancelVideoPublication(VideoPublicationStage? stage) =>
    stage == VideoPublicationStage.uploading ||
    stage == VideoPublicationStage.processing;

bool shouldCancelVideoPublicationOnLifecycleInterruption(
  VideoPublicationStage? stage,
) =>
    stage == VideoPublicationStage.validating ||
    stage == VideoPublicationStage.uploading ||
    stage == VideoPublicationStage.processing;

final class VideoPublicationProgress {
  const VideoPublicationProgress(this.stage, {this.fraction});

  final VideoPublicationStage stage;
  final double? fraction;
}

final class VideoPublicationException implements Exception {
  const VideoPublicationException(this.outcome, {this.ineligibilityReason});

  final VideoServiceOutcome? outcome;
  final VideoUploadIneligibilityReason? ineligibilityReason;
}

typedef VideoUploadOperation =
    Future<VideoServiceResult> Function({
      required String authorizationHeader,
      required CancelToken cancelToken,
      required bool bypassDeduplication,
      required void Function(int sent, int total) onProgress,
    });

typedef VideoPublishOperation =
    Future<void> Function(
      CreatePostVideo proof, {
      required bool allowBlobRecovery,
    });

final class VideoPublicationCoordinator {
  factory VideoPublicationCoordinator({
    required Future<VideoUploadLimits> Function() checkEligibility,
    required Future<VideoUploadAuthorization> Function() authorize,
    required VideoUploadOperation upload,
    required Future<VideoServiceResult> Function(
      String jobId,
      CancelToken cancelToken,
    )
    poll,
    required Future<void> Function(Duration duration) wait,
    required VideoPublishOperation publish,
    required void Function(VideoPublicationProgress progress) onProgress,
    DateTime Function()? clock,
    String? operationAccountDid,
  }) => VideoPublicationCoordinator._(
    checkEligibility,
    authorize,
    upload,
    poll,
    wait,
    publish,
    onProgress,
    clock ?? DateTime.now,
    operationAccountDid,
  );

  VideoPublicationCoordinator._(
    this._checkEligibility,
    this._authorize,
    this._upload,
    this._poll,
    this._wait,
    this._publish,
    this._onProgress,
    this._clock,
    this._operationAccountDid,
  );

  final Future<VideoUploadLimits> Function() _checkEligibility;
  final Future<VideoUploadAuthorization> Function() _authorize;
  final VideoUploadOperation _upload;
  final Future<VideoServiceResult> Function(String, CancelToken) _poll;
  final Future<void> Function(Duration) _wait;
  final VideoPublishOperation _publish;
  final void Function(VideoPublicationProgress) _onProgress;
  final DateTime Function() _clock;
  final String? _operationAccountDid;

  VideoUploadAuthorization? _authorization;
  String? _jobId;
  CancelToken? _cancelToken;
  VideoOperation _operation = VideoOperation.limits;

  bool get hasEphemeralState => _authorization != null || _jobId != null;

  Future<void> publish({
    required String altText,
    required (int, int)? aspectRatio,
  }) async {
    if (_cancelToken != null) {
      throw DiagnosticStateError('Video publication is running');
    }
    final cancelToken = CancelToken();
    _cancelToken = cancelToken;
    _emit(VideoPublicationStage.validating);
    try {
      _operation = VideoOperation.limits;
      final limits = await _checkEligibility();
      _diagnose(VideoOperationOutcome.succeeded);
      _throwIfCanceled(cancelToken);
      if (!limits.canUpload) {
        _diagnose(VideoOperationOutcome.rejected);
        throw VideoPublicationException(
          null,
          ineligibilityReason: limits.reason,
        );
      }
      var bypassDeduplication = false;
      while (true) {
        _operation = VideoOperation.authorization;
        _authorization = await _authorize();
        _diagnose(VideoOperationOutcome.succeeded);
        _throwIfCanceled(cancelToken);
        if (!_authorization!.expiresAt.isAfter(_clock().toUtc())) {
          _diagnose(VideoOperationOutcome.rejected);
          throw const VideoPublicationException(null);
        }
        _emit(VideoPublicationStage.uploading);
        _operation = VideoOperation.upload;
        var result = await _uploadAuthorized(
          cancelToken,
          bypassDeduplication: bypassDeduplication,
        );
        _diagnose(VideoOperationOutcome.succeeded);
        _jobId = result.jobId;
        final startedAt = _clock().toUtc();
        var completedPolls = 0;
        while (result.outcome == VideoServiceOutcome.processing) {
          _operation = VideoOperation.polling;
          _emit(
            VideoPublicationStage.processing,
            fraction: result.progress == null ? null : result.progress! / 100,
          );
          if (isVideoJobExpired(startedAt: startedAt, now: _clock().toUtc())) {
            throw const VideoPublicationException(null);
          }
          await Future.any<void>([
            _wait(
              videoPollingDelay(
                completedPolls,
                retryAfter: result.retryAfter,
              ),
            ),
            cancelToken.whenCancel.then<void>((error) => throw error),
          ]);
          if (cancelToken.cancelError case final error?) throw error;
          try {
            result = await _poll(_jobId!, cancelToken);
            _diagnose(VideoOperationOutcome.succeeded);
          } on VideoTransportException catch (error, stackTrace) {
            if (error.kind != VideoTransportFailure.unavailable) rethrow;
            _diagnose(
              VideoOperationOutcome.retrying,
              error: error,
              stackTrace: stackTrace,
            );
          }
          completedPolls++;
        }
        final blob = result.blob;
        if (result.outcome != VideoServiceOutcome.completed || blob == null) {
          throw VideoPublicationException(result.outcome);
        }
        _emit(VideoPublicationStage.publishing);
        _operation = VideoOperation.publication;
        try {
          await _publish(
            CreatePostVideo(
              jobId: _jobId!,
              blob: CreatePostVideoBlob(
                cid: blob.cid,
                mimeType: blob.mimeType,
                size: blob.size,
              ),
              alt: altText.trim().isEmpty ? null : altText,
              aspectRatio: aspectRatio == null
                  ? null
                  : CreatePostVideoAspectRatio(
                      width: aspectRatio.$1,
                      height: aspectRatio.$2,
                    ),
            ),
            allowBlobRecovery: !bypassDeduplication,
          );
          _diagnose(VideoOperationOutcome.succeeded);
          break;
        } on Object catch (error, stackTrace) {
          if (bypassDeduplication || !_isMissingVideoBlob(error)) rethrow;
          _diagnose(
            VideoOperationOutcome.retrying,
            error: error,
            stackTrace: stackTrace,
          );
          bypassDeduplication = true;
          _jobId = null;
        }
      }
      _emit(VideoPublicationStage.complete);
    } on DioException catch (error, stackTrace) {
      final canceled = CancelToken.isCancel(error);
      _diagnose(
        canceled
            ? VideoOperationOutcome.canceled
            : VideoOperationOutcome.failed,
        error: error,
        stackTrace: stackTrace,
      );
      _emit(
        canceled
            ? VideoPublicationStage.canceled
            : VideoPublicationStage.failed,
      );
      rethrow;
    } on Object catch (error, stackTrace) {
      _diagnose(
        cancelToken.isCancelled
            ? VideoOperationOutcome.canceled
            : VideoOperationOutcome.failed,
        error: error,
        stackTrace: stackTrace,
      );
      _emit(
        cancelToken.isCancelled
            ? VideoPublicationStage.canceled
            : VideoPublicationStage.failed,
      );
      rethrow;
    } finally {
      _authorization = null;
      _jobId = null;
      _cancelToken = null;
    }
  }

  void cancel() => _cancelToken?.cancel('Video publication canceled');

  void _throwIfCanceled(CancelToken cancelToken) {
    if (cancelToken.cancelError case final error?) throw error;
  }

  Future<VideoServiceResult> _uploadAuthorized(
    CancelToken cancelToken, {
    required bool bypassDeduplication,
  }) async {
    try {
      return await _upload(
        authorizationHeader: _authorization!.authorizationHeader,
        cancelToken: cancelToken,
        bypassDeduplication: bypassDeduplication,
        onProgress: (sent, total) => _emit(
          VideoPublicationStage.uploading,
          fraction: total > 0 ? (sent / total).clamp(0, 1) : null,
        ),
      );
    } finally {
      _authorization = null;
    }
  }

  void _emit(VideoPublicationStage stage, {double? fraction}) =>
      _onProgress(VideoPublicationProgress(stage, fraction: fraction));

  void _diagnose(
    VideoOperationOutcome outcome, {
    Object? error,
    StackTrace? stackTrace,
  }) {
    final expected =
        outcome == VideoOperationOutcome.canceled ||
        outcome == VideoOperationOutcome.rejected ||
        (error is VideoPublicationException &&
            (error.ineligibilityReason != null ||
                const {
                  VideoServiceOutcome.emailUnverified,
                  VideoServiceOutcome.quotaExhausted,
                  VideoServiceOutcome.providerUnsupported,
                  VideoServiceOutcome.validationFailed,
                }.contains(error.outcome)));
    final appError = error == null
        ? null
        : AppErrorMapper.map(
            error,
            fallbackKind: AppErrorKind.actionFailed,
            source: 'video',
            fallbackClassification: 'video.failed',
          );
    final context = ReportContext(
      feature: 'VideoPublication',
      operation: 'video.${_operation.name}',
      classification: appError?.sentryClassification ?? 'video.${outcome.name}',
      outcome: expected
          ? DiagnosticOutcome.expected
          : outcome == VideoOperationOutcome.retrying
          ? DiagnosticOutcome.retry
          : DiagnosticOutcome.automatic,
      workflow: error != null && !expected && _operationAccountDid != null
          ? PublicRecordContext(actorDid: _operationAccountDid)
          : null,
      safeDiagnostics: {
        ...?(appError?.safeDiagnostics),
        'failureStage': _operation.name,
        'outcome': outcome.name,
        'retryable': outcome == VideoOperationOutcome.retrying,
        if (error is VideoTransportException) 'providerCode': error.kind.name,
        if (error is VideoTransportException &&
            error.httpStatus != null &&
            error.httpStatus! >= 100 &&
            error.httpStatus! <= 599)
          'providerStatus': error.httpStatus,
        if (error is VideoPublicationException && error.outcome != null)
          'providerCode': error.outcome!.name,
      },
    );
    final message = DiagnosticMessage(
      'video operation outcome',
      context: context,
    );
    switch (outcome) {
      case VideoOperationOutcome.failed:
        _log.severe(message, error, stackTrace);
      case VideoOperationOutcome.rejected ||
          VideoOperationOutcome.canceled ||
          VideoOperationOutcome.retrying:
        _log.warning(message, error, stackTrace);
      case VideoOperationOutcome.succeeded:
        _log.fine(message);
    }
  }

  @override
  String toString() => 'VideoPublicationCoordinator(<redacted>)';
}

bool _isMissingVideoBlob(Object error) =>
    error is ApiException && error.details.appViewError == 'video_blob_missing';
