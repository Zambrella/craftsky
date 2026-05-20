import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_image_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image_picker/image_picker.dart';

void main() {
  test(
    'device picker returns full selection for downstream cap validation',
    () async {
      final picker = DeviceComposerImagePicker(
        picker: _FakeImagePicker(
          files: [
            XFile.fromData(Uint8List.fromList([1]), name: 'one.jpg'),
            XFile.fromData(Uint8List.fromList([2]), name: 'two.jpg'),
          ],
        ),
      );

      final selected = await picker.pickImages(maxImages: 1);

      expect(selected, hasLength(2));
    },
  );
}

class _FakeImagePicker extends ImagePicker {
  _FakeImagePicker({required this.files});

  final List<XFile> files;

  @override
  Future<List<XFile>> pickMultiImage({
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    int? limit,
    bool requestFullMetadata = true,
  }) async {
    return files;
  }
}
