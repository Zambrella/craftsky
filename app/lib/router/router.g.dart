// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'router.dart';

// **************************************************************************
// GoRouterGenerator
// **************************************************************************

List<RouteBase> get $appRoutes => [
  $appShellRoute,
  $welcomeRoute,
  $signInRoute,
  $authCompleteRoute,
  $onboardingRoute,
  $postThreadRoute,
  $userProfileRoute,
];

RouteBase get $appShellRoute => StatefulShellRouteData.$route(
  factory: $AppShellRouteExtension._fromState,
  branches: [
    StatefulShellBranchData.$branch(
      navigatorKey: FeedBranch.$navigatorKey,
      routes: [
        GoRouteData.$route(
          path: '/feed',
          name: 'feed',
          factory: $FeedRoute._fromState,
        ),
      ],
    ),
    StatefulShellBranchData.$branch(
      navigatorKey: ProjectsBranch.$navigatorKey,
      routes: [
        GoRouteData.$route(
          path: '/projects',
          name: 'projects',
          factory: $ProjectsRoute._fromState,
        ),
      ],
    ),
    StatefulShellBranchData.$branch(
      navigatorKey: SearchBranch.$navigatorKey,
      routes: [
        GoRouteData.$route(
          path: '/search',
          name: 'search',
          factory: $SearchRoute._fromState,
          routes: [
            GoRouteData.$route(
              path: 'tags',
              name: 'search-tag',
              factory: $TagSearchRoute._fromState,
            ),
          ],
        ),
      ],
    ),
    StatefulShellBranchData.$branch(
      navigatorKey: NotificationsBranch.$navigatorKey,
      routes: [
        GoRouteData.$route(
          path: '/notifications',
          name: 'notifications',
          factory: $NotificationsRoute._fromState,
          routes: [
            GoRouteData.$route(
              path: 'settings',
              name: 'notification-settings',
              parentNavigatorKey: NotificationSettingsRoute.$parentNavigatorKey,
              factory: $NotificationSettingsRoute._fromState,
            ),
          ],
        ),
      ],
    ),
    StatefulShellBranchData.$branch(
      navigatorKey: ProfileBranch.$navigatorKey,
      routes: [
        GoRouteData.$route(
          path: '/profile',
          name: 'profile',
          factory: $ProfileRoute._fromState,
          routes: [
            GoRouteData.$route(
              path: 'saved',
              name: 'saved',
              factory: $SavedRoute._fromState,
            ),
            GoRouteData.$route(
              path: 'settings',
              name: 'settings',
              parentNavigatorKey: SettingsRoute.$parentNavigatorKey,
              factory: $SettingsRoute._fromState,
            ),
            GoRouteData.$route(
              path: 'playground',
              name: 'playground',
              parentNavigatorKey: PlaygroundRoute.$parentNavigatorKey,
              factory: $PlaygroundRoute._fromState,
            ),
          ],
        ),
      ],
    ),
  ],
);

extension $AppShellRouteExtension on AppShellRoute {
  static AppShellRoute _fromState(GoRouterState state) => const AppShellRoute();
}

