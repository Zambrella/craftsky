import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    as registry_provider;
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-013 publication starts without a metered-network prompt', (
    tester,
  ) async {
    final publicationStarted = Completer<void>();
    final proofReady = Completer<CreatePostVideo>();
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'session-token',
      did: 'did:plc:alice',
      handle: 'alice.example',
    );
    final videos = ComposerVideoController(picker: _NoopPicker())
      ..restoredSelection = _selection();

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
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onCreate: ({required text, reply, images}) async =>
                  _createdPost(text),
            ),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: PostComposerSheet(
              composerId: 'metered-video',
              videoController: videos,
              prepareVideoProof: (_) {
                if (!publicationStarted.isCompleted) {
                  publicationStarted.complete();
                }
                return proofReady.future;
              },
            ),
          ),
        ),
      ),
    );
    final container = ProviderScope.containerOf(
      tester.element(find.byType(MaterialApp)),
    );
    await container.read(registry_provider.sessionRegistryProvider.future);
    await tester.enterText(find.byType(TextField).first, 'Metered publish');
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pump();

    expect(publicationStarted.isCompleted, isTrue);
    expect(find.byType(AlertDialog), findsNothing);

    proofReady.complete(_proof());
    await tester.pumpAndSettle();
  });
}

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i++) {
    await tester.pump(const Duration(milliseconds: 20));
    final finder = find.widgetWithText(ChunkyButton, 'Post');
    if (finder.evaluate().isNotEmpty &&
        tester.widget<ChunkyButton>(finder).onPressed != null) {
      return;
    }
  }
  fail('Timed out waiting for Post button to be enabled');
}

LocalVideoSelection _selection() => LocalVideoSelection(
  displayName: 'shawl.mp4',
  mimeType: 'video/mp4',
  byteLength: 12,
  duration: const Duration(seconds: 2),
  width: 1080,
  height: 1920,
  headerBytes: Uint8List(0),
  openRead: () => const Stream.empty(),
);

CreatePostVideo _proof() => const CreatePostVideo(
  jobId: 'job-1',
  blob: CreatePostVideoBlob(
    cid: 'bafyvideo',
    mimeType: 'video/mp4',
    size: 12,
  ),
);

Post _createdPost(String text) => Post(
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
  author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
);

final class _NoopPicker implements ExistingVideoPicker {
  @override
  Future<LocalVideoSelection?> pickExisting() async => null;
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
