import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-003 project composer primary fields render', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('sheet-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'sheet-composer'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Project post'), findsOneWidget);
    expect(find.text('Fill in the details about your project'), findsOneWidget);
    expect(find.text('Project title'), findsOneWidget);
    expect(find.text('Add a short project title'), findsOneWidget);
    expect(find.text('Finished'), findsOneWidget);
    expect(find.text('Pattern tag or name'), findsOneWidget);
    expect(find.text('Next'), findsOneWidget);
    expect(find.text('Post'), findsNothing);

    expect(find.byKey(const Key('craftType-select-button')), findsOneWidget);
    expect(find.byKey(const Key('status-select-button')), findsOneWidget);

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    expect(
      find.text(
        'This information is optional but will help others find your project',
      ),
      findsOneWidget,
    );
    expect(find.text('Materials'), findsOneWidget);
    expect(find.text('Search colours'), findsOneWidget);
    expect(find.text('Colours'), findsOneWidget);
    expect(find.text('Search design tags'), findsOneWidget);
    expect(find.text('Design tags'), findsOneWidget);
    expect(find.text('More project details'), findsNothing);

    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    expect(
      find.byKey(const Key('project-composer-body-editor')),
      findsOneWidget,
    );
    expect(find.text('Post'), findsOneWidget);

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
        overrides: [
          composerImagesProvider('focus-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'focus-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    final bodyField = find.descendant(
      of: find.byKey(const Key('project-composer-body-editor')),
      matching: find.byType(TextField),
    );
    await tester.ensureVisible(bodyField);
    await tester.pumpAndSettle();
    await tester.tap(bodyField);
    await tester.pump();
    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isTrue);

    final bodyRect = tester.getRect(find.byType(SafeArea).first);
    await tester.tapAt(Offset(bodyRect.left + 8, bodyRect.top + 8));
    await tester.pump();

    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isFalse);
  });

  testWidgets('AT-003 page navigation resets scroll to top', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('scroll-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'scroll-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    final scrollable = find.byType(Scrollable).first;
    await tester.drag(scrollable, const Offset(0, -500));
    await tester.pumpAndSettle();
    expect(
      tester.state<ScrollableState>(scrollable).position.pixels,
      greaterThan(0),
    );

    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    expect(tester.state<ScrollableState>(scrollable).position.pixels, 0);
  });

  testWidgets('AT-003 hidden pages do not inflate scroll extent', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('extent-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'extent-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    final patternField = find.descendant(
      of: find.byKey(const Key('project-composer-pattern-name-editor')),
      matching: find.byType(TextField),
    );
    await tester.enterText(patternField, '#socks');
    await tester.pumpAndSettle();

    expect(find.text('Pattern info'), findsOneWidget);
    final pageOneMaxScrollExtent = tester
        .state<ScrollableState>(find.byType(Scrollable).first)
        .position
        .maxScrollExtent;

    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    final pageThreeMaxScrollExtent = tester
        .state<ScrollableState>(find.byType(Scrollable).first)
        .position
        .maxScrollExtent;
    expect(pageThreeMaxScrollExtent, lessThan(pageOneMaxScrollExtent));
    expect(pageThreeMaxScrollExtent, lessThan(120));
  });

  testWidgets('AT-003 page two fields advance with tab traversal', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('tab-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'tab-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    final materials = _materialTextField();
    final colors = find.byKey(const Key('colors-search-input'));
    final designTags = find.byKey(const Key('designTags-search-input'));
    final backAction = find.byKey(const Key('project-composer-back-action'));
    final primaryAction = find.byKey(
      const Key('project-composer-primary-action'),
    );

    tester.widget<IconButton>(backAction).focusNode?.requestFocus();
    await tester.pump();
    expect(tester.widget<IconButton>(backAction).focusNode?.hasFocus, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(materials).focusNode?.hasFocus, isTrue);

    await tester.tap(materials);
    await tester.pump();
    expect(tester.widget<TextField>(materials).focusNode?.hasFocus, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(materials).focusNode?.hasFocus, isFalse);
    expect(tester.widget<TextField>(colors).focusNode?.hasFocus, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(colors).focusNode?.hasFocus, isFalse);
    expect(tester.widget<TextField>(designTags).focusNode?.hasFocus, isTrue);

    await tester.tap(designTags);
    await tester.pump();
    expect(tester.widget<TextField>(designTags).focusNode?.hasFocus, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(designTags).focusNode?.hasFocus, isFalse);
    expect(
      tester.widget<TextButton>(primaryAction).focusNode?.hasFocus,
      isTrue,
    );
  });

  testWidgets('AT-003 page three body field advances to post action', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('body-tab-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'body-tab-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    final bodyField = find.descendant(
      of: find.byKey(const Key('project-composer-body-editor')),
      matching: find.byType(TextField),
    );
    final postAction = find.byKey(const Key('project-composer-primary-action'));

    await tester.tap(bodyField);
    await tester.pump();
    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isFalse);
    expect(tester.widget<TextButton>(postAction).focusNode?.hasFocus, isTrue);
  });
}

const _readyImagesState = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'image-1',
      fileName: 'project.jpg',
      mimeType: 'image/jpeg',
      altText: 'Finished project photo',
      phase: ImageUploaded(
        UploadedDraftImage(
          cid: 'bafkimage',
          mime: 'image/jpeg',
          size: 123,
        ),
      ),
    ),
  ],
);

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

Finder _materialTextField() {
  return find.descendant(
    of: find.byKey(const Key('materials-custom-input')),
    matching: find.byType(TextField),
  );
}
