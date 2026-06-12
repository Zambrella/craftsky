import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'AT-007 hides pattern fields until pattern tag or name is filled',
    (
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

      expect(find.text('Pattern tag or name'), findsOneWidget);
      expect(find.text('Pattern info'), findsNothing);
      expect(find.text('Link'), findsNothing);
      expect(find.text('Difficulty'), findsNothing);
      expect(find.text('Designer'), findsNothing);
      expect(find.text('Publisher'), findsNothing);

      await tester.enterText(
        find.descendant(
          of: find.byKey(const Key('project-composer-pattern-name-editor')),
          matching: find.byType(TextField),
        ),
        '#SockKAL',
      );
      await tester.pumpAndSettle();

      expect(find.text('Pattern info'), findsOneWidget);
      expect(find.text('Designer'), findsOneWidget);
      expect(find.text('Publisher'), findsOneWidget);
      expect(find.text('Link'), findsOneWidget);
      expect(find.text('Difficulty'), findsOneWidget);
    },
  );

  testWidgets('AT-007 colour metadata enforces count limits and removal', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('metadata-colours-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(
              composerId: 'metadata-colours-composer',
            ),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

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

  testWidgets(
    'AT-007 pattern metadata supports hashtag and mention suggestions',
    (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            facetAutocompleteDebounceProvider.overrideWithValue(Duration.zero),
            hashtagSuggestionRepositoryProvider.overrideWithValue(
              const MockHashtagSuggestionRepository(
                hashtags: [
                  HashtagSuggestion(tag: 'SockKAL', postsLast28Days: 7),
                ],
              ),
            ),
            accountSuggestionRepositoryProvider.overrideWithValue(
              const MockAccountSuggestionRepository(
                accounts: [
                  AccountSuggestion(
                    did: 'did:plc:alice',
                    handle: 'alice.craftsky.social',
                    displayName: 'Alice',
                    avatar: null,
                    isCraftskyProfile: true,
                  ),
                ],
              ),
            ),
          ],
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

      await tester.enterText(
        _facetTextField(const Key('project-composer-pattern-name-editor')),
        '#sock',
      );
      await tester.pumpAndSettle();

      expect(find.text('#SockKAL'), findsOneWidget);
      await tester.tap(find.text('#SockKAL'));
      await tester.pumpAndSettle();

      await tester.enterText(
        _facetTextField(const Key('project-composer-pattern-designer-editor')),
        '@ali',
      );
      await tester.pumpAndSettle();

      expect(find.text('Alice'), findsOneWidget);
      expect(find.text('@alice.craftsky.social'), findsOneWidget);
    },
  );
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

Finder _facetTextField(Key editorKey) {
  return find.descendant(
    of: find.byKey(editorKey),
    matching: find.byType(TextField),
  );
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
