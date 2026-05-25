import 'dart:typed_data';

import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'composer_image_state.mapper.dart';

/// Complete image attachment state for a single post composer instance.
@MappableClass()
class ComposerImagesState with ComposerImagesStateMappable {
  const ComposerImagesState({required this.images, this.notice});

  /// Images currently attached to the composer, in display/submission order.
  final List<ComposerImageDraft> images;

  /// One-shot user feedback emitted by image selection or processing actions.
  final ComposerImageNotice? notice;

  /// Whether all attached images are uploaded and any provided alt text is
  /// valid.
  bool canSubmitImages({MediaConfig config = mediaConfig}) {
    for (final image in images) {
      if (image.phase is! ImageUploaded) return false;

      final alt = image.altText.trim();
      if (alt.length > config.maxAltTextCharacters) {
        return false;
      }
    }

    return true;
  }

  bool get hasImagesMissingAltText {
    return images.any((image) => image.altText.trim().isEmpty);
  }

  /// Builds the API image payload for currently uploaded images.
  List<CreatePostImage>? toCreatePostImages() {
    if (images.isEmpty) return null;
    return images.map((image) {
      final uploaded = (image.phase as ImageUploaded).uploaded;
      return CreatePostImage(
        blob: CreatePostBlob(
          ref: CreatePostBlobRef(link: uploaded.cid),
          mimeType: uploaded.mime,
          size: uploaded.size,
        ),
        alt: image.altText.trim(),
        aspectRatio: uploaded.aspectRatio,
      );
    }).toList();
  }

  @override
  String toString() {
    final imageSummary = images.isEmpty ? '()' : '(${images.join(', ')})';
    return 'ComposerImagesState(images: $imageSummary, notice: $notice)';
  }
}

/// A single selected image and its current lifecycle state.
@MappableClass()
class ComposerImageDraft with ComposerImageDraftMappable {
  const ComposerImageDraft({
    required this.id,
    required this.fileName,
    required this.mimeType,
    required this.altText,
    required this.phase,
    this.previewBytes,
    this.previewAspectRatio,
  });

  /// Stable client-side id used for UI keys and operation tracking.
  final String id;

  /// Original file name reported by the platform picker.
  final String fileName;

  /// MIME type inferred before upload preparation.
  final String mimeType;

  /// User-authored alt text submitted with the post.
  final String altText;

  /// Local preview bytes displayed while processing/uploading.
  final Uint8List? previewBytes;

  /// Aspect ratio for the local preview before the upload response exists.
  final CreatePostImageAspectRatio? previewAspectRatio;

  /// Current lifecycle state for this image.
  final ComposerImagePhase phase;

  @override
  String toString() {
    return 'ComposerImageDraft('
        'id: $id, '
        'fileName: $fileName, '
        'mimeType: $mimeType, '
        'altText: $altText, '
        'phase: $phase, '
        'previewBytes: ${_byteSummary(previewBytes)}, '
        'previewAspectRatio: $previewAspectRatio'
        ')';
  }
}

String _byteSummary(Uint8List? bytes) {
  if (bytes == null) return 'null';
  return '${bytes.length} bytes';
}

/// Sealed lifecycle phase for a selected composer image.
@MappableClass(discriminatorKey: 'type')
sealed class ComposerImagePhase with ComposerImagePhaseMappable {
  const ComposerImagePhase();
}

/// Image was accepted and is waiting for the local pipeline to start.
@MappableClass()
final class ImageQueued extends ComposerImagePhase with ImageQueuedMappable {
  const ImageQueued();
}

/// Original bytes are being read from the picked file.
@MappableClass()
final class ImageReading extends ComposerImagePhase with ImageReadingMappable {
  const ImageReading();
}

/// Image bytes are being decoded, stripped, and re-encoded off the UI isolate.
@MappableClass()
final class ImagePreparing extends ComposerImagePhase
    with ImagePreparingMappable {
  const ImagePreparing();
}

/// Prepared bytes are being uploaded to the AppView.
@MappableClass()
final class ImageUploading extends ComposerImagePhase
    with ImageUploadingMappable {
  const ImageUploading(this.progress);

  /// Combined send/receive transfer progress for this upload.
  final ImageTransferProgress progress;
}

/// Image upload succeeded and can be included in the create-post payload.
@MappableClass()
final class ImageUploaded extends ComposerImagePhase
    with ImageUploadedMappable {
  const ImageUploaded(this.uploaded);

  /// Uploaded blob returned by the AppView.
  final UploadedDraftImage uploaded;
}

/// Image processing or upload failed.
@MappableClass()
final class ImageFailed extends ComposerImagePhase with ImageFailedMappable {
  const ImageFailed(this.failure);

  /// Structured failure reason for UI copy and retry behavior.
  final ComposerImageFailure failure;
}

/// Uploaded image blob metadata used when creating the final post.
@MappableClass()
class UploadedDraftImage with UploadedDraftImageMappable {
  const UploadedDraftImage({
    required this.cid,
    required this.mime,
    required this.size,
    this.aspectRatio,
  });

