import 'package:dart_mappable/dart_mappable.dart';

part 'uploaded_image_blob.mapper.dart';

/// Response payload returned from `POST /v1/blobs/images`.
@MappableClass()
class UploadedImageBlob with UploadedImageBlobMappable {
  const UploadedImageBlob({
    required this.blob,
    required this.cid,
    required this.mime,
    required this.size,
  });

  factory UploadedImageBlob.fromMap(Map<String, dynamic> json) {
    return UploadedImageBlobMapper.fromMap(json);
  }

  /// ATProto blob object for the upload response.
  final UploadedBlob blob;

  /// CID for the uploaded image blob.
  final String cid;

  /// MIME type stored for the uploaded blob.
  final String mime;

  /// Blob size in bytes.
  final int size;
}

/// Canonical blob representation from AppView upload endpoints.
@MappableClass()
class UploadedBlob with UploadedBlobMappable {
  const UploadedBlob({
    required this.type,
    required this.ref,
    required this.mimeType,
    required this.size,
  });

  factory UploadedBlob.fromMap(Map<String, dynamic> json) {
    return UploadedBlobMapper.fromMap(json);
  }

  /// ATProto blob object marker, expected to be `blob`.
  @MappableField(key: r'$type')
  final String type;

  /// Blob CID link wrapper.
  final UploadedBlobRef ref;

  /// MIME type for the uploaded blob.
  final String mimeType;

  /// Blob size in bytes.
  final int size;
}

/// Link container for uploaded blob CIDs.
@MappableClass()
class UploadedBlobRef with UploadedBlobRefMappable {
  const UploadedBlobRef({required this.link});

  factory UploadedBlobRef.fromMap(Map<String, dynamic> json) {
    return UploadedBlobRefMapper.fromMap(json);
  }

  /// CID value used in `$link` ATProto fields.
  @MappableField(key: r'$link')
  final String link;
}
