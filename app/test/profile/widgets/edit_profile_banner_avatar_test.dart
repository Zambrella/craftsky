import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_banner_avatar.dart';
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
    crafts: const [],
  );

  testWidgets('avatar edit button is tappable in the overlap area', (
    tester,
  ) async {
    var tapped = 0;

    await tester.pumpWidget(
      _wrap(
        EditProfileBannerAvatar(
          profile: profile,
          bannerColor: const Color(0xFFCC8866),
          onPickAvatar: () => tapped++,
        ),
      ),
    );

    await tester.tap(find.byTooltip('Change avatar'));
    await tester.pump();

    expect(tapped, 1);
  });

  testWidgets('cover edit button remains tappable', (tester) async {
    var tapped = 0;

    await tester.pumpWidget(
      _wrap(
        EditProfileBannerAvatar(
          profile: profile,
          bannerColor: const Color(0xFFCC8866),
          onPickBanner: () => tapped++,
        ),
      ),
    );

    await tester.tap(find.text('Change cover'));
    await tester.pump();

    expect(tapped, 1);
  });
}
