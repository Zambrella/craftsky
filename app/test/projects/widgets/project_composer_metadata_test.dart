import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_draft_snapshot_adapter.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:crypto/crypto.dart';
import 'package:cupertino_ui/cupertino_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  for (final selfDrafted in [true, false, null]) {
    testWidgets('draft restores self-drafted pattern $selfDrafted', (
      tester,
    ) async {
      const adapter = ProjectDraftSnapshotAdapter();
      final savedFields = adapter.encodeKnownFields({
        ProjectComposerFields.patternDesigner: 'Pattern designer',
        ProjectComposerFields.patternPublisher: 'Pattern publisher',
        ProjectComposerFields.patternSelfDrafted: ?selfDrafted,
      });
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
              home: ProjectComposerSheet(
                draftSeed: _projectDraftSeed(savedFields),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await _openDetail(
        tester,
        const Key('project-composer-pattern-details-action'),
      );

      final toggle = find.byKey(
        const Key('project-composer-self-drafted-switch'),
      );
      expect(
        tester.widget<SwitchListTile>(toggle).value,
        selfDrafted ?? false,
      );

      await tester.ensureVisible(toggle);
      await tester.tap(toggle);
      await tester.pumpAndSettle();
      expect(
        tester.widget<SwitchListTile>(toggle).value,
        !(selfDrafted ?? false),
      );
      await tester.pageBack();
      await tester.pumpAndSettle();
      await _openDetail(
        tester,
        const Key('project-composer-pattern-details-action'),
      );
      expect(
        tester.widget<SwitchListTile>(toggle).value,
        !(selfDrafted ?? false),
      );
    });
  }

  testWidgets('opening a draft preserves one pattern hash prefix', (
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
            home: ProjectComposerSheet(
              composerId: 'pattern-draft-composer',
              draftSeed: _patternDraftSeed(),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.byKey(const Key('project-composer-pattern-details-action')),
      findsOneWidget,
    );
    await _openDetail(
      tester,
      const Key('project-composer-pattern-details-action'),
    );
    final field = tester.widget<TextField>(
      _facetTextField(const Key('project-composer-pattern-name-editor')),
    );

    expect(field.controller?.text, '#SockKAL');
  });

  testWidgets(
    'opening a draft preserves an old subtype as a removable selection',
    (
      tester,
    ) async {
      const composerId = 'page-two-draft-composer';
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
              home: ProjectComposerSheet(
                composerId: composerId,
                draftSeed: _craftDraftSeed(),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Materials and style'), findsNothing);
      expect(find.text('Duration'), findsNothing);
      await _openDetail(
        tester,
        const Key('project-composer-craft-details-action'),
      );

      expect(tester.takeException(), isNull);
      expect(find.widgetWithText(InputChip, 'Knitting'), findsOneWidget);
      expect(find.text('Home goods · Lampshade'), findsOneWidget);
      expect(find.byKey(const Key('project-custom-subtype')), findsNothing);
      await tester.tap(find.byTooltip('Clear Project type'));
      await tester.pumpAndSettle();
      expect(find.text('Home goods · Lampshade'), findsNothing);
    },
  );

  testWidgets(
    'AT-007 pattern details are available without a pattern name',
    (
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
              theme: AppTheme.lightThemeData.copyWith(
                platform: TargetPlatform.iOS,
              ),
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProjectComposerSheet(),
            ),
          ),
        ),
      );

      expect(find.text('Pattern name'), findsNothing);
      expect(find.text('Pattern details'), findsOneWidget);
      expect(find.text('Link'), findsNothing);
      expect(find.text('Difficulty'), findsNothing);
      expect(find.text('Designer'), findsNothing);
      expect(find.text('Publisher'), findsNothing);

      await _openDetail(
        tester,
        const Key('project-composer-pattern-details-action'),
      );

      expect(find.text('Pattern name'), findsOneWidget);
      expect(find.text('Designer'), findsOneWidget);
      expect(find.text('Publisher'), findsOneWidget);
      expect(find.text('Link'), findsOneWidget);
      expect(find.text('Difficulty'), findsOneWidget);
      expect(find.text('Self-drafted pattern'), findsOneWidget);
      final selfDrafted = find.byKey(
        const Key('project-composer-self-drafted-switch'),
      );
      expect(
        find.descendant(of: selfDrafted, matching: find.byType(Switch)),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: selfDrafted,
          matching: find.byType(CupertinoSwitch),
        ),
        findsNothing,
      );
      final themedSwitch = tester.widget<Switch>(
        find.descendant(of: selfDrafted, matching: find.byType(Switch)),
      );
      expect(themedSwitch.value, isFalse);
      expect(
        themedSwitch.activeTrackColor,
        AppTheme.lightThemeData.colorScheme.primary,
      );
      expect(
        themedSwitch.activeThumbColor,
        AppTheme.lightThemeData.colorScheme.onPrimary,
      );
    },
  );

  testWidgets(
    'AT-007 pattern metadata supports hashtag and mention suggestions',
    (
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
            facetAutocompleteDebounceProvider.overrideWithValue(Duration.zero),
            hashtagSuggestionRepositoryProvider.overrideWithValue(
              const MockHashtagSuggestionRepository(
                hashtags: [
                  HashtagSuggestion(tag: 'SockKAL', postCount: 7),
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
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProjectComposerSheet(),
            ),
          ),
        ),
      );

      await _openDetail(
        tester,
        const Key('project-composer-pattern-details-action'),
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

Finder _facetTextField(Key editorKey) {
  return find.descendant(
    of: find.byKey(editorKey),
    matching: find.byType(TextField),
  );
}

Future<void> _openDetail(WidgetTester tester, Key key) async {
  final action = find.byKey(key);
  await tester.ensureVisible(action);
  await tester.pumpAndSettle();
  await tester.tap(action);
  await tester.pumpAndSettle();
}

LocalPostDraftSeed _patternDraftSeed() {
  return _projectDraftSeed(const {
    ProjectComposerFields.patternName: '#SockKAL',
  });
}

LocalPostDraftSeed _craftDraftSeed() {
  const craft = ProjectOptionCatalogs.embroideryCraftToken;
  return _projectDraftSeed({
    ProjectComposerFields.craftType: craft,
    ProjectComposerFields.additionalCrafts: [
      ProjectOptionCatalogs.knittingCraftToken,
    ],
    ProjectComposerFields.projectTypeFor(craft):
        'social.craftsky.project.defs#homeGoods',
    ProjectComposerFields.projectSubtypeFor(craft): 'Lampshade',
    ProjectComposerFields.colours: <dynamic>['blue'],
  }, withMedia: true);
}

LocalPostDraftSeed _projectDraftSeed(
  Map<String, dynamic> knownProjectFieldValues, {
  bool withMedia = false,
}) {
  final bytes = Uint8List.fromList(
    img.encodeJpg(img.Image(width: 1, height: 1)),
  );
  final descriptor = DraftMediaDescriptor(
    mediaId: '00000000-0000-4000-8000-000000000031',
    storageRevision: '00000000-0000-4000-8000-000000000032',
    storageFileName: 'project.jpg',
    displayFileName: 'project.jpg',
    mimeType: 'image/jpeg',
    byteLength: bytes.length,
    sha256: sha256.convert(bytes).toString(),
    width: 1,
    height: 1,
    altText: 'Project image',
    order: 0,
  );
  return LocalPostDraftSeed(
    draft: LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000030',
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.project,
      createdAt: DateTime.utc(2026, 8, 4, 10),
      updatedAt: DateTime.utc(2026, 8, 4, 11),
      content: ProjectDraftContent(
        body: '',
        languages: const ['en'],
        knownProjectFieldValues: knownProjectFieldValues,
      ),
      schedule: const DraftScheduleIntent.now(),
      media: withMedia ? [descriptor] : const [],
    ),
    media: withMedia
        ? [HydratedDraftMedia(descriptor: descriptor, bytes: bytes)]
        : const [],
  );
}
