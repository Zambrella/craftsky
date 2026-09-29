import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('mediaConfig', () {
    test('preserves published image and accessibility limits', () {
      expect(mediaConfig.maxImages, 4);
      expect(mediaConfig.maxImageBytes, 2000000);
      expect(mediaConfig.maxImageAspectRatio, 20);
      expect(mediaConfig.maxAltTextCharacters, 300);
    });
  });
}
