import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/models/video_thumbnail_frame.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:flutter/foundation.dart' show defaultTargetPlatform;
import 'package:flutter/services.dart' show PlatformException;
import 'package:flutter_riverpod/misc.dart' show ProviderException;
import 'package:logging/logging.dart';

final class LocalVideoSelection {
  const new({
    required this.displayName,
    required this.mimeType,
    required this.byteLength,
    required this.duration,
    required this.headerBytes,
    required this.openRead,
    this.width = 1,
    this.height = 1,
    this.posterBytes,
    this.thumbnail,
    this.thumbnailFrame = VideoThumbnailFrame.midpoint,
    this.altText = '',
    this.localUri,
  });

  final String displayName;
  final String mimeType;
  final int byteLength;
  final Duration? duration;
  final int width;
  final int height;
  final Uint8List headerBytes;
  final Stream<List<int>> Function() openRead;
  final Uint8List? posterBytes;
  final Uint8List? thumbnail;
  final VideoThumbnailFrame thumbnailFrame;
  final String altText;
  final Uri? localUri;

  LocalVideoSelection withAltText(String value) =>
      _withDetails(value, thumbnail, thumbnailFrame);

  LocalVideoSelection withThumbnail(Uint8List? value) =>
      _withDetails(altText, value, thumbnailFrame);

  LocalVideoSelection withThumbnailFrame(VideoThumbnailFrame frame) =>
      _withDetails(altText, thumbnail, frame);

  LocalVideoSelection _withDetails(
    String alt,
    Uint8List? cover,
    VideoThumbnailFrame frame,
  ) => LocalVideoSelection(
    displayName: displayName,
    mimeType: mimeType,
    byteLength: byteLength,
    duration: duration,
    width: width,
    height: height,
    headerBytes: headerBytes,
    openRead: openRead,
    posterBytes: posterBytes,
    thumbnail: cover,
    thumbnailFrame: frame,
    altText: alt,
    localUri: localUri,
  );

  @override
  String toString() => 'LocalVideoSelection(<redacted>)';
}

// The interface gives platform pickers a replaceable test seam.
abstract interface class ExistingVideoPicker {
  Future<LocalVideoSelection?> pickExisting();
}

final class ComposerVideoController {
  factory({
    required ExistingVideoPicker picker,
    Future<VideoUploadLimits> Function()? checkEligibility,
    ErrorReporter reporter = const NoopErrorReporter(),
  }) => ComposerVideoController._(
    picker,
    checkEligibility,
    GuardedErrorReporter(reporter),
  );

  new _(
    this._picker,
    this._checkEligibility,
    this._reporter,
  );

  final ExistingVideoPicker _picker;
  final Future<VideoUploadLimits> Function()? _checkEligibility;
  final GuardedErrorReporter _reporter;
  LocalVideoSelection? _selection;

  LocalVideoSelection? get selection => _selection;

  Future<VideoUploadLimits?> selectExisting() async {
    var stage = 'checking_eligibility';
    try {
      final limits = await _checkEligibility?.call();
      if (limits != null && !limits.canUpload) return limits;
      stage = 'selecting_local_video';
      final selected = await _picker.pickExisting();
      if (selected == null) return limits;
      final validation = validateVideoSource(
        sizeBytes: selected.byteLength,
        fileName: selected.displayName,
        mimeType: selected.mimeType,
        headerBytes: selected.headerBytes,
        duration: selected.duration,
      );
      if (!validation.canUpload) {
        throw VideoSourceException(validation.rejectedReason!);
      }
      _selection = selected;
      return limits;
    } on Object catch (error, stack) {
      _reportSelectionFailure(error, stack, stage);
      rethrow;
    }
  }

  void _reportSelectionFailure(Object error, StackTrace stack, String stage) {
    if (error is ApiException ||
        error is ProviderException ||
        error is VideoSourceException) {
      return;
    }
    final context = ReportContext(
      feature: 'Video',
      operation: 'select',
      classification: 'video.selection.failed',
      safeDiagnostics: {
        'failureStage': stage,
        'platform': defaultTargetPlatform.name,
      },
    );
    final diagnostic = DiagnosticMessage(
      'Video selection failed',
      context: context,
    );
    final logger = Logger('ComposerVideoController');
    if (isExpectedDiagnostic(error, context) ||
        (error is PlatformException &&
            const {
              'photo_access_denied',
              'photo_access_restricted',
            }.contains(error.code))) {
      logger.warning(diagnostic, error, stack);
      return;
    }
    logger.severe(diagnostic, error, stack);
    unawaited(
      _reporter.captureException(error, context: context, stackTrace: stack),
    );
  }

  Future<VideoUploadLimits?> replace() => selectExisting();

  LocalVideoSelection? get restoredSelection => _selection;
  set restoredSelection(LocalVideoSelection value) => _selection = value;

  void remove() => _selection = null;

  void setThumbnail(Uint8List? value) =>
      _selection = _selection?.withThumbnail(value);

  void setThumbnailFrame(VideoThumbnailFrame frame) =>
      _selection = _selection?.withThumbnailFrame(frame);

  void setAltText(String value) {
    if (!RecordTextLimits.videoAlt.accepts(value)) {
      throw ArgumentError('Video alt text is too long');
    }
    _selection = _selection?.withAltText(value);
  }

  Future<VideoUploadLimits?> recheckEligibility() async =>
      await _checkEligibility?.call();
}
