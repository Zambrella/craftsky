import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_card.dart';
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';
import '../fakes/fake_profile_repository.dart';

Widget _wrap(Widget child, {List<dynamic> overrides = const []}) {
  return ProviderScope(
    overrides: List.from(overrides),
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: Center(child: child)),
    ),
  );
}

Profile _profile() {
  return Profile(
    did: 'did:plc:alice',
    handle: 'alice.craftsky.social',
    displayName: 'Alice',
    crafts: const ['knitting', 'sewing'],
    createdAt: DateTime.now().subtract(const Duration(days: 370)),
    postsLast7Days: 3,
    projectCount: 12,
  );
}

void main() {
  group('ProfileCard', () {
    testWidgets(
      'TDD-001 scopes a supplied primary colour and defaults decorations off',
      (tester) async {
        const customPrimary = Color(0xFF9A4DFF);

        await tester.pumpWidget(
          _wrap(
            ProfileCard(
              profile: _profile(),
              primaryColor: customPrimary,
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
        final framedAvatar = find.byType(ProfileFramedAvatar);
        final avatarRim = tester.widget<DecoratedBox>(
          find
              .descendant(
                of: framedAvatar,
                matching: find.byType(DecoratedBox),
              )
              .first,
        );
        final avatarRimDecoration = avatarRim.decoration as BoxDecoration;
        expect(header.color, customPrimary);
        expect(
          avatarRimDecoration.color,
          Theme.of(tester.element(framedAvatar)).colorScheme.surface,
        );
        expect(
          find.byKey(const Key('profile-card-background-illustration')),
          findsNothing,
        );
        expect(
          find.byKey(const Key('profile-card-avatar-frame')),
          findsNothing,
        );
      },
    );

    testWidgets(
      'TDD-002 renders independently selected curated decorations',
      (tester) async {
        expect(ProfileBackgroundIllustration.values, hasLength(3));
        expect(ProfileAvatarFrame.values, hasLength(3));

        await tester.pumpWidget(
          _wrap(
            ProfileCard(
              profile: _profile(),
              backgroundIllustration: ProfileBackgroundIllustration.botanical,
              avatarFrame: ProfileAvatarFrame.stitched,
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
          findsOneWidget,
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

      final avatarContainers = tester.widgetList<Container>(
        find.descendant(
          of: find.byType(ProfileAvatar),
          matching: find.byType(Container),
        ),
      );
      final avatarDecoration = avatarContainers
          .map((container) => container.decoration)
          .whereType<BoxDecoration>()
          .firstWhere(
            (decoration) => decoration.shape == BoxShape.circle,
          );

      expect(avatarDecoration.boxShadow, isEmpty);
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
        final theme = AppTheme.lightThemeData;

        expect(
          unfollowButton.backgroundColor,
          theme.extension<BrandSwatchTheme>()!.paper3,
        );
        expect(unfollowButton.foregroundColor, theme.colorScheme.onSurface);
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

  group('showUserProfileCard', () {
    testWidgets('TDD-004A loads the complete profile into the modal', (
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
          Builder(
            builder: (context) => TextButton(
              onPressed: () => showUserProfileCard(
                context,
                handleOrDid: 'alice.craftsky.social',
              ),
              child: const Text('Open card'),
            ),
          ),
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(SignedInAuthSession.new),
          ],
        ),
      );

      await tester.tap(find.text('Open card'));
      await tester.pumpAndSettle();

      expect(fetchedKey, 'alice.craftsky.social');
      expect(find.byType(ProfileCard), findsOneWidget);
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
          Builder(
            builder: (context) => TextButton(
              onPressed: () => showUserProfileCard(
                context,
                handleOrDid: 'alice.craftsky.social',
              ),
              child: const Text('Open card'),
            ),
          ),
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(SignedInAuthSession.new),
          ],
        ),
      );

      await tester.tap(find.text('Open card'));
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
          Builder(
            builder: (context) => TextButton(
              onPressed: () => showUserProfileCard(
                context,
                handleOrDid: 'alice.craftsky.social',
              ),
              child: const Text('Open card'),
            ),
          ),
          overrides: [
            profileRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(
              () => SignedInAuthSession(did: 'did:plc:alice'),
            ),
          ],
        ),
      );

      await tester.tap(find.text('Open card'));
      await tester.pumpAndSettle();

      expect(find.byType(ProfileCard), findsOneWidget);
      expect(find.text('Visit profile'), findsOneWidget);
      expect(find.text('Edit profile'), findsNothing);
      expect(find.byType(ChunkyButton), findsNothing);
    });
  });
}
