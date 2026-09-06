import 'dart:typed_data';

import 'package:mime/mime.dart';

const maxVideoSourceBytes = 300000000;
const maxVideoDuration = Duration(seconds: 600);

enum VideoSourceRejection { unsupportedType, tooLarge, tooLong }

final class VideoSourceValidationResult {
  const VideoSourceValidationResult({
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
  final detectedType = lookupMimeType(fileName, headerBytes: headerBytes);
  final contentType = lookupMimeType('', headerBytes: headerBytes);
  if (detectedType != 'video/mp4' ||
      mimeType.toLowerCase().trim() != 'video/mp4' ||
      contentType != 'video/mp4') {
    return const VideoSourceValidationResult(
      canUpload: false,
      rejectedReason: VideoSourceRejection.unsupportedType,
    );
  }
  if (sizeBytes < 0 || sizeBytes > maxVideoSourceBytes) {
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
