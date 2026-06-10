import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/projects/models/user_projects_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-013 exposes items, cursor, hasMore, equality and copyWith', () {
    const empty = UserProjectsState(items: []);
    const cursor = UserProjectsState(items: [], cursor: 'next');

    expect(empty.hasMore, isFalse);
    expect(cursor.hasMore, isTrue);
    expect(cursor, const UserProjectsState(items: [], cursor: 'next'));
    expect(cursor.copyWith(cursor: null).hasMore, isFalse);
  });
}
