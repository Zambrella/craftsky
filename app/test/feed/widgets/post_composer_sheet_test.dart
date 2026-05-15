import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
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
    author: const PostAuthor(
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
    author: const PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
    ),
    reply: const PostReply(
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
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
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

      final initial = tester.widget<TextButton>(find.byType(TextButton));
      expect(initial.onPressed, isNull);

      await tester.enterText(find.byType(TextField), 'hello');
      await tester.pump();

      final updated = tester.widget<TextButton>(find.byType(TextButton));
      expect(updated.onPressed, isNotNull);
    });

    testWidgets('successful create dispatches success message', (tester) async {
      final messenger = RecordingMessenger();
      var capturedText = '';
      final repo = FakePostRepository(
        onCreate: ({required text, reply}) async {
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
        onCreate: ({required text, reply}) async {
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
  });
}
