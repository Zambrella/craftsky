import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_meta_section.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

const _cid = 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq';

void main() {
  testWidgets('AT-002 shows plain Business, tagline, and exact action', (
    tester,
  ) async {
    final launched = <Uri>[];
    final profile = Profile(
      did: 'did:plc:maker',
      handle: 'maker.test',
      crafts: const [],
      accountType: AccountType.business,
      business: BusinessProfile(
        cid: _cid,
        tagline: 'Small-batch colour for adventurous knitters',
        primaryAction: const BusinessAction(
          type: 'shop',
          destination: 'https://shop.example/products?source=profile#featured',
        ),
      ),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: SingleChildScrollView(
            child: ProfileMetaSection(
              profile: profile,
              isOwnProfile: false,
              actions: VisitorProfileActionSet(
                isFollowing: false,
                isBusy: false,
                onFollowToggle: () {},
                onShare: () {},
                onReport: () {},
                onMuteToggle: () {},
                onBlockToggle: () {},
              ),
              launchExternal: (uri) async {
                launched.add(uri);
                return true;
              },
              confirmExternal: (_, _) async => true,
            ),
          ),
        ),
      ),
    );

    expect(find.text('Business'), findsOneWidget);
    expect(find.byIcon(Icons.verified), findsNothing);
    expect(find.byIcon(Icons.verified_outlined), findsNothing);
    expect(
      find.text('Small-batch colour for adventurous knitters'),
      findsOneWidget,
    );
    expect(find.widgetWithText(OutlinedButton, 'Shop'), findsOneWidget);

    await tester.tap(find.widgetWithText(OutlinedButton, 'Shop'));
    await tester.pump();
    expect(launched, [
      Uri.parse('https://shop.example/products?source=profile#featured'),
    ]);
  });

  testWidgets('AT-002 regular summary has no business presentation', (
    tester,
  ) async {
    final profile = Profile(
      did: 'did:plc:maker',
      handle: 'maker.test',
      crafts: const [],
      accountType: AccountType.regular,
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ProfileMetaSection(
            profile: profile,
            isOwnProfile: true,
            actions: SelfProfileActionSet(
              onEdit: () {},
              onSettings: () {},
            ),
          ),
        ),
      ),
    );

    expect(find.text('Business'), findsNothing);
  });
}
