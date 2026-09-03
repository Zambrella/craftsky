import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

import '../fakes/recording_messenger.dart';

void main() {
  test(
    'new scheduled media gets a one-minute-style independent budget',
    () async {
      CancelToken? capturedToken;
      final bytes = _pngBytes(width: 2, height: 3);
      final image = ComposerImageDraft(
        id: 'new-ready',
        fileName: 'detail.png',
        mimeType: 'image/png',
        altText: '',
        phase: ImageReady(
          bytes: bytes,
          mimeType: 'image/png',
          width: 2,
          height: 3,
          sha256: sha256.convert(bytes).toString(),
        ),
      );

      await expectLater(
        materializeScheduledComposerMedia(
          [image],
          transferBudget: const Duration(milliseconds: 5),
          stageMedia:
              ({
                required id,
                required bytes,
                required mimeType,
                cancelToken,
              }) {
                capturedToken = cancelToken;
                return Completer<void>().future;
              },
        ),
        throwsA(isA<TimeoutException>()),
      );
      expect(capturedToken?.isCancelled, isTrue);
    },
  );

  test('rejects ready bytes mutated after local preparation', () async {
    var stageCalls = 0;
    final bytes = _pngBytes(width: 2, height: 1);
    final image = ComposerImageDraft(
      id: 'new-ready',
      fileName: 'detail.png',
      mimeType: 'image/png',
      altText: '',
      phase: ImageReady(
        bytes: bytes,
        mimeType: 'image/png',
        width: 2,
        height: 1,
        sha256: sha256.convert(bytes).toString(),
      ),
    );
    bytes[0] ^= 0xff;

    await expectLater(
      materializeScheduledComposerMedia(
        [image],
        stageMedia:
            ({
              required id,
              required bytes,
              required mimeType,
              cancelToken,
            }) async {
              stageCalls += 1;
            },
      ),
      throwsStateError,
    );
    expect(stageCalls, 0);
  });

  test('scheduled ready media reuses canonical prepared bytes', () async {
    final bytes = _pngBytes(width: 20, height: 10);
    final image = ComposerImageDraft(
      id: 'large-ready',
      fileName: 'detail.png',
      mimeType: 'image/png',
      altText: '  project detail  ',
      phase: ImageReady(
        bytes: bytes,
        mimeType: 'image/png',
        width: 20,
        height: 10,
        sha256: sha256.convert(bytes).toString(),
      ),
    );
    Uint8List? stagedBytes;

    final media = await materializeScheduledComposerMedia(
      [image],
      stageMedia:
          ({
            required id,
            required bytes,
            required mimeType,
            cancelToken,
          }) async {
            stagedBytes = Uint8List.fromList(bytes);
          },
    );

    expect(media, [
      {
        'id': 'large-ready',
        'alt': 'project detail',
        'width': 20,
        'height': 10,
      },
    ]);
    expect(stagedBytes, bytes);
  });

  test(
    'AT-007 preserves the final edited media set and stages only new media',
    () async {
      final privateBytes = _pngBytes(width: 3, height: 2);
      final drafts = await hydrateScheduledComposerMedia(
        const [
          {'id': 'old-1', 'alt': 'front', 'width': 3, 'height': 2},
          {'id': 'old-2', 'alt': 'back', 'width': 2, 'height': 3},
        ],
        loadBytes: (_) async => privateBytes,
      );

      final newBytes = _pngBytes(width: 4, height: 3);
      final edited = [
        drafts.first.copyWith(altText: 'updated front'),
        ComposerImageDraft(
          id: 'new-1',
          fileName: 'detail.png',
          mimeType: 'image/png',
          altText: 'detail',
          previewBytes: newBytes,
          phase: const ImageUploaded(
            UploadedDraftImage(
              cid: 'eager-pds-cid',
              mime: 'image/png',
              size: 10,
              aspectRatio: CreatePostImageAspectRatio(width: 4, height: 3),
            ),
          ),
        ),
      ];
      final staged = <String, Uint8List>{};

      final media = await materializeScheduledComposerMedia(
        edited,
        stageMedia:
            ({
              required id,
              required bytes,
              required mimeType,
              cancelToken,
            }) async {
              staged[id] = Uint8List.fromList(bytes);
            },
      );

      expect(media, [
        {'id': 'old-1', 'alt': 'updated front', 'width': 3, 'height': 2},
        {'id': 'new-1', 'alt': 'detail', 'width': 4, 'height': 3},
      ]);
      expect(staged.keys, ['new-1']);
      expect(staged['new-1'], isNotEmpty);
    },
  );

  testWidgets(
    'AT-005 unchanged scheduled external retains thumbnail identity and bytes',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final thumbnailBytes = _pngBytes(width: 2, height: 1);
      final repository = _ScheduledRepository(
        thumbnailBytes,
        detail: _scheduledExternalDetail,
      );
      final previews = _NoFetchPreviewRepository();
      const composerId = 'scheduled-external-editor';
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(previews),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: composerId,
                scheduledPost: _scheduledExternalDetail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      final selected = container
          .read(linkPreviewControllerProvider(composerId, account).notifier)
          .selected;
      expect(repository.mediaRequests, [_externalThumbnailID]);
      expect(selected?.preview.thumbnail?.bytes, thumbnailBytes);
      expect(previews.calls, 0);

      await tester.tap(find.widgetWithText(ChunkyButton, 'Schedule'));
      await tester.pumpAndSettle();

      expect(
        repository.updatedPayload?['external'],
        _scheduledExternalDetail.payload['external'],
      );
      expect(repository.stagedIDs, isEmpty);
      expect(previews.calls, 0);
    },
  );

  testWidgets(
    'IR-012 delayed scheduled thumbnail hydration reseeds frozen selection',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final thumbnailBytes = _pngBytes(width: 2, height: 1);
      final mediaResponse = Completer<Uint8List>();
      final repository = _ScheduledRepository(
        thumbnailBytes,
        detail: _scheduledExternalDetail,
        mediaResponse: mediaResponse,
      );
      final previews = _NoFetchPreviewRepository();
      const composerId = 'scheduled-delayed-external-editor';
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(previews),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: composerId,
                scheduledPost: _scheduledExternalDetail,
              ),
            ),
          ),
        ),
      );
      for (var i = 0; i < 20 && !repository.mediaStarted.isCompleted; i++) {
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(repository.mediaStarted.isCompleted, isTrue);
      await tester.pump();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      final controller = container.read(
        linkPreviewControllerProvider(composerId, account).notifier,
      );
      expect(controller.selected?.preview.thumbnail, isNull);

      mediaResponse.complete(thumbnailBytes);
      await tester.pumpAndSettle();

      expect(controller.selected?.preview.thumbnail?.bytes, thumbnailBytes);
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(
        repository.updatedPayload?['external'],
        _scheduledExternalDetail.payload['external'],
      );
      expect(repository.stagedIDs, isEmpty);
      expect(previews.calls, 0);
    },
  );

  testWidgets(
    'IR-018 stale hydration cannot resurrect a removed frozen selection',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final oldBytes = _pngBytes(width: 2, height: 1);
      final replacementBytes = _pngBytes(width: 3, height: 1);
      final mediaResponse = Completer<Uint8List>();
      final repository = _ScheduledRepository(
        oldBytes,
        detail: _scheduledExternalDetail,
        mediaResponse: mediaResponse,
      );
      final previews = _ReplacementPreviewRepository(replacementBytes);
      const composerId = 'scheduled-stale-hydration-editor';
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(previews),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: composerId,
                scheduledPost: _scheduledExternalDetail,
              ),
            ),
          ),
        ),
      );
      for (var i = 0; i < 20 && !repository.mediaStarted.isCompleted; i++) {
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(repository.mediaStarted.isCompleted, isTrue);

      await tester.enterText(
        find.byType(TextField).first,
        'Use https://replacement.example/pattern ',
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      final controller = container.read(
        linkPreviewControllerProvider(composerId, account).notifier,
      );
      for (
        var i = 0;
        i < 20 && controller.selected?.preview.title != 'Replacement pattern';
        i++
      ) {
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(previews.urls, ['https://replacement.example/pattern']);
      expect(controller.selected?.preview.title, 'Replacement pattern');

      mediaResponse.complete(oldBytes);
      await tester.pumpAndSettle();

      expect(find.text('Replacement pattern'), findsOneWidget);
      expect(find.text('Frozen pattern'), findsNothing);
      expect(
        controller.selected?.candidate.identity,
        Uri.parse('https://replacement.example/pattern'),
      );
    },
  );

  testWidgets(
    'AT-005 dismissing a scheduled external explicitly removes its thumbnail',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final detail = ScheduledPostDetail(
        id: _scheduledExternalDetail.id,
        operationId: _scheduledExternalDetail.operationId,
        status: _scheduledExternalDetail.status,
        scheduledAt: _scheduledExternalDetail.scheduledAt,
        payload: {
          ..._scheduledExternalDetail.payload,
          'text': 'Use https://source.example/pattern for this project',
        },
      );
      final repository = _ScheduledRepository(
        _pngBytes(width: 2, height: 1),
        detail: detail,
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(
              _NoFetchPreviewRepository(),
            ),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: 'scheduled-external-remove',
                scheduledPost: detail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('Dismiss link previews'));
      await tester.pumpAndSettle();
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(repository.updatedPayload, isNotNull);
      expect(repository.updatedPayload, isNot(contains('external')));
      expect(repository.stagedIDs, isEmpty);
    },
  );

  testWidgets(
    'IR-013 existing scheduled images remove external on save',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final detail = ScheduledPostDetail(
        id: _scheduledExternalDetail.id,
        operationId: _scheduledExternalDetail.operationId,
        status: _scheduledExternalDetail.status,
        scheduledAt: _scheduledExternalDetail.scheduledAt,
        payload: {
          ..._scheduledExternalDetail.payload,
          'text': 'Use https://source.example/pattern for this project',
          'media': const [
            {'id': 'old-1', 'alt': 'project', 'width': 2, 'height': 1},
          ],
        },
      );
      final repository = _ScheduledRepository(
        _pngBytes(width: 2, height: 1),
        detail: detail,
      );
      const composerId = 'scheduled-images-win-external';
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(
              _NoFetchPreviewRepository(),
            ),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: composerId,
                scheduledPost: detail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      expect(
        container.read(composerImagesProvider(composerId)).images,
        hasLength(1),
      );
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(repository.updatedPayload, isNotNull);
      expect(repository.updatedPayload?['media'], hasLength(1));
      expect(repository.updatedPayload, isNot(contains('external')));
      expect(repository.stagedIDs, isEmpty);
    },
  );

  testWidgets('IR-013 removing scheduled source removes external on save', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _ScheduledRepository(
      _pngBytes(width: 2, height: 1),
      detail: _scheduledExternalDetail,
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          linkPreviewRepositoryProvider.overrideWithValue(
            _NoFetchPreviewRepository(),
          ),
          accountScheduledPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: _testTheme,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: PostComposerSheet(
              composerId: 'scheduled-external-source-remove',
              scheduledPost: _scheduledExternalDetail,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField).first, 'Source removed');
    await tester.pumpAndSettle();
    tester
        .widget<ChunkyButton>(
          find.byKey(const Key('post-composer-primary-action')),
        )
        .onPressed!();
    await tester.pumpAndSettle();

    expect(repository.updatedPayload, isNot(contains('external')));
    expect(repository.stagedIDs, isEmpty);
  });

  testWidgets(
    'AT-005 replacing a scheduled external stages new bytes under a new ID',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final oldBytes = _pngBytes(width: 2, height: 1);
      final replacementBytes = _pngBytes(width: 3, height: 1);
      final repository = _ScheduledRepository(
        oldBytes,
        detail: _scheduledExternalDetail,
      );
      final previews = _ReplacementPreviewRepository(replacementBytes);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(previews),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: 'scheduled-external-replace',
                scheduledPost: _scheduledExternalDetail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.enterText(
        find.byType(TextField).first,
        'Use https://replacement.example/pattern ',
      );
      await tester.pumpAndSettle();
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(previews.urls, ['https://replacement.example/pattern']);
      expect(repository.stagedIDs, hasLength(1));
      expect(repository.stagedIDs.single, isNot(_externalThumbnailID));
      expect(repository.stagedBytes.single, replacementBytes);
      expect(
        (repository.updatedPayload?['external'] as Map)['thumbMediaId'],
        repository.stagedIDs.single,
      );
    },
  );

  testWidgets(
    'IT-015 replacement staging failure preserves the scheduled card',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final repository = _ScheduledRepository(
        _pngBytes(width: 2, height: 1),
        detail: _scheduledExternalDetail,
        failStage: true,
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(
              _ReplacementPreviewRepository(_pngBytes(width: 3, height: 1)),
            ),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: 'scheduled-external-stage-failure',
                scheduledPost: _scheduledExternalDetail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.enterText(
        find.byType(TextField).first,
        'Use https://replacement.example/pattern ',
      );
      await tester.pumpAndSettle();
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(repository.stagedIDs, hasLength(1));
      expect(repository.updatedPayload, isNull);
      expect(find.text('Replacement pattern'), findsOneWidget);
      expect(
        find.text('Use https://replacement.example/pattern '),
        findsOneWidget,
      );
    },
  );

  testWidgets(
    'IR-014 changed replacement after failed update stages under a fresh ID',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final firstBytes = _pngBytes(width: 3, height: 1);
      final secondBytes = _pngBytes(width: 4, height: 1);
      final repository = _ScheduledRepository(
        _pngBytes(width: 2, height: 1),
        detail: _scheduledExternalDetail,
        failUpdateCount: 1,
      );
      final previews = _ChangingReplacementPreviewRepository([
        firstBytes,
        secondBytes,
      ]);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            linkPreviewRepositoryProvider.overrideWithValue(previews),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: 'scheduled-external-update-retry',
                scheduledPost: _scheduledExternalDetail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.enterText(
        find.byType(TextField).first,
        'Use https://replacement.example/first ',
      );
      await tester.pumpAndSettle();
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(repository.stagedIDs, hasLength(1));
      expect(repository.stagedBytes.single, firstBytes);
      final failedID = repository.stagedIDs.single;

      await tester.enterText(
        find.byType(TextField).first,
        'Use https://replacement.example/second ',
      );
      await tester.pumpAndSettle();
      tester
          .widget<ChunkyButton>(
            find.byKey(const Key('post-composer-primary-action')),
          )
          .onPressed!();
      await tester.pumpAndSettle();

      expect(repository.stagedIDs, hasLength(2));
      expect(repository.stagedIDs.last, isNot(failedID));
      expect(repository.stagedBytes.last, secondBytes);
      expect(
        (repository.updatedPayload?['external'] as Map)['thumbMediaId'],
        repository.stagedIDs.last,
      );
    },
  );

  testWidgets('AT-007 editor retains its slot when all three slots are full', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _ScheduledRepository(
      _pngBytes(width: 3, height: 2),
      scheduledCount: 3,
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          accountScheduledPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: _testTheme,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: PostComposerSheet(
              composerId: 'scheduled-editor',
              scheduledPost: _scheduledDetail,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('of 3 scheduled'), findsNothing);
    expect(find.textContaining("can't schedule another post"), findsNothing);
    expect(
      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Schedule'))
          .onPressed,
      isNotNull,
    );
    expect(find.byKey(const Key('composer-alt-old-1')), findsOneWidget);
    expect(find.byKey(const Key('composer-alt-old-2')), findsOneWidget);
    await tester.enterText(
      find.descendant(
        of: find.byKey(const Key('composer-alt-old-1')),
        matching: find.byType(EditableText),
      ),
      'updated front',
    );
    await tester.ensureVisible(
      find.byKey(const Key('composer-move-down-old-1')),
    );
    await tester.tap(find.byKey(const Key('composer-move-down-old-1')));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Schedule'));
    await tester.pumpAndSettle();

    expect(repository.updatedPayload?['media'], [
      {'id': 'old-2', 'alt': 'back', 'width': 2, 'height': 3},
      {'id': 'old-1', 'alt': 'updated front', 'width': 3, 'height': 2},
    ]);
    expect(repository.stagedIDs, isEmpty);
  });

  testWidgets('AT-007 Post now publishes the final existing media edits', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _ScheduledRepository(_pngBytes(width: 3, height: 2));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          accountScheduledPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: _testTheme,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: PostComposerSheet(
              composerId: 'scheduled-publish-editor',
              scheduledPost: _scheduledDetail,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.enterText(
      find.descendant(
        of: find.byKey(const Key('composer-alt-old-1')),
        matching: find.byType(EditableText),
      ),
      'published front',
    );
    await tester.tap(find.text('When'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Now').last);
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(repository.publishedPayload?['media'], [
      {'id': 'old-1', 'alt': 'published front', 'width': 3, 'height': 2},
      {'id': 'old-2', 'alt': 'back', 'width': 2, 'height': 3},
    ]);
  });

  testWidgets(
    'IT-021 delayed media is discarded after account activation changes',
    (tester) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final aliceRegistry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final owner = aliceRegistry.activeLease!;
      final account = owner.session.account;
      final mediaResponse = Completer<Uint8List>();
      final repository = _ScheduledRepository(
        _pngBytes(width: 3, height: 2),
        mediaResponse: mediaResponse,
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(aliceRegistry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: 'scheduled-delayed-media',
                scheduledPost: _scheduledDetail,
                scheduledOwner: owner,
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump();
      await repository.mediaStarted.future;
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceRegistry.leaseFor(AccountKey('did:plc:bob'))!);
      await tester.pump();
      mediaResponse.complete(_pngBytes(width: 3, height: 2));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.byKey(const Key('composer-alt-old-1')), findsNothing);
    },
  );

  testWidgets(
    'IT-021 delayed mutation is discarded after account activation changes',
    (tester) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final aliceRegistry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final owner = aliceRegistry.activeLease!;
      final updateResponse = Completer<ScheduledPostDetail>();
      final repository = _ScheduledRepository(
        _pngBytes(width: 3, height: 2),
        detail: _scheduledTextDetail,
        updateResponse: updateResponse,
      );
      final messenger = RecordingMessenger();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(aliceRegistry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            accountScheduledPostRepositoryProvider(
              owner.session.account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: PostComposerSheet(
                composerId: 'scheduled-delayed-update',
                scheduledPost: _scheduledTextDetail,
                scheduledOwner: owner,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(ChunkyButton, 'Schedule'));
      await tester.pump();
      await repository.updateStarted.future;
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceRegistry.leaseFor(AccountKey('did:plc:bob'))!);
      await tester.pump();
      updateResponse.complete(_scheduledTextDetail);
      await tester.pumpAndSettle();

      expect(messenger.calls.where((call) => call.$1 == 'info'), isEmpty);
    },
  );

  testWidgets('AT-007 reschedule preserves facets when text is unchanged', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _ScheduledRepository(
      _pngBytes(width: 1, height: 1),
      detail: _scheduledFacetedDetail,
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          facetGeneratorProvider.overrideWithValue(
            const FacetGenerator(mentionResolver: _EmptyMentionResolver()),
          ),
          accountScheduledPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: _testTheme,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: PostComposerSheet(
              composerId: 'scheduled-facet-update',
              scheduledPost: _scheduledFacetedDetail,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Schedule'));
    await tester.pumpAndSettle();

    expect(
      repository.updatedPayload?['facets'],
      _scheduledFacetedDetail.payload['facets'],
    );
  });

  testWidgets('AT-007 Post now preserves facets when text is unchanged', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _ScheduledRepository(
      _pngBytes(width: 1, height: 1),
      detail: _scheduledFacetedDetail,
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          facetGeneratorProvider.overrideWithValue(
            const FacetGenerator(mentionResolver: _EmptyMentionResolver()),
          ),
          accountScheduledPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: _testTheme,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: PostComposerSheet(
              composerId: 'scheduled-facet-publish',
              scheduledPost: _scheduledFacetedDetail,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('When'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Now').last);
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(
      repository.publishedPayload?['facets'],
      _scheduledFacetedDetail.payload['facets'],
    );
  });

  testWidgets('AT-007 project editor retains its slot at full capacity', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final account = registry.activeLease!.session.account;
    final repository = _ScheduledRepository(
      _pngBytes(width: 3, height: 2),
      scheduledCount: 3,
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          accountScheduledPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: _testTheme,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ProjectComposerSheet(
              composerId: 'scheduled-project-editor',
              scheduledPost: _scheduledProjectDetail,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Save draft'), findsNothing);
    expect(find.byKey(const Key('composer-alt-old-1')), findsOneWidget);
    expect(
      find.byKey(const Key('project-composer-pattern-designer-editor')),
      findsOneWidget,
    );
    expect(find.text('A. Maker'), findsOneWidget);

    await tester.tap(
      find.byKey(const Key('project-composer-primary-action')),
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(
        const Key('${ProjectComposerFields.knittingProjectType}-select-button'),
      ),
      findsOneWidget,
    );
    expect(
      find.byKey(const Key('knitting-gauge-stitches-input')),
      findsOneWidget,
    );

    await tester.tap(
      find.byKey(const Key('project-composer-primary-action')),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('of 3 scheduled'), findsNothing);
    expect(find.textContaining("can't schedule another post"), findsNothing);
    expect(
      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Schedule'))
          .onPressed,
      isNotNull,
    );
    await tester.tap(find.widgetWithText(ChunkyButton, 'Schedule'));
    await tester.pumpAndSettle();

    expect(
      repository.updatedPayload?['project'],
      _scheduledProjectDetail.payload['project'],
    );
    expect(repository.updatedPayload?['media'], [
      {'id': 'old-1', 'alt': 'cardigan', 'width': 3, 'height': 2},
    ]);
  });

  for (final fixture in _additionalProjectFixtures.entries) {
    testWidgets('AT-007 ${fixture.key} project saves every field unchanged', (
      tester,
    ) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final account = registry.activeLease!.session.account;
      final detail = fixture.value;
      final repository = _ScheduledRepository(
        _pngBytes(width: 3, height: 2),
        detail: detail,
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            accountScheduledPostRepositoryProvider(
              account,
            ).overrideWith((ref) async => repository),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: _testTheme,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: ProjectComposerSheet(
                composerId: '${fixture.key}-scheduled-editor',
                scheduledPost: detail,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('composer-alt-old-1')), findsOneWidget);
      expect(
        find.byKey(
          const Key('project-composer-pattern-designer-editor'),
        ),
        findsOneWidget,
      );
      expect(find.text('A. Maker'), findsOneWidget);
      await tester.tap(
        find.byKey(const Key('project-composer-primary-action')),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(Key('${fixture.key}ProjectType-select-button')),
        findsOneWidget,
      );
      await tester.tap(
        find.byKey(const Key('project-composer-primary-action')),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(ChunkyButton, 'Schedule'));
      await tester.pumpAndSettle();

      expect(repository.updatedPayload?['project'], detail.payload['project']);
      expect(repository.stagedIDs, isEmpty);
    });
  }
}

Uint8List _pngBytes({required int width, required int height}) =>
    Uint8List.fromList(img.encodePng(img.Image(width: width, height: height)));

final _scheduledDetail = ScheduledPostDetail(
  id: 'schedule-1',
  operationId: 'operation-1',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  payload: const {
    'kind': 'standard',
    'text': 'A scheduled post',
    'langs': ['en'],
    'media': [
      {'id': 'old-1', 'alt': 'front', 'width': 3, 'height': 2},
      {'id': 'old-2', 'alt': 'back', 'width': 2, 'height': 3},
    ],
  },
);

final _scheduledTextDetail = ScheduledPostDetail(
  id: 'schedule-text',
  operationId: 'operation-text',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  payload: const {
    'kind': 'standard',
    'text': 'A scheduled text post',
    'langs': ['en'],
    'media': <Map<String, dynamic>>[],
  },
);

const _externalThumbnailID = '55555555-5555-4555-8555-555555555555';

final _scheduledExternalDetail = ScheduledPostDetail(
  id: 'schedule-external',
  operationId: 'operation-external',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  payload: {
    'kind': 'standard',
    'text': 'Use https://source.example/pattern ',
    'langs': ['en'],
    'media': <Map<String, dynamic>>[],
    'external': {
      'sourceUri': 'https://source.example/pattern',
      'uri': 'https://final.example/pattern#frozen',
      'title': 'Frozen pattern',
      'description': 'Frozen description',
      'thumbMediaId': _externalThumbnailID,
    },
  },
);

final _scheduledFacetedDetail = ScheduledPostDetail(
  id: 'schedule-faceted',
  operationId: 'operation-faceted',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  payload: const {
    'kind': 'standard',
    'text': 'Hello @alice.test',
    'langs': ['en'],
    'facets': [
      {
        'index': {'byteStart': 6, 'byteEnd': 17},
        'features': [
          {
            r'$type': 'app.bsky.richtext.facet#mention',
            'did': 'did:plc:alice',
          },
        ],
      },
    ],
    'media': <Map<String, dynamic>>[],
  },
);

final _scheduledProjectDetail = ScheduledPostDetail(
  id: 'schedule-project',
  operationId: 'operation-project',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  payload: const {
    'kind': 'project',
    'text': 'A scheduled cardigan',
    'langs': ['en'],
    'media': [
      {'id': 'old-1', 'alt': 'cardigan', 'width': 3, 'height': 2},
    ],
    'project': {
      'common': {
        'craftType': ProjectOptionCatalogs.knittingCraftToken,
        'status': ProjectOptionCatalogs.finishedStatusToken,
        'title': 'Cardigan',
        'materials': [
          {'text': 'Merino wool'},
        ],
        'colors': ['blue'],
        'designTags': ['social.craftsky.project.defs#stripes'],
        'pattern': {
          'name': 'Warm cardigan',
          'designer': 'A. Maker',
          'publisher': 'Patterns Ltd',
          'url': 'https://example.com/pattern',
          'difficulty': 'intermediate',
        },
      },
      'details': {
        r'$type': 'social.craftsky.project.knitting#details',
        'projectType': 'garment',
        'projectSubtype': 'cardigan',
        'yarnWeight': 'dk',
        'needleSizeMm': '4',
        'gauge': {
          'stitches': 20,
          'rows': 28,
          'measurement': 10,
          'unit': 'cm',
        },
      },
    },
  },
);

final _additionalProjectFixtures = <String, ScheduledPostDetail>{
  'sewing': _projectFixture(
    id: 'sewing',
    craftType: ProjectOptionCatalogs.sewingCraftToken,
    details: {
      r'$type': 'social.craftsky.project.sewing#details',
      'projectType': 'social.craftsky.project.defs#garment',
      'projectSubtype': 'social.craftsky.project.sewing.defs#dress',
      'sizeMade': 'M',
      'fitNotes': 'Relaxed fit',
    },
  ),
  'crochet': _projectFixture(
    id: 'crochet',
    craftType: ProjectOptionCatalogs.crochetCraftToken,
    details: {
      r'$type': 'social.craftsky.project.crochet#details',
      'projectType': 'social.craftsky.project.defs#accessory',
      'projectSubtype': 'social.craftsky.project.crochet.defs#bag',
      'yarnWeight': 'social.craftsky.project.defs#dk',
      'hookSizeMm': '4.0mm',
      'gauge': {
        'stitches': 18,
        'rows': 22,
        'measurement': 10,
        'unit': 'cm',
      },
      'finishedSize': '30 cm',
    },
  ),
  'quilting': _projectFixture(
    id: 'quilting',
    craftType: ProjectOptionCatalogs.quiltingCraftToken,
    details: {
      r'$type': 'social.craftsky.project.quilting#details',
      'projectType': 'social.craftsky.project.defs#quilt',
      'projectSubtype': 'social.craftsky.project.quilting.defs#throwQuilt',
      'size': '50 x 60 in',
      'piecingTechnique': 'social.craftsky.project.quilting.defs#traditional',
      'quiltingMethod': 'social.craftsky.project.quilting.defs#machineQuilted',
    },
  ),
};

ScheduledPostDetail _projectFixture({
  required String id,
  required String craftType,
  required Map<String, dynamic> details,
}) => ScheduledPostDetail(
  id: id,
  operationId: '$id-operation',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
  payload: {
    'kind': 'project',
    'text': '$id project body',
    'langs': const ['en'],
    'media': const [
      {'id': 'old-1', 'alt': 'project', 'width': 3, 'height': 2},
    ],
    'project': {
      'common': {
        'craftType': craftType,
        'status': ProjectOptionCatalogs.finishedStatusToken,
        'title': '$id title',
        'materials': const [
          {'text': 'Cotton thread'},
        ],
        'colors': const ['blue'],
        'designTags': const ['social.craftsky.project.defs#stripes'],
        'pattern': const {
          'name': 'Pattern name',
          'designer': 'A. Maker',
          'publisher': 'Patterns Ltd',
          'url': 'https://example.com/pattern',
          'difficulty': 'social.craftsky.feed.defs#intermediate',
        },
      },
      'details': details,
    },
  },
);

final class _ScheduledRepository implements ScheduledPostRepository {
  _ScheduledRepository(
    this.bytes, {
    this.detail,
    this.mediaResponse,
    this.updateResponse,
    this.scheduledCount = 0,
    this.failStage = false,
    this.failUpdateCount = 0,
  });

  final Uint8List bytes;
  final ScheduledPostDetail? detail;
  final Completer<Uint8List>? mediaResponse;
  final Completer<ScheduledPostDetail>? updateResponse;
  final int scheduledCount;
  final bool failStage;
  int failUpdateCount;
  final Completer<void> mediaStarted = Completer<void>();
  final Completer<void> updateStarted = Completer<void>();
  Map<String, dynamic>? updatedPayload;
  Map<String, dynamic>? publishedPayload;
  final stagedIDs = <String>[];
  final stagedBytes = <Uint8List>[];
  final mediaRequests = <String>[];

  @override
  Future<List<ScheduledPostSummary>> list() async => List.generate(
    scheduledCount,
    (index) => ScheduledPostSummary(
      id: 'scheduled-$index',
      kind: ScheduledPostKind.standard,
      status: ScheduledPostStatus.scheduled,
      text: 'Scheduled $index',
      scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
    ),
  );

  @override
  Future<ScheduledPostDetail> get(String id) async =>
      detail ??
      (id == _scheduledProjectDetail.id
          ? _scheduledProjectDetail
          : _scheduledDetail);

  @override
  Future<Uint8List> mediaBytes(String id) {
    mediaRequests.add(id);
    if (!mediaStarted.isCompleted) mediaStarted.complete();
    return mediaResponse?.future ?? Future.value(bytes);
  }

  @override
  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) async {
    updatedPayload = payload;
    if (!updateStarted.isCompleted) updateStarted.complete();
    if (failUpdateCount > 0) {
      failUpdateCount -= 1;
      throw StateError('update failed');
    }
    if (updateResponse != null) return updateResponse!.future;
    return detail ?? _scheduledDetail;
  }

  @override
  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
    CancelToken? cancelToken,
  }) async {
    stagedIDs.add(id);
    stagedBytes.add(Uint8List.fromList(bytes));
    if (failStage) throw StateError('stage failed');
  }

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();

  @override
  Future<void> delete(String id) async {}

  @override
  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) async {
    publishedPayload = payload;
  }
}

