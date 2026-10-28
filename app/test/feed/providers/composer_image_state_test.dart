import 'dart:typed_data';

import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-004 published images submit retained indices without bytes or reupload',
    () {
      const state = ComposerImagesState(
        images: [
          ComposerImageDraft(
            id: 'published:1',
            fileName: 'Published image',
            mimeType: 'image/jpeg',
            altText: 'Kept image',
            phase: PublishedImageReady(
              existingIndex: 1,
              url: 'https://cdn.example/image.jpeg',
            ),
          ),
        ],
      );
      expect(state.isSubmitReady, isTrue);
      expect(state.canSaveDraftMedia(), isFalse);
      expect(state.toCreatePostImages(), isNull);
      expect(state.toPostEditImages(), [
        {'existingIndex': 1, 'alt': 'Kept image'},
      ]);
    },
  );
  test('image alt permits 1000 graphemes and enforces 10000 UTF-8 bytes', () {
    for (final (alt, valid) in [
      ('a' * 1000, true),
      ('a' * 1001, false),
      ('é' * 1000, true),
      ('é' * 1001, false),
      ('👨‍👩‍👧‍👦' * 400, true),
      ('👨‍👩‍👧‍👦' * 401, false),
    ]) {
      final state = ComposerImagesState(
        images: [
          ComposerImageDraft(
            id: 'image-1',
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
            altText: alt,
            phase: const ImageUploaded(
              UploadedDraftImage(
                cid: 'bafkimage',
                mime: 'image/jpeg',
                size: 123,
              ),
            ),
          ),
        ],
      );
      expect(state.canSubmitImages(), valid);
    }
  });

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

    test('separates upload readiness from alt text validation', () {
      final state = ComposerImagesState(
        images: [
          ComposerImageDraft(
            id: 'image-1',
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
            altText: List.filled(1001, 'x').join(),
            phase: const ImageUploaded(
              UploadedDraftImage(
                cid: 'bafkimage',
                mime: 'image/jpeg',
                size: 123,
              ),
            ),
          ),
        ],
      );

      expect(state.isSubmitReady, isTrue);
      expect(state.canSubmitImages(), isFalse);
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

  group('Composer image failures', () {
    test('source geometry failures are non-retryable and round-trip', () {
      const failure = ImageSourceDimensionsTooLarge();

      expect(failure.canRetry, isFalse);
      expect(failure.message, 'This image is too large to process');
      expect(
        ImageSourceDimensionsTooLargeMapper.fromJson(failure.toJson()),
        failure,
      );
    });

    test('preparation failures expose explicit retry eligibility', () {
      expect(const ImagePreparationFailed().canRetry, isFalse);
      expect(
        const ImagePreparationFailed(retryable: true).canRetry,
        isTrue,
      );
    });
  });
}
