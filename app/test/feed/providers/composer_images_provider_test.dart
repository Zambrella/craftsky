import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';

void main() {
  test('seeds locally saved ready and unavailable draft images', () async {
    const available = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000002',
      storageRevision: '00000000-0000-4000-8000-000000000012',
      storageFileName: 'available.png',
      displayFileName: 'available.png',
      mimeType: 'image/png',
      byteLength: 3,
      sha256:
          '039058c6f2c0cb492c533b0a4d14ef77'
          'cc0f78abccced5287d84a1a2011cfb81',
      width: 1,
      height: 1,
      altText: 'Available alt',
      order: 0,
    );
    const missing = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000003',
      storageRevision: '00000000-0000-4000-8000-000000000013',
      storageFileName: 'missing.png',
      displayFileName: 'missing.png',
      mimeType: 'image/png',
      byteLength: 3,
      sha256:
          '039058c6f2c0cb492c533b0a4d14ef77'
          'cc0f78abccced5287d84a1a2011cfb81',
      width: 1,
      height: 1,
      altText: 'Missing alt',
      order: 1,
      availability: DraftMediaAvailability.unavailable,
    );
    final draft = LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000001',
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.standard,
      createdAt: DateTime.utc(2026),
      updatedAt: DateTime.utc(2026),
      content: const StandardDraftContent(text: '', languages: ['en']),
      schedule: const DraftScheduleIntent.now(),
      media: const [available, missing],
    );
    final seed = LocalPostDraftSeed(
      draft: draft,
      media: [
        HydratedDraftMedia(
          descriptor: available,
          bytes: Uint8List.fromList([1, 2, 3]),
        ),
        const HydratedDraftMedia(descriptor: missing, bytes: null),
      ],
    );
    final container = ProviderContainer.test();
    addTearDown(container.dispose);

    container
        .read(composerImagesProvider('draft-composer').notifier)
        .seedLocalDraft(seed);
    final state = container.read(composerImagesProvider('draft-composer'));

    expect(state.images.first.phase, isA<ImageReady>());
    expect((state.images.first.phase as ImageReady).storedOrigin, isNotNull);
    expect(state.images.last.phase, isA<ImageUnavailable>());
    expect(state.canSubmitImages(), isFalse);
    expect(state.canSaveDraftMedia(), isFalse);
  });
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
      'prepares accepted images locally without uploading',
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
          (state) => state.images.singleOrNull?.phase is ImageReady,
        );

        final image = state.images.single;
        expect(image.fileName, 'PROJECT.JPG');
        expect(image.mimeType, 'image/jpeg');
        expect(image.previewBytes, isNotEmpty);
        expect(image.previewAspectRatio?.width, 3);
        expect(image.previewAspectRatio?.height, 2);
        expect(api.uploadCount, 0);

        final ready = image.phase as ImageReady;
        expect(ready.bytes, image.previewBytes);
        expect(ready.mimeType, 'image/jpeg');
        expect(ready.width, 3);
        expect(ready.height, 2);
        expect(ready.sha256, isNotEmpty);
      },
    );

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
      expect(
        notice,
        isA<ImageSelectionLimitNotice>()
            .having((notice) => notice.maxImages, 'maxImages', 0)
            .having((notice) => notice.acceptedCount, 'acceptedCount', 0),
      );
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
  int pickCount = 0;

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

class _FakePostApiClient extends PostApiClient {
  _FakePostApiClient()
    : _uploadHandler = null, super(Dio(BaseOptions(baseUrl: 'https://example.com')));

  final Future<UploadedImageBlob> Function({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  })?
  _uploadHandler;
  int uploadCount = 0;
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
  required String mimeType,
  required int size,
  String cid = _testCid,
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
