class UploadedImageBlob {
  const UploadedImageBlob({
    required this.blob,
    required this.cid,
    required this.mime,
    required this.size,
  });

  factory UploadedImageBlob.fromMap(Map<String, dynamic> json) {
    return UploadedImageBlob(
      blob: UploadedBlob.fromMap(json['blob'] as Map<String, dynamic>),
      cid: json['cid'] as String,
      mime: json['mime'] as String,
      size: json['size'] as int,
    );
  }

  final UploadedBlob blob;
  final String cid;
  final String mime;
  final int size;
}

class UploadedBlob {
  const UploadedBlob({
    required this.type,
    required this.ref,
    required this.mimeType,
    required this.size,
  });

  factory UploadedBlob.fromMap(Map<String, dynamic> json) {
    return UploadedBlob(
      type: json[r'$type'] as String,
      ref: UploadedBlobRef.fromMap(json['ref'] as Map<String, dynamic>),
      mimeType: json['mimeType'] as String,
      size: json['size'] as int,
    );
  }

  final String type;
  final UploadedBlobRef ref;
  final String mimeType;
  final int size;
}

class UploadedBlobRef {
  const UploadedBlobRef({required this.link});

  factory UploadedBlobRef.fromMap(Map<String, dynamic> json) {
    return UploadedBlobRef(link: json[r'$link'] as String);
  }

  final String link;
}
