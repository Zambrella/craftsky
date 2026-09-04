import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
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
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-005 freezes and privately stages a scheduled external card', (
    tester,
  ) async {
    final scheduled = _SubmissionRepository();
    await tester.pumpWidget(
      _testApp(
        scheduled: scheduled,
        publicPosts: FakePostRepository(),
        messenger: RecordingMessenger(),
        images: const ComposerImagesState(images: []),
        previews: _PreviewRepository(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'Use https://source.example/pattern#section ',
    );
    await _pumpUntil(
      tester,
      () => find.text('Frozen pattern').evaluate().isNotEmpty,
    );
    await _selectLater(tester);
    await _pumpUntilEnabled(tester, 'Schedule');
    await tester.tap(find.byKey(const Key('post-composer-primary-action')));
    await _pumpUntil(tester, () => scheduled.createCalls == 1);

    final external =
        scheduled.createdPayload!['external'] as Map<String, dynamic>;
    expect(scheduled.events, ['stage:start', 'stage:done', 'create']);
    expect(scheduled.stagedBytes, [1, 2, 3]);
    expect(scheduled.stagedMimeType, 'image/png');
    expect(external, {
      'sourceUri': 'https://source.example/pattern',
      'uri': 'https://final.example/pattern#final',
      'title': 'Frozen pattern',
      'description': 'Frozen description',
      'thumbMediaId': scheduled.stagedID,
    });
  });

  testWidgets('AT-005 freezes metadata-only without staging media', (
    tester,
  ) async {
    final scheduled = _SubmissionRepository();
    await tester.pumpWidget(
      _testApp(
        scheduled: scheduled,
        publicPosts: FakePostRepository(),
        messenger: RecordingMessenger(),
        images: const ComposerImagesState(images: []),
        previews: _PreviewRepository(thumbnail: false),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'Use https://source.example/pattern ',
    );
    await _pumpUntil(
      tester,
      () => find.text('Frozen pattern').evaluate().isNotEmpty,
    );
    await _selectLater(tester);
    await _pumpUntilEnabled(tester, 'Schedule');
    await tester.tap(find.byKey(const Key('post-composer-primary-action')));
    await _pumpUntil(tester, () => scheduled.createCalls == 1);

    expect(scheduled.stageCalls, 0);
    expect(
      scheduled.createdPayload!['external'],
      isNot(contains('thumbMediaId')),
    );
  });

  testWidgets(
    'IR-019 trimmed terminal frozen URL survives create reopen save',
    (tester) async {
      final scheduled = _SubmissionRepository();
      final previews = _PreviewRepository(thumbnail: false);
      await tester.pumpWidget(
        _testApp(
          scheduled: scheduled,
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: previews,
          composerId: 'trimmed-create',
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'Use https://source.example/pattern ',
      );
      await _pumpUntil(
        tester,
        () => find.text('Frozen pattern').evaluate().isNotEmpty,
      );
      await _selectLater(tester);
      await _pumpUntilEnabled(tester, 'Schedule');
      await tester.tap(find.byKey(const Key('post-composer-primary-action')));
      await _pumpUntil(tester, () => scheduled.createCalls == 1);

      final createdPayload = scheduled.createdPayload!;
      expect(createdPayload['text'], 'Use https://source.example/pattern');
      expect(createdPayload['external'], isNotNull);
      final detail = ScheduledPostDetail(
        id: 'trimmed-schedule',
        operationId: 'trimmed-create',
        status: ScheduledPostStatus.scheduled,
        scheduledAt: ScheduledInstant(scheduled.lastScheduledAt!),
        payload: createdPayload,
      );

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
      await tester.pumpWidget(
        _testApp(
          scheduled: scheduled,
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: previews,
          composerId: 'trimmed-reopen',
          scheduledPost: detail,
        ),
      );
      await tester.pumpAndSettle();
      await _pumpUntilEnabled(tester, 'Schedule');
      await tester.tap(find.byKey(const Key('post-composer-primary-action')));
      await _pumpUntil(tester, () => scheduled.updateCalls == 1);

      expect(scheduled.updatedPayload?['external'], createdPayload['external']);
      expect(previews.fetchCalls, 1);
    },
  );

  testWidgets('AT-002 real composer dismissal snackbar restores with Undo', (
    tester,
  ) async {
    await tester.pumpWidget(
      _testApp(
        scheduled: _SubmissionRepository(),
        publicPosts: FakePostRepository(),
        messenger: RecordingMessenger(),
        images: const ComposerImagesState(images: []),
        previews: _PreviewRepository(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'Use https://source.example/pattern ',
    );
    await _pumpUntil(
      tester,
      () => find.text('Frozen pattern').evaluate().isNotEmpty,
    );

    await tester.tap(find.byTooltip('Dismiss link previews'));
    await tester.pump();

    expect(find.text('Frozen pattern'), findsNothing);
    expect(find.text('Link previews hidden'), findsOneWidget);
    expect(find.text('Undo'), findsOneWidget);
    expect(find.byType(CraftskySnackBarContent), findsOneWidget);

    tester
        .widget<CraftskySnackBarContent>(
          find.byType(CraftskySnackBarContent),
        )
        .action!
        .onPressed();
    await tester.pumpAndSettle();

    expect(find.text('Frozen pattern'), findsOneWidget);
  });

  testWidgets(
    'IR-021 earlier snackbar expiry cannot close a later Undo window',
    (tester) async {
      await tester.pumpWidget(
        _testApp(
          scheduled: _SubmissionRepository(),
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: _PreviewRepository(),
          undoDuration: const Duration(seconds: 1),
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'Use https://source.example/pattern ',
      );
      await _pumpUntil(
        tester,
        () => find.text('Frozen pattern').evaluate().isNotEmpty,
      );

      await tester.tap(find.byTooltip('Dismiss link previews'));
      await tester.pump();
      tester
          .widget<CraftskySnackBarContent>(
            find.byType(CraftskySnackBarContent),
          )
          .action!
          .onPressed();
      await tester.pump();
      await tester.tap(find.byTooltip('Dismiss link previews'));
      await tester.pump();

      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
      expect(find.text('Undo'), findsOneWidget);
      tester
          .widget<CraftskySnackBarContent>(
            find.byType(CraftskySnackBarContent),
          )
          .action!
          .onPressed();
      await tester.pump();

      expect(find.text('Frozen pattern'), findsOneWidget);
    },
  );

  testWidgets('AT-001 real composer selection follows URL across edits', (
    tester,
  ) async {
    final previews = _MappedPreviewRepository();
    await tester.pumpWidget(
      _testApp(
        scheduled: _SubmissionRepository(),
        publicPosts: FakePostRepository(),
        messenger: RecordingMessenger(),
        images: const ComposerImagesState(images: []),
        previews: previews,
      ),
    );
    await tester.pumpAndSettle();
    final field = find.byType(TextField).first;
    await tester.enterText(
      field,
      'https://one.example/path https://two.example/path ',
    );
    await _pumpUntil(
      tester,
      () => find.text('Link preview 1 of 2').evaluate().isNotEmpty,
    );

    expect(previews.urls, [
      'https://one.example/path',
      'https://two.example/path',
    ]);
    expect(find.text('One pattern'), findsOneWidget);

    await tester.tap(find.byTooltip('Next link preview'));
    await tester.pump();
    expect(find.text('Two pattern'), findsOneWidget);

    await tester.enterText(
      field,
      'https://two.example/path https://one.example/path ',
    );
    await tester.pump();
    expect(find.text('Two pattern'), findsOneWidget);
    expect(find.text('Link preview 1 of 2'), findsOneWidget);
    expect(previews.urls, hasLength(2));

    await tester.enterText(field, 'https://one.example/path ');
    await tester.pump();
    expect(find.text('One pattern'), findsOneWidget);
    expect(find.text('Two pattern'), findsNothing);
  });

  testWidgets(
    'IR-016 AT-001 real composer bounds five sequential links '
    'and omits failure',
    (tester) async {
      final previews = _SequentialAcceptancePreviewRepository();
      final messenger = RecordingMessenger();
      await tester.pumpWidget(
        _testApp(
          scheduled: _SubmissionRepository(),
          publicPosts: FakePostRepository(),
          messenger: messenger,
          images: const ComposerImagesState(images: []),
          previews: previews,
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'https://one.example/a https://two.example/b '
        'https://three.example/c https://four.example/d '
        'https://five.example/e ',
      );
      await _pumpUntil(tester, () => previews.urls.length == 4);
      await tester.pumpAndSettle();

      expect(previews.urls, [
        'https://one.example/a',
        'https://two.example/b',
        'https://three.example/c',
        'https://four.example/d',
      ]);
      expect(previews.maxActive, 1);
      expect(find.text('One pattern'), findsOneWidget);
      expect(find.text('Two pattern'), findsNothing);
      expect(find.text('Link preview 1 of 3'), findsOneWidget);
      expect(messenger.calls.where((call) => call.$1 == 'error'), isEmpty);

      await tester.tap(find.byTooltip('Next link preview'));
      await tester.pump();
      expect(find.text('Three pattern'), findsOneWidget);
      await tester.tap(find.byTooltip('Next link preview'));
      await tester.pump();
      expect(find.text('Four pattern'), findsOneWidget);
    },
  );

  testWidgets(
    'IR-016 AT-002 real image actions cancel, hide, and restore previews',
    (tester) async {
      final previews = _SuppressionAcceptancePreviewRepository();
      final images = _ToggleComposerImages();
      await tester.pumpWidget(
        _testApp(
          scheduled: _SubmissionRepository(),
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          previews: previews,
          imageNotifier: images,
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'https://one.example/a https://two.example/b ',
      );
      await _pumpUntil(tester, () => previews.tokens.length == 2);
      expect(find.text('One pattern'), findsOneWidget);
      final pendingToken = previews.tokens.last;

      await tester.ensureVisible(find.byKey(const Key('composer-add-image')));
      await tester.tap(find.byKey(const Key('composer-add-image')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('composer-choose-photos')));
      await tester.pumpAndSettle();

      expect(pendingToken.isCancelled, isTrue);
      expect(find.text('One pattern'), findsNothing);
      expect(
        find.byKey(const Key('composer-remove-test-image')),
        findsOneWidget,
      );

      await tester.ensureVisible(
        find.byKey(const Key('composer-remove-test-image')),
      );
      await tester.tap(find.byKey(const Key('composer-remove-test-image')));
      await _pumpUntil(tester, () => previews.tokens.length == 3);
      await tester.pump();
      expect(find.text('One pattern'), findsOneWidget);
      previews.completeLatest();
      await tester.pumpAndSettle();
      expect(find.text('Link preview 1 of 2'), findsOneWidget);
    },
  );

  testWidgets(
    'IR-016 AT-002 Undo expiry holds only for the current composer session',
    (tester) async {
      final previews = _PreviewRepository();
      await tester.pumpWidget(
        _testApp(
          scheduled: _SubmissionRepository(),
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: previews,
          composerId: 'expiring-session',
          undoDuration: const Duration(milliseconds: 100),
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'Use https://source.example/pattern ',
      );
      await _pumpUntil(
        tester,
        () => find.text('Frozen pattern').evaluate().isNotEmpty,
      );
      await tester.tap(find.byTooltip('Dismiss link previews'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump(const Duration(seconds: 1));
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.text('Undo'), findsNothing);
      expect(find.text('Frozen pattern'), findsNothing);
      await tester.enterText(
        find.byType(TextField).first,
        'Use https://source.example/pattern and https://two.example/a ',
      );
      await tester.pump(const Duration(milliseconds: 100));
      expect(previews.fetchCalls, 1);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
      await tester.pumpWidget(
        _testApp(
          scheduled: _SubmissionRepository(),
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: previews,
          composerId: 'new-session',
          undoDuration: const Duration(milliseconds: 100),
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'Use https://source.example/pattern ',
      );
      await _pumpUntil(tester, () => previews.fetchCalls == 2);
      expect(find.text('Frozen pattern'), findsOneWidget);
    },
  );

  testWidgets('AT-002 quote composer suppresses preview requests and embeds', (
    tester,
  ) async {
    final previews = _PreviewRepository();
    final publicPosts = FakePostRepository();
    await tester.pumpWidget(
      _testApp(
        scheduled: _SubmissionRepository(),
        publicPosts: publicPosts,
        messenger: RecordingMessenger(),
        images: const ComposerImagesState(images: []),
        previews: previews,
        quoteTarget: _post('quoted post'),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'Quote https://source.example/pattern ',
    );
    await tester.pump(const Duration(milliseconds: 100));

    expect(previews.fetchCalls, 0);
    expect(find.text('Frozen pattern'), findsNothing);

    await _pumpUntilEnabled(tester, 'Post');
    await tester.tap(find.byKey(const Key('post-composer-primary-action')));
    await tester.pumpAndSettle();

    expect(publicPosts.lastCreateExternal, isNull);
  });

  testWidgets('AT-003 thumbnail upload failure retains real composer state', (
    tester,
  ) async {
    final publicPosts = FakePostRepository();
    final messenger = RecordingMessenger();
    final upload = _FailingPostApiClient();
    await tester.pumpWidget(
      _testApp(
        scheduled: _SubmissionRepository(),
        publicPosts: publicPosts,
        messenger: messenger,
        images: const ComposerImagesState(images: []),
        previews: _PreviewRepository(),
        uploadClient: upload,
      ),
    );
    await tester.pumpAndSettle();
    const text = 'Keep https://source.example/pattern ';
    await tester.enterText(find.byType(TextField).first, text);
    await _pumpUntil(
      tester,
      () => find.text('Frozen pattern').evaluate().isNotEmpty,
    );
    await _pumpUntilEnabled(tester, 'Post');

    await tester.tap(find.byKey(const Key('post-composer-primary-action')));
    await tester.pumpAndSettle();

    expect(upload.uploadCalls, 1);
    expect(publicPosts.lastCreateExternal, isNull);
    expect(find.text(text), findsOneWidget);
    expect(find.text('Frozen pattern'), findsOneWidget);
    expect(
      messenger.calls,
      contains(('error', "Couldn't post.", null)),
    );
  });

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
        .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Schedule'))
        .onPressed!();
    await tester.pump();

    await _pumpUntil(tester, () => scheduled.stageCalls == 1);
    await tester.pump();
    expect(find.text('Scheduling your post…'), findsOneWidget);
    expect(find.byKey(const Key('submission-modal-barrier')), findsOneWidget);
    expect(scheduled.createCalls, 0);
    expect(publicCreateCalls, 0);
    expect(scheduled.stagedBytes, isNotEmpty);
    expect(scheduled.stagedMimeType, 'image/png');

    stageGate.complete();
    await _pumpUntil(tester, () => scheduled.createCalls == 1);
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
    final scheduled = _SubmissionRepository(
      failFirstCreate: true,
      createError: StateError('scheduled-save-dependency-canary'),
    );
    final messenger = RecordingMessenger();
    final reporter = _RecordingErrorReporter();
    await tester.pumpWidget(
      _testApp(
        scheduled: scheduled,
        publicPosts: FakePostRepository(),
        messenger: messenger,
        reporter: reporter,
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'scheduled-save-post-text-canary',
    );
    await _selectLater(tester);
    await _pumpUntilEnabled(tester, 'Schedule');

    tester
        .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Schedule'))
        .onPressed!();
    await tester.pump();
    await _pumpUntil(tester, () => scheduled.createCalls == 1);
    await tester.pump();

    expect(find.text('scheduled-save-post-text-canary'), findsOneWidget);
    expect(scheduled.createCalls, 1);
    expect(reporter.captured, isEmpty);
    expect(reporter.messages, isEmpty);
    expect(reporter.breadcrumbs, isEmpty);
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
        .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Schedule'))
        .onPressed!();
    await tester.pump();
    await _pumpUntil(tester, () => scheduled.createCalls == 2);
    await tester.pump();

    expect(scheduled.createCalls, 2);
    expect(scheduled.operationIDs.toSet(), {'submission'});
  });

  testWidgets(
    'IR-020 new schedule retains identical thumbnail ID '
    'and rotates changed content',
    (tester) async {
      final firstBytes = _pngBytes(width: 3, height: 1);
      final secondBytes = _pngBytes(width: 4, height: 1);
      final scheduled = _SubmissionRepository(failCreateCount: 2);
      final previews = _ChangingThumbnailPreviewRepository(
        firstBytes: firstBytes,
        secondBytes: secondBytes,
      );
      await tester.pumpWidget(
        _testApp(
          scheduled: scheduled,
          publicPosts: FakePostRepository(),
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: previews,
          composerId: 'changed-thumbnail-retry',
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'Use https://first.example/pattern ',
      );
      await _pumpUntil(
        tester,
        () => find.text('First pattern').evaluate().isNotEmpty,
      );
      await _selectLater(tester);
      await _pumpUntilEnabled(tester, 'Schedule');

      await tester.tap(find.byKey(const Key('post-composer-primary-action')));
      await _pumpUntil(tester, () => scheduled.createCalls == 1);
      await _pumpUntilEnabled(tester, 'Schedule');
      await tester.tap(find.byKey(const Key('post-composer-primary-action')));
      await _pumpUntil(tester, () => scheduled.createCalls == 2);

      expect(scheduled.stagedIDs, hasLength(2));
      expect(scheduled.stagedIDs[1], scheduled.stagedIDs[0]);
      expect(scheduled.stagedByteHistory, [firstBytes, firstBytes]);

      await tester.enterText(
        find.byType(TextField).first,
        'Use https://second.example/pattern ',
      );
      await _pumpUntil(
        tester,
        () => find.text('Second pattern').evaluate().isNotEmpty,
      );
      await _pumpUntilEnabled(tester, 'Schedule');
      await tester.tap(find.byKey(const Key('post-composer-primary-action')));
      await _pumpUntil(tester, () => scheduled.createCalls == 3);

      expect(scheduled.stagedIDs, hasLength(3));
      expect(scheduled.stagedIDs[2], isNot(scheduled.stagedIDs[1]));
      expect(scheduled.stagedByteHistory[2], secondBytes);
      expect(
        (scheduled.createdPayload?['external'] as Map)['thumbMediaId'],
        scheduled.stagedIDs[2],
      );
    },
  );

  testWidgets(
    'IT-013 submit invalidates pending preview queue after publish failure',
    (tester) async {
      final previews = _PendingPreviewRepository();
      final publicPosts = FakePostRepository(
        onCreateWithFacets:
            ({required text, reply, project, images, facets}) async =>
                throw StateError('publication failed'),
      );
      await tester.pumpWidget(
        _testApp(
          scheduled: _SubmissionRepository(),
          publicPosts: publicPosts,
          messenger: RecordingMessenger(),
          images: const ComposerImagesState(images: []),
          previews: previews,
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField).first,
        'Try https://one.example/path then https://two.example/path ',
      );
      await _pumpUntil(tester, () => previews.urls.length == 1);
      await _pumpUntilEnabled(tester, 'Post');

      tester
          .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Post'))
          .onPressed!();
      await tester.pump();
      await tester.pump();

      expect(previews.tokens.single.isCancelled, isTrue);
      previews.completers.single.complete(
        LinkPreview(
          url: Uri.parse('https://one.example/final'),
          title: 'Late pattern',
          description: 'Late description',
        ),
      );
      await tester.pumpAndSettle();

      expect(previews.urls, ['https://one.example/path']);
      expect(find.text('Late pattern'), findsNothing);
      expect(find.textContaining('Try https://one.example'), findsOneWidget);
    },
  );
}

Widget _testApp({
  required _SubmissionRepository scheduled,
  required FakePostRepository publicPosts,
  required RecordingMessenger messenger,
  ComposerImagesState? images,
  ComposerImages? imageNotifier,
  LinkPreviewRepository? previews,
  Post? quoteTarget,
  PostApiClient? uploadClient,
  String composerId = 'submission',
  Duration? undoDuration,
  ErrorReporter? reporter,
  ScheduledPostDetail? scheduledPost,
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
      if (imageNotifier != null)
        composerImagesProvider(
          composerId,
        ).overrideWith(() => imageNotifier)
      else
        composerImagesProvider(
          composerId,
        ).overrideWithValue(images ?? _readyImage),
      if (previews != null)
        linkPreviewRepositoryProvider.overrideWithValue(previews),
      if (uploadClient != null)
        postApiClientProvider.overrideWith((ref) => uploadClient),
      if (undoDuration != null)
        linkPreviewUndoDurationProvider.overrideWithValue(undoDuration),
      if (reporter != null) errorReporterProvider.overrideWithValue(reporter),
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
        home: PostComposerSheet(
          composerId: composerId,
          quoteTarget: quoteTarget,
          scheduledPost: scheduledPost,
        ),
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
      phase: ImageReady(
        bytes: _pngBytes(width: 3, height: 2),
        mimeType: 'image/png',
        width: 3,
        height: 2,
        sha256: 'ready-image-digest',
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
    final ready = image.phase as ImageReady;
    await stageMedia(
      id: image.id,
      bytes: ready.bytes,
      mimeType: ready.mimeType,
    );
    media.add({
      'id': image.id,
      'alt': image.altText,
      'width': ready.width,
      'height': ready.height,
    });
    onStaged?.call(media.length);
  }
  return media;
}

Future<void> _selectLater(WidgetTester tester) async {
  await tester.ensureVisible(find.text('When'));
  await tester.pumpAndSettle();
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
    final finder = find.widgetWithText(ChunkyButton, label);
    return finder.evaluate().isNotEmpty &&
        tester.widget<ChunkyButton>(finder).onPressed != null;
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

final class _RecordingErrorReporter implements ErrorReporter {
  final captured = <Object>[];
  final messages = <String>[];
  final breadcrumbs = <SafeBreadcrumb>[];

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) => breadcrumbs.add(breadcrumb);

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    captured.add(error);
    return 'event';
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    messages.add(message);
  }
}

Uint8List _pngBytes({required int width, required int height}) =>
    Uint8List.fromList(img.encodePng(img.Image(width: width, height: height)));

final class _SubmissionRepository implements ScheduledPostRepository {
  _SubmissionRepository({
    this.stageGate,
    bool failFirstCreate = false,
    this.createError,
    int? failCreateCount,
  }) : _remainingCreateFailures = failCreateCount ?? (failFirstCreate ? 1 : 0);

  final Completer<void>? stageGate;
  final Error? createError;
  int _remainingCreateFailures;
  int stageCalls = 0;
  int listCalls = 0;
  int createCalls = 0;
  int updateCalls = 0;
  List<int> stagedBytes = const [];
  String? stagedMimeType;
  String? stagedID;
  final stagedIDs = <String>[];
  final stagedByteHistory = <List<int>>[];
  Map<String, dynamic>? createdPayload;
  Map<String, dynamic>? updatedPayload;
  DateTime? lastScheduledAt;
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
    CancelToken? cancelToken,
  }) async {
    stageCalls += 1;
    stagedBytes = List.unmodifiable(bytes);
    stagedMimeType = mimeType;
    stagedID = id;
    stagedIDs.add(id);
    stagedByteHistory.add(List.unmodifiable(bytes));
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
    lastScheduledAt = scheduledAt;
    operationIDs.add(operationId);
    createdPayload = payload;
    events.add('create');
    if (_remainingCreateFailures > 0) {
      _remainingCreateFailures -= 1;
      throw createError ?? StateError('create failed');
    }
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
  }) async {
    updateCalls += 1;
    updatedPayload = payload;
    return ScheduledPostDetail(
      id: id,
      operationId: 'trimmed-create',
      status: ScheduledPostStatus.scheduled,
      scheduledAt: ScheduledInstant(scheduledAt),
      payload: payload,
    );
  }
}

final class _PreviewRepository implements LinkPreviewRepository {
  _PreviewRepository({this.thumbnail = true});

  final bool thumbnail;
  int fetchCalls = 0;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    fetchCalls += 1;
    return LinkPreview(
      url: Uri.parse('https://final.example/pattern#final'),
      title: 'Frozen pattern',
      description: 'Frozen description',
      thumbnail: thumbnail
          ? LinkPreviewThumbnail(
              bytes: Uint8List.fromList([1, 2, 3]),
              mimeType: 'image/png',
              width: 20,
              height: 10,
            )
          : null,
    );
  }
}

final class _ChangingThumbnailPreviewRepository
    implements LinkPreviewRepository {
  _ChangingThumbnailPreviewRepository({
    required this.firstBytes,
    required this.secondBytes,
  });

  final Uint8List firstBytes;
  final Uint8List secondBytes;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    final first = url.host == 'first.example';
    final name = first ? 'First' : 'Second';
    final bytes = first ? firstBytes : secondBytes;
    return LinkPreview(
      url: Uri.parse('https://final.example/${name.toLowerCase()}'),
      title: '$name pattern',
      description: '$name description',
      thumbnail: LinkPreviewThumbnail(
        bytes: bytes,
        mimeType: 'image/png',
        width: first ? 3 : 4,
        height: 1,
      ),
    );
  }
}

final class _PendingPreviewRepository implements LinkPreviewRepository {
  final urls = <String>[];
  final tokens = <CancelToken>[];
  final completers = <Completer<LinkPreview>>[];

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) {
    urls.add(url.toString());
    tokens.add(cancelToken);
    final completer = Completer<LinkPreview>();
    completers.add(completer);
    return completer.future;
  }
}

final class _MappedPreviewRepository implements LinkPreviewRepository {
  final urls = <String>[];

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    urls.add(url.toString());
    final name = url.host.startsWith('one') ? 'One' : 'Two';
    return LinkPreview(
      url: url,
      title: '$name pattern',
      description: '$name description',
    );
  }
}

final class _SequentialAcceptancePreviewRepository
    implements LinkPreviewRepository {
  final List<String> urls = [];
  int active = 0;
  int maxActive = 0;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    urls.add(url.toString());
    active += 1;
    if (active > maxActive) maxActive = active;
    await Future<void>.delayed(Duration.zero);
    try {
      if (url.host == 'two.example') throw StateError('rate limited');
      final name = url.host.split('.').first;
      return LinkPreview(
        url: Uri.parse('https://final.example/$name'),
        title: '${name[0].toUpperCase()}${name.substring(1)} pattern',
        description: '$name description',
      );
    } finally {
      active -= 1;
    }
  }
}

final class _SuppressionAcceptancePreviewRepository
    implements LinkPreviewRepository {
  final urls = <Uri>[];
  final tokens = <CancelToken>[];
  final pending = <Completer<LinkPreview>>[];

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) {
    urls.add(url);
    tokens.add(cancelToken);
    if (url.host == 'one.example') {
      return Future.value(
        LinkPreview(
          url: url,
          title: 'One pattern',
          description: 'One description',
        ),
      );
    }
    final completer = Completer<LinkPreview>();
    pending.add(completer);
    return completer.future;
  }

  void completeLatest() {
    pending.last.complete(
      LinkPreview(
        url: Uri.parse('https://two.example/final'),
        title: 'Two pattern',
        description: 'Two description',
      ),
    );
  }
}

final class _ToggleComposerImages extends ComposerImages {
  @override
  ComposerImagesState build(String composerId) =>
      const ComposerImagesState(images: []);

  @override
  Future<void> addImages() async {
    final bytes = _pngBytes(width: 2, height: 1);
    state = ComposerImagesState(
      images: [
        ComposerImageDraft(
          id: 'test-image',
          fileName: 'test.png',
          mimeType: 'image/png',
          altText: 'Test image',
          previewBytes: bytes,
          phase: ImageReady(
            bytes: bytes,
            mimeType: 'image/png',
            width: 2,
            height: 1,
            sha256: 'test-image-digest',
          ),
        ),
      ],
    );
  }

  @override
  void remove(String imageId) {
    state = const ComposerImagesState(images: []);
  }
}

final class _FailingPostApiClient extends PostApiClient {
  _FailingPostApiClient() : super(Dio());

  int uploadCalls = 0;

  @override
  Future<Never> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    uploadCalls += 1;
    throw StateError('thumbnail upload failed');
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
