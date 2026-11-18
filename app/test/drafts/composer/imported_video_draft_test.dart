import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/models/video_draft_descriptor.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/models/instagram_post_import_source.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  for (final kind in LocalPostDraftKind.values) {
    testWidgets(
      '${kind.name} import waits for video restore and can save removal alone',
      (tester) async {
        final drafts = await _openImportedDraft(tester, kind);
        final save = find.widgetWithText(TextButton, 'Save changes');
        final remove = find.byKey(const Key('composer-remove-video'));
        expect(drafts.posterReads, 1);
        expect(remove, findsNothing);
        expect(tester.widget<TextButton>(save).onPressed, isNull);

        drafts.finishRestoring();
        await tester.pumpAndSettle();
        expect(remove, findsOneWidget);
        expect(tester.widget<TextButton>(save).onPressed, isNull);

        await tester.ensureVisible(remove);
        await tester.tap(remove);
        await tester.pumpAndSettle();
        expect(remove, findsNothing);
        expect(tester.widget<TextButton>(save).onPressed, isNotNull);

        await tester.tap(save);
        await tester.pumpAndSettle();
        expect(drafts.writes, hasLength(1));
        expect(drafts.writes.single.video, isNull);
        expect(find.text('Open imported draft'), findsOneWidget);
      },
    );
  }

  testWidgets('closing a still-restoring project draft does not edit it', (
    tester,
  ) async {
    final drafts = await _openImportedDraft(tester, LocalPostDraftKind.project);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.text('Save your draft?'), findsNothing);
    expect(find.text('Open imported draft'), findsOneWidget);

    drafts.finishRestoring();
    await tester.pumpAndSettle();
    expect(drafts.writes, isEmpty);
  });
}

Future<_GatedVideoDraftRepository> _openImportedDraft(
  WidgetTester tester,
  LocalPostDraftKind kind,
) async {
  final registry = SessionRegistry.empty().upsertAndActivate(
    token: 'session-token',
    did: 'did:plc:alice',
    handle: 'alice.example',
  );
  final owner = registry.activeLease!;
  final draft = _videoDraft(owner.session.account, kind);
  final drafts = _GatedVideoDraftRepository(draft);
  final seed = LocalPostDraftSeed(draft: draft, media: const []);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        accountLocalPostDraftRepositoryProvider(
          draft.owner,
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
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => Navigator.of(context).push<void>(
                  MaterialPageRoute<void>(
                    builder: (_) => switch (kind) {
                      LocalPostDraftKind.standard => PostComposerSheet(
                        draftSeed: seed,
                        draftOwner: owner,
                      ),
                      LocalPostDraftKind.project => ProjectComposerSheet(
                        draftSeed: seed,
                        draftOwner: owner,
                      ),
                    },
                  ),
                ),
                child: const Text('Open imported draft'),
              ),
            ),
          ),
        ),
      ),
    ),
  );
  final container = ProviderScope.containerOf(
    tester.element(find.byType(MaterialApp)),
  );
  await container.read(sessionRegistryProvider.future);
  await tester.tap(find.text('Open imported draft'));
  await tester.pumpAndSettle();
  return drafts;
}

final class _GatedVideoDraftRepository
    implements LocalPostDraftRepository, LocalVideoDraftRepository {
  new(this._draft);

  final LocalPostDraft _draft;
  final _poster = Completer<Uint8List>();
  final writes = <DraftWriteRequest>[];
  int posterReads = 0;

  void finishRestoring() => _poster.complete(Uint8List.fromList([1]));

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) async {
    writes.add(request);
    return _draft;
  }

  @override
  Future<List<LocalPostDraft>> list() async => [_draft];

  @override
  Future<Uint8List> readVideoPoster(String draftId) {
    posterReads++;
    return _poster.future;
  }

  @override
  Stream<List<int>> openVideoSource(String draftId) => const Stream.empty();

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _RegistryStorage implements SessionRegistryStorage {
  new(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}

LocalPostDraft _videoDraft(AccountKey owner, LocalPostDraftKind kind) =>
    LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000010',
      owner: owner,
      kind: kind,
      createdAt: DateTime.utc(2026, 8, 4, 10),
      updatedAt: DateTime.utc(2026, 8, 4, 11),
      content: switch (kind) {
        LocalPostDraftKind.standard => const StandardDraftContent(
          text: 'Saved caption',
          languages: ['en'],
        ),
        LocalPostDraftKind.project => const ProjectDraftContent(
          body: 'Saved caption',
          languages: ['en'],
          knownProjectFieldValues: {},
        ),
      },
      schedule: const DraftScheduleIntent.now(),
      media: const [],
      video: const VideoDraftDescriptor(
        storageRevision: '00000000-0000-4000-8000-000000000011',
        sourceStorageFileName: 'video.mp4',
        posterStorageFileName: 'video-poster.jpg',
        displayFileName: 'video.mp4',
        mimeType: 'video/mp4',
        sourceByteLength: 12,
        sourceSha256:
            'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789',
        posterByteLength: 1,
        posterSha256:
            'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789',
        posterMimeType: 'image/jpeg',
        width: 1080,
        height: 1920,
        duration: Duration(seconds: 2),
        altText: 'Video of the finished project',
      ),
      instagramImport: InstagramPostImportSource(
        sourceKey: 'abcdef0123456789abcdef0123456789',
        createdAt: DateTime.utc(2021, 8, 4, 12),
      ),
    );
