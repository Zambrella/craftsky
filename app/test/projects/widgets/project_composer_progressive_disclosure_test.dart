import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_detail_page.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/projects/widgets/project_pattern_editor.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('project composer fields follow the posting order', (
    tester,
  ) async {
    await _pumpComposer(tester);

    expect(
      find.byKey(const Key('project-composer-craft-details-action')),
      findsOneWidget,
    );
    expect(
      tester
          .widget<ListTile>(find.widgetWithText(ListTile, 'Craft details'))
          .enabled,
      isFalse,
    );
    await _selectCraft(tester, 'Sewing');
    final fields = [
      find.byType(ComposerImageAttachmentSection),
      find.byKey(const Key('craftType-select-button')),
      find.byKey(const Key('project-composer-craft-details-action')),
      find.byKey(const Key('project-composer-pattern-details-action')),
      find.byKey(const Key('status-select-button')),
      find.byKey(const Key('project-title-input')),
      find.byKey(const Key('project-composer-body-editor')),
      find.byKey(const Key('composer-metadata-controls')),
    ];
    for (var index = 1; index < fields.length; index++) {
      expect(
        tester.getTopLeft(fields[index]).dy,
        greaterThan(tester.getTopLeft(fields[index - 1]).dy),
      );
    }
    await tester.ensureVisible(fields.last);
    await tester.pumpAndSettle();
    expect(
      find.text(
        'Tell everyone about your project.\n\n'
        'You can @mention people, use #hashtags or add https://links.com.',
      ),
      findsOneWidget,
    );
    expect(find.text('Add more details'), findsNothing);
    expect(find.text('Materials and style'), findsNothing);
    expect(
      find.byKey(const Key('composer-schedule-control')),
      findsOneWidget,
    );
    expect(find.widgetWithText(ChunkyButton, 'Post'), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Next'), findsNothing);
  });

  testWidgets('pattern details are disclosed as a subpage', (tester) async {
    await _pumpComposer(tester);

    expect(find.text('Pattern details'), findsOneWidget);
    expect(find.text('Pattern name'), findsNothing);
    expect(find.text('Designer'), findsNothing);

    await tester.ensureVisible(find.text('Pattern details'));
    await tester.tap(find.text('Pattern details'));
    await tester.pumpAndSettle();

    final name = _firstPatternName();
    _expectBlankPatternName(tester, name);
    await tester.enterText(name, '#SockKAL');
    await tester.pumpAndSettle();
    expect(find.text('#SockKAL'), findsOneWidget);
    expect(find.text('Designer'), findsOneWidget);
    expect(find.text('Publisher'), findsOneWidget);
    expect(find.text('Add or @mention pattern designer'), findsOneWidget);
    expect(find.text('Add or @mention pattern publisher'), findsOneWidget);
    expect(find.text('Link'), findsOneWidget);
    expect(find.text('Difficulty'), findsOneWidget);
  });

  testWidgets('pattern details can add, edit and remove a second pattern', (
    tester,
  ) async {
    await _pumpComposer(tester);
    final action = find.text('Pattern details');
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    final add = find.text('Add another pattern');
    final remove = find.text('Remove pattern');
    final divider = find.byType(Divider);
    expect(remove, findsNothing);
    expect(divider, findsNothing);
    await tester.ensureVisible(add);
    await tester.tap(add);
    await tester.pumpAndSettle();

    final firstPattern = find.text('Pattern 1');
    final secondPattern = find.text('Pattern 2');
    expect(firstPattern, findsOneWidget);
    expect(secondPattern, findsOneWidget);
    expect(divider, findsOneWidget);
    final dividerY = tester.getTopLeft(divider).dy;
    expect(dividerY, greaterThan(tester.getBottomLeft(firstPattern).dy));
    expect(dividerY, lessThan(tester.getTopLeft(secondPattern).dy));
    expect(remove, findsNWidgets(2));
    final removeIcons = find.byIcon(CraftskyIconsBold.delete);
    expect(removeIcons, findsNWidgets(2));
    for (final element in removeIcons.evaluate()) {
      expect(IconTheme.of(element).color, Theme.of(element).colorScheme.error);
    }
    expect(find.byIcon(CraftskyIconsBold.add), findsOneWidget);
    expect(find.text('Add or @mention pattern designer'), findsNWidgets(2));
    expect(
      find.text('Add or @mention pattern publisher'),
      findsNWidgets(2),
    );
    expect(find.text('I drafted this pattern'), findsNWidgets(2));
    final additionalName = _additionalPatternName();
    _expectBlankPatternName(tester, _firstPatternName());
    _expectBlankPatternName(tester, additionalName);
    await tester.ensureVisible(additionalName);
    await tester.enterText(additionalName, 'Sleeves');
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(
      tester.widget<TextField>(additionalName).controller?.text,
      'Sleeves',
    );

    await tester.ensureVisible(remove.last);
    await tester.tap(remove.last);
    await tester.pumpAndSettle();
    expect(find.text('Pattern 2'), findsNothing);
    expect(remove, findsNothing);
    expect(divider, findsNothing);
  });

  testWidgets(
    'detail actions float over scrolling fields above safe areas',
    (
      tester,
    ) async {
      await _pumpComposer(tester);
      await _selectCraft(tester, 'Sewing');
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 560);
      tester.view.padding = const FakeViewPadding(bottom: 24);
      addTearDown(tester.view.reset);
      await tester.pumpAndSettle();

      final post = find.byKey(const Key('project-composer-primary-action'));
      final postBounds = tester.getRect(post);
      final details = find.byType(ProjectComposerDetailPage);
      final done = find.byKey(const Key('composer-details-done'));
      final scroll = find.descendant(
        of: details,
        matching: find.byType(SingleChildScrollView),
      );
      for (final label in ['Sewing details', 'Pattern details']) {
        final action = find.text(label);
        await tester.ensureVisible(action);
        await tester.tap(action);
        await tester.pumpAndSettle();

        expect(tester.getRect(done), postBounds);
        expect(tester.getRect(scroll).overlaps(postBounds), isTrue);
        if (label == 'Pattern details') {
          await tester.tap(
            find.descendant(
              of: find.byKey(
                const Key('project-composer-pattern-name-editor'),
              ),
              matching: find.byType(TextField),
            ),
          );
          tester.view.viewInsets = const FakeViewPadding(bottom: 210);
          await tester.pumpAndSettle();
          final keyboardDone = find.byKey(const Key('form-keyboard-done'));
          expect(tester.getBottomRight(keyboardDone).dy, lessThan(350));
          await tester.tap(keyboardDone);
          tester.view.resetViewInsets();
          await tester.pumpAndSettle();
          expect(details, findsOneWidget);
          await tester.drag(scroll, const Offset(0, -700));
          await tester.pumpAndSettle();
          expect(
            find.text('Add another pattern').hitTestable(),
            findsOneWidget,
          );
          expect(tester.getRect(done), postBounds);
        }
        await tester.tap(done);
        await tester.pumpAndSettle();
        expect(details, findsNothing);
        expect(post, findsOneWidget);
      }
    },
  );

  testWidgets('knitting methods allow Hand and Machine together', (
    tester,
  ) async {
    await _pumpComposer(tester);
    await _selectCraft(tester, 'Knitting');
    final action = find.text('Knitting details');
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();

    final methods = find.byKey(const Key('Knitting methods-select-button'));
    expect(find.text('Machine gauge (mm)'), findsNothing);
    await tester.ensureVisible(methods);
    await tester.tap(methods);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('Knitting methods-search-input')),
      'MACHINE KNITTING',
    );
    await tester.pumpAndSettle();
    final machine = find.widgetWithText(CheckboxListTile, 'Machine');
    expect(tester.widget<CheckboxListTile>(machine).subtitle, isNull);
    expect(find.widgetWithText(CheckboxListTile, 'Hand'), findsNothing);
    await tester.tap(machine);
    await tester.pumpAndSettle();
    expect(find.text('Machine gauge (mm)'), findsNothing);
    expect(find.text('Needle sizes'), findsNothing);
    await tester.ensureVisible(methods);
    await tester.tap(methods);
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(CheckboxListTile, 'Hand'));
    await tester.pumpAndSettle();
    expect(find.text('Machine gauge (mm)'), findsNothing);
    expect(find.text('Needle sizes'), findsNothing);
    expect(find.widgetWithText(InputChip, 'Hand'), findsOneWidget);
    expect(find.widgetWithText(InputChip, 'Machine'), findsOneWidget);
  });

  testWidgets('detail edits update the summary and Back restores focus', (
    tester,
  ) async {
    await _pumpComposer(tester);
    await _selectCraft(tester, 'Sewing');
    final action = find.byKey(
      const Key('project-composer-craft-details-action'),
    );
    final actionTile = find.descendant(
      of: action,
      matching: find.byType(ListTile),
    );
    String? summary() =>
        (tester.widget<ListTile>(actionTile).subtitle! as Text).data;

    await tester.ensureVisible(action);
    await tester.pumpAndSettle();
    expect(summary(), 'Add other crafts and details about what you made');
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(find.text('Sewing details'), findsOneWidget);
    expect(find.text('What are you making?'), findsNothing);
    expect(find.text('Craft details'), findsNothing);
    final type = find.byKey(const Key('sewingProjectType-select-button'));
    final additional = find.byKey(
      const Key('additionalCrafts-select-button'),
    );
    expect(
      tester.getBottomLeft(type).dy,
      lessThan(tester.getTopLeft(additional).dy),
    );
    await tester.tap(type);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('sewingProjectType-search-input')),
      'garment',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Garment'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    expect(summary(), 'Garment');

    await tester.ensureVisible(action);
    await tester.pumpAndSettle();
    await tester.tap(action);
    await tester.pumpAndSettle();
    await tester.tap(type);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('sewingProjectType-search-input')),
      'dress',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Dress').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    expect(summary(), 'Garment · Dress');
    expect(tester.widget<ListTile>(actionTile).focusNode?.hasFocus, isTrue);
  });

  testWidgets('Spinning opens craft details for additional crafts', (
    tester,
  ) async {
    await _pumpComposer(tester);

    await _selectCraft(tester, 'Spinning');
    final action = find.byKey(
      const Key('project-composer-craft-details-action'),
    );
    final patterns = find.byKey(
      const Key('project-composer-pattern-details-action'),
    );
    expect(patterns, findsNothing);
    expect(find.text('Additional crafts'), findsNothing);
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(find.text('Spinning details'), findsOneWidget);
    expect(find.text('Project type'), findsNothing);
    final additional = find.byKey(
      const Key('additionalCrafts-select-button'),
    );
    await tester.tap(additional);
    await tester.pumpAndSettle();
    final search = find.byKey(const Key('additionalCrafts-search-input'));
    await tester.enterText(search, 'Knitting');
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(CheckboxListTile, 'Knitting'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(InputChip, 'Knitting'), findsOneWidget);
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    expect(patterns, findsOneWidget);
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(find.widgetWithText(InputChip, 'Knitting'), findsOneWidget);
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    await _selectCraft(tester, 'Knitting');
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(find.widgetWithText(InputChip, 'Knitting'), findsNothing);
  });

  testWidgets('Dyeing keeps entered patterns after changing crafts', (
    tester,
  ) async {
    await _pumpComposer(tester);
    final action = find.byKey(
      const Key('project-composer-pattern-details-action'),
    );
    await _selectCraft(tester, 'Dyeing');
    expect(action, findsNothing);

    await _selectCraft(tester, 'Crochet');
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();

    final name = find.descendant(
      of: find.byKey(const Key('project-composer-pattern-name-editor')),
      matching: find.byType(TextField),
    );
    await tester.enterText(name, '#YarnScarf');
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    await _selectCraft(tester, 'Dyeing');
    expect(action, findsOneWidget);
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(tester.widget<TextField>(name).controller!.text, '#YarnScarf');
    await tester.enterText(name, '#');
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    expect(action, findsNothing);
    expect(
      tester
          .widget<ListTile>(find.widgetWithText(ListTile, 'Dyeing details'))
          .focusNode
          ?.hasFocus,
      isTrue,
    );
  });

  for (final (craft, alternative) in [
    ('Weaving', 'draft'),
    ('Felting', 'template'),
  ]) {
    testWidgets('$craft uses $alternative wording on all pattern rows', (
      tester,
    ) async {
      await _pumpComposer(tester);
      await _selectCraft(tester, craft);
      final action = find.text('Pattern or $alternative');
      await tester.ensureVisible(action);
      await tester.tap(action);
      await tester.pumpAndSettle();

      final name = find.text('Pattern or $alternative name');
      expect(name, findsOneWidget);
      final add = find.text('Add another pattern');
      await tester.ensureVisible(add);
      await tester.tap(add);
      await tester.pumpAndSettle();
      expect(name, findsNWidgets(2));
      _expectBlankPatternName(tester, _firstPatternName());
      _expectBlankPatternName(tester, _additionalPatternName());
    });
  }

  testWidgets(
    'clearing a pattern name retains independent pattern details',
    (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      await _pumpComposer(tester, messenger: messenger);
      final patternName = find.descendant(
        of: find.byKey(const Key('project-composer-pattern-name-editor')),
        matching: find.byType(TextField),
      );
      final action = find.byKey(
        const Key('project-composer-pattern-details-action'),
      );
      await tester.ensureVisible(action);
      await tester.tap(action);
      await tester.pumpAndSettle();
      await tester.enterText(patternName, '#SockKAL');
      await tester.pumpAndSettle();
      await tester.enterText(
        find.descendant(
          of: find.byKey(
            const Key('project-composer-pattern-designer-editor'),
          ),
          matching: find.byType(TextField),
        ),
        '@alice.example',
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();

      await tester.ensureVisible(action);
      await tester.tap(action);
      await tester.pumpAndSettle();
      await tester.ensureVisible(patternName);
      await tester.enterText(patternName, '#');
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();

      expect(action, findsOneWidget);
      expect(
        messenger.calls,
        isNot(contains(('info', 'Pattern details cleared.', null))),
      );

      await tester.ensureVisible(action);
      await tester.tap(action);
      await tester.pumpAndSettle();
      final designer = tester.widget<TextField>(
        find.descendant(
          of: find.byKey(
            const Key('project-composer-pattern-designer-editor'),
          ),
          matching: find.byType(TextField),
        ),
      );
      expect(designer.controller?.text, '@alice.example');
    },
  );
}

Finder _firstPatternName() => find.descendant(
  of: find.byKey(const Key('project-composer-pattern-name-editor')),
  matching: find.byType(TextField),
);

Finder _additionalPatternName() => find
    .descendant(
      of: find.byType(ProjectPatternEditor),
      matching: find.byType(TextField),
    )
    .first;

void _expectBlankPatternName(WidgetTester tester, Finder name) {
  final input = tester.widget<TextField>(name);
  expect(input.controller?.text, isEmpty);
  expect(input.decoration?.hintText, '#PatternName');
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
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectComposerSheet(
            composerId: 'progressive-composer',
          ),
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
  await tester.enterText(
    find.byKey(const Key('craftType-search-input')),
    craftLabel,
  );
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
