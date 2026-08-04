import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  group('PostComposerSheet discard confirmation', () {
    testWidgets(
      'draft save listener keeps an asynchronous save alive until completion',
      (tester) async {
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
        final account = registry.activeLease!.session.account;
        final drafts = _GatedDraftSaveRepository();
        final messenger = RecordingMessenger();

        await _openComposer(
          tester,
          composerId: '96ad7199-292f-4388-a6cd-b4f74230116b',
          registry: registry,
          messenger: messenger,
          overrides: [
            accountLocalPostDraftRepositoryProvider(
              account,
            ).overrideWith((ref) async => drafts),
          ],
        );
        await tester.enterText(
          find.byType(TextField).first,
          'A cardigan WIP',
        );
        await tester.pump();

        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Save draft'))
            .onPressed!();
        await tester.pump(const Duration(milliseconds: 20));
        expect(drafts.request, isNotNull);

        // Cross an additional frame so an unlistened auto-dispose mutation
        // provider is released before its repository future completes.
        await tester.pump(const Duration(milliseconds: 20));
        drafts.complete();
        await tester.pumpAndSettle();

        expect(find.text('Host'), findsOneWidget);
        expect(
          messenger.calls,
          contains(('info', 'Draft saved', null)),
        );
        expect(
          messenger.calls.where((call) => call.$1 == 'error'),
          isEmpty,
        );
      },
    );

    testWidgets('IT-012 shared account guard cancels or closes dirty compose', (
      tester,
    ) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      await _openComposer(tester, registry: registry);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      final owner = registry.activeLease!.session;

      final cancelled = container
          .read(unsavedWorkGuardProvider)
          .confirmLeave(owner);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();
      expect(await cancelled, isFalse);
      expect(find.text('New post'), findsOneWidget);

      final confirmed = container
          .read(unsavedWorkGuardProvider)
          .confirmLeave(owner);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();
      expect(await confirmed, isTrue);
      expect(find.text('Host'), findsOneWidget);
    });

    testWidgets('closes immediately when the composer is unchanged', (
      tester,
    ) async {
      await _openComposer(tester);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
      expect(find.text('Discard draft?'), findsNothing);
    });

    testWidgets('close button confirms before discarding edits', (
      tester,
    ) async {
      await _openComposer(tester);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Save your draft?'), findsOneWidget);
      expect(
        find.text('You can save this work on this device before closing.'),
        findsOneWidget,
      );

      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();

      expect(find.text('Save your draft?'), findsNothing);
      expect(find.text('New post'), findsOneWidget);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
    });

    testWidgets('system back confirms before discarding edits', (tester) async {
      await _openComposer(tester);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();

      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();

      expect(find.text('Save your draft?'), findsOneWidget);

      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
    });
  });

  group('PostComposerSheet alt text warning', () {
    testWidgets('confirms before submitting images without alt text', (
      tester,
    ) async {
      var createCalls = 0;
      List<CreatePostImage>? submittedImages;
      final messenger = RecordingMessenger();
      final repo = FakePostRepository(
        onCreate: ({required text, reply, images}) async {
          createCalls += 1;
          submittedImages = images;
          return _post(text);
        },
      );

      await _openComposer(
        tester,
        composerId: 'composer',
        registry: SessionRegistry.empty().upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        ),
        messenger: messenger,
        overrides: [
          composerImagesProvider('composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'project.jpg',
                  mimeType: 'image/jpeg',
                  altText: '',
                  phase: ImageUploaded(
                    UploadedDraftImage(
                      cid: 'bafkimage',
                      mime: 'image/jpeg',
                      size: 123,
                    ),
                  ),
                ),
              ],
            ),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
      );
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await _pumpUntilPostEnabled(tester);

      await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
      await tester.pumpAndSettle();

      expect(find.text('Some images do not have alt text'), findsOneWidget);
      expect(find.text('Do you wish to post anyway?'), findsOneWidget);
      expect(createCalls, 0);

      await tester.tap(find.text('Post anyway'));
      await tester.pumpAndSettle();

      expect(messenger.calls.where((call) => call.$1 == 'error'), isEmpty);
      expect(createCalls, 1);
      expect(submittedImages, hasLength(1));
      expect(submittedImages!.single.alt, isEmpty);
    });
  });

  testWidgets(
    'IT-015 account switch during standard upload prevents create',
    (tester) async {
      var createCalls = 0;
      final upload = _GatedPostApiClient();
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final bytes = Uint8List.fromList([1, 2, 3]);
      await _openComposer(
        tester,
        composerId: 'account-switch-composer',
        registry: registry,
        overrides: [
          composerImagesProvider(
            'account-switch-composer',
          ).overrideWithValue(
            ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'work.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Work in progress',
                  phase: ImageReady(
                    bytes: bytes,
                    mimeType: 'image/jpeg',
                    width: 1,
                    height: 1,
                    sha256: sha256.convert(bytes).toString(),
                  ),
                ),
              ],
            ),
          ),
          postApiClientProvider.overrideWith((ref) => upload),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onCreate: ({required text, reply, images}) async {
                createCalls += 1;
                return _post(text);
              },
            ),
          ),
        ],
      );
      await tester.enterText(find.byType(TextField).first, 'Alice work');
      await _pumpUntilPostEnabled(tester);

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await _pumpUntil(tester, () => upload.uploadCalls == 1);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostComposerSheet)),
      );
      final bob = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bob);
      upload.complete();
      await tester.pumpAndSettle();

      expect(createCalls, 0);
      expect(upload.uploadCalls, 1);
    },
  );

  testWidgets(
    'origin draft revision advances before retrying a standard submission',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = registry.activeLease!;
      final origin = _originDraft(lease.session.account);
      final drafts = _RevisionDraftRepository(origin);
      var createCalls = 0;
      await _openComposer(
        tester,
        registry: registry,
        draftSeed: LocalPostDraftSeed(draft: origin, media: const []),
        draftOwner: lease,
        overrides: [
          accountLocalPostDraftRepositoryProvider(
            lease.session.account,
          ).overrideWith((ref) async => drafts),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onCreate: ({required text, reply, images}) async {
                createCalls += 1;
                if (createCalls == 1) throw StateError('network unavailable');
                return _post(text);
              },
            ),
          ),
        ],
      );
      await _pumpUntilPostEnabled(tester);

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await tester.pumpAndSettle();

      expect(createCalls, 1);
      expect(drafts.expectedRevisions, [1]);
      expect(drafts.deletedIds, isEmpty);
      expect(find.byType(PostComposerSheet), findsOneWidget);

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await tester.pumpAndSettle();

      expect(createCalls, 2);
      expect(drafts.expectedRevisions, [1, 2]);
      expect(drafts.deletedIds, [origin.id]);
      expect(find.byType(PostComposerSheet), findsNothing);
    },
  );
}

