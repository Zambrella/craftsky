import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_banner_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_banner.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child),
  );
}

void main() {
  final profile = Profile(
    did: 'did:plc:test',
    handle: 'test.craftsky.social',
    displayName: 'Test User',
    banner: 'https://example.test/banner.jpg',
    crafts: const [],
    customisation: const ProfileCustomisation(
      colour: 'orchid',
    ),
  );

  testWidgets('avatar edit button remains tappable', (
    tester,
  ) async {
    var tapped = 0;

    await tester.pumpWidget(
      _wrap(
        EditProfileBannerAvatar(
          profile: profile,
          onPickAvatar: () => tapped++,
        ),
      ),
    );

    await tester.tap(find.byTooltip('Change avatar'));
    await tester.pump();

    expect(tapped, 1);
    final avatar = tester.widget<ProfileAvatar>(find.byType(ProfileAvatar));
    expect(avatar.customisation.colour, 'orchid');
    expect(avatar.showShadow, isFalse);
  });

  testWidgets('banner is absent and avatar is centered without a shadow', (
    tester,
  ) async {
    await tester.pumpWidget(
      _wrap(
        EditProfileBannerAvatar(
          profile: profile,
        ),
      ),
    );

    final header = find.byType(EditProfileBannerAvatar);
    final avatar = find.byType(ProfileAvatar);
    final shadow = tester.widget<DecoratedBox>(
      find.byKey(const Key('profile-avatar-shadow')),
    );
    expect(find.text('Change cover'), findsNothing);
    expect(find.byType(ProfileBanner), findsNothing);
    expect(tester.getCenter(avatar).dx, tester.getCenter(header).dx);
    expect((shadow.decoration as BoxDecoration).boxShadow, isEmpty);
  });
}
