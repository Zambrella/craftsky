import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ImageDraftController', () {
    test('transitions preparing -> uploading -> uploaded', () {
      final controller = ImageDraftController();

      controller.addDraftImage(
        const DraftImageInput(
          id: 'img-1',
          fileName: 'a.jpg',
          mimeType: 'image/jpeg',
        ),
      );
      expect(controller.images.single.lifecycle, DraftImageLifecycle.preparing);

      controller.markPrepared('img-1');
      expect(controller.images.single.lifecycle, DraftImageLifecycle.uploading);

      controller.markUploadProgress('img-1', 0.5);
      expect(controller.images.single.uploadProgress, 0.5);

      controller.markUploaded(
        'img-1',
        const UploadedDraftImage(
          cid: 'bafkimage1',
          mime: 'image/jpeg',
          size: 253496,
        ),
      );

      final image = controller.images.single;
      expect(image.lifecycle, DraftImageLifecycle.uploaded);
      expect(image.uploadProgress, 1);
      expect(image.uploaded?.cid, 'bafkimage1');
    });

    test(
      'moves to failed on preparation or upload failures and supports retry',
      () {
        final controller = ImageDraftController();
        controller.addDraftImage(
          const DraftImageInput(
            id: 'img-1',
            fileName: 'a.jpg',
            mimeType: 'image/jpeg',
          ),
        );

        controller.markPreparationFailed('img-1', 'strip failed');
        expect(controller.images.single.lifecycle, DraftImageLifecycle.failed);
        expect(controller.images.single.errorMessage, 'strip failed');

        controller.retry('img-1');
        expect(
          controller.images.single.lifecycle,
          DraftImageLifecycle.preparing,
        );
        expect(controller.images.single.errorMessage, isNull);

        controller.markPrepared('img-1');
        controller.markUploadFailed('img-1', 'upload failed');
        expect(controller.images.single.lifecycle, DraftImageLifecycle.failed);
        expect(controller.images.single.errorMessage, 'upload failed');
      },
    );

    test('remove excludes image even if delayed upload completion arrives', () {
      final controller = ImageDraftController();
      controller.addDraftImage(
        const DraftImageInput(
          id: 'img-1',
          fileName: 'a.jpg',
          mimeType: 'image/jpeg',
        ),
      );
      controller.markPrepared('img-1');

      controller.remove('img-1');
      expect(controller.images, isEmpty);

      controller.markUploaded(
        'img-1',
        const UploadedDraftImage(
          cid: 'late-cid',
          mime: 'image/jpeg',
          size: 100,
        ),
      );
      expect(controller.images, isEmpty);
    });
  });
}
