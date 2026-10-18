import 'dart:math' as math;
import 'dart:typed_data';

import 'package:craftsky_app/feed/media/heif_image_info.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/media/platform_image_compression.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:image_size_getter/image_size_getter.dart' as image_size;
import 'package:mime/mime.dart';

final class ImageSourceDimensionsTooLargeException implements Exception {
  const new({
    required this.width,
    required this.height,
    required this.maxSide,
    required this.maxPixels,
  });

  final int width;
  final int height;
  final int maxSide;
  final int maxPixels;
}

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

/// The AppView hard geometry policy for private scheduled images.
const defaultScheduledImageLimits = ScheduledImageLimits(
  maxWidth: 8192,
  maxHeight: 8192,
  maxPixels: 16000000,
  maxAspectRatio: defaultMaxImageAspectRatio,
);

/// Coordinates image-related logic used by the composer pipeline.
class ComposerImageMediaService {
  const new({
    this.config = mediaConfig,
    this.scheduledImageLimits = defaultScheduledImageLimits,
  });

  /// Limits and validation settings for composer media.
  final MediaConfig config;

  /// Geometry policy applied only when preparing private scheduled media.
  final ScheduledImageLimits scheduledImageLimits;

  /// Maximum images accepted by a post.
  int get maxImages => config.maxImages;

  /// Maximum prepared image size accepted by upload.
  int get maxImageBytes => config.maxImageBytes;

  /// Infers a MIME type from a picked file name.
  String mimeTypeForFileName(String fileName) {
    final lower = fileName.toLowerCase();
    if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
    if (lower.endsWith('.png')) return 'image/png';
    if (supportsAppleHeic &&
        (lower.endsWith('.heic') || lower.endsWith('.heif'))) {
      return 'image/heic';
    }
    return '';
  }

  /// Validates an original local file before the decode/re-encode step.
  OriginalImageValidationResult validateOriginalImage({
    required int sizeBytes,
    required String fileName,
    required String mimeType,
    required Uint8List headerBytes,
  }) {
    if (sizeBytes > config.maxSourceImageBytes) {
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
        allowHeic: supportsAppleHeic,
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

  /// Starts asynchronous image preparation using the platform codec.
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
      config: config,
      allowHeic: supportsAppleHeic,
    );
    return ImagePreparationJob._(
      _prepareWithPlatform(request),
    );
  }

  /// Prepares scheduled media and proportionally downsizes it to AppView's
  /// private-media geometry policy before upload.
  ImagePreparationJob prepareScheduledImage({
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
      config: config,
      allowHeic: supportsAppleHeic,
    );
    final limits = scheduledImageLimits;
    return ImagePreparationJob._(
      _prepareWithPlatform(request, scheduledLimits: limits),
    );
  }

  /// Reads oriented dimensions for the local preview before it is displayed.
  ImageInspectionJob inspectImage({
    required Uint8List bytes,
    required String fileName,
    required String mimeType,
  }) {
    final request = ImagePreparationRequest(
      bytes: bytes,
      fileName: fileName,
      mimeType: mimeType,
      metadata: const {},
      config: config,
      allowHeic: supportsAppleHeic,
    );
    return ImageInspectionJob._(
      Future.sync(() => ComposerImageMediaService._inspectForPreview(request)),
    );
  }

