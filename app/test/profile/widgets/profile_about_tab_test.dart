import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_about_tab.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

const _cid = 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq';

void main() {
  testWidgets('AT-002 About renders every hydrated business detail', (
    tester,
  ) async {
    await _pumpAbout(
      tester,
      BusinessProfile(
        cid: _cid,
        businessTypes: const [
          BusinessOpenValue(value: 'dyer', known: true),
          BusinessOpenValue(value: 'loom-consulting', known: false),
        ],
        offerings: const [
          BusinessOpenValue(value: 'yarn', known: true),
          BusinessOpenValue(value: 'fiber-processing', known: true),
        ],
        location: const BusinessLocation(country: 'GB', locality: 'Bristol'),
        serviceArea: 'Ships across the UK',
        hoursNote: 'Tuesday to Saturday, 10–4',
      ),
    );

    expect(find.text('Business types'), findsOneWidget);
    expect(find.text('Dyer'), findsOneWidget);
    expect(find.text('Other: Loom consulting'), findsOneWidget);
    expect(find.text('Offerings'), findsOneWidget);
    expect(find.text('Yarn'), findsOneWidget);
    expect(find.text('Fiber processing'), findsOneWidget);
    expect(find.text('Location'), findsOneWidget);
    expect(find.text('Bristol, United Kingdom'), findsOneWidget);
    expect(find.text('Service area'), findsOneWidget);
    expect(find.text('Ships across the UK'), findsOneWidget);
    expect(find.text('Hours'), findsOneWidget);
    expect(find.text('Tuesday to Saturday, 10–4'), findsOneWidget);
  });

  testWidgets('AT-002 About omits absent business sections', (tester) async {
    await _pumpAbout(tester, BusinessProfile(cid: _cid));

    expect(find.text('Business types'), findsNothing);
    expect(find.text('Offerings'), findsNothing);
    expect(find.text('Location'), findsNothing);
    expect(find.text('Service area'), findsNothing);
    expect(find.text('Hours'), findsNothing);
  });
}

Future<void> _pumpAbout(
  WidgetTester tester,
  BusinessProfile business,
) => tester.pumpWidget(
  MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(
      body: CustomScrollView(
        slivers: [
          ProfileAboutTab(
            profile: Profile(
              did: 'did:plc:maker',
              handle: 'maker.test',
              crafts: const [],
              accountType: AccountType.business,
              business: business,
            ),
          ),
        ],
      ),
    ),
  ),
);
