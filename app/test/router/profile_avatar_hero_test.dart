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
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_hero.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/rendering.dart';
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
  Profile? profile,
  double textScale = 1,
  TargetPlatform platform = TargetPlatform.android,
  bool dark = false,
}) async {
  final identity = profile ?? _profile;
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
                                did: identity.did.toString(),
                                handle: identity.handle.toString(),
                                displayName: identity.displayName,
                                crafts: identity.crafts,
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
      child: RepaintBoundary(
        key: const Key('profile-hero-scene'),
        child: MaterialApp.router(
          theme: (dark ? AppTheme.darkThemeData : AppTheme.lightThemeData)
              .copyWith(
                platform: platform,
              ),
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              disableAnimations: reducedMotion,
              textScaler: TextScaler.linear(textScale),
            ),
            child: child!,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

Future<int> _paintedIconPixels(
  WidgetTester tester,
  Finder icon,
  Color ink,
) async {
  final rect = tester.getRect(icon);
  final boundary = tester.renderObject<RenderRepaintBoundary>(
    find.byKey(const Key('profile-hero-scene')),
  );
  return (await tester.runAsync(() async {
    final image = await boundary.toImage();
    try {
      final pixels = (await image.toByteData())!;
      var count = 0;
      for (var y = rect.top.ceil(); y < rect.bottom.floor(); y++) {
        for (var x = rect.left.ceil(); x < rect.right.floor(); x++) {
          final offset = (y * image.width + x) * 4;
          if ((pixels.getUint8(offset) - ink.r * 255).abs() < 32 &&
              (pixels.getUint8(offset + 1) - ink.g * 255).abs() < 32 &&
              (pixels.getUint8(offset + 2) - ink.b * 255).abs() < 32) {
            count++;
          }
        }
      }
      return count;
    } finally {
      image.dispose();
    }
  }))!;
}

void main() {
  testWidgets('the selected duplicate avatar flies through nested navigation, '
      'cold loading, and Back', (tester) async {
    final pending = Completer<Profile>();
    final router = await _pumpScene(tester, pending);
    final allHeroes = tester.widgetList<Hero>(find.byType(Hero)).toList();
    expect(allHeroes.map((hero) => hero.tag).toSet(), hasLength(6));
    final heroes = tester
        .widgetList<Hero>(
          find.descendant(
            of: find.byType(ProfileAvatarHero),
            matching: find.byType(Hero),
          ),
        )
        .toList();
    expect(heroes, hasLength(2));
    expect(heroes.first.tag, isNot(heroes.last.tag));
    final sourceRect = tester.getRect(find.byType(ProfileAvatar).last);
    await tester.tap(find.byType(ProfileAccountListTile).last);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));

    final flight = find.byKey(const Key('profile-avatar-flight'));
    expect(flight, findsOneWidget);
    expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
    expect(find.byKey(const Key('profile-handle-flight')), findsOneWidget);
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
    final transition = router.state.extra! as ProfileAvatarTransition;
    final nameCenter = tester.getCenter(
      find.byKey(transition.nameDestinationKey),
    );
    final handleCenter = tester.getCenter(
      find.byKey(transition.handleDestinationKey),
    );
    pending.complete(_profile);
    await tester.pumpAndSettle();
    expect(tester.getCenter(find.byType(ProfileFramedAvatar)), loadingCenter);
    expect(
      tester.getCenter(find.byKey(transition.nameDestinationKey)),
      nameCenter,
    );
    expect(
      tester.getCenter(find.byKey(transition.handleDestinationKey)),
      handleCenter,
    );
    expect(find.byType(ProfileTabBar), findsOneWidget);

    await tester.pageBack();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(flight, findsOneWidget);
    expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
    expect(find.byKey(const Key('profile-handle-flight')), findsOneWidget);
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
    expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
    expect(find.byKey(const Key('profile-handle-flight')), findsOneWidget);
    pending.complete(_profile);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 16));
    expect(find.byKey(const Key('profile-avatar-flight')), findsOneWidget);
    expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
    expect(find.byKey(const Key('profile-handle-flight')), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(ProfileFramedAvatar),
        matching: find.byType(ProfileAvatar),
      ),
      findsNothing,
      reason: 'the destination stays hidden until its flying avatar lands',
    );
    for (final text in ['Alice', '@alice.craftsky.social']) {
      expect(
        find.descendant(
          of: find.byType(ProfileIdentity),
          matching: find.text(text),
        ),
        findsNothing,
        reason: 'the arriving text must stay hidden until its hero lands',
      );
    }
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
    expect(find.byKey(const Key('profile-name-flight')), findsNothing);
    expect(find.byKey(const Key('profile-handle-flight')), findsNothing);
    expect(find.byKey(const Key('profile-avatar-flight')), findsNothing);
    pending.complete(_profile);
    await tester.pumpAndSettle();
    expect(find.byType(ProfileTabBar), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'a profile without a display name animates its primary label once',
    (tester) async {
      final profile = _profile.copyWith(displayName: null);
      final pending = Completer<Profile>();
      await _pumpScene(tester, pending, profile: profile);
      await tester.tap(find.byType(ProfileAccountListTile).first);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));
      expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
      expect(find.byKey(const Key('profile-handle-flight')), findsNothing);
      pending.complete(profile);
      await tester.pumpAndSettle();
      expect(find.byType(ProfileIdentity), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'long identity text animates at large text scale on a narrow screen',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final profile = _profile.copyWith(
        displayName: 'Alice the enthusiastic knitter and textile designer',
        handle: 'alice.the.enthusiastic.knitter.craftsky.social',
      );
      final pending = Completer<Profile>();
      await _pumpScene(tester, pending, profile: profile, textScale: 2);
      await tester.tap(find.byType(ProfileAccountListTile).first);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));
      expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
      expect(find.byKey(const Key('profile-handle-flight')), findsOneWidget);
      pending.complete(profile);
      await tester.pumpAndSettle();
      expect(find.byType(ProfileIdentity), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('craft icons grow into labeled pills and shrink on Back', (
    tester,
  ) async {
    final profile = _profile.copyWith(crafts: ['knitting', 'sewing']);
    final pending = Completer<Profile>();
    final router = await _pumpScene(tester, pending, profile: profile);
    final heroes = tester.widgetList<Hero>(find.byType(Hero)).toList();
    expect(heroes.map((hero) => hero.tag).toSet(), hasLength(10));
    final sourceIcon = find.descendant(
      of: find.byType(ProfileAccountListTile).last,
      matching: find.byWidgetPredicate(
        (widget) => widget is CraftIcon && widget.craft == 'knitting',
      ),
    );
    final sourceRect = tester.getRect(sourceIcon);
    await tester.tap(find.byType(ProfileAccountListTile).last);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    for (final craft in profile.crafts) {
      expect(
        find.byKey(ValueKey('profile-craft-flight-$craft')),
        findsOneWidget,
      );
    }
    final knittingFlight = find.byKey(
      const ValueKey('profile-craft-flight-knitting'),
    );
    final flyingRect = tester.getRect(knittingFlight);
    expect(flyingRect.width, greaterThan(sourceRect.width));
    expect(flyingRect.center, isNot(sourceRect.center));
    final labelOpacity = find
        .ancestor(
          of: find.descendant(
            of: knittingFlight,
            matching: find.text('Knitting'),
          ),
          matching: find.byType(Opacity),
        )
        .first;
    final opacity = tester.widget<Opacity>(labelOpacity).opacity;
    expect(opacity, greaterThan(0));
    expect(opacity, lessThan(1));

    // Keep the destination heroes when the fetched profile replaces the header.
    pending.complete(profile);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 16));
    expect(knittingFlight, findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(ProfileCraftChips),
        matching: find.text('Knitting'),
      ),
      findsNothing,
    );
    await tester.pumpAndSettle();
    expect(knittingFlight, findsNothing);
    expect(find.text('Knitting'), findsOneWidget);
    expect(find.text('Sewing'), findsOneWidget);
    final transition = router.state.extra! as ProfileAvatarTransition;
    final chipWidth = tester
        .getSize(find.byKey(transition.craftDestinationKey('knitting')))
        .width;
    expect(chipWidth, greaterThan(flyingRect.width));

    await tester.pageBack();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(knittingFlight, findsOneWidget);
    final returningWidth = tester.getSize(knittingFlight).width;
    expect(returningWidth, lessThan(chipWidth));
    expect(returningWidth, greaterThan(sourceRect.width));
    await tester.pumpAndSettle();
    expect(knittingFlight, findsNothing);
    expect(find.byType(ProfileAccountListTile), findsNWidgets(2));
    expect(tester.takeException(), isNull);
  });

  testWidgets('overflowed crafts stay counted and only visible icons fly', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final profile = _profile.copyWith(
      displayName: 'Hadewijch',
      crafts: [
        'knitting',
        'crochet',
        'sewing',
        'quilting',
        'embroidery',
        'unknown',
      ],
    );
    final pending = Completer<Profile>();
    await _pumpScene(tester, pending, profile: profile, textScale: 1.5);
    final tile = find.byType(ProfileAccountListTile).first;
    final visibleCrafts = tester
        .widgetList<CraftIcon>(
          find.descendant(of: tile, matching: find.byType(CraftIcon)),
        )
        .map((icon) => icon.craft)
        .toSet();
    expect(visibleCrafts, isNotEmpty);
    expect(visibleCrafts.length, lessThan(profile.crafts.length));
    expect(
      find.descendant(
        of: tile,
        matching: find.text('+${profile.crafts.length - visibleCrafts.length}'),
      ),
      findsOneWidget,
    );
    await tester.tap(tile);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    for (final craft in profile.crafts) {
      expect(
        find.byKey(ValueKey('profile-craft-flight-$craft')),
        visibleCrafts.contains(craft) ? findsOneWidget : findsNothing,
      );
    }
    pending.complete(profile);
    await tester.pumpAndSettle();
    expect(find.text('Unknown'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'craft morphs respect reduced motion',
    (tester) async {
      final profile = _profile.copyWith(
        crafts: ['knitting', 'knitting', 'crochet'],
      );
      final pending = Completer<Profile>();
      final router = await _pumpScene(
        tester,
        pending,
        profile: profile,
        reducedMotion: true,
      );
      await tester.tap(find.byType(ProfileAccountListTile).first);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));
      expect(router.state.extra, isNull);
      expect(
        find.byKey(const ValueKey('profile-craft-flight-knitting')),
        findsNothing,
      );
      pending.complete(profile);
      await tester.pumpAndSettle();
      expect(find.text('Knitting'), findsNWidgets(2));
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'duplicate normalized craft IDs share a single flight per source',
    (tester) async {
      final profile = _profile.copyWith(
        crafts: ['knitting', 'KNITTING', 'crochet'],
      );
      final pending = Completer<Profile>();
      await _pumpScene(tester, pending, profile: profile);
      await tester.tap(find.byType(ProfileAccountListTile).last);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));
      expect(
        find.byKey(const ValueKey('profile-craft-flight-knitting')),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('profile-craft-flight-crochet')),
        findsOneWidget,
      );
      pending.complete(profile);
      await tester.pumpAndSettle();
      expect(find.text('Knitting'), findsOneWidget);
      expect(find.text('KNITTING'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  for (final dark in [false, true]) {
    testWidgets(
      'iOS returning craft stays visible as its background fades (dark: $dark)',
      (tester) async {
        final profile = _profile.copyWith(crafts: ['knitting']);
        final pending = Completer<Profile>()..complete(profile);
        await _pumpScene(
          tester,
          pending,
          profile: profile,
          platform: TargetPlatform.iOS,
          dark: dark,
        );
        final tile = find.byType(ProfileAccountListTile).last;
        final sourceIcon = find.descendant(
          of: tile,
          matching: find.byType(CraftIcon),
        );
        final sourceColor = IconTheme.of(tester.element(sourceIcon)).color!;
        final surface = Theme.of(tester.element(sourceIcon))
            .scaffoldBackgroundColor;
        await tester.tap(tile);
        await tester.pumpAndSettle();
        await tester.pageBack();
        await tester.pump();
        final flight = find.byKey(
          const ValueKey('profile-craft-flight-knitting'),
        );
        for (var i = 0; i < 6; i++) {
          await tester.pump(const Duration(milliseconds: 40));
          expect(flight, findsOneWidget);
          final icon = find.descendant(
            of: flight,
            matching: find.byType(CraftIcon),
          );
          final iconRect = tester.getRect(icon);
          expect(iconRect.width, closeTo(16, .1));
          expect(iconRect.height, closeTo(16, .1));
          final background = tester
              .widget<ColoredBox>(
                find.descendant(of: flight, matching: find.byType(ColoredBox)),
              )
              .color;
          final foreground = tester.widget<CraftIcon>(icon).color!;
          final backdrop = Color.alphaBlend(
            background,
            surface,
          ).computeLuminance();
          final ratio =
              (foreground.computeLuminance() + .05) / (backdrop + .05);
          expect(ratio >= 1 ? ratio : 1 / ratio, greaterThanOrEqualTo(3));
          if (i == 1 || i == 4) {
            expect(
              await _paintedIconPixels(tester, icon, foreground),
              greaterThan(8),
              reason: 'the SVG glyph must paint before the return finishes',
            );
          }
          if (i >= 2) {
            expect(
              background.a,
              lessThan(.1),
              reason:
                  'the chip background should disappear early '
                  'in the return flight',
            );
            expect(tester.widget<CraftIcon>(icon).color, sourceColor);
          }
        }
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('iOS swipe-back sheds the chip before its icon lands', (
    tester,
  ) async {
    final profile = _profile.copyWith(crafts: ['knitting']);
    final pending = Completer<Profile>()..complete(profile);
    final router = await _pumpScene(
      tester,
      pending,
      profile: profile,
      platform: TargetPlatform.iOS,
    );
    await tester.tap(find.byType(ProfileAccountListTile).last);
    await tester.pumpAndSettle();
    final gesture = await tester.startGesture(const Offset(1, 300));
    await gesture.moveBy(const Offset(20, 0));
    await tester.pump();
    await gesture.moveBy(Offset(tester.view.physicalSize.width * .6, 0));
    await tester.pump();
    final flight = find.byKey(const ValueKey('profile-craft-flight-knitting'));
    expect(flight, findsOneWidget);
    final background = tester
        .widget<ColoredBox>(
          find.descendant(of: flight, matching: find.byType(ColoredBox)),
        )
        .color;
    expect(background.a, 0);
    final icon = find.descendant(of: flight, matching: find.byType(CraftIcon));
    expect(tester.getSize(icon), const Size(16, 16));
    await gesture.up();
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/');
    expect(find.byType(ProfileAccountListTile), findsNWidgets(2));
    expect(tester.takeException(), isNull);
  });

  testWidgets('craft flight allows its icon to exceed a small animation box', (
    tester,
  ) async {
    final profile = _profile.copyWith(crafts: ['knitting']);
    final pending = Completer<Profile>()..complete(profile);
    final router = await _pumpScene(tester, pending, profile: profile);
    final source = tester.element(
      find.descendant(
        of: find.byType(ProfileCraftHero).last,
        matching: find.byType(Hero),
      ),
    );
    await tester.tap(find.byType(ProfileAccountListTile).last);
    await tester.pumpAndSettle();
    final transition = router.state.extra! as ProfileAvatarTransition;
    final destination = tester.element(
      find.byKey(transition.craftDestinationKey('knitting')),
    );
    final hero = destination.widget as Hero;
    final shuttle = hero.flightShuttleBuilder!(
      destination,
      const AlwaysStoppedAnimation(.5),
      HeroFlightDirection.push,
      source,
      destination,
    );
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        home: Scaffold(
          body: Center(child: SizedBox(width: 20, height: 20, child: shuttle)),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    expect(tester.getSize(find.byType(CraftIcon)), const Size(16, 16));
  });
}
