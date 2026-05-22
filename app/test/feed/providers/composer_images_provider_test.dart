import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';

void main() {
  group('ComposerImages', () {
    test('surfaces picker failures without changing the draft', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(() async => throw Exception('permission denied')),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      final state = container.read(composerImagesProvider('composer'));
      expect(state.images, isEmpty);
      expect(state.notice, isA<ImagePickerFailedNotice>());
    });

    test('rejects WebP selections before adding draft images', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              Uint8List.fromList([
                0x52,
                0x49,
                0x46,
                0x46,
                0x00,
                0x00,
                0x00,
                0x00,
                0x57,
                0x45,
                0x42,
                0x50,
              ]),
              name: 'project.webp',
              mimeType: 'image/webp',
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      final state = container.read(composerImagesProvider('composer'));
      expect(state.images, isEmpty);
      expect(state.notice, isA<UnsupportedImagesNotice>());
    });

    test('fails oversized originals before reading full bytes', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              Uint8List.fromList(List<int>.filled(16, 0xff)),
              name: 'large.jpg',
              mimeType: 'image/jpeg',
              length: mediaConfig.maxImageBytes + 1,
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) =>
            state.images.length == 1 &&
            state.images.single.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect(phase, isA<ImageFailed>());
      expect((phase as ImageFailed).failure, isA<ImageTooLarge>());
    });

    test(
      'uploads accepted images and stores preview and aspect ratio',
      () async {
        final originalBytes = _jpegBytes(width: 3, height: 2);
        final api = _FakePostApiClient();
        final container = _containerWithPicker(
          _FakeImagePicker(
            () async => [
              XFile.fromData(
                originalBytes,
                path: '/tmp/PROJECT.JPG',
              ),
            ],
          ),
          api: api,
        );
        addTearDown(container.dispose);
        final sub = _listenComposer(container);
        addTearDown(sub.close);

        await container
            .read(composerImagesProvider('composer').notifier)
            .addImages();
        final state = await _waitForState(
          container,
          (state) => state.images.singleOrNull?.phase is ImageUploaded,
        );

        final image = state.images.single;
        expect(image.fileName, 'PROJECT.JPG');
        expect(image.mimeType, 'image/jpeg');
        expect(image.previewBytes, originalBytes);
        expect(api.uploadCount, 1);
        expect(api.lastMimeType, 'image/jpeg');
        expect(api.lastBytes, isNotEmpty);

        final uploaded = (image.phase as ImageUploaded).uploaded;
        expect(uploaded.cid, _testCid);
        expect(uploaded.mime, 'image/jpeg');
        expect(uploaded.aspectRatio?.width, 3);
        expect(uploaded.aspectRatio?.height, 2);
      },
    );

    test('reports upload progress while an upload is in flight', () async {
      final uploadCompleter = Completer<UploadedImageBlob>();
      final api = _FakePostApiClient(
        uploadHandler:
            ({
              required bytes,
              required mimeType,
              onSendProgress,
              onReceiveProgress,
              cancelToken,
            }) {
              onSendProgress?.call(5, 10);
              onReceiveProgress?.call(1, 5);
              return uploadCompleter.future;
            },
      );
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _pngBytes(width: 1, height: 1),
              name: 'project.png',
              mimeType: 'image/png',
            ),
          ],
        ),
        api: api,
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final uploading = await _waitForState(
        container,
        (state) {
          final phase = state.images.singleOrNull?.phase;
          final progress = phase is ImageUploading ? phase.progress : null;
          return progress is TransferBytes && progress.received == 1;
        },
      );

      final progress =
          (uploading.images.single.phase as ImageUploading).progress;
      expect(progress, isA<TransferBytes>());
      expect((progress as TransferBytes).sent, 5);
      expect(progress.sendTotal, 10);
      expect(progress.received, 1);
      expect(progress.receiveTotal, 5);
      expect(progress.indicatorValue, closeTo(0.4, 0.0001));

      uploadCompleter.complete(
        _uploadedBlob(mimeType: api.lastMimeType!, size: api.lastBytes!.length),
      );
      await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageUploaded,
      );
    });

    test('throttles rapid upload progress updates', () async {
      final uploadCompleter = Completer<UploadedImageBlob>();
      final progressUpdates = <TransferBytes>[];
      final api = _FakePostApiClient(
        uploadHandler:
            ({
              required bytes,
              required mimeType,
              onSendProgress,
              onReceiveProgress,
              cancelToken,
            }) {
              for (var sent = 1; sent <= 100; sent += 1) {
                onSendProgress?.call(sent, 100);
              }
              return uploadCompleter.future;
            },
      );
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _pngBytes(width: 1, height: 1),
              name: 'project.png',
              mimeType: 'image/png',
            ),
          ],
        ),
        api: api,
      );
      addTearDown(container.dispose);
      final provider = composerImagesProvider('composer');
      final sub = container.listen(provider, (_, state) {
        final phase = state.images.singleOrNull?.phase;
        final progress = phase is ImageUploading ? phase.progress : null;
        if (progress is TransferBytes) progressUpdates.add(progress);
      }, fireImmediately: true);
      addTearDown(sub.close);

      await container.read(provider.notifier).addImages();
      await _waitForState(
        container,
        (state) => progressUpdates.length == 2,
      );

      expect(progressUpdates.map((progress) => progress.sent), [1, 100]);

      uploadCompleter.complete(
        _uploadedBlob(mimeType: api.lastMimeType!, size: api.lastBytes!.length),
      );
      await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageUploaded,
      );
    });

    test('maps unsupported original headers to image failure', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _jpegBytes(width: 1, height: 1),
              name: 'spoofed.jpg',
              mimeType: 'image/jpeg',
            ),
          ],
        ),
        media: const _UnsupportedOriginalMediaService(),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect((phase as ImageFailed).failure, isA<UnsupportedImageType>());
    });

    test('maps corrupt accepted bytes to preparation failure', () async {
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              Uint8List.fromList([1, 2, 3, 4]),
              name: 'corrupt.jpg',
              mimeType: 'image/jpeg',
            ),
          ],
        ),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect((phase as ImageFailed).failure, isA<ImagePreparationFailed>());
    });

    test('maps prepared size failures before upload starts', () async {
      final api = _FakePostApiClient();
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _pngBytes(width: 1, height: 1),
              name: 'project.png',
              mimeType: 'image/png',
            ),
          ],
        ),
        api: api,
        media: const _PreparedTooLargeMediaService(),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final state = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );

      final phase = state.images.single.phase;
      expect((phase as ImageFailed).failure, isA<ImageTooLarge>());
      expect(api.uploadCount, 0);
    });

    test('retries upload failures using retained preview bytes', () async {
      var failUpload = true;
      final api = _FakePostApiClient(
        uploadHandler:
            ({
              required bytes,
              required mimeType,
              onSendProgress,
              onReceiveProgress,
              cancelToken,
            }) async {
              if (failUpload) throw const ApiNetworkError('offline');
              return _uploadedBlob(mimeType: mimeType, size: bytes.length);
            },
      );
      final container = _containerWithPicker(
        _FakeImagePicker(
          () async => [
            XFile.fromData(
              _jpegBytes(width: 1, height: 1),
              name: 'project.jpg',
              mimeType: 'image/jpeg',
            ),
          ],
        ),
        api: api,
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();
      final failed = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageFailed,
      );
      final imageId = failed.images.single.id;
      final failure = (failed.images.single.phase as ImageFailed).failure;
      expect(failure, isA<ImageUploadFailed>());
      expect(failure.canRetry, isTrue);
      expect(failed.images.single.previewBytes, isNotNull);

      failUpload = false;
      container
          .read(composerImagesProvider('composer').notifier)
          .retry(imageId);
      final uploaded = await _waitForState(
        container,
        (state) => state.images.singleOrNull?.phase is ImageUploaded,
      );

      expect(api.uploadCount, 2);
      expect(uploaded.images.single.id, imageId);
      expect(uploaded.images.single.phase, isA<ImageUploaded>());
    });

    test(
      'removing an uploading image cancels and ignores late completion',
      () async {
        final uploadCompleter = Completer<UploadedImageBlob>();
        final api = _FakePostApiClient(
          uploadHandler:
              ({
                required bytes,
                required mimeType,
                onSendProgress,
                onReceiveProgress,
                cancelToken,
              }) {
                return uploadCompleter.future;
              },
        );
        final container = _containerWithPicker(
          _FakeImagePicker(
            () async => [
              XFile.fromData(
                _pngBytes(width: 1, height: 1),
                name: 'project.png',
                mimeType: 'image/png',
              ),
            ],
          ),
          api: api,
        );
        addTearDown(container.dispose);
        final sub = _listenComposer(container);
        addTearDown(sub.close);

        await container
            .read(composerImagesProvider('composer').notifier)
            .addImages();
        final uploading = await _waitForState(
          container,
          (state) => state.images.singleOrNull?.phase is ImageUploading,
        );
        final imageId = uploading.images.single.id;

        container
            .read(composerImagesProvider('composer').notifier)
            .remove(imageId);

        expect(
          container.read(composerImagesProvider('composer')).images,
          isEmpty,
        );
        expect(api.lastCancelToken?.isCancelled, isTrue);

        uploadCompleter.complete(
          _uploadedBlob(
            mimeType: api.lastMimeType!,
            size: api.lastBytes!.length,
          ),
        );
        await Future<void>.delayed(const Duration(milliseconds: 20));

        expect(
          container.read(composerImagesProvider('composer')).images,
          isEmpty,
        );
      },
    );

    test('emits limit notice without opening picker when full', () async {
      final picker = _FakeImagePicker(
        () async => [
          XFile.fromData(
            _pngBytes(width: 1, height: 1),
            name: 'project.png',
            mimeType: 'image/png',
          ),
        ],
      );
      final container = _containerWithPicker(
        picker,
        media: const _NoSlotsMediaService(),
      );
      addTearDown(container.dispose);
      final sub = _listenComposer(container);
      addTearDown(sub.close);

      await container
          .read(composerImagesProvider('composer').notifier)
          .addImages();

      final notice = container.read(composerImagesProvider('composer')).notice;
      expect(notice, isA<ImageSelectionLimitNotice>());
      expect((notice as ImageSelectionLimitNotice).maxImages, 0);
      expect(notice.acceptedCount, 0);
      expect(picker.pickCount, 0);
    });
  });
}

