import 'dart:async';
import 'dart:math';
import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
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

  test(
    'compresses a large profile photo below the PDS profile limit',
    () async {
      final image = img.Image(width: 1200, height: 1200);
      final random = Random(42);
      for (final pixel in image) {
        pixel.setRgb(
          random.nextInt(256),
          random.nextInt(256),
          random.nextInt(256),
        );
      }
      final original = Uint8List.fromList(img.encodeJpg(image, quality: 95));
      expect(original.length, greaterThan(1000000));
      final container = ProviderContainer.test(
        overrides: [
          imagePickerProvider.overrideWithValue(
            _RecordingImagePicker(
              XFile.fromData(
                original,
                name: 'photo.jpg',
                mimeType: 'image/jpeg',
              ),
            ),
          ),
          blobApiClientProvider.overrideWithValue(BlobApiClient(Dio())),
        ],
      );
      addTearDown(container.dispose);
      final prepared = await container
          .read(profileImagePickerProvider)
          .pickAndPrepare(source: ImageSource.gallery, onPreviewReady: (_) {});
      expect(prepared, isNotNull);
      expect(prepared!.bytes.length, lessThanOrEqualTo(1000000));
      expect(prepared.mimeType, 'image/jpeg');
      expect(img.decodeImage(prepared.bytes), isNotNull);
    },
  );

  test('resizes a 48 MP profile photo below the upload limit', () async {
    final original = Uint8List.fromList(
      img.encodeJpg(img.Image(width: 8000, height: 6000)),
    );
    final container = ProviderContainer.test(
      overrides: [
        imagePickerProvider.overrideWithValue(
          _RecordingImagePicker(
            XFile.fromData(original, name: 'photo.jpg', mimeType: 'image/jpeg'),
          ),
        ),
        blobApiClientProvider.overrideWithValue(BlobApiClient(Dio())),
      ],
    );
    final prepared = await container
        .read(profileImagePickerProvider)
        .pickAndPrepare(
          source: ImageSource.gallery,
          onPreviewReady: (_) {},
        );
    expect(prepared, isNotNull);
    expect(prepared!.bytes.length, lessThanOrEqualTo(1000000));
    final decoded = img.decodeJpg(prepared.bytes)!;
    expect(decoded.width, 4000);
    expect(decoded.height, 3000);
  });

  test('accepts 48 MP source geometry and files up to 100 MB', () async {
    final container = ProviderContainer.test();
    final media = container.read(profileImageMediaServiceProvider);
    final inspected = await media
        .inspectImage(
          bytes: _jpegHeader(width: 8000, height: 6000),
          fileName: 'photo.jpg',
          mimeType: 'image/jpeg',
        )
        .future;
    expect(inspected.width, 8000);
    expect(inspected.height, 6000);
    expect(
      media
          .validateOriginalImage(
            sizeBytes: 100000000,
            fileName: 'photo.jpg',
            mimeType: 'image/jpeg',
            headerBytes: _jpegHeader(width: 8000, height: 6000),
          )
          .canPrepare,
      isTrue,
    );
    expect(
      media
          .validateOriginalImage(
            sizeBytes: 100000001,
            fileName: 'photo.jpg',
            mimeType: 'image/jpeg',
            headerBytes: _jpegHeader(width: 8000, height: 6000),
          )
          .canPrepare,
      isFalse,
    );
  });

  test('rejects over 48 MP before decoding or contacting the server', () async {
    final container = ProviderContainer.test(
      overrides: [
        imagePickerProvider.overrideWithValue(
          _RecordingImagePicker(
            XFile.fromData(
              _jpegHeader(width: 8000, height: 6001),
              name: 'photo.jpg',
              mimeType: 'image/jpeg',
            ),
          ),
        ),
        blobApiClientProvider.overrideWithValue(BlobApiClient(Dio())),
      ],
    );
    await expectLater(
      container
          .read(profileImagePickerProvider)
          .pickAndUpload(source: ImageSource.gallery, onPreviewReady: (_) {}),
      throwsA(
        isA<ProfileImagePickException>().having(
          (error) => error.reason,
          'reason',
          ProfileImageFailure.tooManyPixels,
        ),
      ),
    );
  });

  test('reports a file over 100 MB before reading the full image', () async {
    final container = ProviderContainer.test(
      overrides: [
        imagePickerProvider.overrideWithValue(
          _RecordingImagePicker(_OversizedImageFile()),
        ),
        blobApiClientProvider.overrideWithValue(BlobApiClient(Dio())),
      ],
    );
    await expectLater(
      container
          .read(profileImagePickerProvider)
          .pickAndPrepare(
            source: ImageSource.gallery,
            onPreviewReady: (_) => fail('oversized file must not be previewed'),
          ),
      throwsA(
        isA<ProfileImagePickException>().having(
          (error) => error.reason,
          'reason',
          ProfileImageFailure.fileTooLarge,
        ),
      ),
    );
  });

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

Uint8List _pngBytes() =>
    Uint8List.fromList(img.encodePng(img.Image(width: 2, height: 1)));

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

// Only geometry is needed: oversized input must fail before full decoding.
Uint8List _jpegHeader({required int width, required int height}) =>
    Uint8List.fromList([
      0xff,
      0xd8,
      0xff,
      0xc0,
      0x00,
      0x11,
      0x08,
      height >> 8,
      height & 0xff,
      width >> 8,
      width & 0xff,
      0x03,
      0x01,
      0x11,
      0x00,
      0x02,
      0x11,
      0x00,
      0x03,
      0x11,
      0x00,
      0xff,
      0xd9,
    ]);

class _OversizedImageFile extends XFile {
  new() : super.fromData(_pngBytes(), name: 'photo.png', mimeType: 'image/png');

  @override
  Future<int> length() async => 100000001;
}
