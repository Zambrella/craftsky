import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/about_page.dart';
import 'package:craftsky_app/settings/settings_links.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('About exposes legal links, cache action, and build version', (
    tester,
  ) async {
    Uri? opened;
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MessengerScope(
            messenger: RecordingMessenger(),
            child: AboutPage(
              version: '1.2.3',
              buildNumber: '45',
              linkLauncher: (uri) async {
                opened = uri;
                return true;
              },
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Terms'), findsOneWidget);
    expect(find.text('Privacy policy'), findsOneWidget);
    expect(find.text('Clear image cache'), findsOneWidget);
    expect(find.text('Version'), findsOneWidget);
    expect(find.text('1.2.3 (45)'), findsOneWidget);
    expect(find.byIcon(CraftskyIconsBold.externalLink), findsNWidgets(2));

    await tester.tap(find.text('Terms'));
    await tester.pump();
    expect(opened, settingsTermsUri);
  });
}
