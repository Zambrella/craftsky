import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-003 project composer primary fields render', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(),
          ),
        ),
      ),
    );

    expect(find.text('Project post'), findsOneWidget);
    expect(find.text('Photos'), findsOneWidget);
    expect(find.text('Add a photo'), findsOneWidget);
    expect(find.text('What are you making?'), findsOneWidget);
    expect(find.text('Project title'), findsOneWidget);
    expect(find.text('Craft type'), findsOneWidget);
    expect(find.text('Status'), findsOneWidget);
    expect(find.text('Finished'), findsOneWidget);
    expect(find.text('Materials'), findsOneWidget);
    expect(find.text('Colours'), findsOneWidget);
    expect(find.text('Design tags'), findsOneWidget);
    expect(find.text('Add pattern'), findsOneWidget);
    expect(find.text('More project details'), findsOneWidget);
    expect(find.text('Post'), findsOneWidget);

    const finishedStatusKey = Key(
      'status-radio-social.craftsky.feed.defs#finished',
    );
    final finishedRadio = tester.widget<RadioListTile<String>>(
      find.byKey(finishedStatusKey),
    );
    // Flutter's RadioGroup replacement is still migrating; this verifies the
    // current RadioListTile-backed field keeps the Finished token selected.
    // ignore: deprecated_member_use
    expect(finishedRadio.groupValue, ProjectOptionCatalogs.finishedStatusToken);
  });
}
