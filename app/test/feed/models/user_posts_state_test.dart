import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('UserPostsState', () {
    test('hasMore is true when cursor is non-null', () {
      const state = UserPostsState(items: [], cursor: 'abc');
      expect(state.hasMore, isTrue);
    });

    test('hasMore is false when cursor is null', () {
      const state = UserPostsState(items: []);
      expect(state.hasMore, isFalse);
    });

    test('copyWith preserves untouched fields', () {
      const state = UserPostsState(items: [], cursor: 'abc');
      final next = state.copyWith(cursor: null);
      expect(next.items, state.items);
      expect(next.cursor, isNull);
    });

    test('toString summarizes list state', () {
      const state = UserPostsState(items: [], cursor: 'abc');
      expect(state.toString(), 'UserPostsState(items: 0, hasMore: true)');
    });
  });
}
