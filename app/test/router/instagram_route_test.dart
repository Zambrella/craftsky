import 'package:craftsky_app/router/router.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-016 Instagram migration has a typed settings route', () {
    expect(
      const InstagramMigrationRoute().location,
      '/profile/settings/instagram',
    );
  });
}
