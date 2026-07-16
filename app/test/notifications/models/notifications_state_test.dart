import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('generated copyWith can explicitly clear the pagination cursor', () {
    const state = NotificationsState(
      items: [],
      cursor: 'opaque-next',
      renderToken: 1,
    );

    final updated = state.copyWith(cursor: null);

    expect(updated.cursor, isNull);
    expect(updated.hasMore, isFalse);
    expect(updated.renderToken, 1);
    expect(
      updated,
      const NotificationsState(items: [], renderToken: 1),
    );
  });
}
