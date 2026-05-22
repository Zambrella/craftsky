import 'dart:isolate';
import 'dart:typed_data';

import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:image/image.dart' as img;
import 'package:mime/mime.dart';

const _supportedMimeTypes = <String>{'image/jpeg', 'image/png'};
const _supportedExtensions = <String>{'.jpg', '.jpeg', '.png'};
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

/// Coordinates image-related logic used by the composer pipeline.
class ComposerImageMediaService {
  const ComposerImageMediaService({this.config = mediaConfig});

  /// Limits and validation settings for composer media.
  final MediaConfig config;

  /// Maximum images accepted by a post.
  int get maxImages => config.maxImages;

  /// Maximum prepared image size accepted by upload.
  int get maxImageBytes => config.maxImageBytes;

  /// Infers a MIME type from a picked file name.
  String mimeTypeForFileName(String fileName) {
    final lower = fileName.toLowerCase();
    if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
    if (lower.endsWith('.png')) return 'image/png';
    return '';
  }

  /// Validates an original local file before the decode/re-encode step.
  OriginalImageValidationResult validateOriginalImage({
    required int sizeBytes,
    required String fileName,
    required String mimeType,
    required Uint8List headerBytes,
  }) {
    if (sizeBytes > config.maxImageBytes) {
      return const OriginalImageValidationResult(
        canPrepare: false,
        rejectedReason: OriginalImageRejection.tooLarge,
      );
    }

    try {
      _resolveSupportedImageFormat(
        fileName: fileName,
        mimeType: mimeType,
        headerBytes: headerBytes,
      );
    } on FormatException {
      return const OriginalImageValidationResult(
        canPrepare: false,
        rejectedReason: OriginalImageRejection.unsupportedType,
      );
    }

    return const OriginalImageValidationResult(
      canPrepare: true,
      rejectedReason: null,
    );
  }

  /// Validates a new selection against existing composer images.
  ImageSelectionValidationResult validateSelection({
    required List<LocalImageSelection> existing,
    required List<LocalImageSelection> incoming,
  }) {
    var remainingSlots = config.maxImages - existing.length;
    final accepted = <LocalImageSelection>[];
    final rejected = <RejectedImageSelection>[];

    for (final candidate in incoming) {
      if (!_isSupportedType(candidate)) {
        rejected.add(
          RejectedImageSelection(
            image: candidate,
            reason: ImageSelectionRejection.unsupportedType,
          ),
        );
        continue;
      }
      if (remainingSlots <= 0) {
        rejected.add(
          RejectedImageSelection(
            image: candidate,
            reason: ImageSelectionRejection.imageLimitExceeded,
          ),
        );
        continue;
      }

      accepted.add(candidate);
      remainingSlots -= 1;
    }

    return ImageSelectionValidationResult(
      accepted: accepted,
      rejected: rejected,
    );
  }

  /// Starts off-main-isolate image preparation.
  ImagePreparationJob prepareImage({
    required Uint8List bytes,
    required String fileName,
    required String mimeType,
    Map<String, String> metadata = const {},
  }) {
    final request = ImagePreparationRequest(
      bytes: bytes,
      fileName: fileName,
      mimeType: mimeType,
      metadata: metadata,
    );
    return ImagePreparationJob._(
      Isolate.run(() => ComposerImageMediaService._prepareForUpload(request)),
    );
  }

  /// Validates final bytes before upload.
  PreparedUploadValidationResult validatePreparedUploadBytes({
    required int originalBytes,
    required int preparedBytes,
  }) {
    if (preparedBytes > config.maxImageBytes) {
      return const PreparedUploadValidationResult(
        canUpload: false,
        rejectedReason: PreparedUploadRejection.tooLarge,
      );
    }
    return const PreparedUploadValidationResult(
      canUpload: true,
      rejectedReason: null,
    );
  }

  /// Converts decoded dimensions into optional AT Protocol aspect-ratio data.
  CreatePostImageAspectRatio? aspectRatioFor({
    required int width,
    required int height,
  }) {
    if (width <= 0 || height <= 0) return null;
    return CreatePostImageAspectRatio(width: width, height: height);
  }

  bool _isSupportedType(LocalImageSelection candidate) {
    final mime = candidate.mimeType.trim().toLowerCase();
    if (_supportedMimeTypes.contains(mime)) return true;

    final name = candidate.name.toLowerCase();
    return _supportedExtensions.any(name.endsWith);
  }

