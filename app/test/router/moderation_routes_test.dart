import 'package:craftsky_app/router/router.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  test('moderation typed routes use canonical settings locations', () {
    expect(
      const AccountStandingRoute().location,
      '/profile/settings/moderation',
    );
  });

  test('settings exposes one moderation route', () {
    final routes = _flattenRoutes($appRoutes).whereType<GoRoute>();
    final settings = routes.singleWhere((route) => route.name == 'settings');

    expect(
      settings.routes
          .whereType<GoRoute>()
          .where((route) => route.path.startsWith('moderation'))
          .map((route) => route.name),
      ['account-standing'],
    );
  });
}

Iterable<RouteBase> _flattenRoutes(Iterable<RouteBase> routes) sync* {
  for (final route in routes) {
    yield route;
    yield* _flattenRoutes(route.routes);
  }
}
