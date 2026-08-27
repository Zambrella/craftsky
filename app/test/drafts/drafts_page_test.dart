import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/pages/drafts_page.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows draft previews, kinds, thumbnails, and damaged state', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(800, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final owner = AccountKey('did:plc:alice');
    final items = [
      _draft(
        'standard',
        owner,
        const StandardDraftContent(
          text: 'A knitted blue cardigan',
          languages: ['en'],
        ),
        media: const [_media],
      ),
      _draft(
        'project',
        owner,
        const ProjectDraftContent(
          body: 'Project notes',
          languages: ['en'],
          knownProjectFieldValues: {'title': 'Cardigan project'},
        ),
        kind: LocalPostDraftKind.project,
      ),
      LocalPostDraft.unavailable(id: 'damaged', owner: owner),
    ];
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: DraftsPageContent(
            items: items,
            onRefresh: () async {},
            onEdit: (_) async {},
            onDelete: (_) async {},
            thumbnailBuilder: (draftId, mediaId) => Text(
              'thumbnail:$draftId:$mediaId',
            ),
          ),
        ),
      ),
    );

    expect(find.text('Drafts'), findsOneWidget);
    expect(find.text('A knitted blue cardigan'), findsOneWidget);
    expect(find.text('Cardigan project'), findsOneWidget);
    expect(find.text('Project notes'), findsOneWidget);
    expect(find.text('Draft unavailable'), findsOneWidget);
    expect(find.textContaining('Standard ·'), findsOneWidget);
    expect(find.textContaining('Project ·'), findsOneWidget);
    expect(find.text('thumbnail:standard:${_media.mediaId}'), findsOneWidget);
    expect(find.byTooltip('Edit draft'), findsNWidgets(2));
    expect(find.byTooltip('Delete draft'), findsNWidgets(3));
  });

  testWidgets(
    'IT-004 opens a readable damaged draft with replacement controls',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = registry.activeLease!;
      final damaged =
          _draft(
            '00000000-0000-4000-8000-000000000001',
            lease.session.account,
            const StandardDraftContent(
              text: 'Recover this work',
              languages: ['en'],
            ),
            media: const [
              DraftMediaDescriptor(
                mediaId: '00000000-0000-4000-8000-000000000002',
                storageRevision: '00000000-0000-4000-8000-000000000003',
                storageFileName: 'missing.jpg',
                displayFileName: 'missing.jpg',
                mimeType: 'image/jpeg',
                byteLength: 3,
                sha256:
                    '039058c6f2c0cb492c533b0a4d14ef77'
                    'cc0f78abccced5287d84a1a2011cfb81',
                width: 1,
                height: 1,
                altText: 'Preserved description',
                order: 0,
                availability: DraftMediaAvailability.unavailable,
              ),
            ],
          ).withStorageState(
            availability: LocalPostDraftAvailability.unavailable,
            media: const [
              DraftMediaDescriptor(
                mediaId: '00000000-0000-4000-8000-000000000002',
                storageRevision: '00000000-0000-4000-8000-000000000003',
                storageFileName: 'missing.jpg',
                displayFileName: 'missing.jpg',
                mimeType: 'image/jpeg',
                byteLength: 3,
                sha256:
                    '039058c6f2c0cb492c533b0a4d14ef77'
                    'cc0f78abccced5287d84a1a2011cfb81',
                width: 1,
                height: 1,
                altText: 'Preserved description',
                order: 0,
                availability: DraftMediaAvailability.unavailable,
              ),
            ],
          );
      final repository = _MemoryRepository([damaged]);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            activeAccountInitializationProvider.overrideWith(
              (ref) => ActiveAccountInitialization(
                lease: lease,
                languagePreferences: const LanguagePreferences(
                  primaryLanguage: 'en',
                  contentLanguages: ['en'],
                ),
              ),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            accountLocalPostDraftRepositoryProvider(
              lease.session.account,
            ).overrideWith((ref) async => repository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const DraftsPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('Edit draft'));
      await tester.pumpAndSettle();

      expect(find.byType(PostComposerSheet), findsOneWidget);
      expect(find.text('Recover this work'), findsOneWidget);
      expect(find.text('Image unavailable'), findsOneWidget);
      expect(find.text('Replace image'), findsOneWidget);
      expect(find.text('Preserved description'), findsOneWidget);
    },
  );

  testWidgets('AT-006 reopened draft derives a fresh preview session', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = registry.activeLease!;
    final repository = _MemoryRepository([
      _draft(
        '00000000-0000-4000-8000-000000000010',
        lease.session.account,
        const StandardDraftContent(
          text: 'Resume https://source.example/pattern ',
          languages: ['en'],
        ),
      ),
    ]);
    final previews = _DraftPreviewRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeAccountInitializationProvider.overrideWith(
            (ref) => ActiveAccountInitialization(
              lease: lease,
              languagePreferences: const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          accountLocalPostDraftRepositoryProvider(
            lease.session.account,
          ).overrideWith((ref) async => repository),
          linkPreviewRepositoryProvider.overrideWithValue(previews),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const DraftsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Edit draft'));
    await tester.pumpAndSettle();

    expect(find.text('Fresh preview'), findsOneWidget);
    expect(previews.urls, ['https://source.example/pattern']);
  });
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

final class _MemoryRepository implements LocalPostDraftRepository {
  _MemoryRepository(this.items);

  final List<LocalPostDraft> items;

  @override
  Future<List<LocalPostDraft>> list() async => items;

  @override
  Future<LocalPostDraft> get(String draftId) async =>
      items.singleWhere((draft) => draft.id == draftId);

  @override
  Future<Uint8List> readMedia(String draftId, String mediaId) async =>
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.unavailable,
      );

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _DraftPreviewRepository implements LinkPreviewRepository {
  final urls = <String>[];

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    urls.add(url.toString());
    return LinkPreview(
      url: url,
      title: 'Fresh preview',
      description: 'Fresh description',
    );
  }
}

LocalPostDraft _draft(
  String id,
  AccountKey owner,
  LocalDraftContent content, {
  LocalPostDraftKind kind = LocalPostDraftKind.standard,
  List<DraftMediaDescriptor> media = const [],
}) => LocalPostDraft(
  id: id,
  owner: owner,
  kind: kind,
  createdAt: DateTime.utc(2026, 8, 3, 10),
  updatedAt: DateTime.utc(2026, 8, 3, 11),
  content: content,
  schedule: const DraftScheduleIntent.now(),
  media: media,
);

const _media = DraftMediaDescriptor(
  mediaId: '00000000-0000-4000-8000-000000000002',
  storageRevision: '00000000-0000-4000-8000-000000000003',
  storageFileName: 'image.jpg',
  displayFileName: 'image.jpg',
  mimeType: 'image/jpeg',
  byteLength: 1,
  sha256:
      '0123456789abcdef0123456789abcdef'
      '0123456789abcdef0123456789abcdef',
  width: 1,
  height: 1,
  altText: '',
  order: 0,
);
