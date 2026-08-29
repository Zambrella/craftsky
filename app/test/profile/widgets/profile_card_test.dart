import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_card.dart';
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/profile/widgets/profile_route_presentation.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/cupertino.dart'
    show CupertinoPageTransition, CupertinoPageTransitionsBuilder;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../feed/fakes/fake_post_repository.dart';
import '../fakes/fake_profile_repository.dart';

final _emptyPostRepository = FakePostRepository(
  onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
  onListProjectsByAuthor: (_, {cursor, limit}) async =>
      const PostPage(items: []),
  onListCommentsByAuthor: (_, {cursor, limit}) async =>
      const PostPage(items: []),
);

class _MarkedPageTransitionsBuilder extends CupertinoPageTransitionsBuilder {
  const _MarkedPageTransitionsBuilder();

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    if (route.settings is! ProfilePresentationPage) {
      return super.buildTransitions(
        route,
        context,
        animation,
        secondaryAnimation,
        child,
      );
    }
    return KeyedSubtree(
      key: const Key('test-material-page-transition'),
      child: super.buildTransitions(
        route,
        context,
        animation,
        secondaryAnimation,
        child,
      ),
    );
  }
}

Widget _wrap(
  Widget child, {
  List<dynamic> overrides = const [],
  ThemeMode themeMode = ThemeMode.light,
}) {
  return ProviderScope(
    overrides: List.from(overrides),
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      darkTheme: AppTheme.darkThemeData,
      themeMode: themeMode,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: Center(child: child)),
    ),
  );
}

Profile _profile({
  ProfileCustomisation customisation = ProfileCustomisation.defaults,
}) {
  return Profile(
    did: 'did:plc:alice',
    handle: 'alice.craftsky.social',
    displayName: 'Alice',
    description: 'A maker bio.',
    crafts: const ['knitting', 'sewing'],
    createdAt: DateTime.now().subtract(const Duration(days: 370)),
    postsLast7Days: 3,
    projectCount: 12,
    customisation: customisation,
  );
}

Map<String, double> _profileVerticalGaps(
  WidgetTester tester, {
  bool includeTabs = true,
}) {
  final avatar = tester.getRect(find.byType(ProfileFramedAvatar));
  final identity = tester.getRect(find.byType(ProfileIdentity));
  final crafts = tester.getRect(find.byType(ProfileCraftChips));
  final stats = tester.getRect(find.byType(ProfileStats));
  final action = tester.getRect(
    find
        .ancestor(
          of: find.text('Follow'),
          matching: find.byType(ChunkyButton),
        )
        .first,
  );
  final bio = tester.getRect(find.byType(ProfileBio));

  return {
    'avatar to identity': identity.top - avatar.bottom,
    'identity to crafts': crafts.top - identity.bottom,
    'crafts to stats': stats.top - crafts.bottom,
    'stats to actions': action.top - stats.bottom,
    'actions to bio': bio.top - action.bottom,
    if (includeTabs)
      'bio to tabs':
          tester.getRect(find.byType(ProfileTabBar)).top - bio.bottom,
  };
}

