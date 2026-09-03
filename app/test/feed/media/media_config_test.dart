import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('mediaConfig', () {
    test('exposes centralized image limits', () {
      expect(mediaConfig.maxImages, 4);
      expect(mediaConfig.maxSourceImageBytes, 50000000);
      expect(mediaConfig.maxSourceImageSide, 8192);
      expect(mediaConfig.maxSourceImagePixels, 16000000);
      expect(mediaConfig.maxImageBytes, 2000000);
      expect(mediaConfig.targetImageBytes, 1950000);
      expect(mediaConfig.maxImageWidth, 4000);
      expect(mediaConfig.maxImageHeight, 4000);
      expect(mediaConfig.maxImageAspectRatio, 20);
      expect(mediaConfig.maxAltTextCharacters, 300);
    });
  });
}