  static PreparedImagePayload _prepareForUpload(
    ImagePreparationRequest request,
  ) {
    final format = _resolveSupportedImageFormat(
      fileName: request.fileName,
      mimeType: request.mimeType,
      headerBytes: request.bytes,
    );

    final img.Image? decoded;
    try {
      decoded = img.decodeImage(request.bytes);
    } on Object {
      throw const FormatException('Unsupported or corrupt image bytes');
    }
    if (decoded == null) {
      throw const FormatException('Unsupported or corrupt image bytes');
    }

    final preparedImage = _stripEmbeddedMetadata(
      img.bakeOrientation(decoded),
    );

    final stripped = _stripNonEssentialMetadata(
      format: format,
      metadata: request.metadata,
      hasTransparency: preparedImage.hasAlpha,
    );

    final preparedBytes = switch (format) {
      SupportedImageFormat.jpeg => Uint8List.fromList(
        img.encodeJpg(preparedImage),
      ),
      SupportedImageFormat.png => Uint8List.fromList(
        img.encodePng(preparedImage),
      ),
    };

    return PreparedImagePayload(
      bytes: preparedBytes,
      mimeType: _mimeTypeFor(format),
      width: preparedImage.width,
      height: preparedImage.height,
      metadata: stripped.metadata,
      hasTransparency: stripped.hasTransparency,
    );
  }

  static img.Image _stripEmbeddedMetadata(img.Image image) {
    image
      ..exif.clear()
      ..textData = null;
    return image;
  }

  static ({Map<String, String> metadata, bool hasTransparency})
  _stripNonEssentialMetadata({
    required SupportedImageFormat format,
    required Map<String, String> metadata,
    required bool hasTransparency,
  }) {
    final retained = <String, String>{
      for (final entry in metadata.entries)
        if (!_metadataKeysToStrip.contains(entry.key)) entry.key: entry.value,
    };
    return (
      metadata: retained,
      hasTransparency: format == SupportedImageFormat.png && hasTransparency,
    );
  }

  static SupportedImageFormat _resolveSupportedImageFormat({
    required String fileName,
    required String mimeType,
    Uint8List? headerBytes,
  }) {
    final normalizedMime = mimeType.trim().toLowerCase();
    final detectedMime = lookupMimeType(fileName, headerBytes: headerBytes);
    final resolvedMime = detectedMime ?? normalizedMime;

    if (resolvedMime == 'image/jpeg') return SupportedImageFormat.jpeg;
    if (resolvedMime == 'image/png') return SupportedImageFormat.png;

    throw const FormatException('Unsupported image format');
  }

  static String _mimeTypeFor(SupportedImageFormat format) => switch (format) {
    SupportedImageFormat.jpeg => 'image/jpeg',
    SupportedImageFormat.png => 'image/png',
  };
}

/// Why a selected local image was rejected before processing.
enum ImageSelectionRejection { unsupportedType, imageLimitExceeded }

/// Lightweight image identity used while validating a picker selection.
class LocalImageSelection {
  const LocalImageSelection({required this.name, required this.mimeType});

  final String name;
  final String mimeType;
}

/// A selected image rejected before entering the upload pipeline.
class RejectedImageSelection {
  const RejectedImageSelection({required this.image, required this.reason});

  final LocalImageSelection image;
  final ImageSelectionRejection reason;
}

/// Result of validating a local multi-image selection.
class ImageSelectionValidationResult {
  const ImageSelectionValidationResult({
    required this.accepted,
    required this.rejected,
  });

  final List<LocalImageSelection> accepted;
  final List<RejectedImageSelection> rejected;
}

/// Reason prepared bytes cannot be uploaded.
enum PreparedUploadRejection { tooLarge }

/// Reason an original picked file cannot enter the preparation pipeline.
enum OriginalImageRejection { unsupportedType, tooLarge }

/// Result of original local file validation before decode/re-encode.
class OriginalImageValidationResult {
  const OriginalImageValidationResult({
    required this.canPrepare,
    required this.rejectedReason,
  });

  final bool canPrepare;
  final OriginalImageRejection? rejectedReason;
}

/// Result of final upload-size validation.
class PreparedUploadValidationResult {
  const PreparedUploadValidationResult({
    required this.canUpload,
    required this.rejectedReason,
  });

  final bool canUpload;
  final PreparedUploadRejection? rejectedReason;
}

/// Image formats accepted by the local preparation pipeline.
enum SupportedImageFormat { jpeg, png }

/// Sendable request object for off-main-isolate image preparation.
class ImagePreparationRequest {
  const ImagePreparationRequest({
    required this.bytes,
    required this.fileName,
    required this.mimeType,
    required this.metadata,
  });

  final Uint8List bytes;
  final String fileName;
  final String mimeType;
  final Map<String, String> metadata;
}

/// Prepared image data returned from the isolate.
class PreparedImagePayload {
  const PreparedImagePayload({
    required this.bytes,
    required this.mimeType,
    required this.width,
    required this.height,
    required this.metadata,
    required this.hasTransparency,
  });

  final Uint8List bytes;
  final String mimeType;
  final int width;
  final int height;
  final Map<String, String> metadata;
  final bool hasTransparency;
}

/// Handle for an in-flight image preparation task.
class ImagePreparationJob {
  ImagePreparationJob._(this.future);

  final Future<PreparedImagePayload> future;

  void cancel() {}
}
