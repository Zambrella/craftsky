class CreatePostImage {
  const CreatePostImage({
    required this.blob,
    required this.alt,
    this.aspectRatio,
  });

  final CreatePostBlob blob;
  final String alt;
  final CreatePostImageAspectRatio? aspectRatio;

  Map<String, dynamic> toMap() => {
    'blob': blob.toMap(),
    'alt': alt,
    if (aspectRatio != null) 'aspectRatio': aspectRatio!.toMap(),
  };
}

class CreatePostBlob {
  const CreatePostBlob({
    required this.link,
    required this.mimeType,
    required this.size,
  });

  final String link;
  final String mimeType;
  final int size;

  Map<String, dynamic> toMap() => {
    r'$type': 'blob',
    'ref': {r'$link': link},
    'mimeType': mimeType,
    'size': size,
  };
}

class CreatePostImageAspectRatio {
  const CreatePostImageAspectRatio({required this.width, required this.height});

  final int width;
  final int height;

  Map<String, dynamic> toMap() => {'width': width, 'height': height};
}
