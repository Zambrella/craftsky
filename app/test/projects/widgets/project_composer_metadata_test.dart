import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-007 hides pattern fields until Pattern expands', (
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

    expect(find.text('Pattern'), findsOneWidget);
    expect(find.text('Name'), findsNothing);
    expect(find.text('Link'), findsNothing);
    expect(find.text('Pattern difficulty'), findsNothing);
    expect(find.text('Designer'), findsNothing);
    expect(find.text('Publisher'), findsNothing);

    await tester.ensureVisible(find.text('Pattern'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Pattern'));
    await tester.pumpAndSettle();

    expect(find.text('Name'), findsOneWidget);
    expect(find.text('Link'), findsOneWidget);
    expect(find.text('Pattern difficulty'), findsOneWidget);
    expect(find.text('Designer'), findsOneWidget);
    expect(find.text('Publisher'), findsOneWidget);
  });

  testWidgets('AT-007 colour metadata enforces count limits and removal', (
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

    final firstTen = ProjectOptionCatalogs.colours.take(10).toList();
    final eleventh = ProjectOptionCatalogs.colours[10];

    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
    );
    await tester.pumpAndSettle();

    for (final option in firstTen) {
      await _searchAndTapColour(tester, option);
    }

    expect(find.text(firstTen.first.label), findsWidgets);

    await _searchAndTapColour(tester, eleventh);

    expect(find.text('You can choose up to 10.'), findsOneWidget);
    expect(find.text(eleventh.label), findsWidgets);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();

    final firstRemoveButton = find.byKey(
      Key('${ProjectComposerFields.colours}-remove-${firstTen.first.value}'),
    );
    await tester.ensureVisible(firstRemoveButton);
    await tester.pumpAndSettle();
    await tester.tap(firstRemoveButton);
    await tester.pumpAndSettle();

    await _searchAndTapColour(tester, eleventh);

    expect(find.text('You can choose up to 10.'), findsNothing);
    expect(find.text(eleventh.label), findsWidgets);
  });
}

Future<void> _searchAndTapColour(
  WidgetTester tester,
  ProjectOption option,
) async {
  final searchInput = find.byKey(
    const Key('${ProjectComposerFields.colours}-search-input'),
  );
  final optionFinder = find.byKey(
    Key('${ProjectComposerFields.colours}-option-${option.value}'),
  );
  await tester.ensureVisible(searchInput);
  await tester.pumpAndSettle();
  await tester.tap(searchInput);
  await tester.pump();
  await tester.enterText(searchInput, option.label);
  await tester.pumpAndSettle();
  await tester.tap(optionFinder);
  await tester.pumpAndSettle();
}
