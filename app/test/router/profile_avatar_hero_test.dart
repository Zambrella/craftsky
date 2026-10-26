import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

final _profile = Profile(
  did: 'did:plc:alice',
  handle: 'alice.craftsky.social',
  displayName: 'Alice',
  crafts: const [],
);

Future<GoRouter> _pumpScene(
  WidgetTester tester,
  Completer<Profile> pending, {
  bool reducedMotion = false,
}) async {
  final router = GoRouter(
    routes: [
      ShellRoute(
        builder: (_, _, child) => child,
        routes: [
          StatefulShellRoute.indexedStack(
            builder: (_, _, shell) => shell,
            branches: [
              StatefulShellBranch(
                routes: [
                  GoRoute(
                    path: '/',
                    builder: (_, _) => Scaffold(
                      body: ListView(
                        children: [
                          for (var i = 0; i < 2; i++)
                            ProfileAccountListTile(
                              account: ProfileAccountSummary(
                                did: _profile.did.toString(),
                                handle: _profile.handle.toString(),
                                displayName: _profile.displayName,
                                isCraftskyProfile: true,
                              ),
                            ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
          GoRoute(
            path: '/profiles/:did',
            builder: (context, state) => UserProfileRoute(
              did: Did.parse(state.pathParameters['did']!),
              $extra: state.extra as ProfileAvatarTransition?,
            ).build(context, state),
          ),
        ],
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        profileRepositoryProvider.overrideWithValue(
          FakeProfileRepository(onFetch: (_) => pending.future),
        ),
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(
            onListByAuthor: (_, {cursor, limit}) async =>
                const PostPage(items: []),
            onListProjectsByAuthor: (_, {cursor, limit}) async =>
                const PostPage(items: []),
            onListCommentsByAuthor: (_, {cursor, limit}) async =>
                const PostPage(items: []),
          ),
        ),
      ],
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData.copyWith(
          platform: TargetPlatform.android,
        ),
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        routerConfig: router,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context)
              .copyWith(disableAnimations: reducedMotion),
          child: child!,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

void main() {
  testWidgets('the selected duplicate avatar flies through nested navigation, '
      'cold loading, and Back', (tester) async {
    final pending = Completer<Profile>();
    final router = await _pumpScene(tester, pending);
    final heroes = tester.widgetList<Hero>(find.byType(Hero)).toList();
    expect(heroes, hasLength(2));
    expect(heroes.first.tag, isNot(heroes.last.tag));
    final sourceRect = tester.getRect(find.byType(ProfileAvatar).last);
    await tester.tap(find.byType(ProfileAccountListTile).last);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));

    final flight = find.byKey(const Key('profile-avatar-flight'));
    expect(flight, findsOneWidget);
    expect(
      (router.state.extra! as ProfileAvatarTransition).tag,
      heroes.last.tag,
    );
    final flyingRect = tester.getRect(flight);
    expect(flyingRect.width, greaterThan(sourceRect.width));
    expect(flyingRect.width, lessThan(ProfileAvatarSize.large.outerDimension));
    expect(flyingRect.center, isNot(sourceRect.center));

    // The destination is available before the network request completes.
    await tester.pump(const Duration(milliseconds: 400));
    expect(flight, findsNothing);
    final loadingCenter = tester.getCenter(find.byType(ProfileFramedAvatar));
    pending.complete(_profile);
    await tester.pumpAndSettle();
    expect(tester.getCenter(find.byType(ProfileFramedAvatar)), loadingCenter);
    expect(find.byType(ProfileTabBar), findsOneWidget);

    await tester.pageBack();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(flight, findsOneWidget);
    expect(
      tester.getSize(flight).width,
      lessThan(ProfileAvatarSize.large.outerDimension),
    );
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/');
    expect(find.byType(ProfileAccountListTile), findsNWidgets(2));
    expect(flight, findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('a response during the flight does not paint a second avatar', (
    tester,
  ) async {
    final pending = Completer<Profile>();
    await _pumpScene(tester, pending);
    await tester.tap(find.byType(ProfileAccountListTile).first);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(find.byKey(const Key('profile-avatar-flight')), findsOneWidget);
    pending.complete(_profile);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 16));
    expect(find.byKey(const Key('profile-avatar-flight')), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(ProfileFramedAvatar),
        matching: find.byType(ProfileAvatar),
      ),
      findsNothing,
      reason: 'the destination stays hidden until its flying avatar lands',
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('profile-avatar-flight')), findsNothing);
    expect(
      find.descendant(
        of: find.byType(ProfileFramedAvatar),
        matching: find.byType(ProfileAvatar),
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('reduced motion opens the profile without an avatar flight', (
    tester,
  ) async {
    final pending = Completer<Profile>();
    final router = await _pumpScene(tester, pending, reducedMotion: true);
    await tester.tap(find.byType(ProfileAccountListTile).first);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(router.state.extra, isNull);
    expect(find.byKey(const Key('profile-avatar-flight')), findsNothing);
    pending.complete(_profile);
    await tester.pumpAndSettle();
    expect(find.byType(ProfileTabBar), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
