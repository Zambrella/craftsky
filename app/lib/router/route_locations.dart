/// Canonical route path strings. Both the router definitions and the redirect
/// logic reference these so the two can't drift.
class RouteLocations {
  RouteLocations._();

  static const welcome = '/welcome';
  static const signIn = '/sign-in';
  static const authComplete = '/auth/complete';
  static const onboarding = '/onboarding';
  static const feed = '/feed';
  // Alias: the post-auth home landing. Keep as a const reference to `feed`
  // so renaming the branch in one place updates both usages.
  static const String home = feed;
  static const projects = '/projects';
  static const search = '/search';
  static const notifications = '/notifications';
  static const postThread = '/posts/:did/:rkey';
  static const profile = '/profile';
  static const savedChild = 'saved';
  static const settingsChild = 'settings';
  static const playgroundChild = 'playground';
}
