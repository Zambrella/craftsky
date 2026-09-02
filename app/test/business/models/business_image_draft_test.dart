import 'dart:typed_data';

import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 unchanged image reconstructs exact mutation metadata only', () {
    final image = BusinessImageView(
      cid: 'bafy-image',
      mime: 'image/webp',
      size: 123,
      alt: 'Woven cloth',
      thumb: 'https://cdn.example/thumb',
      fullsize: 'https://cdn.example/full',
      aspectRatio: BusinessImageAspectRatio(width: 4, height: 3),
    );

    expect(ExistingBusinessImageDraft(image).toJson(), {
      'image': {
        r'$type': 'blob',
        'ref': {r'$link': 'bafy-image'},
        'mimeType': 'image/webp',
        'size': 123,
      },
      'alt': 'Woven cloth',
      'aspectRatio': {'width': 4, 'height': 3},
    });
  });

  test(
    'UT-014 image states distinguish missing, existing, and replacement',
    () {
      const missing = MissingBusinessImageDraft();
      const uploaded = UploadedBusinessImageDraft(
        cid: 'bafy-new',
        mime: 'image/png',
        size: 99,
        alt: 'New cloth',
      );

      expect(missing.toJson(), isNull);
      expect(uploaded.toJson(), {
        'image': {
          r'$type': 'blob',
          'ref': {r'$link': 'bafy-new'},
          'mimeType': 'image/png',
          'size': 99,
        },
        'alt': 'New cloth',
      });
    },
  );

  test('UT-014 alt text uses the business 1000-character limit', () {
    expect(
      const UploadedBusinessImageDraft(
        cid: 'bafy-new',
        mime: 'image/png',
        size: 99,
        alt: 'valid',
      ).isValid,
      isTrue,
    );
    expect(
      UploadedBusinessImageDraft(
        cid: 'bafy-new',
        mime: 'image/png',
        size: 99,
        alt: List.filled(1001, 'a').join(),
      ).isValid,
      isFalse,
    );
  });

  test('UT-014 local preview never enters reconstructed mutation JSON', () {
    final previewBytes = Uint8List.fromList([1, 2, 3]);
    final image = BusinessImageView.localPreview(
      cid: 'bafy-local',
      mime: 'image/png',
      size: 3,
      alt: 'Local preview',
      previewBytes: previewBytes,
    );

    expect(image.previewBytes, same(previewBytes));
    expect(ExistingBusinessImageDraft(image).toJson(), {
      'image': {
        r'$type': 'blob',
        'ref': {r'$link': 'bafy-local'},
        'mimeType': 'image/png',
        'size': 3,
      },
      'alt': 'Local preview',
    });
  });
}
