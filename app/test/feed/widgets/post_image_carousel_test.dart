import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('computeBoundedImageHeight', () {
    test('uses stable fallback when aspect ratio is missing', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: null,
      );

      expect(height, 320);
    });

    test('keeps 1:1 images within bounds', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 1, height: 1),
      );

      expect(height, 320);
    });

    test('clamps very tall images to max height', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 919, height: 2000),
      );

      expect(height, 420);
    });

    test('clamps very wide images to min height', () {
      final height = computeBoundedImageHeight(
        availableWidth: 320,
        aspectRatio: const PostImageAspectRatio(width: 2000, height: 919),
      );

      expect(height, 160);
    });
  });
}
