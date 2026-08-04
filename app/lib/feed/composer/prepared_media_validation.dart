import 'dart:typed_data';

import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:crypto/crypto.dart';

String verifyPreparedMediaBytes({
  required Uint8List bytes,
  required String mimeType,
  required int width,
  required int height,
  required String altText,
  required ComposerImageMediaService mediaService,
  String? expectedSha256,
}) {
  final normalizedMime = mimeType.trim().toLowerCase();
  final sizeValidation = mediaService.validatePreparedUploadBytes(
    originalBytes: bytes.length,
    preparedBytes: bytes.length,
  );
  if (!sizeValidation.canUpload ||
      bytes.isEmpty ||
      (normalizedMime != 'image/jpeg' && normalizedMime != 'image/png') ||
      width <= 0 ||
      height <= 0 ||
      altText.trim().length > mediaService.config.maxAltTextCharacters) {
    throw StateError('prepared image is no longer valid');
  }
  final digest = sha256.convert(bytes).toString();
  if (expectedSha256 != null && digest != expectedSha256) {
    throw StateError('prepared image bytes changed');
  }
  return digest;
}