Future<void> _pumpUntil(WidgetTester tester, bool Function() condition) async {
  for (var attempt = 0; attempt < 100; attempt++) {
    await tester.pump(const Duration(milliseconds: 20));
    if (condition()) return;
  }
  fail('Condition did not become true');
}

final class _GatedPostApiClient extends PostApiClient {
  _GatedPostApiClient() : super(Dio());

  final _gate = Completer<void>();
  int uploadCalls = 0;

  void complete() => _gate.complete();

  @override
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    uploadCalls += 1;
    await _gate.future;
    return const UploadedImageBlob(
      blob: UploadedBlob(
        type: 'blob',
        ref: UploadedBlobRef(link: 'bafkimage'),
        mimeType: 'image/jpeg',
        size: 3,
      ),
      cid: 'bafkimage',
      mime: 'image/jpeg',
      size: 3,
    );
  }
}

Future<void> _openComposer(
  WidgetTester tester, {
  List<dynamic> overrides = const [],
  String? composerId,
  SessionRegistry? registry,
  RecordingMessenger? messenger,
  LocalPostDraftSeed? draftSeed,
  ActiveAccountLease? draftOwner,
}) async {
  final providerOverrides = <dynamic>[
    activeLanguagePreferencesProvider.overrideWith(
      (ref) => const LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      ),
    ),
    ...overrides,
  ];
  if (registry != null) {
    providerOverrides.add(
      secureSessionRegistryStorageProvider.overrideWithValue(
        _RegistryStorage(registry),
      ),
    );
  }
  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from(providerOverrides),
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: _testTheme,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) {
              return Scaffold(
                body: Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Text('Host'),
                      ElevatedButton(
                        onPressed: () {
                          unawaited(
                            Navigator.of(context).push<Post?>(
                              MaterialPageRoute<Post?>(
                                fullscreenDialog: true,
                                builder: (_) => PostComposerSheet(
                                  composerId: composerId,
                                  draftSeed: draftSeed,
                                  draftOwner: draftOwner,
                                ),
                              ),
                            ),
                          );
                        },
                        child: const Text('Open composer'),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
      ),
    ),
  );

  if (registry != null) {
    final container = ProviderScope.containerOf(
      tester.element(find.byType(MaterialApp)),
    );
    await container.read(sessionRegistryProvider.future);
  }

  await tester.tap(find.text('Open composer'));
  await tester.pumpAndSettle();
}

