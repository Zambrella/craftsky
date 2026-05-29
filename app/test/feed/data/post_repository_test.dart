import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  group('PostRepository.listTimeline', () {
    test('fake exposes timeline method without handle or DID input', () async {
      String? seenCursor;
      int? seenLimit;
      final repo = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          seenCursor = cursor;
          seenLimit = limit;
          return const PostPage(items: [], cursor: 'next');
        },
      );

      final asInterface = repo as PostRepository;
      final page = await asInterface.listTimeline(cursor: 'c1', limit: 20);

      expect(seenCursor, 'c1');
      expect(seenLimit, 20);
      expect(page.cursor, 'next');
    });
  });
}
