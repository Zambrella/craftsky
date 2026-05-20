import 'dart:typed_data';

import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_service.dart';
import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

void main() {
  test(
    'pipeline strips metadata before uploader receives prepared image',
    () async {
      final source = img.Image(width: 2, height: 1);
      final originalBytes = Uint8List.fromList(img.encodeJpg(source));

      final picker = _FakePicker(
        images: [
          SelectedComposerImage(
            id: 'img-1',
            fileName: 'shawl.jpg',
            mimeType: 'image/jpeg',
            bytes: originalBytes,
            metadata: const {
              'gpsLatitude': '47.60',
              'gpsLongitude': '-122.33',
              'cameraMake': 'camera',
              'orientation': '1',
            },
          ),
        ],
      );
      final uploader = _RecordingUploader();
      final service = DefaultComposerImageService(
        picker: picker,
        preparer: const DefaultComposerImagePreparer(),
        uploader: uploader,
      );
      final controller = ImageDraftController();

      await service.addImages(controller);

      expect(uploader.uploaded, hasLength(1));
      final prepared = uploader.uploaded.single;
      expect(prepared.originalBytes, originalBytes.length);
      expect(prepared.preparedBytes, isNotEmpty);
      expect(prepared.strippedMetadata.containsKey('gpsLatitude'), isFalse);
      expect(prepared.strippedMetadata.containsKey('gpsLongitude'), isFalse);
      expect(prepared.strippedMetadata.containsKey('cameraMake'), isFalse);
      expect(prepared.strippedMetadata['orientation'], '1');

      expect(controller.images, hasLength(1));
      expect(controller.images.single.lifecycle, DraftImageLifecycle.uploaded);
    },
  );

  test(
    'pipeline rejects webp in production path when metadata cannot be proven stripped',
    () async {
      final source = img.Image(width: 2, height: 1);
      final originalBytes = Uint8List.fromList(img.encodePng(source));

      final picker = _FakePicker(
        images: [
          SelectedComposerImage(
            id: 'img-1',
            fileName: 'shawl.webp',
            mimeType: 'image/webp',
            bytes: originalBytes,
            metadata: const {},
          ),
        ],
      );
      final uploader = _RecordingUploader();
      final service = DefaultComposerImageService(
        picker: picker,
        preparer: const DefaultComposerImagePreparer(),
        uploader: uploader,
      );
      final controller = ImageDraftController();

      await service.addImages(controller);

      expect(uploader.uploaded, isEmpty);
      expect(controller.images, hasLength(1));
      expect(controller.images.single.lifecycle, DraftImageLifecycle.failed);
      expect(controller.images.single.errorMessage, 'Could not prepare image');
    },
  );
}

class _FakePicker implements ComposerImagePicker {
  const _FakePicker({required this.images});

  final List<SelectedComposerImage> images;

  @override
  Future<List<SelectedComposerImage>> pickImages({
    required int maxImages,
  }) async {
    return images.take(maxImages).toList();
  }
}

class _RecordingUploader implements ComposerImageUploader {
  final List<PreparedComposerImage> uploaded = [];

  @override
  Future<UploadedImageBlob> upload(
    PreparedComposerImage image, {
    ProgressCallback? onSendProgress,
  }) async {
    uploaded.add(image);
    onSendProgress?.call(
      image.preparedBytes.length,
      image.preparedBytes.length,
    );
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
