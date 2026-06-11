import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
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
    expect(find.text('Pattern name'), findsNothing);
    expect(find.text('Pattern URL'), findsNothing);
    expect(find.text('Pattern difficulty'), findsNothing);
    expect(find.text('Designer'), findsNothing);
    expect(find.text('Publisher'), findsNothing);

    await tester.ensureVisible(find.text('Pattern'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Pattern'));
    await tester.pumpAndSettle();

    expect(find.text('Pattern name'), findsOneWidget);
    expect(find.text('Pattern URL'), findsOneWidget);
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
      await tester.enterText(
        find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
        option.label,
      );
      await tester.pumpAndSettle();
      await tester.tap(
        find.byKey(
          Key('${ProjectComposerFields.colours}-option-${option.value}'),
        ),
      );
      await tester.pumpAndSettle();
    }

    expect(find.text(firstTen.first.label), findsWidgets);

    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
      eleventh.label,
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(
        Key('${ProjectComposerFields.colours}-option-${eleventh.value}'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('You can choose up to 10.'), findsOneWidget);
    expect(find.text(eleventh.label), findsWidgets);

    await tester.tap(
      find.byKey(
        Key('${ProjectComposerFields.colours}-remove-${firstTen.first.value}'),
      ),
    );
    await tester.pumpAndSettle();

    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
      eleventh.label,
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(
        Key('${ProjectComposerFields.colours}-option-${eleventh.value}'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('You can choose up to 10.'), findsNothing);
    expect(find.text(eleventh.label), findsWidgets);
  });
}
