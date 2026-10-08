/// Canonical route path strings. Both the router definitions and the redirect
/// logic reference these so the two can't drift.
class RouteLocations {
  RouteLocations._();

  static const welcome = '/welcome';
  static const signIn = '/sign-in';
  static const addAccount = '/add-account';
  static const authComplete = '/auth/complete';
  static const accountDeletionReauthComplete =
      '/account-deletion/reauth-complete';
  static const onboarding = '/onboarding';
  static const accountEligibility = '/account-eligibility';
  static const feed = '/feed';
  // Alias: the post-auth home landing. Keep as a const reference to `feed`
  // so renaming the branch in one place updates both usages.
  static const String home = feed;
  static const projects = '/projects';
  static const search = '/search';
  static const searchTagsChild = 'tags';
  static const notifications = '/notifications';
  static const notificationSettingsChild = 'settings';
  static const postThread = '/posts/:did/:rkey';
  static const postLikesChild = 'likes';
  static const postRepostsChild = 'reposts';
  static const postQuotesChild = 'quotes';
  static const businessEvent = '/events/:did/:rkey';
  static const profile = '/profile';
  static const profiles = '/profiles';
  static const settingsChild = 'settings';
  static const settings = '$profile/$settingsChild';
  static const growthChild = 'growth';
  static const appearanceChild = 'appearance';
  static const languagesChild = 'languages';
  static const customisationChild = 'customisation';
  static const accountChild = 'account';
  static const subscriptionsChild = 'subscriptions';
  static const subscriptions = '$settings/$subscriptionsChild';
  static const moderationChild = 'moderation';
  static const aboutChild = 'about';
  static const productsChild = 'products';
  static const eventsChild = 'events';
  static const instagramMigrationChild = 'instagram';
  static const scheduledPosts = '/scheduled';
  static const drafts = '/drafts';
  static const savedPosts = '/saved';
  static const savedPostFolderChild = 'folder';
  static const followersChild = 'followers';
  static const followingChild = 'following';
  static const mutedAccountsChild = 'muted';
  static const blockedAccountsChild = 'blocked';
  static const playgroundChild = 'playground';
}
