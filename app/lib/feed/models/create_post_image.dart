import 'package:dart_mappable/dart_mappable.dart';

part 'create_post_image.mapper.dart';

/// Image payload for top-level `POST /v1/posts` create requests.
@MappableClass(ignoreNull: true)
class CreatePostImage with CreatePostImageMappable {
  const CreatePostImage({
    required this.blob,
    required this.alt,
    this.aspectRatio,
  });

  /// Uploaded blob reference accepted by AppView.
  @MappableField(key: 'image')
  final CreatePostBlob blob;

  /// User-provided alternative text for accessibility.
  final String alt;

  /// Optional display aspect ratio metadata.
  final CreatePostImageAspectRatio? aspectRatio;

  @override
  Map<String, dynamic> toMap() => {
    'image': blob.toMap(),
    'alt': alt,
    if (aspectRatio != null) 'aspectRatio': aspectRatio!.toMap(),
  };
}

/// AT Protocol blob object embedded in a create-post request.
@MappableClass()
class CreatePostBlob with CreatePostBlobMappable {
  const CreatePostBlob({
    required this.ref,
    required this.mimeType,
    required this.size,
    this.type = 'blob',
  });

  /// AT Protocol blob type discriminator.
  @MappableField(key: r'$type')
  final String type;

  /// CID link wrapper for the uploaded blob.
  final CreatePostBlobRef ref;

  /// CID link for callers that do not need the AT Protocol wrapper.
  String get link => ref.link;

  /// MIME type returned from image upload.
  final String mimeType;

  /// Blob size in bytes.
  final int size;

  @override
  Map<String, dynamic> toMap() => {
    r'$type': type,
    'ref': ref.toMap(),
    'mimeType': mimeType,
    'size': size,
  };
}

/// AT Protocol `$link` wrapper for uploaded blob CIDs.
@MappableClass()
class CreatePostBlobRef with CreatePostBlobRefMappable {
  const CreatePostBlobRef({required this.link});

  /// CID link value for the uploaded blob.
  @MappableField(key: r'$link')
  final String link;
}

/// Optional aspect-ratio metadata attached to an uploaded image.
@MappableClass()
class CreatePostImageAspectRatio with CreatePostImageAspectRatioMappable {
  const CreatePostImageAspectRatio({required this.width, required this.height});

  /// Image width in pixels.
  final int width;

  /// Image height in pixels.
  final int height;
}
