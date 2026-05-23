import 'dart:typed_data';

import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Composer image state string output', () {
    test('summarizes draft preview bytes', () {
      final draft = ComposerImageDraft(
        id: 'image-1',
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        altText: '',
        phase: const ImageUploading(
          TransferBytes(sent: 1, sendTotal: 4, received: 0, receiveTotal: 0),
        ),
        previewBytes: Uint8List.fromList([255, 216, 255, 224]),
      );

      final text = draft.toString();

      expect(text, contains('previewBytes: 4 bytes'));
      expect(text, isNot(contains('255, 216, 255, 224')));
    });

    test('uses summarized draft output in composer state', () {
      final state = ComposerImagesState(
        images: [
          ComposerImageDraft(
            id: 'image-1',
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
            altText: '',
            phase: const ImageQueued(),
            previewBytes: Uint8List.fromList([255, 216, 255, 224]),
          ),
        ],
        notice: const UnsupportedImagesNotice(id: 1, count: 2),
      );

      final text = state.toString();

      expect(text, contains('previewBytes: 4 bytes'));
      expect(text, contains('UnsupportedImagesNotice'));
      expect(text, isNot(contains('255, 216, 255, 224')));
    });
  });

  group('Composer image submission', () {
    test('allows uploaded images without alt text', () {
      const state = ComposerImagesState(
        images: [
          ComposerImageDraft(
            id: 'image-1',
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
            altText: '',
            phase: ImageUploaded(
              UploadedDraftImage(
                cid: 'bafkimage',
                mime: 'image/jpeg',
                size: 123,
              ),
            ),
          ),
        ],
      );

      expect(state.canSubmitImages(), isTrue);
      expect(state.hasImagesMissingAltText, isTrue);
      expect(state.toCreatePostImages(), [
        const CreatePostImage(
          blob: CreatePostBlob(
            ref: CreatePostBlobRef(link: 'bafkimage'),
            mimeType: 'image/jpeg',
            size: 123,
          ),
        ),
      ]);
    });
  });
}