final class _NoFetchPreviewRepository implements LinkPreviewRepository {
  int calls = 0;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    calls += 1;
    throw StateError('scheduled frozen preview must not refetch');
  }
}

final class _ReplacementPreviewRepository implements LinkPreviewRepository {
  _ReplacementPreviewRepository(this.bytes);

  final Uint8List bytes;
  final urls = <String>[];

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    urls.add(url.toString());
    return LinkPreview(
      url: Uri.parse('https://replacement.example/final'),
      title: 'Replacement pattern',
      description: 'Replacement description',
      thumbnail: LinkPreviewThumbnail(
        bytes: bytes,
        mimeType: 'image/png',
        width: 3,
        height: 1,
      ),
    );
  }
}

final class _ChangingReplacementPreviewRepository
    implements LinkPreviewRepository {
  _ChangingReplacementPreviewRepository(this.bytes);

  final List<Uint8List> bytes;
  int calls = 0;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    final index = calls++;
    return LinkPreview(
      url: Uri.parse('https://replacement.example/final-$index'),
      title: 'Replacement pattern $index',
      description: 'Replacement description $index',
      thumbnail: LinkPreviewThumbnail(
        bytes: bytes[index],
        mimeType: 'image/png',
        width: index + 3,
        height: 1,
      ),
    );
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}

final class _EmptyMentionResolver implements MentionResolver {
  const _EmptyMentionResolver();

  @override
  Future<String?> didForHandle(String handle) async => null;
}

final ThemeData _testTheme =
    ThemeData.from(
      colorScheme: const ColorScheme.light(
        primary: BrandColors.cobalt,
        onSurface: BrandColors.ink,
        onSurfaceVariant: BrandColors.ink2,
        outline: BrandColors.ink3,
        outlineVariant: BrandColors.ink4,
        error: BrandColors.red,
      ),
    ).copyWith(
      scaffoldBackgroundColor: BrandColors.paper,
      extensions: const [
        SpacingTheme(),
        RadiusTheme(),
        DurationTheme(),
        BrandShadowTheme(),
        BrandSwatchTheme(),
        SemanticColorsTheme(
          error: BrandColors.red,
          warning: BrandColors.butter,
          success: BrandColors.moss,
          info: BrandColors.cobalt,
          errorSurface: BrandColors.redSoft,
          warningSurface: BrandColors.butter,
          successSurface: BrandColors.moss,
          infoSurface: BrandColors.cobaltSoft,
        ),
      ],
    );
