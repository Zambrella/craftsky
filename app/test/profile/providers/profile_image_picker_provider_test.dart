import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';

void main() {
  for (final source in [ImageSource.gallery, ImageSource.camera]) {
    test('prepares an image selected from ${source.name}', () async {
      final picker = _RecordingImagePicker(
        XFile.fromData(
          _pngBytes(),
          name: 'selected.png',
          mimeType: 'image/png',
        ),
      );
      final container = ProviderContainer.test(
        overrides: [
          imagePickerProvider.overrideWithValue(picker),
          blobApiClientProvider.overrideWithValue(
            BlobApiClient(Dio(BaseOptions(baseUrl: 'https://example.com'))),
          ),
        ],
      );
      addTearDown(container.dispose);
      Uint8List? preview;

      final prepared = await container
          .read(profileImagePickerProvider)
          .pickAndPrepare(
            source: source,
            onPreviewReady: (bytes) => preview = bytes,
          );

      expect(picker.lastSource, source);
      expect(preview, isNotEmpty);
      expect(prepared, isNotNull);
      expect(prepared!.mimeType, 'image/png');
      expect(prepared.width, 2);
      expect(prepared.height, 1);
    });
  }

  test('returns null when camera capture is cancelled', () async {
    final picker = _RecordingImagePicker(null);
    final container = ProviderContainer.test(
      overrides: [
        imagePickerProvider.overrideWithValue(picker),
        blobApiClientProvider.overrideWithValue(
          BlobApiClient(Dio(BaseOptions(baseUrl: 'https://example.com'))),
        ),
      ],
    );
    addTearDown(container.dispose);

    final prepared = await container
        .read(profileImagePickerProvider)
        .pickAndPrepare(
          source: ImageSource.camera,
          onPreviewReady: (_) => fail('cancelled capture has no preview'),
        );

    expect(prepared, isNull);
    expect(picker.lastSource, ImageSource.camera);
  });
}

class _RecordingImagePicker extends ImagePicker {
  _RecordingImagePicker(this.file);

  final XFile? file;
  ImageSource? lastSource;

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) async {
    lastSource = source;
    return file;
  }
}

Uint8List _pngBytes() => Uint8List.fromList(
  img.encodePng(img.Image(width: 2, height: 1)),
);
