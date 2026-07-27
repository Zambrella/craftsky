import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/pages/follow_list_page.dart';
import 'package:craftsky_app/settings/pages/relationship_list_page.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  test('Settings list routes use canonical typed locations', () {
    expect(
      const FollowersRoute().location,
      '/profile/settings/followers',
    );
    expect(
      const FollowingRoute().location,
      '/profile/settings/following',
    );
    expect(
      const MutedAccountsRoute().location,
      '/profile/settings/muted',
    );
    expect(
      const BlockedAccountsRoute().location,
      '/profile/settings/blocked',
    );
  });

  for (final routeCase in _routeCases) {
    testWidgets(
      'Settings opens ${routeCase.label} through the production router',
      (tester) async {
        final container = _container();
        addTearDown(container.dispose);
        final routerSubscription = container.listen(
          goRouterProvider,
          (_, _) {},
          fireImmediately: true,
        );
        addTearDown(routerSubscription.close);
        final router = routerSubscription.read()
          ..go(const SettingsRoute().location);

        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MessengerScope(
                messenger: RecordingMessenger(),
                child: FormFactorWidget(child: child!),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();

        await tester.tap(find.text(routeCase.label));
        await tester.pumpAndSettle();

        expect(router.state.matchedLocation, routeCase.location);
        expect(
          find.byWidgetPredicate(routeCase.matchesPage),
          findsOneWidget,
        );
        expect(find.byType(SettingsPage), findsNothing);

        await tester.pageBack();
        await tester.pumpAndSettle();
        expect(find.byType(SettingsPage), findsOneWidget);
      },
    );
  }
}

ProviderContainer _container() => ProviderContainer.test(
  overrides: [
    authSessionProvider.overrideWith(SignedInAuthSession.new),
    onboardingStatusProvider.overrideWith2(
      (_) => CompletedOnboardingStatus(),
    ),
    profileRepositoryProvider.overrideWithValue(
      FakeProfileRepository(
        onFetch: (_) async => Profile(
          did: 'did:plc:test',
          handle: 'test.bsky.social',
          crafts: const [],
        ),
        onListFollowersMe: ({limit, cursor}) async => _emptyAccountPage,
        onListFollowingMe: ({limit, cursor}) async => _emptyAccountPage,
        onListMutedProfiles: ({limit, cursor}) async => _emptyAccountPage,
        onListBlockedProfiles: ({limit, cursor}) async => _emptyAccountPage,
      ),
    ),
    postRepositoryProvider.overrideWithValue(
      FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
      ),
    ),
  ],
);

const _emptyAccountPage = ProfileAccountPage(
  items: [],
  totalCount: 0,
);

final _routeCases = <_SettingsRouteCase>[
  _SettingsRouteCase(
    label: 'Followers',
    location: '/profile/settings/followers',
    matchesPage: (widget) =>
        widget is FollowListPage && widget.kind == FollowListKind.followers,
  ),
  _SettingsRouteCase(
    label: 'Following',
    location: '/profile/settings/following',
    matchesPage: (widget) =>
        widget is FollowListPage && widget.kind == FollowListKind.following,
  ),
  _SettingsRouteCase(
    label: 'Muted accounts',
    location: '/profile/settings/muted',
    matchesPage: (widget) =>
        widget is RelationshipListPage &&
        widget.kind == RelationshipListKind.muted,
  ),
  _SettingsRouteCase(
    label: 'Blocked accounts',
    location: '/profile/settings/blocked',
    matchesPage: (widget) =>
        widget is RelationshipListPage &&
        widget.kind == RelationshipListKind.blocked,
  ),
];

final class _SettingsRouteCase {
  const _SettingsRouteCase({
    required this.label,
    required this.location,
    required this.matchesPage,
  });

  final String label;
  final String location;
  final bool Function(Widget widget) matchesPage;
}
