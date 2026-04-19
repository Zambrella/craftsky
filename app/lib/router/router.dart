import 'package:craftsky_app/router/error_screen.dart';
import 'package:craftsky_app/router/home_page.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'router.g.dart';

class _NavigatorKeys {
  _NavigatorKeys._();

  static GlobalKey<NavigatorState>? _rootKey;

  static GlobalKey<NavigatorState> get rootNavigatorKey =>
      _rootKey ??= GlobalKey<NavigatorState>(debugLabel: 'rootNavigator');
}

@riverpod
GoRouter goRouter(Ref ref) {
  return GoRouter(
    initialLocation: '/',
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    routes: $appRoutes,
    errorBuilder: (context, state) =>
        ErrorScreen(error: state.error ?? 'Unknown routing error'),
  );
}

@TypedGoRoute<HomeRoute>(path: '/', name: 'home')
class HomeRoute extends GoRouteData with $HomeRoute {
  const HomeRoute();

  @override
  Widget build(BuildContext context, GoRouterState state) => const HomePage();
}

extension GoRouterExtension on GoRouter {
  /// Pops any existing stack and replaces the current location.
  void clearStackAndNavigate(String location) {
    while (canPop()) {
      pop();
    }
    pushReplacement(location);
  }
}
