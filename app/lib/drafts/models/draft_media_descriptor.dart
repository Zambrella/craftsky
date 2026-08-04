import 'package:craftsky_app/drafts/models/draft_manifest_error.dart';

enum DraftMediaAvailability { available, unavailable }

/// Persisted metadata for one immutable, app-owned draft image.
final class DraftMediaDescriptor {
  const DraftMediaDescriptor({
    required this.mediaId,
    required this.storageRevision,
    required this.storageFileName,
    required this.displayFileName,
    required this.mimeType,
    required this.byteLength,
    required this.sha256,
    required this.width,
    required this.height,
    required this.altText,
    required this.order,
    this.availability = DraftMediaAvailability.available,
  });

  final String mediaId;
  final String storageRevision;
  final String storageFileName;
  final String displayFileName;
  final String mimeType;
  final int byteLength;
  final String sha256;
  final int width;
  final int height;
  final String altText;
  final int order;
  final DraftMediaAvailability availability;

  DraftMediaDescriptor withAvailability(DraftMediaAvailability value) =>
      DraftMediaDescriptor(
        mediaId: mediaId,
        storageRevision: storageRevision,
        storageFileName: storageFileName,
        displayFileName: displayFileName,
        mimeType: mimeType,
        byteLength: byteLength,
        sha256: sha256,
        width: width,
        height: height,
        altText: altText,
        order: order,
        availability: value,
      );

  void validate() {
    final isLeafFileName =
        storageFileName.isNotEmpty &&
        !storageFileName.contains('/') &&
        !storageFileName.contains(r'\') &&
        !storageFileName.contains('..');
    final valid =
        _isUuid(mediaId) &&
        _isUuid(storageRevision) &&
        isLeafFileName &&
        displayFileName.isNotEmpty &&
        displayFileName.length <= 255 &&
        (mimeType == 'image/jpeg' || mimeType == 'image/png') &&
        byteLength > 0 &&
        _sha256Pattern.hasMatch(sha256) &&
        width > 0 &&
        height > 0 &&
        altText.length <= 1000 &&
        order >= 0 &&
        order < 4;
    if (!valid) {
      throw const DraftManifestException(
        DraftManifestFailureReason.invalidMedia,
      );
    }
  }

  @override
  String toString() => 'DraftMediaDescriptor(<redacted>)';
}

final _uuidPattern = RegExp(
  r'^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
  caseSensitive: false,
);
final _sha256Pattern = RegExp(r'^[0-9a-f]{64}$', caseSensitive: false);

bool _isUuid(String value) => _uuidPattern.hasMatch(value);
