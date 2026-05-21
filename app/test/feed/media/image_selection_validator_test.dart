import 'package:craftsky_app/feed/media/image_selection_validator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('validateImageSelection', () {
    test('accepts supported image types within cap', () {
      final result = validateImageSelection(
        existing: const [],
        incoming: const [
          LocalImageSelection(name: 'one.jpg', mimeType: 'image/jpeg'),
          LocalImageSelection(name: 'two.png', mimeType: 'image/png'),
          LocalImageSelection(name: 'three.webp', mimeType: 'image/webp'),
        ],
      );

      expect(result.accepted, hasLength(3));
      expect(result.rejected, isEmpty);
    });

    test('falls back to extension when mime type is unavailable', () {
      final result = validateImageSelection(
        existing: const [],
        incoming: const [LocalImageSelection(name: 'loom.WEBP', mimeType: '')],
      );

      expect(result.accepted.single.name, 'loom.WEBP');
      expect(result.rejected, isEmpty);
    });

    test('rejects unsupported types before upload', () {
      final result = validateImageSelection(
        existing: const [],
        incoming: const [
          LocalImageSelection(name: 'anim.gif', mimeType: 'image/gif'),
          LocalImageSelection(name: 'clip.mp4', mimeType: 'video/mp4'),
        ],
      );

      expect(result.accepted, isEmpty);
      expect(
        result.rejected.map((item) => item.reason),
        everyElement(ImageSelectionRejection.unsupportedType),
      );
    });

    test('rejects images that exceed the configured cap', () {
      final result = validateImageSelection(
        existing: const [
          LocalImageSelection(name: '1.jpg', mimeType: 'image/jpeg'),
          LocalImageSelection(name: '2.jpg', mimeType: 'image/jpeg'),
          LocalImageSelection(name: '3.jpg', mimeType: 'image/jpeg'),
          LocalImageSelection(name: '4.jpg', mimeType: 'image/jpeg'),
        ],
        incoming: const [
          LocalImageSelection(name: '5.jpg', mimeType: 'image/jpeg'),
        ],
      );

      expect(result.accepted, isEmpty);
      expect(
        result.rejected.single.reason,
        ImageSelectionRejection.imageLimitExceeded,
      );
    });
  });
}
