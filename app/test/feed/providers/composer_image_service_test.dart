import 'dart:typed_data';

import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_service.dart';
import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:dio/dio.dart';
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

  test(
    'service keeps draft at max when partial selection exceeds remaining slots',
    () async {
      final controller = ImageDraftController();
      for (var i = 1; i <= 3; i++) {
        final id = 'existing-$i';
        controller.addDraftImage(
          DraftImageInput(
            id: id,
            fileName: 'existing-$i.jpg',
            mimeType: 'image/jpeg',
          ),
        );
        controller.markPrepared(id);
        controller.markUploaded(
          id,
          UploadedDraftImage(
            cid: 'cid-existing-$i',
            mime: 'image/jpeg',
            size: 128,
          ),
        );
      }

      final service = DefaultComposerImageService(
        picker: _FakeSelectedImagePicker(
          images: [
            _selected('new-1', 'new-1.jpg'),
            _selected('new-2', 'new-2.jpg'),
            _selected('new-3', 'new-3.jpg'),
          ],
        ),
        preparer: const _FakePreparer(),
        uploader: _RecordingUploader(),
      );

      await expectLater(
        service.addImages(controller),
        throwsA(isA<ImageSelectionLimitExceededException>()),
      );

      expect(controller.images, hasLength(4));
      expect(
        controller.images.any((image) => image.fileName == 'new-2.jpg'),
        isFalse,
      );
      expect(
        controller.images.any((image) => image.fileName == 'new-3.jpg'),
        isFalse,
      );
    },
  );
}

SelectedComposerImage _selected(String id, String fileName) {
  return SelectedComposerImage(
    id: id,
    fileName: fileName,
    mimeType: 'image/jpeg',
    bytes: Uint8List.fromList([1, 2, 3]),
    metadata: const {},
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

class _FakeSelectedImagePicker implements ComposerImagePicker {
  const _FakeSelectedImagePicker({required this.images});

  final List<SelectedComposerImage> images;

  @override
  Future<List<SelectedComposerImage>> pickImages({
    required int maxImages,
  }) async {
    return images;
  }
}

class _FakePreparer implements ComposerImagePreparer {
  const _FakePreparer();

  @override
  Future<PreparedComposerImage> prepare(SelectedComposerImage image) async {
    return PreparedComposerImage(
      id: image.id,
      fileName: image.fileName,
      mimeType: image.mimeType,
      originalBytes: image.bytes.length,
      preparedBytes: image.bytes,
      aspectRatio: null,
      strippedMetadata: const {},
    );
  }
}

class _RecordingUploader implements ComposerImageUploader {
  @override
  Future<UploadedImageBlob> upload(
    PreparedComposerImage image, {
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    return const UploadedImageBlob(
      blob: UploadedBlob(
        type: 'blob',
        ref: UploadedBlobRef(link: 'bafkimg1'),
        mimeType: 'image/jpeg',
        size: 128,
      ),
      cid: 'bafkimg1',
      mime: 'image/jpeg',
      size: 128,
    );
  }
}
