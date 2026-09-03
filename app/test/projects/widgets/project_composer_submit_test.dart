import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-004 submits a valid common-only embroidery project', (
    tester,
  ) async {
    String? capturedText;
    PostReply? capturedReply;
    Project? capturedProject;
    List<CreatePostImage>? capturedImages;
    List<Map<String, dynamic>>? capturedFacets;
    final repo = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            capturedText = text;
            capturedReply = reply;
            capturedProject = project;
            capturedImages = images;
            capturedFacets = facets;
            return _post(text: text, project: project);
          },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          composerImagesProvider('project-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'hoop.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Finished embroidery hoop on a table',
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
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'project-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await _goNext(tester);
    await _goNext(tester);
    await tester.enterText(
      _bodyTextField(),
      'Finished my hoop #embroidery',
    );
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(capturedText, 'Finished my hoop #embroidery');
    expect(capturedReply, isNull);
    expect(capturedImages, hasLength(1));
    expect(capturedImages!.single.alt, 'Finished embroidery hoop on a table');
    expect(capturedProject, isNotNull);
    expect(
      capturedProject!.common.craftType,
      ProjectOptionCatalogs.embroideryCraftToken,
    );
    expect(capturedProject!.details, isNull);
    expect(capturedProject!.common.title, isNull);
    expect(capturedProject!.common.materials, isNull);
    expect(capturedProject!.common.colors, isNull);
    expect(capturedProject!.common.designTags, isNull);
    expect(capturedFacets, isNotNull);
    expect(
      capturedFacets!.expand((facet) => facet['features']! as List<dynamic>),
      contains(
        predicate<Object?>(
          (feature) =>
              feature is Map<String, dynamic> &&
              feature[r'$type'] == 'app.bsky.richtext.facet#tag' &&
              feature['tag'] == 'embroidery',
          'tag facet for #embroidery',
        ),
      ),
    );
  });

  testWidgets('AT-007 submits non-empty metadata and pattern fields', (
    tester,
  ) async {
    Project? capturedProject;
    final repo = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            capturedProject = project;
            return _post(text: text, project: project);
          },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          composerImagesProvider('metadata-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'dress.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Blue handmade dress',
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
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'metadata-composer'),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Sewing');
    await tester.enterText(_patternNameTextField(), 'Garden dress');
    await tester.pumpAndSettle();
    await _openDetail(
      tester,
      const Key('project-composer-pattern-details-action'),
    );
    await tester.enterText(
      find.byKey(const Key('pattern-url-input')),
      'https://patterns.example/garden-dress',
    );
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    await _openDetail(
      tester,
      const Key('project-composer-common-details-action'),
    );

    await tester.ensureVisible(_materialTextField());
    await tester.enterText(_materialTextField(), 'Cotton lawn');
    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.materials}-add-custom')),
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('${ProjectComposerFields.materials}-add-custom')),
    );
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
      'Blue',
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('${ProjectComposerFields.colours}-option-blue')),
    );
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.designTags}-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.designTags}-search-input')),
      'Floral',
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(
        const Key(
          '${ProjectComposerFields.designTags}-option-'
          'social.craftsky.project.defs#floral',
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    await tester.enterText(_bodyTextField(), 'Finished a dress');
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(capturedProject, isNotNull);
    final common = capturedProject!.common;
    expect(common.materials, const [ProjectMaterial(text: 'Cotton lawn')]);
    expect(common.colors, ['blue']);
    expect(common.designTags, ['social.craftsky.project.defs#floral']);
    expect(common.pattern, isNotNull);
    expect(common.pattern!.name, 'Garden dress');
    expect(common.pattern!.url, 'https://patterns.example/garden-dress');
  });

  testWidgets(
    'IT-015 account switch during project upload prevents create',
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
            composerImagesProvider(
              'project-account-switch',
            ).overrideWithValue(
              ComposerImagesState(
                images: [
                  ComposerImageDraft(
                    id: 'image-1',
                    fileName: 'work.jpg',
                    mimeType: 'image/jpeg',
                    altText: 'Project work',
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
                onCreateWithFacets:
                    ({required text, reply, project, images, facets}) async {
                      createCalls += 1;
                      return _post(text: text, project: project);
                    },
              ),
            ),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProjectComposerSheet(
                composerId: 'project-account-switch',
              ),
            ),
          ),
        ),
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ProjectComposerSheet)),
      );
      await container.read(sessionRegistryProvider.future);
      await tester.pumpAndSettle();
      await _selectCraft(tester, 'Embroidery');
      await _goNext(tester);
      await _goNext(tester);
      await tester.enterText(_bodyTextField(), 'Alice project');
      await _pumpUntilPostEnabled(tester);

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await _pumpUntil(tester, () => upload.uploadCalls == 1);
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
    'origin draft revision advances before retrying a project submission',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = registry.activeLease!;
      final projectBytes = Uint8List.fromList(
        img.encodePng(img.Image(width: 1, height: 1)),
      );
      final origin = _projectOriginDraft(
        lease.session.account,
        projectBytes,
      );
      final drafts = _RevisionDraftRepository(origin);
      final upload = _ImmediatePostApiClient();
      var createCalls = 0;

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
            accountLocalPostDraftRepositoryProvider(
              lease.session.account,
            ).overrideWith((ref) async => drafts),
            postApiClientProvider.overrideWith((ref) => upload),
            postRepositoryProvider.overrideWithValue(
              FakePostRepository(
                onCreateWithFacets:
                    ({required text, reply, project, images, facets}) async {
                      createCalls += 1;
                      if (createCalls == 1) {
                        throw StateError('network unavailable');
                      }
                      return _post(text: text, project: project);
                    },
              ),
            ),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: ProjectComposerSheet(
                composerId: 'project-retry',
                draftSeed: LocalPostDraftSeed(
                  draft: origin,
                  media: [
                    HydratedDraftMedia(
                      descriptor: origin.media.single,
                      bytes: projectBytes,
                    ),
                  ],
                ),
                draftOwner: lease,
              ),
            ),
          ),
        ),
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ProjectComposerSheet)),
      );
      await container.read(sessionRegistryProvider.future);
      await tester.pumpAndSettle();
      await _goNext(tester);
      await _goNext(tester);
      await _pumpUntilPostEnabled(tester);

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await tester.pumpAndSettle();

      expect(createCalls, 1);
      expect(drafts.expectedRevisions, [1]);
      expect(drafts.deletedIds, isEmpty);
      expect(find.byType(ProjectComposerSheet), findsOneWidget);

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await tester.pumpAndSettle();

      expect(createCalls, 2);
      expect(upload.uploadCalls, 1);
      expect(drafts.expectedRevisions, [1, 2]);
      expect(drafts.deletedIds, [origin.id]);
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

