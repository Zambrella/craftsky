import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:craftsky_app/feed/providers/image_post_submit_gate.dart';
import 'package:flutter_test/flutter_test.dart';

DraftImageState _image({
  required String id,
  required DraftImageLifecycle lifecycle,
  String altText = 'alt text',
}) {
  return DraftImageState(
    id: id,
    fileName: '$id.jpg',
    mimeType: 'image/jpeg',
    lifecycle: lifecycle,
    uploadProgress: lifecycle == DraftImageLifecycle.uploaded ? 1 : 0,
    uploaded: lifecycle == DraftImageLifecycle.uploaded
        ? const UploadedDraftImage(cid: 'cid', mime: 'image/jpeg', size: 1)
        : null,
    altText: altText,
  );
}

void main() {
  group('canSubmitImagePostDraft', () {
    test('rejects empty or over-limit post text', () {
      expect(canSubmitImagePostDraft(text: '', images: const []), isFalse);
      expect(
        canSubmitImagePostDraft(text: 'x' * 2001, images: const []),
        isFalse,
      );
    });

    test('allows valid text-only top-level post', () {
      expect(
        canSubmitImagePostDraft(text: 'valid text', images: const []),
        isTrue,
      );
    });

    test('rejects preparing, uploading, or failed images', () {
      expect(
        canSubmitImagePostDraft(
          text: 'valid text',
          images: [_image(id: 'a', lifecycle: DraftImageLifecycle.preparing)],
        ),
        isFalse,
      );
      expect(
        canSubmitImagePostDraft(
          text: 'valid text',
          images: [_image(id: 'a', lifecycle: DraftImageLifecycle.uploading)],
        ),
        isFalse,
      );
      expect(
        canSubmitImagePostDraft(
          text: 'valid text',
          images: [_image(id: 'a', lifecycle: DraftImageLifecycle.failed)],
        ),
        isFalse,
      );
    });

    test('rejects missing or over-length alt text', () {
      expect(
        canSubmitImagePostDraft(
          text: 'valid text',
          images: [
            _image(
              id: 'a',
              lifecycle: DraftImageLifecycle.uploaded,
              altText: '   ',
            ),
          ],
        ),
        isFalse,
      );
      expect(
        canSubmitImagePostDraft(
          text: 'valid text',
          images: [
            _image(
              id: 'a',
              lifecycle: DraftImageLifecycle.uploaded,
              altText: 'x' * 301,
            ),
          ],
        ),
        isFalse,
      );
    });

    test('allows valid uploaded images with valid alt text', () {
      expect(
        canSubmitImagePostDraft(
          text: 'valid text',
          images: [
            _image(
              id: 'a',
              lifecycle: DraftImageLifecycle.uploaded,
              altText: 'Blue shawl draped over blocking mats',
            ),
          ],
        ),
        isTrue,
      );
    });
  });
}