ProviderSubscription<ComposerImagesState> _listenComposer(
  ProviderContainer container,
) {
  return container.listen(
    composerImagesProvider('composer'),
    (_, _) {},
    fireImmediately: true,
  );
}

ProviderContainer _containerWithPicker(
  ImagePicker picker, {
  PostApiClient? api,
  ComposerImageMediaService? media,
}) {
  return ProviderContainer.test(
    overrides: [
      imagePickerProvider.overrideWithValue(picker),
      composerImageMediaServiceProvider.overrideWithValue(
        media ?? const ComposerImageMediaService(),
      ),
      postApiClientProvider.overrideWith(
        (ref) =>
            api ??
            PostApiClient(Dio(BaseOptions(baseUrl: 'https://example.com'))),
      ),
    ],
  );
}

Future<ComposerImagesState> _waitForState(
  ProviderContainer container,
  bool Function(ComposerImagesState state) predicate,
) async {
  final provider = composerImagesProvider('composer');
  for (var i = 0; i < 50; i += 1) {
    final state = container.read(provider);
    if (predicate(state)) return state;
    await Future<void>.delayed(const Duration(milliseconds: 10));
  }
  fail(
    'Timed out waiting for composer image state: ${container.read(provider)}',
  );
}

class _FakeImagePicker extends ImagePicker {
  _FakeImagePicker(this._pick);

