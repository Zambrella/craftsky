import 'dart:async';
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
import 'package:image/image.dart' as img;

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
        final jpgBytes = Uint8List.fromList(
          img.encodeJpg(img.Image(width: 1, height: 1)),
        );
        final pngBytes = Uint8List.fromList(
          img.encodePng(img.Image(width: 1, height: 1)),
        );
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
              bytes: jpgBytes,
              metadata: const {},
            ),
            SelectedComposerImage(
              id: 'img-2',
              fileName: 'two.png',
              mimeType: 'image/png',
              bytes: pngBytes,
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

    testWidgets(
      'default service flow shows local preview and upload progress',
      (
        tester,
      ) async {
        final messenger = RecordingMessenger();
        final uploader = _ControllableComposerImageUploader();
        final previewBytes = Uint8List.fromList(
          img.encodeJpg(img.Image(width: 1, height: 1)),
        );

        await _pump(
          tester,
          repo: FakePostRepository(),
          messenger: messenger,
          picker: _FakeComposerImagePicker(
            images: [
              SelectedComposerImage(
                id: 'img-1',
                fileName: 'one.jpg',
                mimeType: 'image/jpeg',
                bytes: previewBytes,
                metadata: const {},
              ),
            ],
          ),
          preparer: const _FakeComposerImagePreparer(),
          uploader: uploader,
        );

        await tester.tap(find.byKey(const Key('composer-add-image')));
        await tester.pump();

        final preview = find.byKey(const Key('composer-preview-img-1'));
        final progress = find.byKey(
          const Key('composer-upload-progress-img-1'),
        );

        expect(preview, findsOneWidget);
        expect(progress, findsOneWidget);

        final progressWidget = tester.widget<LinearProgressIndicator>(progress);
        expect(progressWidget.value, greaterThan(0));

        uploader.completeSuccess();
        await tester.pumpAndSettle();

        expect(find.text('Uploaded'), findsOneWidget);
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

    testWidgets('composer shows feedback when image cap is reached', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      var addCalls = 0;

      final imageService = FakeComposerImageService(
        onAddImages: (controller) {
          addCalls += 1;
          if (addCalls != 1) return;

          for (var i = 1; i <= 4; i++) {
            final id = 'img-$i';
            controller.addDraftImage(
              DraftImageInput(
                id: id,
                fileName: 'image-$i.jpg',
                mimeType: 'image/jpeg',
              ),
            );
            controller.markPrepared(id);
            controller.markUploaded(
              id,
              UploadedDraftImage(
                cid: 'bafkimage$i',
                mime: 'image/jpeg',
                size: 2048,
              ),
            );
          }
        },
      );

      await _pump(
        tester,
        repo: FakePostRepository(),
        messenger: messenger,
        imageService: imageService,
      );

      await tester.tap(find.byKey(const Key('composer-add-image')));
      await tester.pump();

      await tester.tap(find.byKey(const Key('composer-add-image')));
      await tester.pump();

      expect(addCalls, 1);
      expect(
        messenger.calls.any(
          (call) =>
              call.$1 == 'error' &&
              call.$2.contains('You can add up to 4 images'),
        ),
        isTrue,
      );
    });

    testWidgets(
      'composer rejects excess partial selection without exceeding draft max',
      (tester) async {
        final messenger = RecordingMessenger();
        final bytes = Uint8List.fromList(
          img.encodeJpg(img.Image(width: 1, height: 1)),
        );
        final picker = _QueueComposerImagePicker(
          selections: [
            [
              SelectedComposerImage(
                id: 'base-1',
                fileName: 'base-1.jpg',
                mimeType: 'image/jpeg',
                bytes: bytes,
                metadata: const {},
              ),
              SelectedComposerImage(
                id: 'base-2',
                fileName: 'base-2.jpg',
                mimeType: 'image/jpeg',
                bytes: bytes,
                metadata: const {},
              ),
              SelectedComposerImage(
                id: 'base-3',
                fileName: 'base-3.jpg',
                mimeType: 'image/jpeg',
                bytes: bytes,
                metadata: const {},
              ),
            ],
            [
              SelectedComposerImage(
                id: 'plus-1',
                fileName: 'plus-1.jpg',
                mimeType: 'image/jpeg',
                bytes: bytes,
                metadata: const {},
              ),
              SelectedComposerImage(
                id: 'plus-2',
                fileName: 'plus-2.jpg',
                mimeType: 'image/jpeg',
                bytes: bytes,
                metadata: const {},
              ),
              SelectedComposerImage(
                id: 'plus-3',
                fileName: 'plus-3.jpg',
                mimeType: 'image/jpeg',
                bytes: bytes,
                metadata: const {},
              ),
            ],
          ],
        );

        await _pump(
          tester,
          repo: FakePostRepository(),
          messenger: messenger,
          picker: picker,
          preparer: const _FakeComposerImagePreparer(),
          uploader: const _FakeComposerImageUploader(),
        );

        await tester.tap(find.byKey(const Key('composer-add-image')));
        await tester.pumpAndSettle();

        expect(find.text('base-1.jpg'), findsOneWidget);
        expect(find.text('base-2.jpg'), findsOneWidget);
        expect(find.text('base-3.jpg'), findsOneWidget);

        await tester.tap(find.byKey(const Key('composer-add-image')));
        await tester.pumpAndSettle();

        expect(find.text('plus-1.jpg'), findsOneWidget);
        expect(find.text('plus-2.jpg'), findsNothing);
        expect(find.text('plus-3.jpg'), findsNothing);
        expect(find.byIcon(Icons.close), findsNWidgets(4));
        expect(
          messenger.calls.any(
            (call) =>
                call.$1 == 'error' &&
                call.$2.contains('You can add up to 4 images'),
          ),
          isTrue,
        );
      },
    );
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

class _QueueComposerImagePicker implements ComposerImagePicker {
  _QueueComposerImagePicker({required this.selections});

  final List<List<SelectedComposerImage>> selections;
  var _callCount = 0;

  @override
  Future<List<SelectedComposerImage>> pickImages({
    required int maxImages,
  }) async {
    if (_callCount >= selections.length) {
      return const [];
    }
    final next = selections[_callCount];
    _callCount += 1;
    return next;
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

class _ControllableComposerImageUploader implements ComposerImageUploader {
  final Completer<void> _completer = Completer<void>();

  void completeSuccess() {
    if (!_completer.isCompleted) {
      _completer.complete();
    }
  }

  @override
  Future<UploadedImageBlob> upload(
    PreparedComposerImage image, {
    ProgressCallback? onSendProgress,
  }) async {
    onSendProgress?.call(1, 2);
    await _completer.future;
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
