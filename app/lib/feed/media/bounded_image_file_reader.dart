import 'dart:typed_data';

import 'package:image_picker/image_picker.dart';

class ImageSourceTooLargeException implements Exception {
  const ImageSourceTooLargeException();
}

Future<Uint8List> readBoundedImageFile(
  XFile file, {
  required int maxBytes,
}) async {
  if (await file.length() > maxBytes) {
    throw const ImageSourceTooLargeException();
  }

  final bytes = BytesBuilder(copy: false);
  var length = 0;
  await for (final chunk in file.openRead()) {
    length += chunk.length;
    if (length > maxBytes) {
      throw const ImageSourceTooLargeException();
    }
    bytes.add(chunk);
  }
  return bytes.takeBytes();
}
