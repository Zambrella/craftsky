// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'router.dart';

// **************************************************************************
// GoRouterGenerator
// **************************************************************************

List<RouteBase> get $appRoutes => [
  $accountDeletionReauthCompleteRoute,
  $accountEligibilityRoute,
  $authenticatedShellRoute,
  $welcomeRoute,
  $signInRoute,
  $addAccountRoute,
  $authCompleteRoute,
  $onboardingRoute,
];

RouteBase get $accountDeletionReauthCompleteRoute => GoRouteData.$route(
  path: '/account-deletion/reauth-complete',
  name: 'account-deletion-reauth-complete',
  hasOverriddenOnExit: false,
  factory: $AccountDeletionReauthCompleteRoute._fromState,
);

mixin $AccountDeletionReauthCompleteRoute on GoRouteData {
  static AccountDeletionReauthCompleteRoute _fromState(GoRouterState state) =>
      AccountDeletionReauthCompleteRoute(
        jobId: state.uri.queryParameters['job-id']!,
        proof: state.uri.queryParameters['proof']!,
      );

  AccountDeletionReauthCompleteRoute get _self =>
      this as AccountDeletionReauthCompleteRoute;

  @override
  String get location => GoRouteData.$location(
    '/account-deletion/reauth-complete',
    queryParams: {'job-id': _self.jobId, 'proof': _self.proof},
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

RouteBase get $accountEligibilityRoute => GoRouteData.$route(
  path: '/account-eligibility',
  name: 'account-eligibility',
  hasOverriddenOnExit: false,
  factory: $AccountEligibilityRoute._fromState,
);

mixin $AccountEligibilityRoute on GoRouteData {
  static AccountEligibilityRoute _fromState(GoRouterState state) =>
      const AccountEligibilityRoute();

  @override
  String get location => GoRouteData.$location('/account-eligibility');

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

RouteBase get $authenticatedShellRoute => ShellRouteData.$route(
  navigatorKey: AuthenticatedShellRoute.$navigatorKey,
  observers: AuthenticatedShellRoute.$observers,
  factory: $AuthenticatedShellRouteExtension._fromState,
  routes: [
    StatefulShellRouteData.$route(
      factory: $AppShellRouteExtension._fromState,
      branches: [
        StatefulShellBranchData.$branch(
          navigatorKey: FeedBranch.$navigatorKey,
          observers: FeedBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/feed',
              name: 'feed',
              hasOverriddenOnExit: false,
              factory: $FeedRoute._fromState,
            ),
            GoRouteData.$route(
              path: '/projects',
              name: 'projects',
              hasOverriddenOnExit: false,
              factory: $ProjectsRoute._fromState,
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: SearchBranch.$navigatorKey,
          observers: SearchBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/search',
              name: 'search',
              hasOverriddenOnExit: false,
              factory: $SearchRoute._fromState,
              routes: [
                GoRouteData.$route(
                  path: 'tags',
                  name: 'search-tag',
                  hasOverriddenOnExit: false,
                  parentNavigatorKey: TagSearchRoute.$parentNavigatorKey,
                  factory: $TagSearchRoute._fromState,
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: MessagesBranch.$navigatorKey,
          observers: MessagesBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/messages',
              name: 'messages',
              hasOverriddenOnExit: false,
              factory: $ChatInboxRoute._fromState,
              routes: [
                GoRouteData.$route(
                  path: 'requests',
                  name: 'message-requests',
                  hasOverriddenOnExit: false,
                  factory: $ChatRequestsRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'new',
                  name: 'new-message',
                  hasOverriddenOnExit: false,
                  factory: $NewMessageRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'conversations/:conversationId',
                  name: 'message-conversation',
                  hasOverriddenOnExit: false,
                  parentNavigatorKey: ChatConversationRoute.$parentNavigatorKey,
                  factory: $ChatConversationRoute._fromState,
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: NotificationsBranch.$navigatorKey,
          observers: NotificationsBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/notifications',
              name: 'notifications',
              hasOverriddenOnExit: false,
              factory: $NotificationsRoute._fromState,
              routes: [
                GoRouteData.$route(
                  path: 'settings',
                  name: 'notification-settings',
                  hasOverriddenOnExit: false,
                  parentNavigatorKey:
                      NotificationSettingsRoute.$parentNavigatorKey,
                  factory: $NotificationSettingsRoute._fromState,
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: ProfileBranch.$navigatorKey,
          observers: ProfileBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/profile',
              name: 'profile',
              hasOverriddenOnExit: false,
              factory: $ProfileRoute._fromState,
              routes: [
                GoRouteData.$route(
                  path: 'playground',
                  name: 'playground',
                  hasOverriddenOnExit: false,
                  parentNavigatorKey: PlaygroundRoute.$parentNavigatorKey,
                  factory: $PlaygroundRoute._fromState,
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: SavedPostsBranch.$navigatorKey,
          observers: SavedPostsBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/saved',
              name: 'saved-posts',
              hasOverriddenOnExit: false,
              factory: $SavedPostsRoute._fromState,
              routes: [
                GoRouteData.$route(
                  path: 'folder',
                  name: 'saved-post-folder',
                  hasOverriddenOnExit: false,
                  factory: $SavedPostFolderRoute._fromState,
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: ScheduledPostsBranch.$navigatorKey,
          observers: ScheduledPostsBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/scheduled',
              name: 'scheduled-posts',
              hasOverriddenOnExit: false,
              factory: $ScheduledPostsRoute._fromState,
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: DraftsBranch.$navigatorKey,
          observers: DraftsBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/drafts',
              name: 'drafts',
              hasOverriddenOnExit: false,
              factory: $DraftsRoute._fromState,
            ),
          ],
        ),
        StatefulShellBranchData.$branch(
          navigatorKey: SettingsBranch.$navigatorKey,
          observers: SettingsBranch.$observers,
          routes: [
            GoRouteData.$route(
              path: '/settings',
              name: 'settings',
              hasOverriddenOnExit: false,
              factory: $SettingsRoute._fromState,
              routes: [
                GoRouteData.$route(
                  path: 'messaging',
                  name: 'messaging-settings',
                  hasOverriddenOnExit: false,
                  factory: $ChatSettingsRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'appearance',
                  name: 'appearance',
                  hasOverriddenOnExit: false,
                  factory: $AppearanceRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'privacy',
                  name: 'settings-privacy',
                  hasOverriddenOnExit: false,
                  factory: $PrivacyRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'growth',
                  name: 'follower-growth',
                  hasOverriddenOnExit: false,
                  factory: $FollowerGrowthRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'customisation',
                  name: 'profile-customisation',
                  hasOverriddenOnExit: false,
                  factory: $ProfileCustomisationRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'account',
                  name: 'settings-account',
                  hasOverriddenOnExit: false,
                  factory: $AccountRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'subscriptions',
                  name: 'settings-subscriptions',
                  hasOverriddenOnExit: false,
                  factory: $SubscriptionsRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'moderation',
                  name: 'account-standing',
                  hasOverriddenOnExit: false,
                  factory: $AccountStandingRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'about',
                  name: 'settings-about',
                  hasOverriddenOnExit: false,
                  factory: $AboutRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'products',
                  name: 'settings-business-products',
                  hasOverriddenOnExit: false,
                  factory: $BusinessProductsRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'events',
                  name: 'settings-business-events',
                  hasOverriddenOnExit: false,
                  factory: $BusinessEventsRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'languages',
                  name: 'languages',
                  hasOverriddenOnExit: false,
                  factory: $LanguagesRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'instagram',
                  name: 'instagram-migration',
                  hasOverriddenOnExit: false,
                  factory: $InstagramMigrationRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'followers',
                  name: 'settings-followers',
                  hasOverriddenOnExit: false,
                  factory: $FollowersRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'following',
                  name: 'settings-following',
                  hasOverriddenOnExit: false,
                  factory: $FollowingRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'muted',
                  name: 'settings-muted-accounts',
                  hasOverriddenOnExit: false,
                  factory: $MutedAccountsRoute._fromState,
                ),
                GoRouteData.$route(
                  path: 'blocked',
                  name: 'settings-blocked-accounts',
                  hasOverriddenOnExit: false,
                  factory: $BlockedAccountsRoute._fromState,
                ),
              ],
            ),
          ],
        ),
      ],
    ),
    GoRouteData.$route(
      path: '/posts/:did/:rkey',
      name: 'post-thread',
      hasOverriddenOnExit: false,
      factory: $PostThreadRoute._fromState,
      routes: [
        GoRouteData.$route(
          path: 'likes',
          name: 'post-likes',
          hasOverriddenOnExit: false,
          factory: $PostLikesRoute._fromState,
        ),
        GoRouteData.$route(
          path: 'reposts',
          name: 'post-reposts',
          hasOverriddenOnExit: false,
          factory: $PostRepostsRoute._fromState,
        ),
        GoRouteData.$route(
          path: 'quotes',
          name: 'post-quotes',
          hasOverriddenOnExit: false,
          factory: $PostQuotesRoute._fromState,
        ),
      ],
    ),
    GoRouteData.$route(
      path: '/events/:did/:rkey',
      name: 'business-event',
      hasOverriddenOnExit: false,
      factory: $BusinessEventRoute._fromState,
    ),
    GoRouteData.$route(
      path: '/profiles/@:handle',
      name: 'profile-alias',
      hasOverriddenOnExit: false,
      factory: $ProfileAliasRoute._fromState,
    ),
    GoRouteData.$route(
      path: '/profiles/:did',
      name: 'user-profile',
      hasOverriddenOnExit: false,
      factory: $UserProfileRoute._fromState,
      routes: [
        GoRouteData.$route(
          path: 'mutual-followers',
          name: 'profile-mutual-followers',
          hasOverriddenOnExit: false,
          factory: $ProfileMutualFollowersRoute._fromState,
        ),
      ],
    ),
  ],
);

extension $AuthenticatedShellRouteExtension on AuthenticatedShellRoute {
  static AuthenticatedShellRoute _fromState(GoRouterState state) =>
      const AuthenticatedShellRoute();
}

extension $AppShellRouteExtension on AppShellRoute {
  static AppShellRoute _fromState(GoRouterState state) => const AppShellRoute();
}

mixin $FeedRoute on GoRouteData {
  static FeedRoute _fromState(GoRouterState state) =>
      FeedRoute(craft: state.uri.queryParameters['craft']);

  FeedRoute get _self => this as FeedRoute;

  @override
  String get location => GoRouteData.$location(
    '/feed',
    queryParams: {if (_self.craft != null) 'craft': _self.craft},
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

mixin $ProjectsRoute on GoRouteData {
  static ProjectsRoute _fromState(GoRouterState state) =>
      ProjectsRoute(craft: state.uri.queryParameters['craft']);

  ProjectsRoute get _self => this as ProjectsRoute;

  @override
  String get location => GoRouteData.$location(
    '/projects',
    queryParams: {if (_self.craft != null) 'craft': _self.craft},
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

mixin $ChatInboxRoute on GoRouteData {
  static ChatInboxRoute _fromState(GoRouterState state) =>
      const ChatInboxRoute();

  @override
  String get location => GoRouteData.$location('/messages');

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

mixin $ChatRequestsRoute on GoRouteData {
  static ChatRequestsRoute _fromState(GoRouterState state) =>
      const ChatRequestsRoute();

  @override
  String get location => GoRouteData.$location('/messages/requests');

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

mixin $NewMessageRoute on GoRouteData {
  static NewMessageRoute _fromState(GoRouterState state) =>
      NewMessageRoute(recipientDid: state.uri.queryParameters['recipient-did']);

  NewMessageRoute get _self => this as NewMessageRoute;

  @override
  String get location => GoRouteData.$location(
    '/messages/new',
    queryParams: {
      if (_self.recipientDid != null) 'recipient-did': _self.recipientDid,
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

mixin $ChatConversationRoute on GoRouteData {
  static ChatConversationRoute _fromState(GoRouterState state) =>
      ChatConversationRoute(
        conversationId: state.pathParameters['conversationId']!,
      );

  ChatConversationRoute get _self => this as ChatConversationRoute;

  @override
  String get location => GoRouteData.$location(
    '/messages/conversations/${Uri.encodeComponent(_self.conversationId)}',
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

mixin $SavedPostsRoute on GoRouteData {
  static SavedPostsRoute _fromState(GoRouterState state) =>
      const SavedPostsRoute();

  @override
  String get location => GoRouteData.$location('/saved');

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

mixin $SavedPostFolderRoute on GoRouteData {
  static SavedPostFolderRoute _fromState(GoRouterState state) =>
      SavedPostFolderRoute($extra: state.extra as SavedPostFolderRouteData?);

  SavedPostFolderRoute get _self => this as SavedPostFolderRoute;

  @override
  String get location => GoRouteData.$location('/saved/folder');

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

mixin $ScheduledPostsRoute on GoRouteData {
  static ScheduledPostsRoute _fromState(GoRouterState state) =>
      const ScheduledPostsRoute();

  @override
  String get location => GoRouteData.$location('/scheduled');

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

mixin $DraftsRoute on GoRouteData {
  static DraftsRoute _fromState(GoRouterState state) => const DraftsRoute();

  @override
  String get location => GoRouteData.$location('/drafts');

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
  String get location => GoRouteData.$location('/settings');

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

mixin $ChatSettingsRoute on GoRouteData {
  static ChatSettingsRoute _fromState(GoRouterState state) =>
      const ChatSettingsRoute();

  @override
  String get location => GoRouteData.$location('/settings/messaging');

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

mixin $AppearanceRoute on GoRouteData {
  static AppearanceRoute _fromState(GoRouterState state) =>
      const AppearanceRoute();

  @override
  String get location => GoRouteData.$location('/settings/appearance');

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

mixin $PrivacyRoute on GoRouteData {
  static PrivacyRoute _fromState(GoRouterState state) => const PrivacyRoute();

  @override
  String get location => GoRouteData.$location('/settings/privacy');

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

mixin $FollowerGrowthRoute on GoRouteData {
  static FollowerGrowthRoute _fromState(GoRouterState state) =>
      const FollowerGrowthRoute();

  @override
  String get location => GoRouteData.$location('/settings/growth');

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

mixin $ProfileCustomisationRoute on GoRouteData {
  static ProfileCustomisationRoute _fromState(GoRouterState state) =>
      const ProfileCustomisationRoute();

  @override
  String get location => GoRouteData.$location('/settings/customisation');

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

mixin $AccountRoute on GoRouteData {
  static AccountRoute _fromState(GoRouterState state) => const AccountRoute();

  @override
  String get location => GoRouteData.$location('/settings/account');

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

mixin $SubscriptionsRoute on GoRouteData {
  static SubscriptionsRoute _fromState(GoRouterState state) =>
      const SubscriptionsRoute();

  @override
  String get location => GoRouteData.$location('/settings/subscriptions');

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

mixin $AccountStandingRoute on GoRouteData {
  static AccountStandingRoute _fromState(GoRouterState state) =>
      const AccountStandingRoute();

  @override
  String get location => GoRouteData.$location('/settings/moderation');

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

mixin $AboutRoute on GoRouteData {
  static AboutRoute _fromState(GoRouterState state) => const AboutRoute();

  @override
  String get location => GoRouteData.$location('/settings/about');

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

mixin $BusinessProductsRoute on GoRouteData {
  static BusinessProductsRoute _fromState(GoRouterState state) =>
      const BusinessProductsRoute();

  @override
  String get location => GoRouteData.$location('/settings/products');

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

mixin $BusinessEventsRoute on GoRouteData {
  static BusinessEventsRoute _fromState(GoRouterState state) =>
      const BusinessEventsRoute();

  @override
  String get location => GoRouteData.$location('/settings/events');

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

mixin $LanguagesRoute on GoRouteData {
  static LanguagesRoute _fromState(GoRouterState state) =>
      const LanguagesRoute();

  @override
  String get location => GoRouteData.$location('/settings/languages');

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

mixin $InstagramMigrationRoute on GoRouteData {
  static InstagramMigrationRoute _fromState(GoRouterState state) =>
      const InstagramMigrationRoute();

  @override
  String get location => GoRouteData.$location('/settings/instagram');

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

mixin $FollowersRoute on GoRouteData {
  static FollowersRoute _fromState(GoRouterState state) =>
      const FollowersRoute();

  @override
  String get location => GoRouteData.$location('/settings/followers');

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

mixin $FollowingRoute on GoRouteData {
  static FollowingRoute _fromState(GoRouterState state) =>
      const FollowingRoute();

  @override
  String get location => GoRouteData.$location('/settings/following');

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

mixin $MutedAccountsRoute on GoRouteData {
  static MutedAccountsRoute _fromState(GoRouterState state) =>
      const MutedAccountsRoute();

  @override
  String get location => GoRouteData.$location('/settings/muted');

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

mixin $BlockedAccountsRoute on GoRouteData {
  static BlockedAccountsRoute _fromState(GoRouterState state) =>
      const BlockedAccountsRoute();

  @override
  String get location => GoRouteData.$location('/settings/blocked');

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

mixin $PostThreadRoute on GoRouteData {
  static PostThreadRoute _fromState(GoRouterState state) => PostThreadRoute(
    did: state.pathParameters['did']!,
    rkey: state.pathParameters['rkey']!,
    focus: state.uri.queryParameters['focus'],
    imageCid: state.uri.queryParameters['image-cid'],
    showComments:
        _$convertMapValue(
          'show-comments',
          state.uri.queryParameters,
          _$boolConverter,
        ) ??
        false,
    $extra: state.extra as Post?,
  );

  PostThreadRoute get _self => this as PostThreadRoute;

  @override
  String get location => GoRouteData.$location(
    '/posts/${Uri.encodeComponent(_self.did)}/${Uri.encodeComponent(_self.rkey)}',
    queryParams: {
      if (_self.focus != null) 'focus': _self.focus,
      if (_self.imageCid != null) 'image-cid': _self.imageCid,
      if (_self.showComments != false)
        'show-comments': _self.showComments.toString(),
    },
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

mixin $PostLikesRoute on GoRouteData {
  static PostLikesRoute _fromState(GoRouterState state) => PostLikesRoute(
    did: state.pathParameters['did']!,
    rkey: state.pathParameters['rkey']!,
  );

  PostLikesRoute get _self => this as PostLikesRoute;

  @override
  String get location => GoRouteData.$location(
    '/posts/${Uri.encodeComponent(_self.did)}/${Uri.encodeComponent(_self.rkey)}/likes',
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

mixin $PostRepostsRoute on GoRouteData {
  static PostRepostsRoute _fromState(GoRouterState state) => PostRepostsRoute(
    did: state.pathParameters['did']!,
    rkey: state.pathParameters['rkey']!,
  );

  PostRepostsRoute get _self => this as PostRepostsRoute;

  @override
  String get location => GoRouteData.$location(
    '/posts/${Uri.encodeComponent(_self.did)}/${Uri.encodeComponent(_self.rkey)}/reposts',
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

mixin $PostQuotesRoute on GoRouteData {
  static PostQuotesRoute _fromState(GoRouterState state) => PostQuotesRoute(
    did: state.pathParameters['did']!,
    rkey: state.pathParameters['rkey']!,
  );

  PostQuotesRoute get _self => this as PostQuotesRoute;

  @override
  String get location => GoRouteData.$location(
    '/posts/${Uri.encodeComponent(_self.did)}/${Uri.encodeComponent(_self.rkey)}/quotes',
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

mixin $BusinessEventRoute on GoRouteData {
  static BusinessEventRoute _fromState(GoRouterState state) =>
      BusinessEventRoute(
        did: state.pathParameters['did']!,
        rkey: state.pathParameters['rkey']!,
      );

  BusinessEventRoute get _self => this as BusinessEventRoute;

  @override
  String get location => GoRouteData.$location(
    '/events/${Uri.encodeComponent(_self.did)}/${Uri.encodeComponent(_self.rkey)}',
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

mixin $ProfileAliasRoute on GoRouteData {
  static ProfileAliasRoute _fromState(GoRouterState state) => ProfileAliasRoute(
    handle: _decodeHandleRouteParameter(state.pathParameters['handle']!),
  );

  ProfileAliasRoute get _self => this as ProfileAliasRoute;

  @override
  String get location => GoRouteData.$location(
    '/profiles/@${Uri.encodeComponent(_encodeHandleRouteParameter(_self.handle))}',
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

mixin $UserProfileRoute on GoRouteData {
  static UserProfileRoute _fromState(GoRouterState state) => UserProfileRoute(
    did: _decodeDidRouteParameter(state.pathParameters['did']!),
    $extra: state.extra as ProfileAvatarTransition?,
  );

  UserProfileRoute get _self => this as UserProfileRoute;

  @override
  String get location => GoRouteData.$location(
    '/profiles/${Uri.encodeComponent(_encodeDidRouteParameter(_self.did))}',
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

mixin $ProfileMutualFollowersRoute on GoRouteData {
  static ProfileMutualFollowersRoute _fromState(GoRouterState state) =>
      ProfileMutualFollowersRoute(
        did: _decodeDidRouteParameter(state.pathParameters['did']!),
      );

  ProfileMutualFollowersRoute get _self => this as ProfileMutualFollowersRoute;

  @override
  String get location => GoRouteData.$location(
    '/profiles/${Uri.encodeComponent(_encodeDidRouteParameter(_self.did))}/mutual-followers',
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

bool _$boolConverter(String value) {
  switch (value) {
    case 'true':
      return true;
    case 'false':
      return false;
    default:
      throw UnsupportedError('Cannot convert "$value" into a bool.');
  }
}

RouteBase get $welcomeRoute => GoRouteData.$route(
  path: '/welcome',
  name: 'welcome',
  hasOverriddenOnExit: false,
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
  hasOverriddenOnExit: false,
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

RouteBase get $addAccountRoute => GoRouteData.$route(
  path: '/add-account',
  name: 'add-account',
  hasOverriddenOnExit: false,
  parentNavigatorKey: AddAccountRoute.$parentNavigatorKey,
  factory: $AddAccountRoute._fromState,
);

mixin $AddAccountRoute on GoRouteData {
  static AddAccountRoute _fromState(GoRouterState state) =>
      const AddAccountRoute();

  @override
  String get location => GoRouteData.$location('/add-account');

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
  hasOverriddenOnExit: false,
  parentNavigatorKey: AuthCompleteRoute.$parentNavigatorKey,
  factory: $AuthCompleteRoute._fromState,
);

mixin $AuthCompleteRoute on GoRouteData {
  static AuthCompleteRoute _fromState(GoRouterState state) => AuthCompleteRoute(
    code: state.uri.queryParameters['code'],
    error: state.uri.queryParameters['error'],
  );

  AuthCompleteRoute get _self => this as AuthCompleteRoute;

  @override
  String get location => GoRouteData.$location(
    '/auth/complete',
    queryParams: {
      if (_self.code != null) 'code': _self.code,
      if (_self.error != null) 'error': _self.error,
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

RouteBase get $onboardingRoute => GoRouteData.$route(
  path: '/onboarding',
  name: 'onboarding',
  hasOverriddenOnExit: false,
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

String _$goRouterHash() => r'94d3f4e4dcf34678c8126ad1c74fe514650668df';
