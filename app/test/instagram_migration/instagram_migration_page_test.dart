import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:archive/archive.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_published_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/pages/instagram_migration_page.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_picker.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/instagram_migration/widgets/instagram_post_import_row.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:image/image.dart' as image;
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';
import '../test_support/deterministic_pump.dart';

void main() {
  testWidgets('IT-016 import controls stay hidden until verification', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final repository = _Repository(
      imports: InstagramImportPage(items: const [], cursor: null),
    );
    final semantics = tester.ensureSemantics();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const InstagramMigrationPage(),
        ),
      ),
    );
    await pumpUntilFound(
      tester,
      find.text('Instagram verification is unavailable right now.'),
      description: 'the unavailable verification state',
    );

    expect(find.text('Instagram'), findsOneWidget);
    expect(find.byKey(const Key('instagram-post-import-open')), findsNothing);
    expect(find.byType(Card), findsNothing);
    expect(find.byType(CraftskyCard), findsWidgets);
    expect(
      find.text('Instagram verification is unavailable right now.'),
      findsOneWidget,
    );
    expect(
      find.text(
        'Imports become available after Instagram verification is '
        'configured and your account is verified.',
      ),
      findsOneWidget,
    );
    expect(
      find.text('Complete verification to import the accounts you follow.'),
      findsOneWidget,
    );
    expect(
      find.byKey(const Key('instagram-import-composer-card')),
      findsNothing,
    );
    expect(find.byKey(const Key('instagram-imports-card')), findsNothing);
    expect(find.byKey(const Key('instagram-suggestions-card')), findsNothing);
    expect(find.byKey(const Key('instagram-import-help')), findsNothing);
    expect(
      tester.widget<CustomScrollView>(find.byType(CustomScrollView)).physics,
      isA<BouncingScrollPhysics>(),
    );
    semantics.dispose();
  });

  testWidgets('unverified Instagram deep links return to verification', (
    tester,
  ) async {
    final router = _instagramRouter();
    addTearDown(router.dispose);
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
    );
    await _pumpMigrationRouter(tester, router, repository);

    for (final location in [
      const InstagramPostImportRoute().location,
      const InstagramFollowingImportRoute().location,
      const InstagramSuggestionsRoute().location,
      const InstagramAccountRoute().location,
    ]) {
      router.go(location);
      await tester.pumpAndSettle();
      expect(
        router.routerDelegate.currentConfiguration.uri.path,
        const InstagramMigrationRoute().location,
      );
      expect(find.text('Create verification challenge'), findsOneWidget);
      expect(
        find.byKey(const Key('instagram-post-import-select-zip')),
        findsNothing,
      );
      expect(
        find.byKey(const Key('instagram-import-composer-card')),
        findsNothing,
      );
      expect(find.byKey(const Key('instagram-suggestions-card')), findsNothing);
      expect(
        find.byKey(const Key('instagram-discoverable-switch')),
        findsNothing,
      );
    }
  });

  testWidgets(
    'an ownership conflict blocks both post and following imports',
    (
      tester,
    ) async {
      final router = _instagramRouter();
      addTearDown(router.dispose);
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'actual_maker',
            discoverable: false,
            conflictPending: true,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 22),
          ),
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
      );
      await _pumpMigrationRouter(tester, router, repository);

      final l10n = AppLocalizations.of(
        tester.element(find.byType(InstagramMigrationPage)),
      );
      expect(find.text(l10n.instagramConflictPending), findsOneWidget);
      await tester.tap(find.byKey(const Key('instagram-post-import-open')));
      await tester.pumpAndSettle();
      expect(
        find.widgetWithText(AppBar, 'Import posts'),
        findsOneWidget,
      );
      expect(
        find.byKey(const Key('instagram-post-import-select-zip')),
        findsNothing,
      );

      router.pop();
      await tester.pumpAndSettle();
      await tester.tap(
        find.byKey(const Key('instagram-following-import-open')),
      );
      await tester.pumpAndSettle();
      expect(find.text(l10n.instagramImportAccountBlocked), findsOneWidget);
      expect(
        find.byKey(const Key('instagram-import-composer-card')),
        findsNothing,
      );
    },
  );

  testWidgets(
    'review checks new rows once and rechecks published rows after viewing',
    (
      tester,
    ) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final directory = Directory.systemTemp.createTempSync(
        'instagram-review-',
      );
      final selected = InstagramPostExportFile(
        directory: directory,
        name: 'example.zip',
      );
      addTearDown(selected.dispose);
      final json = utf8.encode(
        jsonEncode([
          for (var index = 0; index < 21; index++)
            {
              'title': index == 20 ? 'Latest finished quilt' : 'Post $index',
              'creation_timestamp': 1660000000 + index,
              'media': [
                {'uri': 'media/posts/photo-$index.jpg'},
              ],
            },
        ]),
      );
      final zip = Archive()
        ..addFile(
          ArchiveFile(
            'your_instagram_activity/media/posts_1.json',
            json.length,
            json,
          ),
        );
      File(selected.archivePath)
          .writeAsBytesSync(ZipEncoder().encodeBytes(zip));
      final lookups = <List<String>>[];
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'alice_instagram',
            discoverable: false,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 19),
          ),
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
        onLookupPostImports: (keys) async {
          lookups.add(keys);
          if (lookups.length == 2 || lookups.length == 4) {
            throw const ApiNetworkError('offline');
          }
          if (lookups.length > 1) return [];
          return [
            InstagramPublishedImport(
              sourceKey: keys.first,
              uri: AtUri.parse(
                'at://did:plc:alice/social.craftsky.feed.post/3key',
              ),
              isProject: true,
            ),
          ];
        },
      );
      final router = _instagramRouter(withPostThread: true);
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
            instagramPostExportFilePickerProvider.overrideWithValue(
              () async => selected,
            ),
          ],
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            routerConfig: router,
          ),
        ),
      );
      final open = find.byKey(const Key('instagram-post-import-open'));
      await pumpUntilFound(
        tester,
        open,
        description: 'the post importer entry',
      );
      await tester.tap(open);
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Import posts'), findsOneWidget);
      final card = find.byKey(const Key('instagram-post-import-card'));
      final choose = find.byKey(const Key('instagram-post-import-select-zip'));
      expect(tester.widget(choose), isA<FilledButton>());
      expect(find.descendant(of: card, matching: choose), findsOneWidget);
      expect(
        find.descendant(
          of: choose,
          matching: find.text('Select Instagram export'),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: choose,
          matching: find.byIcon(CraftskyIconsBold.openFile),
        ),
        findsOneWidget,
      );
      final l10n = AppLocalizations.of(tester.element(card));
      final instructions = find.text(l10n.instagramPostImportInstructions);
      final privacy = find.text(l10n.instagramPostImportPrivacy);
      expect(
        tester.getBottomLeft(instructions).dy,
        lessThan(tester.getTopLeft(choose).dy),
      );
      expect(
        tester.getBottomLeft(choose).dy,
        lessThan(tester.getTopLeft(privacy).dy),
      );
      await tester.tap(choose);
      // The archive isolate and file operations run outside the fake clock.
      final projectCreated = find.text('Project created');
      for (var i = 0; i < 100 && projectCreated.evaluate().isEmpty; i++) {
        await tester.runAsync(
          () => Future<void>.delayed(const Duration(milliseconds: 10)),
        );
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(lookups.single, hasLength(20));
      expect(lookups.single, everyElement(matches(RegExp(r'^[0-9a-f]{32}$'))));
      await tester.pumpAndSettle();
      expect(projectCreated, findsOneWidget);
      expect(find.text('View project'), findsOneWidget);
      final publishedRow = find.ancestor(
        of: projectCreated,
        matching: find.byType(InstagramPostImportRow),
      );
      final rowCard = tester.widget<CraftskyCard>(
        find.descendant(of: publishedRow, matching: find.byType(CraftskyCard)),
      );
      expect(rowCard.clipBehavior, Clip.none);
      expect(
        find.descendant(
          of: publishedRow,
          matching: find.text('Latest finished quilt'),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(of: publishedRow, matching: find.text('Create post')),
        findsNothing,
      );
      expect(
        find.descendant(
          of: publishedRow,
          matching: find.text('Create project'),
        ),
        findsNothing,
      );

      final more = find.byKey(const Key('instagram-post-import-more'));
      await tester.scrollUntilVisible(more, 500);
      await tester.tap(more);
      await tester.pumpAndSettle();
      expect(lookups.map((keys) => keys.length), [20, 1]);
      expect(lookups.first, isNot(contains(lookups.last.single)));

      final clear = find.byKey(const Key('instagram-post-import-clear'));
      await tester.scrollUntilVisible(clear, -500);
      final retry = find.text('Try again');
      await tester.ensureVisible(retry);
      await tester.tap(retry);
      await tester.pumpAndSettle();
      expect(lookups.map((keys) => keys.length), [20, 1, 1]);
      expect(lookups[1], lookups[2]);
      expect(retry, findsNothing);

      final viewProject = find.text('View project');
      await tester.ensureVisible(viewProject);
      await tester.tap(viewProject);
      await tester.pumpAndSettle();
      expect(find.text('Imported project thread'), findsOneWidget);
      router.pop();
      await tester.pumpAndSettle();
      expect(lookups.map((keys) => keys.length), [20, 1, 1, 21]);
      expect(viewProject, findsOneWidget);

      await tester.scrollUntilVisible(retry, -200);
      await tester.tap(retry);
      await tester.pumpAndSettle();
      expect(lookups[4], lookups[3]);
      expect(lookups[4].first, lookups[0].first);
      expect(viewProject, findsNothing);
      final availableRow = find.ancestor(
        of: find.text('Latest finished quilt'),
        matching: find.byType(InstagramPostImportRow),
      );
      expect(
        find.descendant(of: availableRow, matching: find.text('Create post')),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: availableRow,
          matching: find.text('Create project'),
        ),
        findsOneWidget,
      );
      await tester.scrollUntilVisible(clear, -400);
      await tester.tap(clear);
      await tester.runAsync(() async {
        for (var i = 0; i < 100 && directory.existsSync(); i++) {
          await Future<void>.delayed(const Duration(milliseconds: 10));
        }
        expect(directory.existsSync(), isFalse);
      });
      await tester.pumpAndSettle();
      expect(find.text('Latest finished quilt'), findsNothing);
      expect(choose, findsOneWidget);
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(open, findsOneWidget);
      expect(choose, findsNothing);
    },
  );

  for (final hasVideo in [false, true]) {
    testWidgets(
      hasVideo
          ? 'an overlong import video can be replaced in the composer'
          : 'selected import photos survive other attachment failures',
      (tester) async {
        final initial = registry.SessionRegistry.empty().upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
        final account = initial.activeLease!.session.account;
        final directory = Directory.systemTemp.createTempSync(
          'instagram-review-media-',
        );
        addTearDown(() => directory.deleteSync(recursive: true));
        final selected = InstagramPostExportFile(
          directory: directory.createTempSync(),
          name: 'example.zip',
        );
        final drafts = FileLocalPostDraftRepository(
          documentsRoot: directory.path,
          owner: account,
        );
        final png = image.encodePng(image.Image(width: 12, height: 12));
        final files = hasVideo
            ? {
                'media/posts/long.mp4': [0, 0, 0, 20, ...'ftypisom'.codeUnits],
              }
            : {
                'media/posts/valid.png': png,
                'media/posts/corrupt.png': [1, 2, 3],
                'media/posts/too-wide.png': image.encodePng(
                  image.Image(width: 128, height: 1),
                ),
              };
        final frames = <MethodCall>[];
        if (hasVideo) {
          const channel = MethodChannel('social.craftsky/local_video');
          final messenger =
              TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
                ..setMockMethodCallHandler(
                  channel,
                  (call) async {
                    if (call.method == 'midpointFrame') {
                      frames.add(call);
                      return png;
                    }
                    expect(call.method, 'probe');
                    return {
                      'width': 1080,
                      'height': 1920,
                      'durationMs': 601000,
                      'posterBytes': png,
                    };
                  },
                );
          addTearDown(() {
            messenger.setMockMethodCallHandler(channel, null);
          });
        }

        const caption = 'Keep my original caption';
        final json = utf8.encode(
          jsonEncode([
            {
              'title': caption,
              'creation_timestamp': 1660000000,
              'media': [
                for (final path in files.keys) {'uri': path},
              ],
            },
          ]),
        );
        final zip = Archive()
          ..addFile(
            ArchiveFile.bytes(
              'your_instagram_activity/media/posts_1.json',
              json,
            ),
          );
        for (final entry in files.entries) {
          zip.addFile(ArchiveFile.bytes(entry.key, entry.value));
        }
        File(selected.archivePath)
            .writeAsBytesSync(ZipEncoder().encodeBytes(zip));

        final repository = _verifiedImportRepository([]);
        final router = _instagramRouter();
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              secureSessionRegistryStorageProvider.overrideWithValue(
                _RegistryStorage(initial),
              ),
              instagramMigrationRepositoryProvider.overrideWith(
                (ref, _) async => repository,
              ),
              instagramVerificationStorageProvider.overrideWithValue(
                _EmptyVerificationStorage(),
              ),
              instagramPostExportFilePickerProvider.overrideWithValue(
                () async => selected,
              ),
              accountLocalPostDraftRepositoryProvider(
                account,
              ).overrideWith((ref) async => drafts),
              activeLanguagePreferencesProvider.overrideWith(
                (ref) => const LanguagePreferences(
                  primaryLanguage: 'en',
                  contentLanguages: ['en'],
                ),
              ),
            ],
            child: MessengerScope(
              messenger: RecordingMessenger(),
              child: MaterialApp.router(
                theme: AppTheme.lightThemeData,
                localizationsDelegates: appLocalizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                routerConfig: router,
              ),
            ),
          ),
        );

        Future<void> waitForNativeWork(Finder finder) async {
          for (
            var attempt = 0;
            attempt < 200 && finder.evaluate().isEmpty;
            attempt++
          ) {
            await tester.runAsync(
              () => Future<void>.delayed(const Duration(milliseconds: 10)),
            );
            await tester.pump(const Duration(milliseconds: 10));
          }
          expect(finder, findsOneWidget);
        }

        final open = find.byKey(const Key('instagram-post-import-open'));
        await pumpUntilFound(tester, open, description: 'post importer');
        await tester.tap(open);
        await tester.pumpAndSettle();
        await tester.tap(
          find.byKey(const Key('instagram-post-import-select-zip')),
        );
        if (hasVideo) {
          await waitForNativeWork(
            find.descendant(
              of: find.byType(InstagramPostImportRow),
              matching: find.byType(Image),
            ),
          );
          expect(frames.single.arguments, containsPair('timeUs', 0));
        }
        final create = find.text('Create post');
        final enabledCreate = find.ancestor(
          of: create,
          matching: find.byWidgetPredicate(
            (widget) => widget is TextButton && widget.onPressed != null,
          ),
        );
        await waitForNativeWork(enabledCreate);
        expect(
          find.descendant(
            of: enabledCreate,
            matching: find.byIcon(CraftskyIconsBold.textPost),
          ),
          findsOneWidget,
        );
        expect(
          find.descendant(
            of: find.widgetWithText(TextButton, 'Create project'),
            matching: find.byIcon(CraftskyIconsBold.projectPost),
          ),
          findsOneWidget,
        );
        await tester.pump(const Duration(milliseconds: 400));
        await tester.ensureVisible(create);
        await tester.tap(create);

        final confirm = find.text(
          hasVideo ? 'Continue without media' : 'Continue with 1 photo',
        );
        await waitForNativeWork(confirm);
        await tester.pump(const Duration(milliseconds: 400));
        expect(await tester.runAsync(drafts.list), isEmpty);
        expect(find.byType(PostComposerSheet), findsNothing);
        if (hasVideo) {
          expect(find.textContaining('ten minutes or shorter'), findsOneWidget);
        } else {
          await tester.tap(find.text('Cancel'));
          await waitForNativeWork(enabledCreate);
          await tester.pump(const Duration(milliseconds: 400));
          expect(await tester.runAsync(drafts.list), isEmpty);
          await tester.tap(create);
          await waitForNativeWork(confirm);
          await tester.pump(const Duration(milliseconds: 400));
        }

        await tester.tap(confirm);
        await waitForNativeWork(find.byType(PostComposerSheet));
        await tester.pump(const Duration(milliseconds: 400));
        final saved = (await tester.runAsync(drafts.list))!.single;
        expect((saved.content as StandardDraftContent).text, caption);
        expect(saved.media, hasLength(hasVideo ? 0 : 1));
        expect(saved.video, isNull);
        expect(saved.instagramImport, isNotNull);
        await tester.pumpWidget(const SizedBox.shrink());
      },
      variant: hasVideo
          ? const TargetPlatformVariant({TargetPlatform.iOS})
          : const DefaultTestVariant(),
    );
  }

  testWidgets('IT-023 following imports use the export picker only', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    Uri? helpUri;
    final sentRequests = <InstagramImportRequest>[];
    final messenger = RecordingMessenger();
    final repository = _Repository(
      status: InstagramAccountStatus(
        integrationAvailable: true,
        account: InstagramAccountLink(
          state: InstagramAccountLinkState.active,
          username: 'alice_instagram',
          discoverable: true,
          conflictPending: false,
          reactivationRequired: false,
          verifiedAt: DateTime.utc(2026, 7, 19),
        ),
      ),
      imports: InstagramImportPage(
        items: [
          InstagramImportSummary(
            importId: 'legacy-manual-import',
            state: InstagramImportState.active,
            sourceType: InstagramImportSourceType.manual,
            followingCount: 1,
            createdAt: DateTime.utc(2026, 7, 18),
          ),
        ],
        cursor: null,
      ),
      onCreateImport: (request) async {
        sentRequests.add(request);
        return InstagramImportCreateResult(
          import: InstagramImportSummary(
            importId: 'import-new',
            state: InstagramImportState.active,
            sourceType: request.sourceType,
            followingCount: request.entries.length,
            createdAt: DateTime.utc(2026, 7, 19),
          ),
          followingCount: request.entries.length,
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          instagramHelpLauncherProvider.overrideWithValue((uri) async {
            helpUri = uri;
            return true;
          }),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramExportFilePickerProvider.overrideWithValue(
            () async => const InstagramImportParseResult(
              entries: [InstagramImportEntry(username: 'bobmaker')],
            ),
          ),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(
              view: InstagramMigrationView.following,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final exportHelp = find.byKey(const Key('instagram-import-help'));
    final importCard = find.byKey(const Key('instagram-import-composer-card'));
    final helpTheme = Theme.of(tester.element(exportHelp));
    final helpButton = tester.widget<ChunkyButton>(exportHelp);
    expect(
      helpButton.backgroundColor,
      helpTheme.extension<BrandSwatchTheme>()!.lilac,
    );
    expect(helpButton.foregroundColor, helpTheme.colorScheme.onSurface);
    expect(
      find.descendant(
        of: exportHelp,
        matching: find.byIcon(CraftskyIcons.help),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: exportHelp,
        matching: find.byIcon(CraftskyIcons.externalLink),
      ),
      findsOneWidget,
    );
    expect(find.text('How to export'), findsOneWidget);
    final semantics = tester.ensureSemantics();
    expect(
      find.bySemanticsLabel('How to export your Instagram following'),
      findsOneWidget,
    );
    semantics.dispose();
    expect(find.ancestor(of: exportHelp, matching: importCard), findsNothing);
    expect(tester.getTopLeft(exportHelp).dx, tester.getTopLeft(importCard).dx);
    expect(
      tester.getSize(exportHelp).width,
      lessThan(tester.getSize(importCard).width),
    );
    expect(
      tester.getBottomLeft(exportHelp).dy,
      lessThan(tester.getTopLeft(importCard).dy),
    );
    await tester.ensureVisible(exportHelp);
    await tester.tap(exportHelp);
    await tester.pump();
    expect(helpUri, Uri.https('craftsky.social', '/instagram-import'));

    final cards = tester.widgetList<CraftskyCard>(find.byType(CraftskyCard));
    expect(cards, isNotEmpty);
    expect(cards.every((card) => card.clipBehavior == Clip.none), isTrue);
    expect(find.text('Accounts that follow me'), findsNothing);
    final jsonDescription = find.text(
      'Choose JSON format and All time when exporting from Instagram, then '
      'select the downloaded ZIP here. CraftSky processes it on this device '
      'and uploads only the usernames of accounts you follow. Everything '
      'else stays on your device.',
    );
    expect(find.byType(SegmentedButton<dynamic>), findsNothing);
    expect(find.text('Enter handles'), findsNothing);
    expect(find.byType(TextField), findsNothing);
    expect(jsonDescription, findsOneWidget);
    final selectExport = find.widgetWithText(
      FilledButton,
      'Select Instagram export',
    );
    expect(
      find.text(
        'Importing creates private suggestions only. You choose whether to '
        'follow each account.',
      ),
      findsOneWidget,
    );
    expect(find.text('Manual handles'), findsOneWidget);
    await tester.ensureVisible(selectExport);
    await tester.pumpAndSettle();
    await tester.tap(selectExport);
    await tester.pumpAndSettle();

    expect(
      sentRequests.single.sourceType,
      InstagramImportSourceType.instagramJson,
    );
    expect(sentRequests.single.entries.single.username, 'bobmaker');
    expect(messenger.calls, [('info', 'Instagram import created', null)]);
  });

  testWidgets(
    'IT-023 late export result is discarded after switching away and back',
    (tester) async {
      final initial = registry.SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final picker = Completer<InstagramImportParseResult?>();
      final sentRequests = <InstagramImportRequest>[];
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'synthetic_instagram',
            discoverable: true,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 23),
          ),
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
        onCreateImport: (request) async {
          sentRequests.add(request);
          return InstagramImportCreateResult(
            import: InstagramImportSummary(
              importId: 'unexpected-import',
              state: InstagramImportState.active,
              sourceType: request.sourceType,
              followingCount: request.entries.length,
              createdAt: DateTime.utc(2026, 7, 23),
            ),
            followingCount: request.entries.length,
          );
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
            instagramExportFilePickerProvider.overrideWithValue(
              () => picker.future,
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(
              view: InstagramMigrationView.following,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Select Instagram export'));
      await tester.tap(find.text('Select Instagram export'));
      await tester.pump();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(InstagramMigrationPage)),
      );
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      await tester.pump();
      final aliceLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:alice'))!;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceLease);
      await tester.pump();
      picker.complete(
        const InstagramImportParseResult(
          entries: [InstagramImportEntry(username: 'private_alice_only')],
        ),
      );
      await tester.pumpAndSettle();

      expect(sentRequests, isEmpty);
    },
  );

  testWidgets('IT-023 cancellation and safe export errors stay local', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var picker = () async => null as InstagramImportParseResult?;
    final sentRequests = <InstagramImportRequest>[];
    final repository = _verifiedImportRepository(sentRequests);

    await _pumpVerifiedExportPage(
      tester,
      initial: initial,
      repository: repository,
      picker: () => picker(),
    );

    final selectExport = find.widgetWithText(
      FilledButton,
      'Select Instagram export',
    );
    await tester.tap(selectExport);
    await tester.pumpAndSettle();

    expect(sentRequests, isEmpty);
    expect(tester.widget<FilledButton>(selectExport).onPressed, isNotNull);

    const cases = <(InstagramImportParseErrorCode, String)>[
      (
        InstagramImportParseErrorCode.invalidJson,
        'This file is not valid JSON.',
      ),
      (
        InstagramImportParseErrorCode.unsupportedShape,
        'This is not a supported Instagram accounts-followed export. Choose '
            'an export containing Accounts you follow.',
      ),
      (
        InstagramImportParseErrorCode.unsupportedFormat,
        "This Instagram export uses a format CraftSky can't read.",
      ),
      (
        InstagramImportParseErrorCode.invalidArchive,
        'This Instagram ZIP is incomplete or damaged. Download a new export '
            'and try again.',
      ),
      (
        InstagramImportParseErrorCode.archiveTooLarge,
        'This Instagram ZIP contains too many files to process safely.',
      ),
      (
        InstagramImportParseErrorCode.fileTooLarge,
        'The accounts-followed data is larger than 20 MiB.',
      ),
      (
        InstagramImportParseErrorCode.tooManyEntries,
        'This import contains more than 10,000 unique handles.',
      ),
    ];
    for (final (code, message) in cases) {
      picker = () async => throw InstagramImportParseException(code);
      await tester.tap(selectExport);
      await tester.pumpAndSettle();
      expect(find.text(message), findsOneWidget);
    }

    picker = () async => throw StateError('synthetic picker failure');
    await tester.tap(selectExport);
    await tester.pumpAndSettle();
    expect(
      find.text("The Instagram export couldn't be opened on this device."),
      findsOneWidget,
    );
    expect(sentRequests, isEmpty);
  });

  testWidgets('IT-023 late export error is discarded after switch away/back', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final picker = Completer<InstagramImportParseResult?>();
    final sentRequests = <InstagramImportRequest>[];

    await _pumpVerifiedExportPage(
      tester,
      initial: initial,
      repository: _verifiedImportRepository(sentRequests),
      picker: () => picker.future,
    );
    await tester.tap(
      find.widgetWithText(FilledButton, 'Select Instagram export'),
    );
    await tester.pump();

    final container = ProviderScope.containerOf(
      tester.element(find.byType(InstagramMigrationPage)),
    );
    final registryNotifier = container.read(sessionRegistryProvider.notifier);
    final bobLease = container
        .read(sessionRegistryProvider)
        .requireValue
        .leaseFor(AccountKey('did:plc:bob'))!;
    await registryNotifier.activate(bobLease);
    await tester.pump();
    final aliceLease = container
        .read(sessionRegistryProvider)
        .requireValue
        .leaseFor(AccountKey('did:plc:alice'))!;
    await registryNotifier.activate(aliceLease);
    await tester.pump();

    picker.completeError(
      const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text(
        'This Instagram ZIP is incomplete or damaged. Download a new export '
        'and try again.',
      ),
      findsNothing,
    );
    expect(sentRequests, isEmpty);
  });

  testWidgets('IT-023 page disposal discards a late export result', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final picker = Completer<InstagramImportParseResult?>();
    final sentRequests = <InstagramImportRequest>[];

    await _pumpVerifiedExportPage(
      tester,
      initial: initial,
      repository: _verifiedImportRepository(sentRequests),
      picker: () => picker.future,
    );
    await tester.tap(
      find.widgetWithText(FilledButton, 'Select Instagram export'),
    );
    await tester.pump();
    await tester.pumpWidget(const SizedBox.shrink());

    picker.complete(
      const InstagramImportParseResult(
        entries: [InstagramImportEntry(username: 'private_disposed_result')],
      ),
    );
    await tester.pumpAndSettle();

    expect(sentRequests, isEmpty);
  });

  testWidgets('FR-024 challenge can be copied opened and cancelled', (
    tester,
  ) async {
    String? clipboardText;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          clipboardText =
              (call.arguments as Map<Object?, Object?>)['text'] as String?;
        }
        return null;
      },
    );
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var cancelCalls = 0;
    Uri? opened;
    final messenger = RecordingMessenger();
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateVerification: () async => InstagramVerificationAttempt(
        verificationId: 'verification-a',
        state: InstagramVerificationState.pendingDm,
        expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
        challenge: 'CRAFT-TEST-123',
        dmUrl: Uri.parse('https://instagram.example/dm'),
      ),
      onCancelVerification: (_) async => cancelCalls++,
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramDmLauncherProvider.overrideWithValue((uri) async {
            opened = uri;
            return true;
          }),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Create verification challenge'));
    await tester.pump();
    await tester.pump();
    expect(find.text('CRAFT-TEST-123'), findsOneWidget);

    final copyButton = find.widgetWithText(OutlinedButton, 'Copy challenge');
    final openButton = find.widgetWithText(FilledButton, 'Open Instagram DM');
    final cancelButton = find.widgetWithText(TextButton, 'Cancel verification');
    final copyRect = tester.getRect(copyButton);
    final openRect = tester.getRect(openButton);
    final cancelRect = tester.getRect(cancelButton);
    expect(openRect.width, copyRect.width);
    expect(cancelRect.width, copyRect.width);
    expect(copyRect.top, lessThan(openRect.top));
    expect(openRect.top, lessThan(cancelRect.top));

    await tester.tap(find.text('Copy challenge'));
    await tester.pump();
    expect(clipboardText, 'CRAFT-TEST-123');
    expect(messenger.calls, [('info', 'Challenge copied', null)]);
    await tester.tap(find.text('Open Instagram DM'));
    await tester.pump();
    expect(opened, Uri.parse('https://instagram.example/dm'));

    await tester.tap(find.text('Cancel verification'));
    await tester.pump();
    await tester.pump();
    expect(cancelCalls, 1);
    expect(
      find.text('This verification challenge is no longer active.'),
      findsOneWidget,
    );
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      null,
    );
  });

  testWidgets('FR-024 candidate defaults to discovery and explains choices', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    bool? confirmedDiscoverable;
    var cancelCalls = 0;
    var createCalls = 0;
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateVerification: () async {
        createCalls++;
        return InstagramVerificationAttempt(
          verificationId: 'verification-$createCalls',
          state: InstagramVerificationState.pendingConfirmation,
          expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
          candidateUsername: 'actual_maker',
        );
      },
      onCancelVerification: (_) async => cancelCalls++,
      onConfirmVerification: (_, {required discoverable}) async {
        confirmedDiscoverable = discoverable;
        return InstagramVerificationConfirmation(
          state: InstagramVerificationState.confirmed,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'actual_maker',
            discoverable: discoverable,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 19),
          ),
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const InstagramMigrationPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create verification challenge'));
    await tester.pump();
    await tester.pump();

    final accountFinder = find.byWidgetPredicate(
      (widget) =>
          widget is RichText &&
          widget.text.toPlainText() == 'Account: @actual_maker',
    );
    expect(accountFinder, findsOneWidget);
    final accountText = tester.widget<RichText>(accountFinder);
    final accountSpan = accountText.text as TextSpan;
    final handleSpan = _textSpans(
      accountSpan,
    ).singleWhere((span) => span.text == '@actual_maker');
    expect(handleSpan.text, '@actual_maker');
    expect(handleSpan.style?.fontWeight, FontWeight.w600);
    expect(find.text('Allow discovery'), findsOneWidget);
    var selector = tester.widget<SegmentedButton<bool>>(
      find.byType(SegmentedButton<bool>),
    );
    expect(selector.selected, {true});
    expect(selector.emptySelectionAllowed, isFalse);
    expect(
      find.text(
        'When enabled, eligible CraftSky members who imported your Instagram '
        'username may see a private suggestion to follow you.',
      ),
      findsOneWidget,
    );
    expect(
      tester.getTopLeft(accountFinder).dy,
      lessThan(tester.getTopLeft(find.text('Allow discovery')).dy),
    );
    expect(
      tester.getTopLeft(find.text('Allow discovery')).dy,
      lessThan(
        tester
            .getTopLeft(
              find.text(
                'When enabled, eligible CraftSky members who imported your '
                'Instagram username may see a private suggestion to follow '
                'you.',
              ),
            )
            .dy,
      ),
    );
    var confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNotNull);

    await tester.tap(find.text('Keep private'));
    await tester.pump();
    selector = tester.widget<SegmentedButton<bool>>(
      find.byType(SegmentedButton<bool>),
    );
    expect(selector.selected, {false});
    expect(
      find.text(
        'When enabled, eligible CraftSky members who imported your Instagram '
        'username may see a private suggestion to follow you.',
      ),
      findsNothing,
    );
    expect(
      find.text(
        'Your Instagram account remains verified, but CraftSky will not match '
        'it with people who imported your username.',
      ),
      findsOneWidget,
    );
    expect(find.text('Cancel verification'), findsOneWidget);

    await tester.tap(find.text('Cancel verification'));
    await tester.pump();
    expect(cancelCalls, 1);
    expect(
      find.text('This verification challenge is no longer active.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Try again'));
    await tester.pump();
    await tester.pump();
    confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNotNull);
    await tester.tap(find.text('Keep private'));
    await tester.pump();
    confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNotNull);
    await tester.tap(find.text('Confirm this account'));
    await tester.pump();
    await tester.pump();

    expect(confirmedDiscoverable, isFalse);
    expect(find.text('Instagram account confirmed.'), findsOneWidget);
  });

  testWidgets(
    'IT-022 current server attempt is shown without creating a replacement',
    (tester) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final repository = _Repository(
        status: const InstagramAccountStatus(
          integrationAvailable: true,
          account: null,
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
        currentVerification: InstagramVerificationAttempt(
          verificationId: 'verification-current',
          state: InstagramVerificationState.processing,
          expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
        ),
        onCancelVerification: (_) async {},
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Create verification challenge'), findsNothing);
      expect(find.text('Checking your message…'), findsOneWidget);
      expect(find.text('Cancel verification'), findsOneWidget);
      expect(find.text('Copy challenge'), findsOneWidget);
      final copy = tester.widget<OutlinedButton>(
        find.widgetWithText(OutlinedButton, 'Copy challenge'),
      );
      expect(copy.onPressed, isNull);
    },
  );

  testWidgets(
    'IT-016 verified landing links to separate tasks and keeps native controls',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final router = _instagramRouter();
      addTearDown(router.dispose);
      final date = DateTime.utc(2026, 7, 22);
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'actual_maker',
            discoverable: true,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: date,
          ),
        ),
        imports: InstagramImportPage(
          items: [
            InstagramImportSummary(
              importId: 'import-a',
              state: InstagramImportState.active,
              sourceType: InstagramImportSourceType.instagramJson,
              followingCount: 1,
              createdAt: date,
            ),
          ],
          cursor: null,
        ),
        suggestions: InstagramSuggestionPage(
          items: [
            InstagramSuggestion(
              suggestionId: 'suggestion-a',
              target: const InstagramSuggestionTarget(
                did: 'did:plc:synthetic',
                handle: 'synthetic.test',
                displayName: 'Suggested Maker',
              ),
              createdAt: date,
            ),
          ],
          cursor: 'next-suggestions',
        ),
      );

      await _pumpMigrationRouter(tester, router, repository);

      final linkedFinder = find.byWidgetPredicate(
        (widget) =>
            widget is RichText &&
            widget.text.toPlainText() == 'Verified as @actual_maker',
      );
      expect(linkedFinder, findsOneWidget);
      final linkedText = tester.widget<RichText>(linkedFinder);
      final handleSpan = _textSpans(
        linkedText.text as TextSpan,
      ).singleWhere((span) => span.text == '@actual_maker');
      expect(handleSpan.style?.fontWeight, FontWeight.w600);
      final manage = find.byKey(const Key('instagram-account-manage'));
      final following = find.byKey(
        const Key('instagram-following-import-open'),
      );
      final posts = find.byKey(const Key('instagram-post-import-open'));
      final suggestions = find.byKey(const Key('instagram-suggestions-open'));
      expect(manage, findsOneWidget);
      expect(following, findsOneWidget);
      expect(posts, findsOneWidget);
      expect(suggestions, findsOneWidget);
      expect(
        tester.getTopLeft(following).dy,
        lessThan(tester.getTopLeft(posts).dy),
      );
      expect(
        find.descendant(
          of: following,
          matching: find.byIcon(CraftskyIcons.addPeople),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(of: suggestions, matching: find.text('1+')),
        findsOneWidget,
      );
      expect(find.byType(Switch), findsNothing);
      expect(find.byKey(const Key('instagram-import-help')), findsNothing);
      expect(find.text('Your imports'), findsNothing);
      expect(find.text('Suggested Maker'), findsNothing);
      expect(find.text('Revoke Instagram verification'), findsNothing);
      await tester.tap(manage);
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Manage account'), findsOneWidget);
      expect(linkedFinder, findsOneWidget);
      final manageContext = tester.element(find.byType(InstagramMigrationPage));
      final verifiedDate = MaterialLocalizations.of(
        manageContext,
      ).formatShortDate(date.toLocal());
      expect(find.text('Verified $verifiedDate'), findsNothing);
      expect(find.text('Your imports'), findsNothing);

      final theme = Theme.of(manageContext);
      final errorColor = theme.colorScheme.error;
      expect(find.byType(SwitchListTile), findsNothing);
      final discoverySwitch = tester.widget<Switch>(
        find.byKey(const Key('instagram-discoverable-switch')),
      );
      expect(discoverySwitch.activeThumbColor, isNull);
      expect(discoverySwitch.activeTrackColor, isNull);
      final revokeButton = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Revoke Instagram verification'),
      );
      expect(revokeButton.style?.foregroundColor?.resolve({}), errorColor);
      expect(revokeButton.style?.iconColor?.resolve({}), errorColor);

      await tester.pageBack();
      await tester.pumpAndSettle();
      await tester.tap(following);
      await tester.pumpAndSettle();
      expect(
        find.widgetWithText(AppBar, 'Import accounts you follow'),
        findsOneWidget,
      );
      expect(find.byKey(const Key('instagram-import-help')), findsOneWidget);
      expect(find.text('Your imports'), findsOneWidget);
      expect(find.text('Suggested Maker'), findsNothing);
      expect(find.byType(Switch), findsNothing);

      final deleteButtonFinder = find.widgetWithIcon(
        IconButton,
        CraftskyIconsBold.delete,
      );
      await tester.ensureVisible(deleteButtonFinder);
      await tester.pumpAndSettle();
      final deleteButton = tester.widget<IconButton>(deleteButtonFinder);
      expect(deleteButton.tooltip, 'Delete import');
      expect(deleteButton.style?.foregroundColor?.resolve({}), errorColor);
      expect(find.widgetWithText(TextButton, 'Delete import'), findsNothing);
      final importRow = tester.widget<Row>(
        find.ancestor(of: deleteButtonFinder, matching: find.byType(Row)),
      );
      expect(importRow.children.last, isA<IconButton>());

      await tester.pageBack();
      await tester.pumpAndSettle();
      await tester.tap(suggestions);
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Suggested accounts'), findsOneWidget);
      expect(find.text('Suggested Maker'), findsOneWidget);
      expect(find.text('Your imports'), findsNothing);
      expect(find.byType(Switch), findsNothing);
    },
  );

  testWidgets('IT-016 revoking Instagram verification requires confirmation', (
    tester,
  ) async {
    final router = _instagramRouter();
    addTearDown(router.dispose);
    var revokeCalls = 0;
    final repository = _Repository(
      status: InstagramAccountStatus(
        integrationAvailable: true,
        account: InstagramAccountLink(
          state: InstagramAccountLinkState.active,
          username: 'actual_maker',
          discoverable: true,
          conflictPending: false,
          reactivationRequired: false,
          verifiedAt: DateTime.utc(2026, 7, 22),
        ),
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onRevokeAccount: () async => revokeCalls += 1,
    );

    await _pumpMigrationRouter(tester, router, repository);
    await tester.tap(find.byKey(const Key('instagram-account-manage')));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Revoke Instagram verification'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Revoke Instagram verification'));
    await tester.pumpAndSettle();

    expect(find.text('Revoke Instagram verification?'), findsOneWidget);
    expect(
      find.text(
        'This removes your Instagram verification and deletes your imported '
        'handles. Existing CraftSky follows will not be affected.',
      ),
      findsOneWidget,
    );
    expect(revokeCalls, 0);

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(revokeCalls, 0);
    expect(find.text('Verified as @actual_maker'), findsOneWidget);

    await tester.ensureVisible(find.text('Revoke Instagram verification'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Revoke Instagram verification'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Revoke Instagram verification').last);
    await tester.pumpAndSettle();

    expect(revokeCalls, 1);
    expect(find.text('Verified as @actual_maker'), findsNothing);
    expect(find.text('Create verification challenge'), findsOneWidget);
  });
}