  /// Validates final bytes before upload.
  PreparedUploadValidationResult validatePreparedUploadBytes({
    required int originalBytes,
    required int preparedBytes,
    required int width,
    required int height,
  }) {
    if (preparedBytes > config.maxImageBytes) {
      return const PreparedUploadValidationResult(
        canUpload: false,
        rejectedReason: PreparedUploadRejection.tooLarge,
      );
    }
    if (width <= 0 ||
        height <= 0 ||
        width > config.maxImageWidth ||
        height > config.maxImageHeight ||
        !_hasValidAspectRatio(width, height, config.maxImageAspectRatio)) {
      return const PreparedUploadValidationResult(
        canUpload: false,
        rejectedReason: PreparedUploadRejection.invalidDimensions,
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
    if (supportsAppleHeic &&
        (mime == 'image/heic' ||
            mime == 'image/heif' ||
            candidate.name.toLowerCase().endsWith('.heic') ||
            candidate.name.toLowerCase().endsWith('.heif'))) {
      return true;
    }

    final name = candidate.name.toLowerCase();
    return _supportedExtensions.any(name.endsWith);
  }

  // Serialize preparation to avoid decoding several 48 MP sources at once.
  static Future<void> _platformTail = Future<void>.value();

  Future<PreparedImagePayload> _prepareWithPlatform(
    ImagePreparationRequest request, {
    ScheduledImageLimits? scheduledLimits,
  }) {
    final operation = _platformTail.then(
      (_) => _encodeWithPlatform(request, scheduledLimits),
    );
    _platformTail = operation.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    return operation;
  }

  static void _validateSourceDimensions(
    ImagePreparationRequest request,
    int width,
    int height,
  ) {
    if (width <= 0 || height <= 0) {
      throw const FormatException('Invalid image dimensions');
    }
    if (width > request.config.maxSourceImageSide ||
        height > request.config.maxSourceImageSide ||
        width * height > request.config.maxSourceImagePixels) {
      throw ImageSourceDimensionsTooLargeException(
        width: width,
        height: height,
        maxSide: request.config.maxSourceImageSide,
        maxPixels: request.config.maxSourceImagePixels,
      );
    }
  }

  static Future<PreparedImagePayload> _encodeWithPlatform(
    ImagePreparationRequest request,
    ScheduledImageLimits? scheduledLimits,
  ) async {
    if (request.bytes.isEmpty ||
        request.bytes.length > request.config.maxSourceImageBytes) {
      throw const FormatException('Invalid source image size');
    }
    final format = _resolveSupportedImageFormat(
      fileName: request.fileName,
      mimeType: request.mimeType,
      headerBytes: request.bytes,
      allowHeic: request.allowHeic,
    );
    final dimensions = _inspectDimensions(request, format);
    final sourceWidth = dimensions.orientedWidth;
    final sourceHeight = dimensions.orientedHeight;
    if (!_hasValidAspectRatio(
      sourceWidth,
      sourceHeight,
      request.config.maxImageAspectRatio,
    )) {
      throw const FormatException('Image aspect ratio is too extreme');
    }
    final scale = math.min(
      1,
      math.min(
        request.config.maxImageWidth / sourceWidth,
        request.config.maxImageHeight / sourceHeight,
      ),
    );
    var width = math.max(1, (sourceWidth * scale).floor());
    var height = math.max(1, (sourceHeight * scale).floor());
    if (scheduledLimits != null) {
      final fitted = scheduledLimits.fit(width: width, height: height);
      width = fitted.width;
      height = fitted.height;
    }
    // Preserve PNG alpha, including palette transparency, without decoding a
    // full bitmap merely to decide the output format.
    var preservePng = format == SupportedImageFormat.png;
    final hasAlpha = preservePng && _pngMayHaveTransparency(request.bytes);
    PreparedImagePayload? hardLimitCandidate;
    for (var round = 0; round <= request.config.maxResizeRounds; round++) {
      var low = request.config.minJpegQuality;
      var high = request.config.maxJpegQuality;
      var smallest = request.bytes.length;
      PreparedImagePayload? best;
      for (
        var attempt = 0;
        attempt < (preservePng ? 1 : request.config.maxJpegQualityAttempts) &&
            low <= high;
        attempt++
      ) {
        final quality = (low + high) ~/ 2;
        final encoded = await compressPlatformImage(
          bytes: request.bytes,
          width: width,
          height: height,
          quality: quality,
          png: preservePng,
        );
        final outputFormat = preservePng
            ? SupportedImageFormat.png
            : SupportedImageFormat.jpeg;
        final outputDimensions = _inspectDimensions(
          ImagePreparationRequest(
            bytes: encoded,
            fileName: preservePng ? 'output.png' : 'output.jpg',
            mimeType: _mimeTypeFor(outputFormat),
            metadata: const {},
            config: request.config,
          ),
          outputFormat,
        );
        final outWidth = outputDimensions.orientedWidth;
        final outHeight = outputDimensions.orientedHeight;
        if (outWidth > request.config.maxImageWidth ||
            outHeight > request.config.maxImageHeight ||
            (scheduledLimits != null &&
                !scheduledLimits.allows(width: outWidth, height: outHeight))) {
          throw const FormatException(
            'Compressed image exceeds dimension limits',
          );
        }
        final payload = PreparedImagePayload(
          bytes: encoded,
          mimeType: _mimeTypeFor(outputFormat),
          width: outWidth,
          height: outHeight,
          metadata: _stripNonEssentialMetadata(
            format: outputFormat,
            metadata: request.metadata,
            hasTransparency: hasAlpha,
          ).metadata,
          hasTransparency: hasAlpha,
        );
        smallest = math.min(smallest, encoded.length);
        if (encoded.length <= request.config.maxImageBytes) {
          hardLimitCandidate = payload;
        }
        if (encoded.length <= request.config.targetImageBytes) {
          best = payload;
          low = quality + 1;
        } else if (preservePng && !hasAlpha) {
          preservePng = false;
          low = request.config.minJpegQuality;
          high = request.config.maxJpegQuality;
          attempt = -1;
        } else {
          high = quality - 1;
        }
      }
      if (best != null) return best;
      if (round == request.config.maxResizeRounds) break;
      final shrink = math.min(
        0.9,
        math.sqrt(request.config.targetImageBytes / smallest) * 0.9,
      );
      width = math.max(1, (width * shrink).floor());
      height = math.max(1, (height * shrink).floor());
    }
    if (hardLimitCandidate != null) return hardLimitCandidate;
    throw const FormatException(
      'Image could not be compressed below the upload limit',
    );
  }

  static bool _pngMayHaveTransparency(Uint8List bytes) {
    if (bytes.length < 26) return false;
    if (bytes[25] == 4 || bytes[25] == 6) return true;
    final data = ByteData.sublistView(bytes);
    var offset = 8;
    while (offset + 12 <= bytes.length) {
      final size = data.getUint32(offset);
      if (size > bytes.length - offset - 12) break;
      if (String.fromCharCodes(bytes.sublist(offset + 4, offset + 8)) ==
          'tRNS') {
        return true;
      }
      offset += size + 12;
    }
    return false;
  }

  static ImageInspectionPayload _inspectForPreview(
    ImagePreparationRequest request,
  ) {
    final format = _resolveSupportedImageFormat(
      fileName: request.fileName,
      mimeType: request.mimeType,
      headerBytes: request.bytes,
      allowHeic: request.allowHeic,
    );

    final inspection = _inspectDimensions(request, format);
    return ImageInspectionPayload(
      width: inspection.orientedWidth,
      height: inspection.orientedHeight,
    );
  }

  static ({
    int rawWidth,
    int rawHeight,
    int orientedWidth,
    int orientedHeight,
  })
  _inspectDimensions(
    ImagePreparationRequest request,
    SupportedImageFormat format,
  ) {
    try {
      if (format == SupportedImageFormat.heic) {
        final info = HeifImageInfo.read(request.bytes);
        _validateSourceDimensions(request, info.width, info.height);
        return (
          rawWidth: info.width,
          rawHeight: info.height,
          orientedWidth: info.rotated ? info.height : info.width,
          orientedHeight: info.rotated ? info.width : info.height,
        );
      }
      final input = image_size.MemoryInput(request.bytes);
      final image_size.BaseDecoder decoder = switch (format) {
        SupportedImageFormat.jpeg => const image_size.JpegDecoder(),
        SupportedImageFormat.png => const image_size.PngDecoder(),
        SupportedImageFormat.heic => throw const FormatException(
          'HEIC requires Apple decoding',
        ),
      };
      if (!decoder.isValid(input)) {
        throw const FormatException('Unsupported or corrupt image bytes');
      }
      final dimensions = decoder.getSize(input);
      if (dimensions.width <= 0 || dimensions.height <= 0) {
        throw const FormatException('Unsupported or corrupt image bytes');
      }
      if (dimensions.width > request.config.maxSourceImageSide ||
          dimensions.height > request.config.maxSourceImageSide ||
          dimensions.width * dimensions.height >
              request.config.maxSourceImagePixels) {
        throw ImageSourceDimensionsTooLargeException(
          width: dimensions.width,
          height: dimensions.height,
          maxSide: request.config.maxSourceImageSide,
          maxPixels: request.config.maxSourceImagePixels,
        );
      }
      return (
        rawWidth: dimensions.width,
        rawHeight: dimensions.height,
        orientedWidth: dimensions.needRotate
            ? dimensions.height
            : dimensions.width,
        orientedHeight: dimensions.needRotate
            ? dimensions.width
            : dimensions.height,
      );
    } on ImageSourceDimensionsTooLargeException {
      rethrow;
    } on FormatException {
      rethrow;
    } on Object {
      throw const FormatException('Unsupported or corrupt image bytes');
    }
  }

  static bool _hasValidAspectRatio(int width, int height, int maxRatio) {
    return width > 0 &&
        height > 0 &&
        width <= height * maxRatio &&
        height <= width * maxRatio;
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
    bool allowHeic = false,
  }) {
    if (allowHeic &&
        headerBytes != null &&
        HeifImageInfo.hasHeicBrand(headerBytes)) {
      return SupportedImageFormat.heic;
    }
    final normalizedMime = mimeType.trim().toLowerCase();
    final detectedMime = lookupMimeType(fileName, headerBytes: headerBytes);
    final resolvedMime = detectedMime ?? normalizedMime;

    if (resolvedMime == 'image/jpeg') return SupportedImageFormat.jpeg;
    if (resolvedMime == 'image/png') return SupportedImageFormat.png;
    if (allowHeic &&
        (headerBytes == null || headerBytes.isEmpty) &&
        (resolvedMime == 'image/heic' || resolvedMime == 'image/heif')) {
      return SupportedImageFormat.heic;
    }

    throw const FormatException('Unsupported image format');
  }

  static String _mimeTypeFor(SupportedImageFormat format) => switch (format) {
    SupportedImageFormat.jpeg => 'image/jpeg',
    SupportedImageFormat.png => 'image/png',
    SupportedImageFormat.heic => 'image/heic',
  };
}

/// Scheduled-image geometry limits mirrored from AppView's non-disableable
/// server policy. Tests may use lower limits to exercise resize behavior.
class ScheduledImageLimits {
  const new({
    required this.maxWidth,
    required this.maxHeight,
    required this.maxPixels,
    required this.maxAspectRatio,
  }) : assert(maxWidth > 0, 'maxWidth must be positive'),
       assert(maxHeight > 0, 'maxHeight must be positive'),
       assert(maxPixels > 0, 'maxPixels must be positive'),
       assert(maxAspectRatio > 0, 'maxAspectRatio must be positive');

