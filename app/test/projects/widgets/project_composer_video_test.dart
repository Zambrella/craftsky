import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-013 project composer accepts video as its required media', (
    tester,
  ) async {
    final videos = ComposerVideoController(picker: _NoopVideoPicker())
      ..restoredSelection = LocalVideoSelection(
        displayName: 'project.mp4',
        mimeType: 'video/mp4',
        byteLength: 12,
        duration: const Duration(seconds: 2),
        width: 1080,
        height: 1920,
        headerBytes: Uint8List(0),
        openRead: () => Stream.value(List.filled(12, 1)),
        posterBytes: Uint8List.fromList([1]),
        altText: 'Project in progress',
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
          composerImagesProvider('project-video').overrideWithValue(
            const ComposerImagesState(images: []),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ProjectComposerSheet(
              composerId: 'project-video',
              videoController: videos,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(ComposerVideoAttachmentCard), findsOneWidget);
    expect(find.text('Add photos'), findsNothing);
  });

  testWidgets('AT-001 project publication preserves project and video proof', (
    tester,
  ) async {
    final videos = ComposerVideoController(picker: _NoopVideoPicker())
      ..restoredSelection = LocalVideoSelection(
        displayName: 'project.mp4',
        mimeType: 'video/mp4',
        byteLength: 12,
        duration: const Duration(seconds: 2),
        width: 1080,
        height: 1920,
        headerBytes: Uint8List(0),
        openRead: () => const Stream.empty(),
        altText: 'Project in progress',
      );
    var attempts = 0;
    final messenger = RecordingMessenger();
    final repository = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async => Post(
            uri: 'at://did:plc:alice/social.craftsky.feed.post/3video',
            cid: 'bafypost',
            rkey: '3video',
            text: text,
            tags: const [],
            likeCount: 0,
            repostCount: 0,
            replyCount: 0,
            viewerHasLiked: false,
            viewerHasReposted: false,
            viewerHasSaved: false,
            createdAt: DateTime(2026),
            indexedAt: DateTime(2026),
            author: PostAuthor(
              did: 'did:plc:alice',
              handle: 'alice.example',
            ),
            project: project,
          ),
    );
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'session-token',
      did: 'did:plc:alice',
      handle: 'alice.example',
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
          composerImagesProvider('project-video-publish').overrideWithValue(
            const ComposerImagesState(images: []),
          ),
          postRepositoryProvider.overrideWithValue(repository),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ProjectComposerSheet(
              composerId: 'project-video-publish',
              videoController: videos,
              prepareVideoProof: (_) async {
                attempts++;
                if (attempts == 1) {
                  throw const VideoPublicationException(
                    VideoServiceOutcome.quotaExhausted,
                  );
                }
                return const CreatePostVideo(
                  jobId: 'job-project',
                  blob: CreatePostVideoBlob(
                    cid: 'bafyvideo',
                    mimeType: 'video/mp4',
                    size: 12,
                  ),
                  alt: 'Project in progress',
                  aspectRatio: CreatePostVideoAspectRatio(
                    width: 1080,
                    height: 1920,
                  ),
                );
              },
            ),
          ),
        ),
      ),
    );
    final container = ProviderScope.containerOf(
      tester.element(find.byType(MaterialApp)),
    );
    await container.read(sessionRegistryProvider.future);
    await tester.pumpAndSettle();
    await _selectCraft(tester, 'Embroidery');
    for (var i = 0; i < 3 && _bodyTextField().evaluate().isEmpty; i++) {
      await _goNext(tester);
    }
    expect(_bodyTextField(), findsOneWidget);
    await tester.enterText(_bodyTextField(), 'Finished my project');
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    final failure = messenger.calls.last;
    expect(failure.$2, contains('daily video limit'));
    expect(failure.$3?.label, 'Retry');
    expect(videos.selection, isNotNull);
    failure.$3!.onPressed();
    await tester.pumpAndSettle();

    expect(attempts, 2);
    expect(repository.lastCreateVideo?.jobId, 'job-project');
    expect(repository.lastCreateVideo?.blob.cid, 'bafyvideo');
  });
}

final class _NoopVideoPicker implements ExistingVideoPicker {
  @override
  Future<LocalVideoSelection?> pickExisting() async => null;
}

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i++) {
    await tester.pump(const Duration(milliseconds: 20));
    final button = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Post'),
    );
    if (button.onPressed != null) return;
  }
  fail('Timed out waiting for Post button to be enabled');
}

Finder _bodyTextField() => find.descendant(
  of: find.byKey(const Key('project-composer-body-editor')),
  matching: find.byType(TextField),
);

Future<void> _goNext(WidgetTester tester) async {
  await tester.tap(find.byKey(const Key('project-composer-primary-action')));
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final dropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(dropdown);
  await tester.pumpAndSettle();
  await tester.tap(dropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}