final class _RevisionDraftRepository implements LocalPostDraftRepository {
  _RevisionDraftRepository(this.current);

  LocalPostDraft current;
  final List<int?> expectedRevisions = [];
  final List<String> deletedIds = [];

  @override
  Future<List<LocalPostDraft>> list() async => [current];

  @override
  Future<LocalPostDraft> get(String draftId) async => current;

  @override
  Future<Uint8List> readMedia(String draftId, String mediaId) async =>
      throw UnimplementedError();

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) async {
    expectedRevisions.add(request.expectedRevision);
    if (request.expectedRevision != current.revision) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.conflict,
      );
    }
    return current = LocalPostDraft(
      id: current.id,
      owner: current.owner,
      kind: request.kind,
      createdAt: current.createdAt,
      updatedAt: current.updatedAt.add(const Duration(seconds: 1)),
      content: request.content,
      schedule: request.schedule,
      media: const [],
      revision: current.revision + 1,
    );
  }

  @override
  Future<void> delete(String draftId) async {
    deletedIds.add(draftId);
  }
}

final class _GatedDraftSaveRepository implements LocalPostDraftRepository {
  final _gate = Completer<void>();
  DraftWriteRequest? request;
  LocalPostDraft? saved;

  void complete() => _gate.complete();

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) async {
    this.request = request;
    await _gate.future;
    return saved = LocalPostDraft(
      id: request.id,
      owner: request.owner,
      kind: request.kind,
      createdAt: DateTime.utc(2026, 8, 4),
      updatedAt: DateTime.utc(2026, 8, 4),
      content: request.content,
      schedule: request.schedule,
      media: const [],
    );
  }

  @override
  Future<List<LocalPostDraft>> list() async => [?saved];

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

LocalPostDraft _originDraft(AccountKey owner) => LocalPostDraft(
  id: '00000000-0000-4000-8000-000000000010',
  owner: owner,
  kind: LocalPostDraftKind.standard,
  createdAt: DateTime.utc(2026, 8, 4, 10),
  updatedAt: DateTime.utc(2026, 8, 4, 11),
  content: const StandardDraftContent(
    text: 'Retry this post',
    languages: ['en'],
  ),
  schedule: const DraftScheduleIntent.now(),
  media: const [],
);

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

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i += 1) {
    await tester.pump(const Duration(milliseconds: 20));
    final buttons = find.widgetWithText(ChunkyButton, 'Post').evaluate();
    if (buttons.isEmpty) continue;
    final button = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Post'),
    );
    if (button.onPressed != null) return;
  }
  fail('Timed out waiting for Post button to be enabled');
}

Post _post(String text) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafy123',
    rkey: '3lf2abc',
    text: text,
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    createdAt: DateTime(2026, 5, 22, 12),
    indexedAt: DateTime(2026, 5, 22, 12, 1),
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
  );
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