void main() {
  group('ProfileCard', () {
    testWidgets(
      'TDD-001 scopes the profile colour and defaults the texture off',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            ProfileCard(
              profile: _profile(
                customisation: const ProfileCustomisation(colour: 'orchid'),
              ),
              isOwnProfile: false,
              onClose: () {},
              onVisitProfile: () {},
              onPrimaryAction: () {},
            ),
          ),
        );

        final header = tester.widget<ColoredBox>(
          find.byKey(const Key('profile-card-header')),
        );
        expect(header.color, const Color(0xFFB615D6));
        expect(
          find.byKey(const Key('profile-card-background-illustration')),
          findsNothing,
        );
        expect(
          find.byKey(const Key('profile-card-avatar-frame')),
          findsNothing,
        );
        expect(
          tester
              .widget<ProfileAvatar>(find.byType(ProfileAvatar))
              .customisation,
          const ProfileCustomisation(colour: 'orchid'),
        );
      },
    );

    testWidgets(
      'TDD-002 renders the selected local texture without a second frame',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            ProfileCard(
              profile: _profile(
                customisation: const ProfileCustomisation(
                  background: 'bayerdark',
                  border: 'thick',
                ),
              ),
              isOwnProfile: false,
              onClose: () {},
              onVisitProfile: () {},
              onPrimaryAction: () {},
            ),
          ),
        );

        expect(
          find.byKey(const Key('profile-card-background-illustration')),
          findsOneWidget,
        );
        expect(
          find.byKey(const Key('profile-card-avatar-frame')),
          findsNothing,
        );
      },
    );

    testWidgets('TDD-002B renders the card avatar without a shadow', (
      tester,
    ) async {
      await tester.pumpWidget(
        _wrap(
          ProfileCard(
            profile: _profile(),
            isOwnProfile: false,
            onClose: () {},
            onVisitProfile: () {},
            onPrimaryAction: () {},
          ),
        ),
      );

      final shadowLayer = tester.widget<DecoratedBox>(
        find.descendant(
          of: find.byType(ProfileAvatar),
          matching: find.byKey(const Key('profile-avatar-shadow')),
        ),
      );
      final avatarDecoration = shadowLayer.decoration as BoxDecoration;

      expect(avatarDecoration.boxShadow, isEmpty);
    });

    testWidgets('compact close button has a neutral contrast surface', (
      tester,
    ) async {
      await tester.pumpWidget(
        _wrap(
          ProfileCard(
            profile: _profile(
              customisation: const ProfileCustomisation(colour: 'amber'),
            ),
            isOwnProfile: false,
            onClose: () {},
            onVisitProfile: () {},
            onPrimaryAction: () {},
          ),
        ),
      );

      final closeButton = tester.widget<IconButton>(
        find.byKey(const Key('profile-card-close')),
      );
      final closeSurface = tester.widget<Material>(
        find.byKey(const Key('profile-card-close-surface')),
      );

      expect(
        closeButton.style?.backgroundColor?.resolve({}),
        Colors.transparent,
      );
      expect(closeSurface.color, AppTheme.lightThemeData.colorScheme.surface);
    });

    testWidgets('compact close button uses a light foreground in dark mode', (
      tester,
    ) async {
      await tester.pumpWidget(
        _wrap(
          ProfileCard(
            profile: _profile(),
            isOwnProfile: false,
            onClose: () {},
            onVisitProfile: () {},
            onPrimaryAction: () {},
          ),
          themeMode: ThemeMode.dark,
        ),
      );

      final closeButton = tester.widget<IconButton>(
        find.byKey(const Key('profile-card-close')),
      );

      expect(
        closeButton.style?.foregroundColor?.resolve({}),
        AppTheme.darkThemeData.colorScheme.onSurface,
      );
    });

    testWidgets('TDD-003 uses the public shared profile presentation widgets', (
      tester,
    ) async {
      await tester.pumpWidget(
        _wrap(
          ProfileCard(
            profile: _profile(),
            isOwnProfile: false,
            onClose: () {},
            onVisitProfile: () {},
            onPrimaryAction: () {},
          ),
        ),
      );

      expect(find.byType(ProfileHeaderBackground), findsOneWidget);
      expect(find.byType(ProfileFramedAvatar), findsOneWidget);
      expect(find.byType(ProfileIdentity), findsOneWidget);
      expect(find.byType(ProfileCraftChips), findsOneWidget);
      expect(find.byType(ProfileStats), findsOneWidget);
    });

    testWidgets(
      'TDD-003A renders current profile summary and visitor actions',
      (tester) async {
        var visitTaps = 0;
        var followTaps = 0;

        await tester.pumpWidget(
          _wrap(
            ProfileCard(
              profile: _profile(),
              isOwnProfile: false,
              onClose: () {},
              onVisitProfile: () => visitTaps++,
              onPrimaryAction: () => followTaps++,
            ),
          ),
        );

        expect(find.text('Alice'), findsOneWidget);
        expect(find.text('@alice.craftsky.social'), findsOneWidget);
        expect(find.text('Knitting'), findsOneWidget);
        expect(find.text('Sewing'), findsOneWidget);
        expect(find.text('1y'), findsOneWidget);
        expect(find.text('here'), findsOneWidget);
        expect(find.text('3 posts'), findsOneWidget);
        expect(find.text('7 days'), findsOneWidget);
        expect(find.text('12'), findsOneWidget);
        expect(find.text('projects'), findsOneWidget);

        await tester.tap(find.text('Visit profile'));
        await tester.tap(find.text('Follow'));

        expect(visitTaps, 1);
        expect(followTaps, 1);
      },
    );

    testWidgets(
      'TDD-003B gives unfollow the quiet main-profile treatment',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            ProfileCard(
              profile: _profile().copyWith(viewerIsFollowing: true),
              isOwnProfile: false,
              onClose: () {},
              onVisitProfile: () {},
              onPrimaryAction: () {},
            ),
          ),
        );

        final unfollowButton = tester.widget<ChunkyButton>(
          find.ancestor(
            of: find.text('Unfollow'),
            matching: find.byType(ChunkyButton),
          ),
        );
        expect(unfollowButton.variant, ChunkyButtonVariant.secondary);
        expect(unfollowButton.backgroundColor, isNull);
        expect(unfollowButton.foregroundColor, isNull);
      },
    );

    testWidgets('TDD-006 remains scrollable on a short viewport', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 480);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final profile = _profile().copyWith(
        crafts: const [
          'knitting',
          'sewing',
          'quilting',
          'crochet',
          'embroidery',
        ],
      );

      await tester.pumpWidget(
        _wrap(
          ProfileCard(
            profile: profile,
            isOwnProfile: false,
            onClose: () {},
            onVisitProfile: () {},
            onPrimaryAction: () {},
          ),
        ),
      );

      expect(tester.takeException(), isNull);
      expect(find.byType(SingleChildScrollView), findsOneWidget);
    });
  });

  group('compact profile route', () {
    testWidgets('TDD-004A loads the complete profile into the card', (
      tester,
    ) async {
      var fetchedKey = '';
      final repository = FakeProfileRepository(
        onFetch: (key) async {
          fetchedKey = key;
          return _profile();
        },
      );

      await tester.pumpWidget(
        _wrap(
          const ProfileRoutePresentation(
            handle: 'alice.craftsky.social',
            startsCompact: true,
          ),
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(SignedInAuthSession.new),
          ],
        ),
      );

      await tester.pumpAndSettle();

      expect(fetchedKey, 'alice.craftsky.social');
      expect(find.byType(ProfileCard), findsOneWidget);
      expect(
        find.byKey(const Key('profile-card-transition-surface')),
        findsOneWidget,
      );
      expect(find.text('Alice'), findsOneWidget);
    });

    testWidgets('TDD-004B follow action updates the open card', (tester) async {
      var followedKey = '';
      final repository = FakeProfileRepository(
        onFetch: (_) async => _profile(),
        onFollow: (key) async {
          followedKey = key;
          return _profile().copyWith(viewerIsFollowing: true);
        },
      );

      await tester.pumpWidget(
        _wrap(
          const ProfileRoutePresentation(
            handle: 'alice.craftsky.social',
            startsCompact: true,
          ),
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(SignedInAuthSession.new),
          ],
        ),
      );

      await tester.pumpAndSettle();
      await tester.tap(find.text('Follow'));
      await tester.pumpAndSettle();

      expect(followedKey, 'alice.craftsky.social');
      expect(find.text('Unfollow'), findsOneWidget);
    });

    testWidgets('TDD-004C own profile card only offers profile navigation', (
      tester,
    ) async {
      final repository = FakeProfileRepository(
        onFetch: (_) async => _profile(),
      );

      await tester.pumpWidget(
        _wrap(
          const ProfileRoutePresentation(
            handle: 'alice.craftsky.social',
            startsCompact: true,
          ),
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(
              () => SignedInAuthSession(did: 'did:plc:alice'),
            ),
          ],
        ),
      );

      await tester.pumpAndSettle();

      expect(find.byType(ProfileCard), findsOneWidget);
      expect(find.text('Visit profile'), findsOneWidget);
      expect(find.text('Edit profile'), findsNothing);
      expect(find.byType(ChunkyButton), findsNothing);
    });

    testWidgets('TDD-005A opening a summary enters the profile route', (
      tester,
    ) async {
      final repository = FakeProfileRepository(
        onFetch: (_) async => _profile(),
      );
      Object? navigationExtra;
      final router = GoRouter(
        initialLocation: '/',
        routes: [
          GoRoute(
            path: '/',
            builder: (context, state) => Scaffold(
              body: Center(
                child: TextButton(
                  onPressed: () => showUserProfileCard(
                    context,
                    handleOrDid: 'alice.craftsky.social',
                  ),
                  child: const Text('Open card'),
                ),
              ),
            ),
          ),
          GoRoute(
            path: '/profile/:handle',
            builder: (context, state) {
              navigationExtra = state.extra;
              return const Scaffold(body: Text('Profile route'));
            },
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(SignedInAuthSession.new),
          ],
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            routerConfig: router,
          ),
        ),
      );

      await tester.tap(find.text('Open card'));
      await tester.pumpAndSettle();

      expect(
        router.state.uri.path,
        '/profile/alice.craftsky.social',
      );
      expect(navigationExtra, isNotNull);
    });

    testWidgets(
      'TDD-005B Visit profile expands without changing route',
      (tester) async {
        final repository = FakeProfileRepository(
          onFetch: (_) async => _profile(),
        );
        Object? navigationExtra;
        late final GoRouter router;
        router = GoRouter(
          initialLocation: '/',
          routes: [
            GoRoute(
              path: '/',
              builder: (context, state) => Scaffold(
                body: Center(
                  child: TextButton(
                    onPressed: () => showUserProfileCard(
                      context,
                      handleOrDid: 'alice.craftsky.social',
                    ),
                    child: const Text('Open card'),
                  ),
                ),
              ),
            ),
            GoRoute(
              path: '/profile/:handle',
              pageBuilder: (context, state) {
                navigationExtra = state.extra;
                final request = state.extra as ProfilePresentationRequest?;
                return ProfilePresentationPage(
                  key: state.pageKey,
                  startsCompact: request?.startsCompact ?? false,
                  child: ProfileRoutePresentation(
                    handle: state.pathParameters['handle']!,
                    startsCompact: request?.startsCompact ?? false,
                  ),
                );
              },
            ),
          ],
        );
        addTearDown(router.dispose);

        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              profileRepositoryProvider.overrideWithValue(repository),
              authSessionProvider.overrideWith(SignedInAuthSession.new),
            ],
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData.copyWith(
                platform: TargetPlatform.iOS,
                pageTransitionsTheme: const PageTransitionsTheme(
                  builders: {
                    TargetPlatform.iOS: _MarkedPageTransitionsBuilder(),
                  },
                ),
              ),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              routerConfig: router,
            ),
          ),
        );

        final originRect = tester.getRect(find.text('Open card'));
        await tester.tap(find.text('Open card'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 120));
        expect(
          tester.getRect(find.text('Open card')),
          originRect,
          reason: 'opening the compact modal must not move its origin route',
        );
        await tester.pumpAndSettle();
        expect(find.byType(ProfileCard), findsOneWidget);
        expect(
          find.byKey(const Key('test-material-page-transition')),
          findsNothing,
        );
        final routeBeforeExpansion = router.state.uri;
        final compactCardRect = tester.getRect(
          find.byKey(const Key('profile-card-transition-surface')),
        );

        await tester.tap(find.text('Visit profile'));
        await tester.pump();

        expect(
          find.byKey(const Key('profile-route-expansion')),
          findsOneWidget,
        );

        await tester.tapAt(const Offset(8, 8));
        await tester.pump();
        expect(router.state.uri, routeBeforeExpansion);

        await tester.pump(
          ProfileRoutePresentation.expansionDuration * 0.5,
        );
        final responsiveSurface = find.byKey(
          const Key('profile-card-transition-surface'),
        );
        expect(responsiveSurface, findsOneWidget);
        expect(
          tester.getRect(responsiveSurface).width,
          greaterThan(compactCardRect.width),
        );
        expect(
          tester.getRect(responsiveSurface).height,
          greaterThan(compactCardRect.height),
        );
        expect(
          tester
              .getRect(
                find.byKey(const Key('profile-card-action-section')),
              )
              .width,
          lessThanOrEqualTo(420),
        );
        final paintedSurface = find
            .descendant(
              of: responsiveSurface,
              matching: find.byType(Material),
            )
            .first;
        expect(
          tester.getRect(paintedSurface).bottom,
          moreOrLessEquals(
            tester.getRect(responsiveSurface).bottom,
            epsilon: 0.01,
          ),
        );
        expect(find.byType(ProfileCard), findsOneWidget);
        for (final label in [
          'Alice',
          '@alice.craftsky.social',
          'Knitting',
          'Sewing',
          'Visit profile',
          'Follow',
        ]) {
          expect(find.text(label), findsOneWidget);
        }
        final opacityAncestors = find
            .ancestor(of: responsiveSurface, matching: find.byType(Opacity))
            .evaluate()
            .map((element) => element.widget)
            .whereType<Opacity>();
        expect(
          opacityAncestors.every((opacity) => opacity.opacity == 1),
          isTrue,
        );
        final expandedOnly = tester.widget<Opacity>(
          find.byKey(const Key('profile-card-expanded-only')),
        );
        expect(expandedOnly.opacity, greaterThan(0));
        expect(expandedOnly.opacity, lessThan(1));
        expect(
          find.descendant(
            of: find.byKey(const Key('profile-card-expanded-only')),
            matching: find.byType(ProfileBio),
          ),
          findsOneWidget,
        );
        expect(
          find.descendant(
            of: find.byKey(const Key('profile-card-expanded-only')),
            matching: find.byType(ProfileTabBar),
          ),
          findsOneWidget,
        );
        expect(find.byType(RawImage), findsNothing);

        await tester.pump(
          ProfileRoutePresentation.expansionDuration * 0.5 -
              const Duration(milliseconds: 1),
        );
        expect(find.byType(ProfileCard), findsOneWidget);
        expect(
          tester
              .getRect(
                find.byKey(const Key('profile-card-transition-surface')),
              )
              .height,
          moreOrLessEquals(
            tester.view.physicalSize.height / tester.view.devicePixelRatio,
            epsilon: 1,
          ),
        );
        expect(
          tester.getRect(paintedSurface).bottom,
          moreOrLessEquals(
            tester
                .getRect(
                  find.byKey(
                    const Key('profile-card-transition-surface'),
                  ),
                )
                .bottom,
            epsilon: 0.01,
          ),
        );
        for (final key in [
          const Key('profile-card-compact-close'),
          const Key('profile-card-compact-visit'),
        ]) {
          final compactOnly = tester.widget<Opacity>(find.byKey(key));
          expect(compactOnly.opacity, lessThan(0.05));
        }
        expect(
          tester
              .widget<Opacity>(
                find.byKey(const Key('profile-card-expanded-only')),
              )
              .opacity,
          greaterThan(0.95),
        );
        final expandedCardGaps = _profileVerticalGaps(tester);

        await tester.pump(const Duration(milliseconds: 2));
        await tester.pump();
        expect(router.state.uri, routeBeforeExpansion);
        final request = navigationExtra! as ProfilePresentationRequest;
        expect(request.startsCompact, isTrue);
        expect(find.byType(ProfileCard), findsNothing);
        expect(find.byKey(const Key('profile-route-expanded')), findsOneWidget);
        final fullProfileGaps = _profileVerticalGaps(tester);
        for (final MapEntry(:key, :value) in expandedCardGaps.entries) {
          expect(
            fullProfileGaps[key],
            moreOrLessEquals(value, epsilon: 1),
            reason:
                '$key spacing should not jump at the handoff: '
                '$expandedCardGaps -> $fullProfileGaps',
          );
        }
        final materialTransition = find.byKey(
          const Key('test-material-page-transition'),
        );
        expect(
          materialTransition,
          findsOneWidget,
          reason:
              'the expanded route must rebuild with its iOS page transition',
        );
        expect(
          find.descendant(
            of: materialTransition,
            matching: find.byType(CupertinoPageTransition),
          ),
          findsOneWidget,
        );

        await tester.dragFrom(
          const Offset(1, 300),
          const Offset(700, 0),
        );
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/');
        expect(find.text('Open card'), findsOneWidget);
        expect(find.byType(ProfileCard), findsNothing);
      },
    );
  });

  testWidgets('TDD-005C direct profile presentation starts expanded', (
    tester,
  ) async {
    final repository = FakeProfileRepository(
      onFetch: (_) async => _profile(),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          postRepositoryProvider.overrideWithValue(_emptyPostRepository),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProfileRoutePresentation(
            handle: 'alice.craftsky.social',
            startsCompact: false,
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byKey(const Key('profile-route-expanded')), findsOneWidget);
    expect(find.byType(ProfileCard), findsNothing);
  });

  testWidgets('TDD-005D final handoff spacing scales with larger text', (
    tester,
  ) async {
    final repository = FakeProfileRepository(
      onFetch: (_) async => _profile(),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          postRepositoryProvider.overrideWithValue(_emptyPostRepository),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: const TextScaler.linear(2)),
            child: child!,
          ),
          home: const ProfileRoutePresentation(
            handle: 'alice.craftsky.social',
            startsCompact: true,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Visit profile'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Visit profile'));
    await tester.pump();
    await tester.pump(
      ProfileRoutePresentation.expansionDuration -
          const Duration(milliseconds: 1),
    );
    final expandedCardGaps = _profileVerticalGaps(
      tester,
      includeTabs: false,
    );

    await tester.pump(const Duration(milliseconds: 2));
    await tester.pump();
    final fullProfileGaps = _profileVerticalGaps(
      tester,
      includeTabs: false,
    );

    for (final MapEntry(:key, :value) in expandedCardGaps.entries) {
      expect(
        fullProfileGaps[key],
        moreOrLessEquals(value, epsilon: 1),
        reason: '$key spacing should remain stable with larger text',
      );
    }
  });

  testWidgets('TDD-005E full profile presentation uses a Material route', (
    tester,
  ) async {
    late Route<void> route;

    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) {
            route = const ProfilePresentationPage(
              startsCompact: false,
              child: SizedBox.shrink(),
            ).createRoute(context);
            return const SizedBox.shrink();
          },
        ),
      ),
    );

    expect(route, isA<MaterialRouteTransitionMixin<void>>());
  });
}
