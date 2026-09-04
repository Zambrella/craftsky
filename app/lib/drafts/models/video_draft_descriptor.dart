import 'package:craftsky_app/drafts/models/draft_manifest_error.dart';

enum DraftVideoAvailability { available, unavailable }

/// Persisted metadata for an immutable video source and generated poster.
final class VideoDraftDescriptor {
  const VideoDraftDescriptor({
    required this.storageRevision,
    required this.sourceStorageFileName,
    required this.posterStorageFileName,
    required this.displayFileName,
    required this.mimeType,
    required this.sourceByteLength,
    required this.sourceSha256,
    required this.posterByteLength,
    required this.posterSha256,
    required this.posterMimeType,
    required this.width,
    required this.height,
    required this.duration,
    required this.altText,
    this.availability = DraftVideoAvailability.available,
  });

  final String storageRevision;
  final String sourceStorageFileName;
  final String posterStorageFileName;
  final String displayFileName;
  final String mimeType;
  final int sourceByteLength;
  final String sourceSha256;
  final int posterByteLength;
  final String posterSha256;
  final String posterMimeType;
  final int width;
  final int height;
  final Duration? duration;
  final String altText;
  final DraftVideoAvailability availability;

  VideoDraftDescriptor withAvailability(DraftVideoAvailability value) =>
      VideoDraftDescriptor(
        storageRevision: storageRevision,
        sourceStorageFileName: sourceStorageFileName,
        posterStorageFileName: posterStorageFileName,
        displayFileName: displayFileName,
        mimeType: mimeType,
        sourceByteLength: sourceByteLength,
        sourceSha256: sourceSha256,
        posterByteLength: posterByteLength,
        posterSha256: posterSha256,
        posterMimeType: posterMimeType,
        width: width,
        height: height,
        duration: duration,
        altText: altText,
        availability: value,
      );

  void validate() {
    final valid =
        _uuidPattern.hasMatch(storageRevision) &&
        _isLeaf(sourceStorageFileName) &&
        _isLeaf(posterStorageFileName) &&
        sourceStorageFileName != posterStorageFileName &&
        displayFileName.isNotEmpty &&
        displayFileName.length <= 255 &&
        mimeType.startsWith('video/') &&
        sourceByteLength > 0 &&
        _sha256Pattern.hasMatch(sourceSha256) &&
        posterByteLength > 0 &&
        _sha256Pattern.hasMatch(posterSha256) &&
        (posterMimeType == 'image/jpeg' || posterMimeType == 'image/png') &&
        width > 0 &&
        height > 0 &&
        (duration == null || duration! > Duration.zero) &&
        altText.length <= 1000;
    if (!valid) {
      throw const DraftManifestException(
        DraftManifestFailureReason.invalidMedia,
      );
    }
  }

  @override
  String toString() => 'VideoDraftDescriptor(<redacted>)';
}

final _uuidPattern = RegExp(
  r'^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
  caseSensitive: false,
);
final _sha256Pattern = RegExp(r'^[0-9a-f]{64}$', caseSensitive: false);

bool _isLeaf(String value) =>
    value.isNotEmpty &&
    !value.contains('/') &&
    !value.contains(r'\') &&
    !value.contains('..');
