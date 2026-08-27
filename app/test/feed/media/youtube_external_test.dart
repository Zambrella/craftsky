import 'package:craftsky_app/feed/media/youtube_external.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('parseYouTubeExternal', () {
    test('recognizes supported YouTube URL forms', () {
      final cases = {
        'https://www.youtube.com/watch?v=dQw4w9WgXcQ': ('dQw4w9WgXcQ', false),
        'https://youtu.be/dQw4w9WgXcQ': ('dQw4w9WgXcQ', false),
        'https://m.youtube.com/shorts/dQw4w9WgXcQ': ('dQw4w9WgXcQ', true),
        'https://youtube.com/live/dQw4w9WgXcQ': ('dQw4w9WgXcQ', false),
        'https://music.youtube.com/watch?v=dQw4w9WgXcQ': ('dQw4w9WgXcQ', false),
      };

      for (final MapEntry(key: url, value: expected) in cases.entries) {
        final parsed = parseYouTubeExternal(Uri.parse(url));
        expect(parsed?.videoId, expected.$1, reason: url);
        expect(parsed?.isShort, expected.$2, reason: url);
      }
    });

    test('parses numeric and duration timestamps', () {
      expect(
        parseYouTubeExternal(
          Uri.parse('https://youtu.be/dQw4w9WgXcQ?t=90'),
        )?.startSeconds,
        90,
      );
      expect(
        parseYouTubeExternal(
          Uri.parse('https://youtube.com/watch?v=dQw4w9WgXcQ&t=1h2m3s'),
        )?.startSeconds,
        3723,
      );
    });

    test('rejects deceptive hosts, unsupported paths, and invalid IDs', () {
      for (final url in [
        'https://youtube.com.example.org/watch?v=dQw4w9WgXcQ',
        'https://notyoutube.com/watch?v=dQw4w9WgXcQ',
        'https://youtube.com/embed/dQw4w9WgXcQ',
        'https://youtube.com/watch?v=too-short',
        'javascript:alert(1)',
      ]) {
        expect(parseYouTubeExternal(Uri.parse(url)), isNull, reason: url);
      }
    });
  });
}