  final Future<List<XFile>> Function() _pick;
  var pickCount = 0;

  @override
  Future<List<XFile>> pickMultiImage({
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    int? limit,
    bool requestFullMetadata = true,
  }) {
    pickCount += 1;
    return _pick();
  }
}

typedef _UploadHandler =
    Future<UploadedImageBlob> Function({
      required List<int> bytes,
      required String mimeType,
      ProgressCallback? onSendProgress,
      ProgressCallback? onReceiveProgress,
      CancelToken? cancelToken,
    });

class _FakePostApiClient extends PostApiClient {
  _FakePostApiClient({_UploadHandler? uploadHandler})
    : _uploadHandler = uploadHandler,
      super(Dio(BaseOptions(baseUrl: 'https://example.com')));

  final _UploadHandler? _uploadHandler;
  var uploadCount = 0;
  List<int>? lastBytes;
  String? lastMimeType;
  CancelToken? lastCancelToken;

  @override
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    uploadCount += 1;
    lastBytes = bytes;
    lastMimeType = mimeType;
    lastCancelToken = cancelToken;

    final handler = _uploadHandler;
    if (handler != null) {
      return handler(
        bytes: bytes,
        mimeType: mimeType,
        onSendProgress: onSendProgress,
        onReceiveProgress: onReceiveProgress,
        cancelToken: cancelToken,
      );
    }

    onSendProgress?.call(bytes.length, bytes.length);
    return _uploadedBlob(mimeType: mimeType, size: bytes.length);
  }
}

class _PreparedTooLargeMediaService extends ComposerImageMediaService {
  const _PreparedTooLargeMediaService();

  @override
  PreparedUploadValidationResult validatePreparedUploadBytes({
    required int originalBytes,
    required int preparedBytes,
  }) {
    return const PreparedUploadValidationResult(
      canUpload: false,
      rejectedReason: PreparedUploadRejection.tooLarge,
    );
  }
}

class _UnsupportedOriginalMediaService extends ComposerImageMediaService {
  const _UnsupportedOriginalMediaService();

  @override
  OriginalImageValidationResult validateOriginalImage({
    required int sizeBytes,
    required String fileName,
    required String mimeType,
    required Uint8List headerBytes,
  }) {
    return const OriginalImageValidationResult(
      canPrepare: false,
      rejectedReason: OriginalImageRejection.unsupportedType,
    );
  }
}

class _NoSlotsMediaService extends ComposerImageMediaService {
  const _NoSlotsMediaService();

  @override
  int get maxImages => 0;
}

const _testCid = 'bafkreicomposerimagetest';

UploadedImageBlob _uploadedBlob({
  String cid = _testCid,
  required String mimeType,
  required int size,
}) {
  return UploadedImageBlob(
    blob: UploadedBlob(
      type: 'blob',
      ref: UploadedBlobRef(link: cid),
      mimeType: mimeType,
      size: size,
    ),
    cid: cid,
    mime: mimeType,
    size: size,
  );
}

Uint8List _jpegBytes({required int width, required int height}) {
  return Uint8List.fromList(
    img.encodeJpg(img.Image(width: width, height: height)),
  );
}

Uint8List _pngBytes({required int width, required int height}) {
  return Uint8List.fromList(
    img.encodePng(img.Image(width: width, height: height)),
  );
}
