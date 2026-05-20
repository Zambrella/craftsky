import 'dart:typed_data';

import 'package:image/image.dart' as img;

enum SupportedImageFormat { jpeg, png, webp }

class ImagePreparationInput {
  const ImagePreparationInput({
    required this.format,
    required this.metadata,
    this.hasTransparency = false,
  });

  final SupportedImageFormat format;
  final Map<String, String> metadata;
  final bool hasTransparency;
}

class PreparedImage {
  const PreparedImage({
    required this.format,
    required this.metadata,
    required this.hasTransparency,
  });

  final SupportedImageFormat format;
  final Map<String, String> metadata;
  final bool hasTransparency;
}

class PreparedUploadImage {
  const PreparedUploadImage({
    required this.bytes,
    required this.mimeType,
    required this.width,
    required this.height,
    required this.format,
    required this.metadata,
    required this.hasTransparency,
  });

  final Uint8List bytes;
  final String mimeType;
  final int width;
  final int height;
  final SupportedImageFormat format;
  final Map<String, String> metadata;
  final bool hasTransparency;
}

const _metadataKeysToStrip = <String>{
  'gpsLatitude',
  'gpsLongitude',
  'cameraMake',
  'cameraModel',
  'captureTimestamp',
  'lens',
  'software',
  'userComment',
  'comment',
};

PreparedImage stripNonEssentialMetadata(ImagePreparationInput input) {
  final retained = <String, String>{
    for (final entry in input.metadata.entries)
      if (!_metadataKeysToStrip.contains(entry.key)) entry.key: entry.value,
  };

  return PreparedImage(
    format: input.format,
    metadata: retained,
    hasTransparency: input.format == SupportedImageFormat.png
        ? input.hasTransparency
        : false,
  );
}

PreparedUploadImage prepareImageForUpload({
  required Uint8List originalBytes,
  required String fileName,
  required String mimeType,
  Map<String, String> metadata = const {},
}) {
  final format = _resolveSupportedImageFormat(
    fileName: fileName,
    mimeType: mimeType,
  );

  final decoded = img.decodeImage(originalBytes);
  if (decoded == null) {
    throw const FormatException('Unsupported or corrupt image bytes');
  }

  final stripped = stripNonEssentialMetadata(
    ImagePreparationInput(
      format: format,
      metadata: metadata,
      hasTransparency: decoded.hasAlpha,
    ),
  );

  final preparedBytes = switch (format) {
    SupportedImageFormat.jpeg => Uint8List.fromList(img.encodeJpg(decoded)),
    SupportedImageFormat.png => Uint8List.fromList(img.encodePng(decoded)),
    SupportedImageFormat.webp => originalBytes,
  };

  return PreparedUploadImage(
    bytes: preparedBytes,
    mimeType: _mimeTypeFor(format),
    width: decoded.width,
    height: decoded.height,
    format: format,
    metadata: stripped.metadata,
    hasTransparency: stripped.hasTransparency,
  );
}

SupportedImageFormat _resolveSupportedImageFormat({
  required String fileName,
  required String mimeType,
}) {
  final normalizedMime = mimeType.trim().toLowerCase();
  switch (normalizedMime) {
    case 'image/jpeg':
      return SupportedImageFormat.jpeg;
    case 'image/png':
      return SupportedImageFormat.png;
    case 'image/webp':
      return SupportedImageFormat.webp;
  }

  final normalizedName = fileName.toLowerCase();
  if (normalizedName.endsWith('.jpg') || normalizedName.endsWith('.jpeg')) {
    return SupportedImageFormat.jpeg;
  }
  if (normalizedName.endsWith('.png')) {
    return SupportedImageFormat.png;
  }
  if (normalizedName.endsWith('.webp')) {
    return SupportedImageFormat.webp;
  }

  throw const FormatException('Unsupported image format');
}

String _mimeTypeFor(SupportedImageFormat format) => switch (format) {
  SupportedImageFormat.jpeg => 'image/jpeg',
  SupportedImageFormat.png => 'image/png',
  SupportedImageFormat.webp => 'image/webp',
};