  final int maxWidth;
  final int maxHeight;
  final int maxPixels;
  final int maxAspectRatio;

  bool allows({required int width, required int height}) {
    if (width <= 0 ||
        height <= 0 ||
        width > maxWidth ||
        height > maxHeight ||
        width * height > maxPixels) {
      return false;
    }
    final longSide = math.max(width, height);
    final shortSide = math.min(width, height);
    return longSide <= shortSide * maxAspectRatio;
  }

  ImageDimensions fit({required int width, required int height}) {
    if (width <= 0 || height <= 0) {
      throw const FormatException('Scheduled image dimensions are invalid');
    }
    final longSide = math.max(width, height);
    final shortSide = math.min(width, height);
    if (longSide > shortSide * maxAspectRatio) {
      throw FormatException(
        'Scheduled image aspect ratio must not exceed $maxAspectRatio:1',
      );
    }
    if (allows(width: width, height: height)) {
      return ImageDimensions(width: width, height: height);
    }

    final scale = math.min(
      1,
      math.min(
        maxWidth / width,
        math.min(maxHeight / height, math.sqrt(maxPixels / (width * height))),
      ),
    );
    final target = ImageDimensions(
      width: math.max(1, (width * scale).floor()),
      height: math.max(1, (height * scale).floor()),
    );
    if (!allows(width: target.width, height: target.height)) {
      throw const FormatException(
        'Scheduled image could not be resized to the supported dimensions',
      );
    }
    return target;
  }
}

/// Integer dimensions selected for a proportional scheduled-image resize.
class ImageDimensions {
  const new({required this.width, required this.height});

