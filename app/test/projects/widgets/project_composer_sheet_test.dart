import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-003 project composer primary fields render on one page', (
    tester,
  ) async {
    await _pumpComposer(tester, 'sheet-composer');

    expect(find.text('Project post'), findsOneWidget);
    expect(find.text('Project title'), findsOneWidget);
    expect(find.text('Pattern tag or name'), findsOneWidget);
    expect(
      find.byKey(const Key('project-composer-body-editor')),
      findsOneWidget,
    );
    expect(
      find.byKey(const Key('project-composer-common-details-action')),
      findsOneWidget,
    );
    expect(find.widgetWithText(ChunkyButton, 'Post'), findsOneWidget);
    expect(find.text('Next'), findsNothing);
  });

  testWidgets('AT-003 tapping scaffold space clears focused field', (
    tester,
  ) async {
    await _pumpComposer(tester, 'focus-composer');

    final bodyField = _bodyTextField();
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

  testWidgets('craft selector shows branded icons in options and selection', (
    tester,
  ) async {
    await _pumpComposer(tester, 'craft-icon-composer');

    final craft = find.byKey(const Key('craftType-select-button'));
    await tester.ensureVisible(craft);
    await tester.pumpAndSettle();
    await tester.tap(craft);
    await tester.pumpAndSettle();

    expect(find.byType(CraftIcon), findsNWidgets(5));

    await tester.tap(find.text('Knitting').last);
    await tester.pumpAndSettle();

    expect(find.byType(CraftIcon), findsOneWidget);
  });

  testWidgets('AT-003 primary fields use normal widget-order tab traversal', (
    tester,
  ) async {
    await _pumpComposer(tester, 'tab-composer');
    await _selectCraft(tester, 'Embroidery');

    final bodyField = _bodyTextField();
    final detailsAction = find.byKey(
      const Key('project-composer-common-details-action'),
    );
    await tester.ensureVisible(bodyField);
    await tester.pumpAndSettle();
    await tester.tap(bodyField);
    await tester.pump();

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isFalse);
    expect(
      tester
          .widget<ListTile>(
            find.descendant(of: detailsAction, matching: find.byType(ListTile)),
          )
          .focusNode
          ?.hasFocus,
      isTrue,
    );
  });
}

Future<void> _pumpComposer(WidgetTester tester, String composerId) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        composerImagesProvider(composerId).overrideWithValue(_readyImagesState),
      ],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ProjectComposerSheet(composerId: composerId),
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

Finder _bodyTextField() => find.descendant(
  of: find.byKey(const Key('project-composer-body-editor')),
  matching: find.byType(TextField),
);

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
