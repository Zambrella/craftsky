import 'dart:typed_data';

import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_service.dart';
import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:dio/dio.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

Post _post() {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/new',
    cid: 'bafy_new',
    rkey: 'new',
    text: 'hello',
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    createdAt: DateTime.now(),
    indexedAt: DateTime.now(),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
    ),
  );
}

Post _replyTarget({String text = 'target'}) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/target',
    cid: 'bafy_target',
    rkey: 'target',
    text: text,
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    createdAt: DateTime.now(),
    indexedAt: DateTime.now(),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
    ),
    reply: PostReply(
      root: PostRef(
        uri: 'at://did:plc:root/social.craftsky.feed.post/root',
        cid: 'bafy_root',
      ),
      parent: PostRef(
        uri: 'at://did:plc:parent/social.craftsky.feed.post/parent',
        cid: 'bafy_parent',
      ),
    ),
  );
}

Future<void> _pump(
  WidgetTester tester, {
  required FakePostRepository repo,
  required RecordingMessenger messenger,
  Post? replyTarget,
  ComposerImageService? imageService,
  ComposerImagePicker? picker,
  ComposerImagePreparer? preparer,
  ComposerImageUploader? uploader,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [
        postRepositoryProvider.overrideWithValue(repo),
        if (imageService != null)
          composerImageServiceProvider.overrideWithValue(imageService),
        if (picker != null)
          composerImagePickerProvider.overrideWithValue(picker),
        if (preparer != null)
          composerImagePreparerProvider.overrideWithValue(preparer),
        if (uploader != null)
          composerImageUploaderProvider.overrideWithValue(uploader),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: PostComposerSheet(replyTarget: replyTarget)),
        ),
      ),
    ),
  );
}

