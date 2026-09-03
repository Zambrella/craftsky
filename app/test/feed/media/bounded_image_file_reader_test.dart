import 'dart:typed_data';

import 'package:craftsky_app/feed/media/bounded_image_file_reader.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image_picker/image_picker.dart';

void main() {
  test('rejects reported source length before opening the full file', () async {
    final file = XFile.fromData(
      Uint8List(1),
      name: 'large.jpg',
      length: 17,
    );

    await expectLater(
      readBoundedImageFile(file, maxBytes: 16),
      throwsA(isA<ImageSourceTooLargeException>()),
    );
  });

  test('rejects actual bytes when reported source length is stale', () async {
    final file = XFile.fromData(
      Uint8List(17),
      name: 'large.jpg',
      length: 16,
    );

    await expectLater(
      readBoundedImageFile(file, maxBytes: 16),
      throwsA(isA<ImageSourceTooLargeException>()),
    );
  });
}