final class _ImmediatePostApiClient extends PostApiClient {
  _ImmediatePostApiClient() : super(Dio());

  int uploadCalls = 0;

  @override
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    uploadCalls += 1;
    return UploadedImageBlob(
      blob: UploadedBlob(
        type: 'blob',
        ref: const UploadedBlobRef(link: 'bafkproject'),
        mimeType: mimeType,
        size: bytes.length,
      ),
      cid: 'bafkproject',
      mime: mimeType,
      size: bytes.length,
    );
  }
}

final class _RevisionDraftRepository implements LocalPostDraftRepository {
  _RevisionDraftRepository(this.current);

  LocalPostDraft current;
  final List<int?> expectedRevisions = [];
  final List<String> deletedIds = [];

  @override
  Future<List<LocalPostDraft>> list() async => [current];

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
      media: current.media,
      revision: current.revision + 1,
    );
  }

  @override
  Future<void> delete(String draftId) async {
    deletedIds.add(draftId);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

LocalPostDraft _projectOriginDraft(AccountKey owner, Uint8List bytes) {
  return LocalPostDraft(
    id: '00000000-0000-4000-8000-000000000020',
    owner: owner,
    kind: LocalPostDraftKind.project,
    createdAt: DateTime.utc(2026, 8, 4, 10),
    updatedAt: DateTime.utc(2026, 8, 4, 11),
    content: const ProjectDraftContent(
      body: 'Retry this project',
      languages: ['en'],
      knownProjectFieldValues: {
        ProjectComposerFields.craftType:
            ProjectOptionCatalogs.embroideryCraftToken,
        ProjectComposerFields.status: 'finished',
      },
    ),
    schedule: const DraftScheduleIntent.now(),
    media: [
      DraftMediaDescriptor(
        mediaId: '00000000-0000-4000-8000-000000000021',
        storageRevision: '00000000-0000-4000-8000-000000000022',
        storageFileName: 'project.png',
        displayFileName: 'project.png',
        mimeType: 'image/png',
        byteLength: bytes.length,
        sha256: sha256.convert(bytes).toString(),
        width: 1,
        height: 1,
        altText: 'Project image',
        order: 0,
      ),
    ],
  );
}

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i += 1) {
    await tester.pump(const Duration(milliseconds: 20));
    final button = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Post'),
    );
    if (button.onPressed != null) return;
  }
  fail('Timed out waiting for Post button to be enabled');
}

Finder _bodyTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-body-editor')),
    matching: find.byType(TextField),
  );
}

Finder _patternNameTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-pattern-name-editor')),
    matching: find.byType(TextField),
  );
}

Finder _materialTextField() {
  return find.descendant(
    of: find.byKey(
      const Key('${ProjectComposerFields.materials}-custom-input'),
    ),
    matching: find.byType(TextField),
  );
}

Future<void> _goNext(WidgetTester tester) async {
  await tester.ensureVisible(
    find.byKey(const Key('project-composer-body-editor')),
  );
  await tester.pumpAndSettle();
}

Future<void> _openDetail(WidgetTester tester, Key key) async {
  final action = find.byKey(key);
  await tester.ensureVisible(action);
  await tester.pumpAndSettle();
  await tester.tap(action);
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

Post _post({required String text, Project? project}) {
  final now = DateTime.utc(2026, 6, 11);
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafyreibazjzrzibga2jwt5co2yus7j2w6p3n3cb6nn4njvkzcxwrlfvula',
    rkey: '3lf2abc',
    text: text,
    tags: const [],
    createdAt: now,
    indexedAt: now,
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    project: project,
  );
}
