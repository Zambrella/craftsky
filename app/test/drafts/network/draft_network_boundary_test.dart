import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final cases = <String, Widget Function(ActiveAccountLease)>{
    'standard': (_) => const PostComposerSheet(composerId: 'boundary'),
    'quote': (_) => PostComposerSheet(
      composerId: 'boundary',
      quoteTarget: _post(),
    ),
    'reply': (_) => PostComposerSheet(
      composerId: 'boundary',
      replyTarget: _post(),
    ),
    'project': (_) => const ProjectComposerSheet(composerId: 'boundary'),
    'new schedule': (_) => const PostComposerSheet(composerId: 'boundary'),
    'scheduled edit': (lease) => PostComposerSheet(
      composerId: 'boundary',
      scheduledPost: _scheduledPost,
      scheduledOwner: lease,
    ),
  };

  for (final MapEntry(key: origin, value: buildComposer) in cases.entries) {
    testWidgets('IT-010 $origin pre-submit actions perform zero transfers', (
      tester,
    ) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final lease = registry.activeLease!;
      final upload = _FailFastPostApiClient();
      final scheduled = _FailFastScheduledRepository();
      final bytesA = Uint8List.fromList([1, 2, 3]);
      final bytesB = Uint8List.fromList([4, 5, 6]);

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
            postApiClientProvider.overrideWith((ref) => upload),
            accountScheduledPostRepositoryProvider(
              lease.session.account,
            ).overrideWith((ref) async => scheduled),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: buildComposer(lease),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(MaterialApp)),
      );
      container.read(composerImagesProvider('boundary').notifier)
        ..seedScheduledImages([
          _readyImage('image-a', bytesA),
          _readyImage('image-b', bytesB),
        ])
        ..setAltText('image-a', 'Edited locally')
        ..reorder(fromIndex: 0, toIndex: 1);
      await tester.pump();

      expect(upload.uploadCalls, 0);
      expect(scheduled.stageCalls, 0);
      expect(scheduled.mutationCalls, 0);
      expect(
        container
            .read(composerImagesProvider('boundary'))
            .images
            .map((image) => image.id),
        ['image-b', 'image-a'],
      );
    });
  }
}

ComposerImageDraft _readyImage(String id, Uint8List bytes) =>
    ComposerImageDraft(
      id: id,
      fileName: '$id.jpg',
      mimeType: 'image/jpeg',
      altText: id,
      phase: ImageReady(
        bytes: bytes,
        mimeType: 'image/jpeg',
        width: 1,
        height: 1,
        sha256: sha256.convert(bytes).toString(),
      ),
    );

final class _FailFastPostApiClient extends PostApiClient {
  _FailFastPostApiClient() : super(Dio());

  int uploadCalls = 0;

  @override
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) {
    uploadCalls += 1;
    throw StateError('unexpected public upload');
  }
}

final class _FailFastScheduledRepository implements ScheduledPostRepository {
  int stageCalls = 0;
  int mutationCalls = 0;

  @override
  Future<List<ScheduledPostSummary>> list() async => const [];

  @override
  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
    CancelToken? cancelToken,
  }) {
    stageCalls += 1;
    throw StateError('unexpected private staging');
  }

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) {
    mutationCalls += 1;
    throw StateError('unexpected schedule create');
  }

  @override
  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) {
    mutationCalls += 1;
    throw StateError('unexpected schedule update');
  }

  @override
  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) {
    mutationCalls += 1;
    throw StateError('unexpected publish now');
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
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

Post _post() => Post(
  uri: 'at://did:plc:bob/social.craftsky.feed.post/3lf2abc',
  cid: 'bafy123',
  rkey: '3lf2abc',
  text: 'Existing post',
  tags: const [],
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  createdAt: DateTime.utc(2026, 8),
  indexedAt: DateTime.utc(2026, 8),
  author: PostAuthor(did: 'did:plc:bob', handle: 'bob.test'),
);

final _scheduledPost = ScheduledPostDetail(
  id: 'schedule-1',
  operationId: 'operation-1',
  status: ScheduledPostStatus.scheduled,
  scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 5, 9, 30)),
  payload: const {
    'text': 'Scheduled locally',
    'langs': ['en'],
    'media': <Map<String, dynamic>>[],
  },
);
