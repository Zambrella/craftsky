import 'dart:typed_data';

import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Composer image state string output', () {
    test('redacts all unpublished image details', () {
      final draft = ComposerImageDraft(
        id: 'image-1',
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        altText: 'private-alt-canary',
        phase: const ImageUploading(
          TransferBytes(sent: 1, sendTotal: 4, received: 0, receiveTotal: 0),
        ),
        previewBytes: Uint8List.fromList([255, 216, 255, 224]),
      );

      final text = draft.toString();

      expect(text, 'ComposerImageDraft(<redacted>)');
      expect(text, isNot(contains('project.jpg')));
      expect(text, isNot(contains('private-alt-canary')));
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

      expect(text, contains('ComposerImageDraft(<redacted>)'));
      expect(text, isNot(contains('project.jpg')));
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

  group('Composer image draft readiness', () {
    test('requires every retained image to be locally ready', () {
      ComposerImageDraft image(ComposerImagePhase phase) => ComposerImageDraft(
        id: 'image-1',
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        altText: '',
        phase: phase,
      );

      final ready = ComposerImagesState(
        images: [
          image(
            ImageReady(
              bytes: Uint8List.fromList([1, 2, 3]),
              mimeType: 'image/jpeg',
              width: 3,
              height: 2,
              sha256: 'digest',
            ),
          ),
        ],
      );
      final preparing = ComposerImagesState(
        images: [image(const ImagePreparing())],
      );
      final failed = ComposerImagesState(
        images: [image(const ImageFailed(ImagePreparationFailed()))],
      );

      expect(ready.canSaveDraftMedia(), isTrue);
      expect(preparing.canSaveDraftMedia(), isFalse);
      expect(failed.canSaveDraftMedia(), isFalse);
    });
  });
}