GoRouter _instagramRouter({bool withPostThread = false}) => GoRouter(
  initialLocation: const InstagramMigrationRoute().location,
  routes: [
    if (withPostThread)
      GoRoute(
        path: RouteLocations.postThread,
        builder: (_, _) =>
            const Scaffold(body: Text('Imported project thread')),
      ),
    GoRoute(
      path: const InstagramMigrationRoute().location,
      builder: const InstagramMigrationRoute().build,
      routes: [
        GoRoute(
          path: RouteLocations.instagramPostImportChild,
          builder: const InstagramPostImportRoute().build,
        ),
        GoRoute(
          path: RouteLocations.instagramFollowingImportChild,
          builder: const InstagramFollowingImportRoute().build,
        ),
        GoRoute(
          path: RouteLocations.instagramSuggestionsChild,
          builder: const InstagramSuggestionsRoute().build,
        ),
        GoRoute(
          path: RouteLocations.instagramAccountChild,
          builder: const InstagramAccountRoute().build,
        ),
      ],
    ),
  ],
);

Future<void> _pumpMigrationRouter(
  WidgetTester tester,
  GoRouter router,
  _Repository repository,
) async {
  final initial = registry.SessionRegistry.empty().upsertAndActivate(
    token: 'token-a',
    did: 'did:plc:alice',
    handle: 'alice.test',
  );
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(initial),
        ),
        instagramVerificationStorageProvider.overrideWithValue(
          _EmptyVerificationStorage(),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, _) async => repository,
        ),
      ],
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        routerConfig: router,
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _pumpVerifiedExportPage(
  WidgetTester tester, {
  required registry.SessionRegistry initial,
  required _Repository repository,
  required InstagramExportFilePicker picker,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(initial),
        ),
        instagramVerificationStorageProvider.overrideWithValue(
          _EmptyVerificationStorage(),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, _) async => repository,
        ),
        instagramExportFilePickerProvider.overrideWithValue(picker),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const InstagramMigrationPage(
          view: InstagramMigrationView.following,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Select Instagram export'));
  await tester.pumpAndSettle();
}

