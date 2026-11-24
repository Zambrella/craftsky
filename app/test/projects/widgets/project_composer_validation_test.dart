import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_link_preview_carousel.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'AT-002 project composer never requests or offers link previews',
    (
      tester,
    ) async {
      final previews = _CountingPreviewRepository();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            composerImagesProvider('link-project').overrideWithValue(
              _readyImagesState,
            ),
            linkPreviewRepositoryProvider.overrideWithValue(previews),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProjectComposerSheet(composerId: 'link-project'),
            ),
          ),
        ),
      );
      await _selectCraft(tester, 'Embroidery');
      await tester.enterText(
        _bodyTextField(),
        'Project https://example.com/pattern ',
      );
      await tester.pump(const Duration(milliseconds: 100));

      expect(previews.fetchCalls, 0);
      expect(find.byType(ComposerLinkPreviewCarousel), findsNothing);
    },
  );

  testWidgets('AT-006 shows all required errors on the primary page', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(),
          ),
        ),
      ),
    );

    await tester.tap(
      find.byKey(const Key('project-composer-primary-action')),
    );
    await tester.pumpAndSettle();

    expect(find.text('Choose a craft type.'), findsOneWidget);
    expect(find.text('Add at least one photo.'), findsOneWidget);
    expect(find.text('Materials and style'), findsNothing);
  });

  testWidgets('AT-006 blocks final submission without caption text', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          composerImagesProvider('body-validation-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(
              composerId: 'body-validation-composer',
            ),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(find.text('Add body text.'), findsOneWidget);
  });
}

final class _CountingPreviewRepository implements LinkPreviewRepository {
  int fetchCalls = 0;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    fetchCalls += 1;
    return LinkPreview(
      url: url,
      title: 'Unexpected',
      description: 'Unexpected',
    );
  }
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

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

Finder _bodyTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-body-editor')),
    matching: find.byType(TextField),
  );
}
