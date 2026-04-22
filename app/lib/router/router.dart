import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/design_playground/pages/design_playground_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/pages/saved_page.dart';
import 'package:craftsky_app/profile/pages/user_profile_page.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/router/error_screen.dart';
import 'package:craftsky_app/router/onboarding_refresh_listener.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'router.g.dart';

/// Singleton navigator keys. Globals that are recreated on hot reload cause
/// go_router to crash; holding them behind a class means hot reload keeps
/// the same instances.
class _NavigatorKeys {
  _NavigatorKeys._();

  static GlobalKey<NavigatorState>? _rootKey;
  static GlobalKey<NavigatorState>? _feedKey;
  static GlobalKey<NavigatorState>? _searchKey;
  static GlobalKey<NavigatorState>? _notificationsKey;
  static GlobalKey<NavigatorState>? _profileKey;

  static GlobalKey<NavigatorState> get rootNavigatorKey =>
      _rootKey ??= GlobalKey<NavigatorState>(debugLabel: 'rootNavigator');
  static GlobalKey<NavigatorState> get feedNavigatorKey =>
      _feedKey ??= GlobalKey<NavigatorState>(debugLabel: 'feedNavigator');
  static GlobalKey<NavigatorState> get searchNavigatorKey =>
      _searchKey ??= GlobalKey<NavigatorState>(debugLabel: 'searchNavigator');
  static GlobalKey<NavigatorState> get notificationsNavigatorKey =>
      _notificationsKey ??= GlobalKey<NavigatorState>(
        debugLabel: 'notificationsNavigator',
      );
  static GlobalKey<NavigatorState> get profileNavigatorKey =>
      _profileKey ??= GlobalKey<NavigatorState>(debugLabel: 'profileNavigator');
}

@riverpod
GoRouter goRouter(Ref ref) {
  final refresh = ChangeNotifier();
  final onboardingListener = OnboardingRefreshListener(
    ref: ref,
    onChange: refresh.notifyListeners,
  );

  ref.onDispose(() {
    onboardingListener.close();
    refresh.dispose();
  });

  ref.listen(authSessionProvider, (_, next) {
    refresh.notifyListeners();
    onboardingListener.update(next.value);
  });

  return GoRouter(
    initialLocation: RouteLocations.welcome,
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    refreshListenable: refresh,
    redirect: (context, state) {
      final loc = state.matchedLocation;
      const unauthenticatedRoutes = [
        RouteLocations.welcome,
        RouteLocations.signIn,
      ];

      final auth = ref.read(authSessionProvider).value;
      if (auth == null) return null; // transient AsyncLoading

      switch (auth) {
        case SignedOut():
          if (loc == RouteLocations.authComplete) return null;
          return unauthenticatedRoutes.contains(loc)
              ? null
              : RouteLocations.welcome;
        case SignedIn(:final did):
          final onboarded = ref.read(onboardingStatusProvider(did));
          if (loc == RouteLocations.authComplete) {
            return onboarded
                ? RouteLocations.home
                : RouteLocations.onboarding;
          }
          if (!onboarded && loc != RouteLocations.onboarding) {
            return RouteLocations.onboarding;
          }
          if (onboarded &&
              (unauthenticatedRoutes.contains(loc) ||
                  loc == RouteLocations.onboarding)) {
            return RouteLocations.home;
          }
          return null;
      }
    },
    routes: $appRoutes,
    errorBuilder: (context, state) =>
        ErrorScreen(error: state.error ?? 'Unknown routing error'),
  );
}

// --- Shell route -----------------------------------------------------------