void main() {
  group('PostComposerSheet', () {
    testWidgets('submit is disabled until text is entered', (tester) async {
      final messenger = RecordingMessenger();
      await _pump(tester, repo: FakePostRepository(), messenger: messenger);

      final initial = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Post'),
      );
      expect(initial.onPressed, isNull);

      await tester.enterText(find.byType(TextField), 'hello');
      await tester.pump();

      final updated = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Post'),
      );
      expect(updated.onPressed, isNotNull);
    });

    testWidgets('successful create dispatches success message', (tester) async {
      final messenger = RecordingMessenger();
      var capturedText = '';
      final repo = FakePostRepository(
        onCreate: ({required text, reply, images}) async {
          capturedText = text;
          return _post();
        },
      );

      await _pump(tester, repo: repo, messenger: messenger);
      await tester.enterText(find.byType(TextField), ' hello ');
      await tester.pump();
      await tester.tap(find.text('Post'));
      await tester.pumpAndSettle();

      expect(capturedText, 'hello');
      expect(messenger.calls.last.$2, 'Posted.');
    });

    testWidgets('successful create returns the created post', (tester) async {
      final messenger = RecordingMessenger();
      final created = _post();
      Post? result;
      final repo = FakePostRepository(
        onCreate: ({required text, reply, images}) async => created,
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Builder(
                builder: (context) => TextButton(
                  onPressed: () async {
                    result = await showPostComposerSheet(context);
                  },
                  child: const Text('Open composer'),
                ),
              ),
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open composer'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'hello');
      await tester.pump();
      await tester.tap(find.text('Post'));
      await tester.pumpAndSettle();

      expect(result?.uri, created.uri);
      expect(messenger.calls.last.$2, 'Posted.');
    });

    testWidgets('reply mode shows reply copy and forwards reply refs', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      var capturedText = '';
      PostReply? capturedReply;
      final target = _replyTarget();
      final repo = FakePostRepository(
        onListCommentBranchReplies: (did, rkey, {cursor, limit}) async =>
            const ReplyPage(loaded: true, items: []),
        onCreate: ({required text, reply, images}) async {
          capturedText = text;
          capturedReply = reply;
          return _post();
        },
      );

      await _pump(
        tester,
        repo: repo,
        messenger: messenger,
        replyTarget: target,
      );

      expect(find.text('Reply'), findsNWidgets(2));
      expect(find.text('Write your reply'), findsOneWidget);
      expect(find.widgetWithText(TextButton, 'Reply'), findsOneWidget);
      expect(find.byKey(const Key('composer-add-image')), findsNothing);
      expect(find.text('Add image'), findsNothing);

      await tester.enterText(find.byType(TextField), ' hello ');
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Reply'));
      await tester.pumpAndSettle();

      expect(capturedText, 'hello');
      expect(capturedReply, isNotNull);
      expect(capturedReply!.root.uri, target.reply!.root.uri);
      expect(capturedReply!.root.cid, target.reply!.root.cid);
      expect(capturedReply!.parent.uri, target.uri);
      expect(capturedReply!.parent.cid, target.cid);
    });

    testWidgets('reply mode shows compact target preview above input', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      final target = _replyTarget();

      await _pump(
        tester,
        repo: FakePostRepository(),
        messenger: messenger,
        replyTarget: target,
      );

      expect(find.text('@alice.craftsky.social'), findsOneWidget);
      expect(find.text('target'), findsOneWidget);
      expect(
        tester.getTopLeft(find.text('target')).dy,
        lessThan(tester.getTopLeft(find.byType(TextField)).dy),
      );
    });

    testWidgets('replying to a reply prefills target author mention', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      final target = _replyTarget();

      await _pump(
        tester,
        repo: FakePostRepository(),
        messenger: messenger,
        replyTarget: target,
      );

      final textField = tester.widget<TextField>(find.byType(TextField));
      expect(textField.controller!.text, startsWith('@alice.craftsky.social'));
    });

    testWidgets('reply target preview limits long text to three lines', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      const longText =
          'This is a very long reply target that should provide enough words '
          'to wrap across more than three lines at phone width so the compact '
          'preview can stay bounded above the composer input.';

      await _pump(
        tester,
        repo: FakePostRepository(),
        messenger: messenger,
        replyTarget: _replyTarget(text: longText),
      );

      final previewText = tester.widget<Text>(find.text(longText));

      expect(previewText.maxLines, 3);
      expect(previewText.overflow, TextOverflow.ellipsis);
    });

    testWidgets(
      'top-level composer image lifecycle gates submit and removes failed image',
      (
        tester,
      ) async {
        final messenger = RecordingMessenger();
        var capturedText = '';
        List<CreatePostImage>? capturedImages;
        final repo = FakePostRepository(
          onCreate: ({required text, reply, images}) async {
            capturedText = text;
            capturedImages = images;
            return _post();
          },
        );

        final imageService = FakeComposerImageService(
          onAddImages: (controller) {
            controller.addDraftImage(
              const DraftImageInput(
                id: 'img-1',
                fileName: 'one.jpg',
                mimeType: 'image/jpeg',
              ),
            );
            controller.markPrepared('img-1');
            controller.markUploaded(
              'img-1',
              const UploadedDraftImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 253496,
              ),
            );

            controller.addDraftImage(
              const DraftImageInput(
                id: 'img-2',
                fileName: 'two.jpg',
                mimeType: 'image/jpeg',
              ),
            );
            controller.markPrepared('img-2');
            controller.markUploadFailed('img-2', 'Upload failed');
          },
        );

        await _pump(
          tester,
          repo: repo,
          messenger: messenger,
          imageService: imageService,
        );

        await tester.tap(find.byKey(const Key('composer-add-image')));
        await tester.pump();

        expect(find.text('one.jpg'), findsOneWidget);
        expect(find.text('two.jpg'), findsOneWidget);
        expect(find.text('Upload failed'), findsOneWidget);

        await tester.enterText(find.byType(TextField).first, ' hello ');
        await tester.enterText(
          find.byKey(const Key('composer-alt-img-1')),
          'Blue shawl draped on a blocking mat',
        );
        await tester.pump();

        var submit = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'Post'),
        );
        expect(submit.onPressed, isNull, reason: 'failed image still present');

        final removeButton = find.byKey(const Key('composer-remove-img-2'));
        await tester.ensureVisible(removeButton);
        await tester.tap(removeButton);
        await tester.pump();

        submit = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'Post'),
        );
        expect(submit.onPressed, isNotNull);

        await tester.tap(find.widgetWithText(TextButton, 'Post'));
        await tester.pumpAndSettle();

        expect(capturedText, 'hello');
        expect(capturedImages, hasLength(1));
        expect(
          capturedImages!.single.alt,
          'Blue shawl draped on a blocking mat',
        );
        expect(capturedImages!.single.blob.link, 'bafkimage1');
      },
    );

    testWidgets(
      'default service flow supports reorder and aspect ratio payload',
      (
        tester,
      ) async {
        final messenger = RecordingMessenger();
        var capturedText = '';
        List<CreatePostImage>? capturedImages;
        final repo = FakePostRepository(
          onCreate: ({required text, reply, images}) async {
            capturedText = text;
            capturedImages = images;
            return _post();
          },
        );

        final picker = _FakeComposerImagePicker(
          images: [
            SelectedComposerImage(
              id: 'img-1',
              fileName: 'one.jpg',
              mimeType: 'image/jpeg',
              bytes: Uint8List.fromList([1, 2, 3]),
              metadata: const {},
            ),
            SelectedComposerImage(
              id: 'img-2',
              fileName: 'two.png',
              mimeType: 'image/png',
              bytes: Uint8List.fromList([4, 5]),
              metadata: const {},
            ),
          ],
        );

        await _pump(
          tester,
          repo: repo,
          messenger: messenger,
          picker: picker,
          preparer: const _FakeComposerImagePreparer(),
          uploader: const _FakeComposerImageUploader(),
        );

        await tester.tap(find.byKey(const Key('composer-add-image')));
        await tester.pumpAndSettle();

        expect(find.text('one.jpg'), findsOneWidget);
        expect(find.text('two.png'), findsOneWidget);

        await tester.enterText(find.byType(TextField).first, ' hello ');
        await tester.enterText(
          find.byKey(const Key('composer-alt-img-1')),
          'alt 1',
        );
        await tester.enterText(
          find.byKey(const Key('composer-alt-img-2')),
          'alt 2',
        );
        await tester.pump();

        final moveUp = find.byKey(const Key('composer-move-up-img-2'));
        await tester.ensureVisible(moveUp);
        await tester.tap(moveUp);
        await tester.pump();

        await tester.tap(find.widgetWithText(TextButton, 'Post'));
        await tester.pumpAndSettle();

        expect(capturedText, 'hello');
        expect(capturedImages, hasLength(2));
        expect(capturedImages!.first.blob.link, 'cid-img-2');
        expect(capturedImages![1].blob.link, 'cid-img-1');
        expect(capturedImages!.first.aspectRatio, isNull);
        expect(capturedImages![1].aspectRatio, isNotNull);
        expect(capturedImages![1].aspectRatio!.width, 4);
        expect(capturedImages![1].aspectRatio!.height, 5);
      },
    );

    testWidgets('image composer copy does not imply private media', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      await _pump(tester, repo: FakePostRepository(), messenger: messenger);

      expect(find.text('Add image'), findsOneWidget);
      expect(find.textContaining('private', findRichText: true), findsNothing);
      expect(
        find.textContaining('only you can see', findRichText: true),
        findsNothing,
      );
    });
  });
}

