import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_staging_progress.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-004 stages private media before creating the schedule', (
    tester,
  ) async {
    final stageGate = Completer<void>();
    final scheduled = _SubmissionRepository(stageGate: stageGate);
    final messenger = RecordingMessenger();
    var publicCreateCalls = 0;
    final publicPosts = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            publicCreateCalls += 1;
            return _post(text);
          },
    );

    await tester.pumpWidget(
      _testApp(
        scheduled: scheduled,
        publicPosts: publicPosts,
        messenger: messenger,
      ),
    );
    await tester.pumpAndSettle();
    expect(scheduled.listCalls, greaterThan(0));
    await tester.enterText(find.byType(TextField).first, 'Schedule with media');
    await _selectLater(tester);
    await _pumpUntilEnabled(tester, 'Schedule');
    tester
        .widget<TextButton>(find.widgetWithText(TextButton, 'Schedule'))
        .onPressed!();
    await tester.pump();

    await _waitForReal(tester, () => scheduled.stageCalls == 1);
    await tester.pump();
    final progress = tester.widget<ScheduledStagingProgress>(
      find.byType(ScheduledStagingProgress),
    );
    expect(progress.completed, 0);
    expect(progress.total, 1);
    expect(scheduled.createCalls, 0);
    expect(publicCreateCalls, 0);
    expect(scheduled.stagedBytes, isNotEmpty);
    expect(scheduled.stagedMimeType, 'image/png');

    stageGate.complete();
    await _waitForReal(tester, () => scheduled.createCalls == 1);
    await tester.pump();

    expect(scheduled.createCalls, 1);
    expect(scheduled.events, ['stage:start', 'stage:done', 'create']);
    expect(scheduled.createdPayload?['media'], [
      {
        'id': 'local-image',
        'alt': 'A tiny project',
        'width': 3,
        'height': 2,
      },
    ]);
    expect(scheduled.createdPayload.toString(), isNot(contains('pds-cid')));
    expect(publicCreateCalls, 0);
  });

  testWidgets('AT-004 failure preserves the composer and retry identity', (
    tester,
  ) async {
    final scheduled = _SubmissionRepository(failFirstCreate: true);
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      _testApp(
        scheduled: scheduled,
        publicPosts: FakePostRepository(),
        messenger: messenger,
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField).first, 'Retry this schedule');
    await _selectLater(tester);
    await _pumpUntilEnabled(tester, 'Schedule');

    tester
        .widget<TextButton>(find.widgetWithText(TextButton, 'Schedule'))
        .onPressed!();
    await tester.pump();
    await _waitForReal(tester, () => scheduled.createCalls == 1);
    await tester.pump();

    expect(find.text('Retry this schedule'), findsOneWidget);
    expect(scheduled.createCalls, 1);
    expect(
      messenger.calls,
      contains(
        (
          'error',
          'Could not schedule post. Your draft is still here.',
          null,
        ),
      ),
    );

    await _pumpUntilEnabled(tester, 'Schedule');
    tester
        .widget<TextButton>(find.widgetWithText(TextButton, 'Schedule'))
        .onPressed!();
    await tester.pump();
    await _waitForReal(tester, () => scheduled.createCalls == 2);
    await tester.pump();

    expect(scheduled.createCalls, 2);
    expect(scheduled.operationIDs.toSet(), {'submission'});
  });
}

Widget _testApp({
  required _SubmissionRepository scheduled,
  required FakePostRepository publicPosts,
  required RecordingMessenger messenger,
}) {
  final registry = SessionRegistry.empty().upsertAndActivate(
    token: 'alice-token',
    did: 'did:plc:alice',
    handle: 'alice.test',
  );
  final account = registry.activeLease!.session.account;
  return ProviderScope(
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
      postRepositoryProvider.overrideWithValue(publicPosts),
      composerImagesProvider('submission').overrideWithValue(_readyImage),
      scheduledComposerMediaMaterializerProvider.overrideWithValue(
        _testMaterializer,
      ),
      accountScheduledPostRepositoryProvider(
        account,
      ).overrideWith((ref) async => scheduled),
    ],
    child: MessengerScope(
      messenger: messenger,
      child: MaterialApp(
        theme: _testTheme,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const PostComposerSheet(composerId: 'submission'),
      ),
    ),
  );
}

