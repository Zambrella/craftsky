import 'dart:convert';
import 'dart:typed_data';

const maxLinkPreviewThumbnailBytes = 1000000;

class LinkPreview {
  const LinkPreview({
    required this.url,
    required this.title,
    required this.description,
    this.thumbnail,
  });

  factory LinkPreview.fromMap(Map<String, dynamic> map) {
    final url = map['url'];
    final title = map['title'];
    final description = map['description'];
    final thumbnail = map['thumbnail'];
    if (url is! String || title is! String || description is! String) {
      throw const FormatException('invalid link preview response');
    }
    final parsed = Uri.tryParse(url);
    if (parsed == null || !parsed.hasAuthority) {
      throw const FormatException('invalid link preview URL');
    }
    return LinkPreview(
      url: parsed,
      title: title,
      description: description,
      thumbnail: thumbnail == null
          ? null
          : LinkPreviewThumbnail.fromMap(
              Map<String, dynamic>.from(thumbnail as Map),
            ),
    );
  }

  final Uri url;
  final String title;
  final String description;
  final LinkPreviewThumbnail? thumbnail;
}

class LinkPreviewThumbnail {
  const LinkPreviewThumbnail({
    required this.bytes,
    required this.mimeType,
    required this.width,
    required this.height,
  });

  factory LinkPreviewThumbnail.fromMap(Map<String, dynamic> map) {
    final encoded = map['bytes'];
    final mimeType = map['mimeType'];
    final width = map['width'];
    final height = map['height'];
    if (encoded is! String ||
        mimeType is! String ||
        width is! int ||
        height is! int ||
        width <= 0 ||
        height <= 0 ||
        encoded.length > ((maxLinkPreviewThumbnailBytes + 2) ~/ 3) * 4 ||
        encoded.length % 4 != 0) {
      throw const FormatException('invalid link preview thumbnail');
    }
    late Uint8List decoded;
    try {
      decoded = base64Decode(encoded);
    } on FormatException {
      throw const FormatException('invalid link preview thumbnail bytes');
    }
    if (decoded.length > maxLinkPreviewThumbnailBytes ||
        base64Encode(decoded) != encoded) {
      throw const FormatException('invalid link preview thumbnail bytes');
    }
    return LinkPreviewThumbnail(
      bytes: decoded,
      mimeType: mimeType,
      width: width,
      height: height,
    );
  }

  final Uint8List bytes;
  final String mimeType;
  final int width;
  final int height;
}