class FakeComposerImageService implements ComposerImageService {
  FakeComposerImageService({required this.onAddImages});

  final void Function(ImageDraftController controller) onAddImages;

  @override
  Future<void> addImages(ImageDraftController controller) async {
    onAddImages(controller);
  }
}

class _FakeComposerImagePicker implements ComposerImagePicker {
  const _FakeComposerImagePicker({required this.images});

  final List<SelectedComposerImage> images;

  @override
  Future<List<SelectedComposerImage>> pickImages({
    required int maxImages,
  }) async {
    return images.take(maxImages).toList();
  }
}

class _FakeComposerImagePreparer implements ComposerImagePreparer {
  const _FakeComposerImagePreparer();

  @override
  Future<PreparedComposerImage> prepare(SelectedComposerImage image) async {
    return PreparedComposerImage(
      id: image.id,
      fileName: image.fileName,
      mimeType: image.mimeType,
      originalBytes: image.bytes.length,
      preparedBytes: image.bytes,
      aspectRatio: image.id == 'img-1'
          ? const CreatePostImageAspectRatio(width: 4, height: 5)
          : null,
      strippedMetadata: const {},
    );
  }
}

class _FakeComposerImageUploader implements ComposerImageUploader {
  const _FakeComposerImageUploader();

  @override
  Future<UploadedImageBlob> upload(
    PreparedComposerImage image, {
    ProgressCallback? onSendProgress,
  }) async {
    onSendProgress?.call(
      image.preparedBytes.length,
      image.preparedBytes.length,
    );
    return UploadedImageBlob(
      blob: UploadedBlob(
        type: 'blob',
        ref: UploadedBlobRef(link: 'cid-${image.id}'),
        mimeType: image.mimeType,
        size: image.preparedBytes.length,
      ),
      cid: 'cid-${image.id}',
      mime: image.mimeType,
      size: image.preparedBytes.length,
    );
  }
}