_Repository _verifiedImportRepository(
  List<InstagramImportRequest> sentRequests,
) => _Repository(
  status: InstagramAccountStatus(
    integrationAvailable: true,
    account: InstagramAccountLink(
      state: InstagramAccountLinkState.active,
      username: 'synthetic_instagram',
      discoverable: true,
      conflictPending: false,
      reactivationRequired: false,
      verifiedAt: DateTime.utc(2026, 7, 23),
    ),
  ),
  imports: InstagramImportPage(items: const [], cursor: null),
  onCreateImport: (request) async {
    sentRequests.add(request);
    return InstagramImportCreateResult(
      import: InstagramImportSummary(
        importId: 'unexpected-import',
        state: InstagramImportState.active,
        sourceType: request.sourceType,
        followingCount: request.entries.length,
        createdAt: DateTime.utc(2026, 7, 23),
      ),
      followingCount: request.entries.length,
    );
  },
);

Iterable<TextSpan> _textSpans(TextSpan span) sync* {
  yield span;
  for (final child in span.children ?? const <InlineSpan>[]) {
    if (child case final TextSpan textSpan) yield* _textSpans(textSpan);
  }
}

final class _EmptyVerificationStorage implements InstagramVerificationStorage {
  @override
  Future<void> delete(AccountKey account, {String? verificationId}) async {}

