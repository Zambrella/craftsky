import 'package:craftsky_app/l10n/generated/app_localizations.dart';
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
    expect(find.text('Project title'), findsOneWidget);
    expect(find.text('Add a short project title'), findsOneWidget);
    expect(find.text('Craft type'), findsOneWidget);
    expect(find.text('Project description'), findsOneWidget);
    expect(find.text('Tell everyone about your project'), findsOneWidget);
    expect(find.text('Status'), findsOneWidget);
    expect(find.text('Finished'), findsOneWidget);
    expect(find.text('Materials'), findsOneWidget);
    expect(find.text('Search colours'), findsOneWidget);
    expect(find.text('Colours'), findsOneWidget);
    expect(find.text('Search design tags'), findsOneWidget);
    expect(find.text('Design tags'), findsOneWidget);
    expect(find.text('Pattern'), findsOneWidget);
    expect(find.text('More project details'), findsOneWidget);
    expect(find.text('Post'), findsOneWidget);

    final craftTop = tester.getTopLeft(find.text('Craft type')).dy;
    final bodyTop = tester.getTopLeft(find.text('Project description')).dy;
    final statusTop = tester.getTopLeft(find.text('Status')).dy;
    expect(bodyTop, greaterThan(craftTop));
    expect(bodyTop, lessThan(statusTop));

    expect(find.byKey(const Key('status-select-button')), findsOneWidget);

    final safeArea = tester.widget<SafeArea>(find.byType(SafeArea).first);
    expect(safeArea.bottom, isFalse);
    expect(
      find.byKey(const Key('project-composer-bottom-safe-space')),
      findsOneWidget,
    );
  });

  testWidgets('AT-003 tapping scaffold space clears focused field', (
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

    final bodyField = find.descendant(
      of: find.byKey(const Key('project-composer-body-editor')),
      matching: find.byType(TextField),
    );
    await tester.tap(bodyField);
    await tester.pump();
    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isTrue);

    final bodyRect = tester.getRect(find.byType(SafeArea).first);
    await tester.tapAt(Offset(bodyRect.left + 8, bodyRect.top + 8));
    await tester.pump();

    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isFalse);
  });
}
