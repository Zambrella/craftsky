import 'dart:typed_data';

import 'package:mime/mime.dart';

const maxVideoSourceBytes = 300000000;
const maxVideoDuration = Duration(seconds: 600);
const supportedVideoSourceMimeTypes = {'video/mp4', 'video/quicktime'};

// mime's built-in signatures recognise MP4 but not the QuickTime ftyp brand.
// Match the container, independently of its filename and video codec.
final _videoMimeTypes = MimeTypeResolver()
  ..addMagicNumber(
    [0, 0, 0, 0, ...'ftypqt  '.codeUnits],
    'video/quicktime',
    mask: [0, 0, 0, 0, ...List<int>.filled(8, 255)],
  );

String? detectVideoSourceMimeType(List<int> headerBytes) {
  final type = _videoMimeTypes.lookup('', headerBytes: headerBytes);
  return supportedVideoSourceMimeTypes.contains(type) ? type : null;
}

enum VideoSourceRejection { unsupportedType, tooLarge, tooLong }

final class VideoSourceException implements Exception {
  const new(this.reason);
  final VideoSourceRejection reason;
  @override
  String toString() => 'VideoSourceException(${reason.name})';
}

final class VideoSourceValidationResult {
  const new({
    required this.canUpload,
    required this.rejectedReason,
  });

  final bool canUpload;
  final VideoSourceRejection? rejectedReason;
}

VideoSourceValidationResult validateVideoSource({
  required int sizeBytes,
  required String fileName,
  required String mimeType,
  required Uint8List headerBytes,
  required Duration? duration,
}) {
  final detectedType = _videoMimeTypes.lookup(
    fileName,
    headerBytes: headerBytes,
  );
  final contentType = detectVideoSourceMimeType(headerBytes);
  if (contentType == null ||
      detectedType != contentType ||
      mimeType.toLowerCase().trim() != contentType) {
    return const VideoSourceValidationResult(
      canUpload: false,
      rejectedReason: VideoSourceRejection.unsupportedType,
    );
  }
  if (sizeBytes <= 0 || sizeBytes > maxVideoSourceBytes) {
    return const VideoSourceValidationResult(
      canUpload: false,
      rejectedReason: VideoSourceRejection.tooLarge,
    );
  }
  if (duration != null && duration > maxVideoDuration) {
    return const VideoSourceValidationResult(
      canUpload: false,
      rejectedReason: VideoSourceRejection.tooLong,
    );
  }
  return const VideoSourceValidationResult(
    canUpload: true,
    rejectedReason: null,
  );
}