mixin $FeedRoute on GoRouteData {
  static FeedRoute _fromState(GoRouterState state) => const FeedRoute();

  @override
  String get location => GoRouteData.$location('/feed');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $ProjectsRoute on GoRouteData {
  static ProjectsRoute _fromState(GoRouterState state) => const ProjectsRoute();

  @override
  String get location => GoRouteData.$location('/projects');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $SearchRoute on GoRouteData {
  static SearchRoute _fromState(GoRouterState state) => SearchRoute(
    q: state.uri.queryParameters['q'],
    tab: _$convertMapValue(
      'tab',
      state.uri.queryParameters,
      _$SearchResultsTabEnumMap._$fromName,
    ),
  );

  SearchRoute get _self => this as SearchRoute;

  @override
  String get location => GoRouteData.$location(
    '/search',
    queryParams: {
      if (_self.q != null) 'q': _self.q,
      if (_self.tab != null) 'tab': _$SearchResultsTabEnumMap[_self.tab!],
    },
  );

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

const _$SearchResultsTabEnumMap = {
  SearchResultsTab.posts: 'posts',
  SearchResultsTab.projects: 'projects',
  SearchResultsTab.profiles: 'profiles',
  SearchResultsTab.tags: 'tags',
};

mixin $TagSearchRoute on GoRouteData {
  static TagSearchRoute _fromState(GoRouterState state) =>
      TagSearchRoute(tag: state.uri.queryParameters['tag']!);

  TagSearchRoute get _self => this as TagSearchRoute;

  @override
  String get location =>
      GoRouteData.$location('/search/tags', queryParams: {'tag': _self.tag});

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $NotificationsRoute on GoRouteData {
  static NotificationsRoute _fromState(GoRouterState state) =>
      const NotificationsRoute();

  @override
  String get location => GoRouteData.$location('/notifications');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $NotificationSettingsRoute on GoRouteData {
  static NotificationSettingsRoute _fromState(GoRouterState state) =>
      const NotificationSettingsRoute();

  @override
  String get location => GoRouteData.$location('/notifications/settings');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $ProfileRoute on GoRouteData {
  static ProfileRoute _fromState(GoRouterState state) => const ProfileRoute();

  @override
  String get location => GoRouteData.$location('/profile');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $SavedRoute on GoRouteData {
  static SavedRoute _fromState(GoRouterState state) => const SavedRoute();

  @override
  String get location => GoRouteData.$location('/profile/saved');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $SettingsRoute on GoRouteData {
  static SettingsRoute _fromState(GoRouterState state) => const SettingsRoute();

  @override
  String get location => GoRouteData.$location('/profile/settings');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

mixin $PlaygroundRoute on GoRouteData {
  static PlaygroundRoute _fromState(GoRouterState state) =>
      const PlaygroundRoute();

  @override
  String get location => GoRouteData.$location('/profile/playground');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

T? _$convertMapValue<T>(
  String key,
  Map<String, String> map,
  T? Function(String) converter,
) {
  final value = map[key];
  return value == null ? null : converter(value);
}

extension<T extends Enum> on Map<T, String> {
  T? _$fromName(String? value) =>
      entries.where((element) => element.value == value).firstOrNull?.key;
}

RouteBase get $welcomeRoute => GoRouteData.$route(
  path: '/welcome',
  name: 'welcome',
  parentNavigatorKey: WelcomeRoute.$parentNavigatorKey,
  factory: $WelcomeRoute._fromState,
);

mixin $WelcomeRoute on GoRouteData {
  static WelcomeRoute _fromState(GoRouterState state) => const WelcomeRoute();

  @override
  String get location => GoRouteData.$location('/welcome');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

RouteBase get $signInRoute => GoRouteData.$route(
  path: '/sign-in',
  name: 'sign-in',
  parentNavigatorKey: SignInRoute.$parentNavigatorKey,
  factory: $SignInRoute._fromState,
);

mixin $SignInRoute on GoRouteData {
  static SignInRoute _fromState(GoRouterState state) => const SignInRoute();

  @override
  String get location => GoRouteData.$location('/sign-in');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

RouteBase get $authCompleteRoute => GoRouteData.$route(
  path: '/auth/complete',
  name: 'auth-complete',
  parentNavigatorKey: AuthCompleteRoute.$parentNavigatorKey,
  factory: $AuthCompleteRoute._fromState,
);

mixin $AuthCompleteRoute on GoRouteData {
  static AuthCompleteRoute _fromState(GoRouterState state) =>
      AuthCompleteRoute(token: state.uri.queryParameters['token']!);

  AuthCompleteRoute get _self => this as AuthCompleteRoute;

  @override
  String get location => GoRouteData.$location(
    '/auth/complete',
    queryParams: {'token': _self.token},
  );

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

RouteBase get $onboardingRoute => GoRouteData.$route(
  path: '/onboarding',
  name: 'onboarding',
  parentNavigatorKey: OnboardingRoute.$parentNavigatorKey,
  factory: $OnboardingRoute._fromState,
);

mixin $OnboardingRoute on GoRouteData {
  static OnboardingRoute _fromState(GoRouterState state) =>
      const OnboardingRoute();

  @override
  String get location => GoRouteData.$location('/onboarding');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

RouteBase get $postThreadRoute => GoRouteData.$route(
  path: '/posts/:did/:rkey',
  name: 'post-thread',
  parentNavigatorKey: PostThreadRoute.$parentNavigatorKey,
  factory: $PostThreadRoute._fromState,
);

mixin $PostThreadRoute on GoRouteData {
  static PostThreadRoute _fromState(GoRouterState state) => PostThreadRoute(
    did: state.pathParameters['did']!,
    rkey: state.pathParameters['rkey']!,
    focus: state.uri.queryParameters['focus'],
    $extra: state.extra as Post?,
  );

  PostThreadRoute get _self => this as PostThreadRoute;

  @override
  String get location => GoRouteData.$location(
    '/posts/${Uri.encodeComponent(_self.did)}/${Uri.encodeComponent(_self.rkey)}',
    queryParams: {if (_self.focus != null) 'focus': _self.focus},
  );

  @override
  void go(BuildContext context) => context.go(location, extra: _self.$extra);

  @override
  Future<T?> push<T>(BuildContext context) =>
      context.push<T>(location, extra: _self.$extra);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location, extra: _self.$extra);

  @override
  void replace(BuildContext context) =>
      context.replace(location, extra: _self.$extra);
}

RouteBase get $userProfileRoute => GoRouteData.$route(
  path: '/profile/:handle',
  name: 'user-profile',
  parentNavigatorKey: UserProfileRoute.$parentNavigatorKey,
  factory: $UserProfileRoute._fromState,
);

mixin $UserProfileRoute on GoRouteData {
  static UserProfileRoute _fromState(GoRouterState state) =>
      UserProfileRoute(handle: state.pathParameters['handle']!);

  UserProfileRoute get _self => this as UserProfileRoute;

  @override
  String get location =>
      GoRouteData.$location('/profile/${Uri.encodeComponent(_self.handle)}');

  @override
  void go(BuildContext context) => context.go(location);

  @override
  Future<T?> push<T>(BuildContext context) => context.push<T>(location);

  @override
  void pushReplacement(BuildContext context) =>
      context.pushReplacement(location);

  @override
  void replace(BuildContext context) => context.replace(location);
}

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(goRouter)
final goRouterProvider = GoRouterProvider._();

final class GoRouterProvider
    extends $FunctionalProvider<GoRouter, GoRouter, GoRouter>
    with $Provider<GoRouter> {
  GoRouterProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'goRouterProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$goRouterHash();

  @$internal
  @override
  $ProviderElement<GoRouter> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  GoRouter create(Ref ref) {
    return goRouter(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(GoRouter value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<GoRouter>(value),
    );
  }
}

String _$goRouterHash() => r'4f281ca69e50d81ef2b5b56c8ecbbc8a051b1ef1';