  @override
  Future<InstagramVerificationSnapshot?> read(AccountKey account) async => null;

  @override
  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  ) async {}
}

final class _RegistryStorage implements SessionRegistryStorage {
  new(this.value);

  registry.SessionRegistry value;

  @override
  Future<registry.SessionRegistry> read() async => value;

  @override
  Future<void> write(registry.SessionRegistry registry) async =>
      value = registry;
}

final class _Repository implements InstagramMigrationRepository {
  const new({
    required this.imports,
    this.suggestions,
    this.status = const InstagramAccountStatus(
      integrationAvailable: false,
      account: null,
    ),
    this.onCreateImport,
    this.onLookupPostImports,
    this.onCreateVerification,
    this.onCancelVerification,
    this.onConfirmVerification,
    this.onRevokeAccount,
    this.currentVerification,
  });

  final InstagramImportPage imports;
  final InstagramSuggestionPage? suggestions;
  final InstagramAccountStatus status;
  final Future<InstagramImportCreateResult> Function(
    InstagramImportRequest request,
  )?
  onCreateImport;
  final Future<List<InstagramPublishedImport>> Function(List<String> keys)?
  onLookupPostImports;
  final Future<InstagramVerificationAttempt> Function()? onCreateVerification;
  final Future<void> Function(String verificationId)? onCancelVerification;
  final Future<InstagramVerificationConfirmation> Function(
    String verificationId, {
    required bool discoverable,
  })?
  onConfirmVerification;
  final Future<void> Function()? onRevokeAccount;
  final InstagramVerificationAttempt? currentVerification;

  @override
  Future<InstagramAccountStatus> getAccount() async => status;

  @override
  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  ) => onCreateImport!.call(request);

  @override
  Future<List<InstagramPublishedImport>> lookupPostImports(
    List<String> keys,
  ) => onLookupPostImports?.call(keys) ?? Future.value(const []);

  @override
  Future<InstagramVerificationAttempt> createVerification() =>
      onCreateVerification!.call();

  @override
  Future<InstagramVerificationAttempt?> getCurrentVerification() async =>
      currentVerification;

  @override
  Future<void> cancelVerification(String verificationId) =>
      onCancelVerification!.call(verificationId);

  @override
  Future<InstagramVerificationConfirmation> confirmVerification(
    String verificationId, {
    required bool discoverable,
  }) => onConfirmVerification!.call(verificationId, discoverable: discoverable);

  @override
  Future<InstagramImportPage> listImports({int? limit, String? cursor}) async =>
      imports;

  @override
  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) async =>
      suggestions ?? InstagramSuggestionPage(items: const [], cursor: null);

  @override
  Future<void> revokeAccount() => onRevokeAccount!.call();

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
