import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('mediaConfig', () {
    test('exposes centralized image limits', () {
      expect(mediaConfig.maxImages, 4);
      expect(mediaConfig.maxImageBytes, 15 * 1024 * 1024);
      expect(mediaConfig.maxAltTextCharacters, 300);
    });
  });
}
