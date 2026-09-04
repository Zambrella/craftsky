import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('project composer keeps the complete posting flow on one page', (
    tester,
  ) async {
    await _pumpComposer(tester);

    await tester.ensureVisible(
      find.byKey(const Key('project-composer-body-editor')),
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const Key('project-composer-body-editor')),
      findsOneWidget,
    );
    await tester.ensureVisible(find.text('Add more details'));
    await tester.pumpAndSettle();
    expect(find.text('Add more details'), findsOneWidget);
    expect(find.text('Materials and style'), findsOneWidget);
    expect(find.text('When'), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Post'), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Next'), findsNothing);
  });

  testWidgets('pattern details are disclosed as a subpage', (tester) async {
    await _pumpComposer(tester);

    expect(find.text('Pattern details'), findsNothing);

    await tester.enterText(
      find.descendant(
        of: find.byKey(const Key('project-composer-pattern-name-editor')),
        matching: find.byType(TextField),
      ),
      '#SockKAL',
    );
    await tester.pumpAndSettle();

    expect(find.text('Pattern details'), findsOneWidget);
    expect(find.text('Designer'), findsNothing);

    await tester.ensureVisible(find.text('Pattern details'));
    await tester.tap(find.text('Pattern details'));
    await tester.pumpAndSettle();

    expect(find.text('Designer'), findsOneWidget);
    expect(find.text('Publisher'), findsOneWidget);
    expect(find.text('Link'), findsOneWidget);
    expect(find.text('Difficulty'), findsOneWidget);
  });

  testWidgets('detail edits update the summary and Back restores focus', (
    tester,
  ) async {
    await _pumpComposer(tester);
    final action = find.byKey(
      const Key('project-composer-common-details-action'),
    );

    await tester.ensureVisible(action);
    await tester.pumpAndSettle();
    await tester.tap(action);
    await tester.pumpAndSettle();
    final materialField = find.descendant(
      of: find.byKey(const Key('materials-custom-input')),
      matching: find.byType(TextField),
    );
    await tester.enterText(materialField, 'Merino wool');
    await tester.pump();
    await tester.tap(find.byKey(const Key('materials-add-custom')));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(action);
    await tester.pumpAndSettle();
    final tile = tester.widget<ListTile>(
      find.descendant(of: action, matching: find.byType(ListTile)),
    );
    expect((tile.subtitle! as Text).data, '1 detail added');
    expect(tile.focusNode?.hasFocus, isTrue);
  });

  testWidgets('craft details only appear for crafts with extra fields', (
    tester,
  ) async {
    await _pumpComposer(tester);

    await _selectCraft(tester, 'Knitting');
    expect(find.text('Knitting details'), findsOneWidget);

    await _selectCraft(tester, 'Embroidery');
    expect(
      find.byKey(const Key('project-composer-craft-details-action')),
      findsNothing,
    );
  });

  testWidgets('removing a pattern clears populated pattern details', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await _pumpComposer(tester, messenger: messenger);
    final patternName = find.descendant(
      of: find.byKey(const Key('project-composer-pattern-name-editor')),
      matching: find.byType(TextField),
    );
    await tester.enterText(patternName, '#SockKAL');
    await tester.pumpAndSettle();
    final action = find.byKey(
      const Key('project-composer-pattern-details-action'),
    );
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.descendant(
        of: find.byKey(const Key('project-composer-pattern-designer-editor')),
        matching: find.byType(TextField),
      ),
      '@alice.example',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(patternName);
    await tester.enterText(patternName, '#');
    await tester.pumpAndSettle();

    expect(action, findsNothing);
    expect(
      messenger.calls,
      contains(('info', 'Pattern details cleared.', null)),
    );
  });
}

Future<void> _pumpComposer(
  WidgetTester tester, {
  RecordingMessenger? messenger,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        composerImagesProvider(
          'progressive-composer',
        ).overrideWithValue(_readyImagesState),
      ],
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectComposerSheet(composerId: 'progressive-composer'),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craft = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craft);
  await tester.pumpAndSettle();
  await tester.tap(craft);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

const _readyImagesState = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'image-1',
      fileName: 'project.jpg',
      mimeType: 'image/jpeg',
      altText: 'Finished project photo',
      phase: ImageUploaded(
        UploadedDraftImage(cid: 'bafkimage', mime: 'image/jpeg', size: 123),
      ),
    ),
  ],
);
