import 'package:craftsky_app/feed/media/image_metadata_stripper.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('stripNonEssentialMetadata', () {
    test('removes removable privacy-sensitive metadata keys', () {
      final prepared = stripNonEssentialMetadata(
        const ImagePreparationInput(
          format: SupportedImageFormat.jpeg,
          metadata: {
            'gpsLatitude': '42.0',
            'gpsLongitude': '-71.0',
            'cameraMake': 'Acme',
            'cameraModel': 'Camera 1',
            'captureTimestamp': '2026-01-01T00:00:00Z',
            'software': 'Some App',
            'comment': 'my note',
            'orientation': '1',
            'colorProfile': 'sRGB',
          },
        ),
      );

      expect(prepared.metadata.keys, isNot(contains('gpsLatitude')));
      expect(prepared.metadata.keys, isNot(contains('gpsLongitude')));
      expect(prepared.metadata.keys, isNot(contains('cameraMake')));
      expect(prepared.metadata.keys, isNot(contains('cameraModel')));
      expect(prepared.metadata.keys, isNot(contains('captureTimestamp')));
      expect(prepared.metadata.keys, isNot(contains('software')));
      expect(prepared.metadata.keys, isNot(contains('comment')));

      expect(prepared.metadata['orientation'], '1');
      expect(prepared.metadata['colorProfile'], 'sRGB');
    });

    test('preserves original supported format whenever possible', () {
      final prepared = stripNonEssentialMetadata(
        const ImagePreparationInput(
          format: SupportedImageFormat.webp,
          metadata: {},
        ),
      );

      expect(prepared.format, SupportedImageFormat.webp);
    });

    test('does not flatten png transparency', () {
      final prepared = stripNonEssentialMetadata(
        const ImagePreparationInput(
          format: SupportedImageFormat.png,
          metadata: {},
          hasTransparency: true,
        ),
      );

      expect(prepared.hasTransparency, isTrue);
    });
  });
}
