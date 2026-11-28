import 'package:craftsky_app/router/router.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-016 Instagram migration views have typed settings routes', () {
    expect(
      const InstagramMigrationRoute().location,
      '/settings/instagram',
    );
    expect(
      const InstagramPostImportRoute().location,
      '/settings/instagram/posts',
    );
    expect(
      const InstagramFollowingImportRoute().location,
      '/settings/instagram/following',
    );
    expect(
      const InstagramSuggestionsRoute().location,
      '/settings/instagram/suggestions',
    );
    expect(
      const InstagramAccountRoute().location,
      '/settings/instagram/account',
    );
  });
}
