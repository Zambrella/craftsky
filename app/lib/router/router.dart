import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/pages/saved_page.dart';
import 'package:craftsky_app/profile/pages/user_profile_page.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/router/error_screen.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
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
  return GoRouter(
    initialLocation: RouteLocations.welcome,
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    redirect: (context, state) {
      final isSignedIn = ref.read(authStatusProvider);
      final isOnboarded = ref.read(onboardingStatusProvider);

      const unauthenticatedRoutes = [
        RouteLocations.welcome,
        RouteLocations.signIn,
      ];
      const onboardingRoute = RouteLocations.onboarding;

      final loc = state.matchedLocation;

      if (!isSignedIn && !unauthenticatedRoutes.contains(loc)) {
        return RouteLocations.welcome;
      }
      if (isSignedIn && !isOnboarded && loc != onboardingRoute) {
        return onboardingRoute;
      }
      if (isSignedIn &&
          isOnboarded &&
          (unauthenticatedRoutes.contains(loc) || loc == onboardingRoute)) {
        return RouteLocations.home;
      }
      return null;
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

@TypedGoRoute<SettingsRoute>(path: RouteLocations.settings, name: 'settings')
class SettingsRoute extends GoRouteData with $SettingsRoute {
  const SettingsRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const SettingsPage();
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
