import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'profile editor picker retains its managed client during selection',
    () async {
      final selected = Completer<XFile?>();
      final opened = Completer<void>();
      var requests = 0;
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            expect(options.path, '/v1/blobs/images/authorization');
            requests++;
            handler.reject(DioException(requestOptions: options));
          },
        ),
      );
      final container = ProviderContainer.test(
        overrides: [
          dioProvider.overrideWithValue(dio),
          imagePickerProvider.overrideWithValue(
            _PendingImagePicker(selected, opened),
          ),
        ],
      );
      var preview = false;
      // Match the profile editor: read the persistent picker, with the real
      // managed blob client rather than an override that hides its lifetime.
      final operation = container
          .read(profileImagePickerProvider)
          .pickAndUpload(
            source: ImageSource.gallery,
            onPreviewReady: (_) => preview = true,
          );
      final assertion = expectLater(operation, throwsA(isA<ApiServerError>()));
      await opened.future;
      await container.pump();
      selected.complete(
        XFile.fromData(
          _pngBytes(),
          name: 'selected.png',
          mimeType: 'image/png',
        ),
      );
      await assertion;
      expect(preview, isTrue);
      expect(requests, 1);
    },
  );

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
  new(this.file);

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

class _PendingImagePicker extends ImagePicker {
  new(this.selected, this.opened);
  final Completer<XFile?> selected;
  final Completer<void> opened;

  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) {
    opened.complete();
    return selected.future;
  }
}
