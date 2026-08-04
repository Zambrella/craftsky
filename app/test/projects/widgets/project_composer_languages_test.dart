import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('IT-027 Project composer starts from Primary', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'fr',
              contentLanguages: ['en'],
            ),
          ),
          composerImagesProvider('project-language').overrideWithValue(
            _readyImages,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(
              composerId: 'project-language',
            ),
          ),
        ),
      ),
    );

    final craft = find.byKey(const Key('craftType-select-button'));
    await tester.ensureVisible(craft);
    await tester.pumpAndSettle();
    await tester.tap(craft);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Embroidery').last);
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('project-composer-primary-action')),
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('project-composer-primary-action')),
    );
    await tester.pumpAndSettle();

    expect(find.text('French'), findsOneWidget);
    expect(find.text('English'), findsNothing);
  });
}

const _readyImages = ComposerImagesState(
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