@TypedStatefulShellRoute<AppShellRoute>(
  branches: [
    TypedStatefulShellBranch<FeedBranch>(
      routes: [
        TypedGoRoute<FeedRoute>(path: RouteLocations.feed, name: 'feed'),
      ],
    ),
    TypedStatefulShellBranch<SearchBranch>(
      routes: [
        TypedGoRoute<SearchRoute>(path: RouteLocations.search, name: 'search'),
      ],
    ),
    TypedStatefulShellBranch<NotificationsBranch>(
      routes: [
        TypedGoRoute<NotificationsRoute>(
          path: RouteLocations.notifications,
          name: 'notifications',
        ),
      ],
    ),
    TypedStatefulShellBranch<ProfileBranch>(
      routes: [
        TypedGoRoute<ProfileRoute>(
          path: RouteLocations.profile,
          name: 'profile',
          routes: [
            TypedGoRoute<SavedRoute>(
              path: RouteLocations.savedChild,
              name: 'saved',
            ),
            TypedGoRoute<SettingsRoute>(
              path: RouteLocations.settingsChild,
              name: 'settings',
            ),
            TypedGoRoute<PlaygroundRoute>(
              path: RouteLocations.playgroundChild,
              name: 'playground',
            ),
          ],
        ),
      ],
    ),
  ],
)
class AppShellRoute extends StatefulShellRouteData {
  const AppShellRoute();

  @override
  Widget builder(
    BuildContext context,
    GoRouterState state,
    StatefulNavigationShell navigationShell,
  ) {
    return AppShell(navigationShell: navigationShell);
  }
}

class FeedBranch extends StatefulShellBranchData {
  const FeedBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.feedNavigatorKey;
}

class SearchBranch extends StatefulShellBranchData {
  const SearchBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.searchNavigatorKey;
}

class NotificationsBranch extends StatefulShellBranchData {
  const NotificationsBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.notificationsNavigatorKey;
}

class ProfileBranch extends StatefulShellBranchData {
  const ProfileBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.profileNavigatorKey;
}

class FeedRoute extends GoRouteData with $FeedRoute {
  const FeedRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) => const FeedPage();
}

class SearchRoute extends GoRouteData with $SearchRoute {
  const SearchRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) => const SearchPage();
}

class NotificationsRoute extends GoRouteData with $NotificationsRoute {
  const NotificationsRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const NotificationsPage();
}

class ProfileRoute extends GoRouteData with $ProfileRoute {
  const ProfileRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const ProfilePage();
}

class SavedRoute extends GoRouteData with $SavedRoute {
  const SavedRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) => const SavedPage();
}

/// Declared as a child of [ProfileRoute] so its path becomes
/// `/profile/settings` and the back arrow pops to `/profile`. The parent
/// navigator key lifts it onto the root navigator so it covers the shell's
/// bottom navigation.
class SettingsRoute extends GoRouteData with $SettingsRoute {
  const SettingsRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const SettingsPage();
}

/// Dev-only design playground. Same shape as [SettingsRoute] — nested under
/// profile for back-button semantics, pushed on the root navigator to cover
/// the bottom nav.
class PlaygroundRoute extends GoRouteData with $PlaygroundRoute {
  const PlaygroundRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const DesignPlaygroundPage();
}

// --- Root-navigator routes (push over the shell) ---------------------------

@TypedGoRoute<WelcomeRoute>(path: RouteLocations.welcome, name: 'welcome')
class WelcomeRoute extends GoRouteData with $WelcomeRoute {
  const WelcomeRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const WelcomePage();
}

@TypedGoRoute<SignInRoute>(path: RouteLocations.signIn, name: 'sign-in')
class SignInRoute extends GoRouteData with $SignInRoute {
  const SignInRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) => const SignInPage();
}

@TypedGoRoute<AuthCompleteRoute>(
  path: RouteLocations.authComplete,
  name: 'auth-complete',
)
class AuthCompleteRoute extends GoRouteData with $AuthCompleteRoute {
  const AuthCompleteRoute({required this.token});

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  final String token;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      AuthCompletePage(token: token);
}

@TypedGoRoute<OnboardingRoute>(
  path: RouteLocations.onboarding,
  name: 'onboarding',
)
class OnboardingRoute extends GoRouteData with $OnboardingRoute {
  const OnboardingRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const OnboardingPage();
}

@TypedGoRoute<UserProfileRoute>(
  path: '${RouteLocations.profile}/:handle',
  name: 'user-profile',
)
class UserProfileRoute extends GoRouteData with $UserProfileRoute {
  const UserProfileRoute({required this.handle});

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  final String handle;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      UserProfilePage(handle: handle);
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
