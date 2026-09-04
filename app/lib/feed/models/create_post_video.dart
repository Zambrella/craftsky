final class CreatePostVideo {
  const CreatePostVideo({
    required this.jobId,
    required this.blob,
    this.alt,
    this.aspectRatio,
  });

  final String jobId;
  final CreatePostVideoBlob blob;
  final String? alt;
  final CreatePostVideoAspectRatio? aspectRatio;

  Map<String, Object> toMap() => {
    'jobId': jobId,
    'blob': blob.toMap(),
    'alt': ?alt,
    'aspectRatio': ?aspectRatio?.toMap(),
  };

  @override
  String toString() => 'CreatePostVideo(<redacted>)';
}

final class CreatePostVideoBlob {
  const CreatePostVideoBlob({
    required this.cid,
    required this.mimeType,
    required this.size,
  });

  final String cid;
  final String mimeType;
  final int size;

  Map<String, Object> toMap() => {
    r'$type': 'blob',
    'ref': {r'$link': cid},
    'mimeType': mimeType,
    'size': size,
  };
}

final class CreatePostVideoAspectRatio {
  const CreatePostVideoAspectRatio({required this.width, required this.height});

  final int width;
  final int height;

  Map<String, int> toMap() => {'width': width, 'height': height};
}
