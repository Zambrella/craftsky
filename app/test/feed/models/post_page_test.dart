import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('PostPage', () {
    test('round-trips with cursor present', () {
      const cursor =
          'eyJpbmRleGVkQXQiOiIyMDI2LTA1LTA0VDE4OjIzOjQ3WiIsInVyaSI6ImF0Oi8vIn0';
      final json = {
        'items': <Map<String, dynamic>>[],
        'cursor': cursor,
      };

      final page = PostPageMapper.fromMap(json);
      expect(page.items, isEmpty);
      expect(page.cursor, cursor);
      expect(page.toMap(), json);
    });

    test('absent cursor decodes as null and re-encodes without the key', () {
      final json = {'items': <Map<String, dynamic>>[]};

      final page = PostPageMapper.fromMap(json);
      expect(page.cursor, isNull);

      // Re-encoding omits the null cursor entirely (matches AppView's
      // pagination contract: `cursor` is omitted, not `null`, when no
      // more pages exist).
      expect(page.toMap(), {'items': <Map<String, dynamic>>[]});
    });
  });
}