final _readyImage = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'local-image',
      fileName: 'project.png',
      mimeType: 'image/png',
      altText: 'A tiny project',
      previewBytes: _pngBytes(width: 3, height: 2),
      phase: const ImageUploaded(
        UploadedDraftImage(
          cid: 'pds-cid',
          mime: 'image/png',
          size: 100,
          aspectRatio: CreatePostImageAspectRatio(width: 3, height: 2),
        ),
      ),
    ),
  ],
);

Future<List<Map<String, dynamic>>> _testMaterializer(
  List<ComposerImageDraft> images, {
  required ScheduledMediaStager stageMedia,
  void Function(int)? onStaged,
}) async {
  final media = <Map<String, dynamic>>[];
  for (final image in images) {
    final uploaded = (image.phase as ImageUploaded).uploaded;
    await stageMedia(
      id: image.id,
      bytes: image.previewBytes!,
      mimeType: image.mimeType,
    );
    media.add({
      'id': image.id,
      'alt': image.altText,
      'width': uploaded.aspectRatio!.width,
      'height': uploaded.aspectRatio!.height,
    });
    onStaged?.call(media.length);
  }
  return media;
}

Future<void> _selectLater(WidgetTester tester) async {
  await tester.tap(find.text('When'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Schedule for later'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('OK'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('OK'));
  await tester.pumpAndSettle();
}

Future<void> _pumpUntilEnabled(WidgetTester tester, String label) async {
  await _pumpUntil(tester, () {
    final finder = find.widgetWithText(TextButton, label);
    return finder.evaluate().isNotEmpty &&
        tester.widget<TextButton>(finder).onPressed != null;
  });
}

Future<void> _pumpUntil(
  WidgetTester tester,
  bool Function() condition,
) async {
  for (var attempt = 0; attempt < 200; attempt++) {
    await tester.pump(const Duration(milliseconds: 20));
    if (condition()) return;
  }
  fail('Condition did not become true');
}

Future<void> _waitForReal(
  WidgetTester tester,
  bool Function() condition,
) async {
  await tester.runAsync(() async {
    for (var attempt = 0; attempt < 200; attempt++) {
      if (condition()) return;
      await Future<void>.delayed(const Duration(milliseconds: 10));
    }
    fail('Real asynchronous condition did not become true');
  });
}

Uint8List _pngBytes({required int width, required int height}) =>
    Uint8List.fromList(img.encodePng(img.Image(width: width, height: height)));

final class _SubmissionRepository implements ScheduledPostRepository {
  _SubmissionRepository({this.stageGate, this.failFirstCreate = false});

  final Completer<void>? stageGate;
  final bool failFirstCreate;
  int stageCalls = 0;
  int listCalls = 0;
  int createCalls = 0;
  List<int> stagedBytes = const [];
  String? stagedMimeType;
  Map<String, dynamic>? createdPayload;
  final operationIDs = <String>[];
  final events = <String>[];

  @override
  Future<List<ScheduledPostSummary>> list() async {
    listCalls += 1;
    return const [];
  }

  @override
  Future<void> stageMedia({
    required String id,
    required List<int> bytes,
    required String mimeType,
  }) async {
    stageCalls += 1;
    stagedBytes = List.unmodifiable(bytes);
    stagedMimeType = mimeType;
    events.add('stage:start');
    await stageGate?.future;
    events.add('stage:done');
  }

  @override
  Future<ScheduledPostDetail> create({
    required String operationId,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) async {
    createCalls += 1;
    operationIDs.add(operationId);
    createdPayload = payload;
    events.add('create');
    if (failFirstCreate && createCalls == 1) throw StateError('create failed');
    return ScheduledPostDetail(
      id: 'scheduled-id',
      operationId: operationId,
      status: ScheduledPostStatus.scheduled,
      scheduledAt: ScheduledInstant(scheduledAt),
      payload: payload,
    );
  }

  @override
  Future<void> delete(String id) async {}

  @override
  Future<ScheduledPostDetail> get(String id) => throw UnimplementedError();

  @override
  Future<Uint8List> mediaBytes(String id) => throw UnimplementedError();

  @override
  Future<void> publishNow({
    required String id,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();

  @override
  Future<ScheduledPostDetail> update({
    required String id,
    required DateTime scheduledAt,
    required Map<String, dynamic> payload,
  }) => throw UnimplementedError();
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

Post _post(String text) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/now',
  cid: 'bafy-now',
  rkey: 'now',
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
  author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
);

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
