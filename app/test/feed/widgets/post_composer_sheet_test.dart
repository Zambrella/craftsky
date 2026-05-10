import 'package:craftsky_app/feed/models/post.dart';
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

Future<void> _pump(
  WidgetTester tester, {
  required FakePostRepository repo,
  required RecordingMessenger messenger,
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
          home: const Scaffold(body: PostComposerSheet()),
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
        onCreate: ({required text}) async {
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
  });
}