  final int width;
  final int height;
}

/// Why a selected local image was rejected before processing.
enum ImageSelectionRejection { unsupportedType, imageLimitExceeded }

/// Lightweight image identity used while validating a picker selection.
class LocalImageSelection {
  const new({required this.name, required this.mimeType});

  final String name;
  final String mimeType;
}

/// A selected image rejected before entering the upload pipeline.
class RejectedImageSelection {
  const new({required this.image, required this.reason});

  final LocalImageSelection image;
  final ImageSelectionRejection reason;
}

/// Result of validating a local multi-image selection.
class ImageSelectionValidationResult {
  const new({
    required this.accepted,
    required this.rejected,
  });

  final List<LocalImageSelection> accepted;
  final List<RejectedImageSelection> rejected;
}

/// Reason prepared bytes cannot be uploaded.
enum PreparedUploadRejection { tooLarge, invalidDimensions }

/// Reason an original picked file cannot enter the preparation pipeline.
enum OriginalImageRejection { unsupportedType, tooLarge }

/// Result of original local file validation before decode/re-encode.
class OriginalImageValidationResult {
  const new({
    required this.canPrepare,
    required this.rejectedReason,
  });

  final bool canPrepare;
  final OriginalImageRejection? rejectedReason;
}

/// Result of final upload-size validation.
class PreparedUploadValidationResult {
  const new({
    required this.canUpload,
    required this.rejectedReason,
  });

  final bool canUpload;
  final PreparedUploadRejection? rejectedReason;
}

/// Image formats accepted by the local preparation pipeline.
enum SupportedImageFormat { jpeg, png, heic }

/// Sendable request object for off-main-isolate image preparation.
class ImagePreparationRequest {
  const new({
    required this.bytes,
    required this.fileName,
    required this.mimeType,
    required this.metadata,
    required this.config,
    this.allowHeic = false,
  });

  final Uint8List bytes;
  final String fileName;
  final String mimeType;
  final Map<String, String> metadata;
  final MediaConfig config;
  final bool allowHeic;
}

/// Prepared image data returned by the processing engine.
class PreparedImagePayload {
  const new({
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

/// Lightweight image data needed before rendering the local preview.
class ImageInspectionPayload {
  const new({required this.width, required this.height});

  final int width;
  final int height;
}

/// Handle for an in-flight image inspection task.
class ImageInspectionJob {
  new _(this.future);

  final Future<ImageInspectionPayload> future;

  void cancel() {}
}

/// Handle for an in-flight image preparation task.
class ImagePreparationJob {
  new _(this.future);

  final Future<PreparedImagePayload> future;

  void cancel() {}
}
