import 'package:craftsky_app/feed/media/image_dimensions.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('toOptionalAspectRatio', () {
    test('returns positive ratio when dimensions are known', () {
      final ratio = toOptionalAspectRatio(width: 919, height: 2000);

      expect(ratio, isNotNull);
      expect(ratio!.width, 919);
      expect(ratio.height, 2000);
    });

    test('returns null when dimensions are unknown', () {
      expect(toOptionalAspectRatio(width: null, height: 2000), isNull);
      expect(toOptionalAspectRatio(width: 2000, height: null), isNull);
    });

    test('returns null for zero or negative dimensions', () {
      expect(toOptionalAspectRatio(width: 0, height: 2000), isNull);
      expect(toOptionalAspectRatio(width: 2000, height: 0), isNull);
      expect(toOptionalAspectRatio(width: -1, height: 2000), isNull);
      expect(toOptionalAspectRatio(width: 2000, height: -1), isNull);
    });
  });
}