  /// Content id for the uploaded blob.
  final String cid;

  /// MIME type returned by the AppView after upload.
  final String mime;

  /// Uploaded blob size in bytes.
  final int size;

  /// Optional pixel aspect ratio preserved in the post record.
  final CreatePostImageAspectRatio? aspectRatio;
}

/// Structured image failure used instead of stringly-typed lifecycle errors.
@MappableClass(discriminatorKey: 'type')
sealed class ComposerImageFailure with ComposerImageFailureMappable {
  const ComposerImageFailure();

  /// User-facing failure copy.
  String get message;

  /// Whether the UI should offer a retry action.
  bool get canRetry => false;
}

/// Selected file type is not supported by the image upload pipeline.
@MappableClass()
final class UnsupportedImageType extends ComposerImageFailure
    with UnsupportedImageTypeMappable {
  const UnsupportedImageType();

  @override
  String get message => 'Unsupported image type';
}

/// Image preparation failed before upload started.
@MappableClass()
final class ImagePreparationFailed extends ComposerImageFailure
    with ImagePreparationFailedMappable {
  const ImagePreparationFailed();

  @override
  String get message => 'Could not prepare image';
}

/// Prepared bytes exceeded the AppView image upload limit.
@MappableClass()
final class ImageTooLarge extends ComposerImageFailure
    with ImageTooLargeMappable {
  const ImageTooLarge(this.maxBytes);

  /// Maximum allowed prepared image size in bytes.
  final int maxBytes;

  @override
  String get message => 'Image exceeds ${maxBytes ~/ (1024 * 1024)} MB limit';
}

/// Upload failed after local preparation succeeded.
@MappableClass()
final class ImageUploadFailed extends ComposerImageFailure
    with ImageUploadFailedMappable {
  const ImageUploadFailed();

  @override
  String get message => 'Upload failed';

  @override
  bool get canRetry => true;
}

/// Combined upload/download progress for an image upload request.
@MappableClass(discriminatorKey: 'type')
sealed class ImageTransferProgress with ImageTransferProgressMappable {
  const ImageTransferProgress();

  /// Progress value for Flutter indicators; `null` means indeterminate.
  double? get indicatorValue;
}

/// Transfer has started, but Dio has not reported byte totals yet.
@MappableClass()
final class TransferStarting extends ImageTransferProgress
    with TransferStartingMappable {
  const TransferStarting();

  @override
  double? get indicatorValue => null;
}

/// Transfer progress with the latest send and receive byte counts.
@MappableClass()
final class TransferBytes extends ImageTransferProgress
    with TransferBytesMappable {
  const TransferBytes({
    required this.sent,
    required this.sendTotal,
    required this.received,
    required this.receiveTotal,
  });

  /// Bytes sent to the server.
  final int sent;

  /// Total bytes Dio expects to send, or `0` if unknown.
  final int sendTotal;

  /// Bytes received from the server response.
  final int received;

  /// Total response bytes Dio expects to receive, or `0` if unknown.
  final int receiveTotal;

  @override
  double? get indicatorValue {
    final knownReceiveTotal = receiveTotal > 0 ? receiveTotal : 0;
    final total = sendTotal + knownReceiveTotal;
    if (total <= 0) return null;
    final completed = sent + (receiveTotal > 0 ? received : 0);
    return (completed / total).clamp(0, 1);
  }
}

/// Upload body finished and the client is waiting for the AppView response.
@MappableClass()
final class TransferFinalizing extends ImageTransferProgress
    with TransferFinalizingMappable {
  const TransferFinalizing();

  @override
  double? get indicatorValue => null;
}

/// One-shot notice emitted by image selection/processing state.
@MappableClass(discriminatorKey: 'type')
sealed class ComposerImageNotice with ComposerImageNoticeMappable {
  const ComposerImageNotice({required this.id});

  /// Monotonic id used by listeners to clear only the notice they consumed.
  final int id;
}

/// User selected more images than the composer can accept.
@MappableClass()
final class ImageSelectionLimitNotice extends ComposerImageNotice
    with ImageSelectionLimitNoticeMappable {
  const ImageSelectionLimitNotice({
    required super.id,
    required this.maxImages,
    required this.acceptedCount,
  });

  /// Maximum images allowed in a single post.
  final int maxImages;

  /// Number of selected images accepted before the limit was reached.
  final int acceptedCount;
}

/// One or more selected images had unsupported MIME types.
@MappableClass()
final class UnsupportedImagesNotice extends ComposerImageNotice
    with UnsupportedImagesNoticeMappable {
  const UnsupportedImagesNotice({required super.id, required this.count});

  /// Number of rejected unsupported images.
  final int count;
}

/// The platform picker could not complete image selection.
@MappableClass()
final class ImagePickerFailedNotice extends ComposerImageNotice
    with ImagePickerFailedNoticeMappable {
  const ImagePickerFailedNotice({required super.id});
}
