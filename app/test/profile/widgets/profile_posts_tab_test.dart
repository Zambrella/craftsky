import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_posts_tab.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

Post _post(String rkey) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: 'post $rkey',
    tags: const [],
    createdAt: DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: const PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
    ),
  );
}

Future<void> _pump(
  WidgetTester tester, {
  required FakePostRepository repo,
  required bool isOwnProfile,
  RecordingMessenger? messenger,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: CustomScrollView(
              slivers: [
                ProfilePostsTab(
                  handle: 'alice.craftsky.social',
                  isOwnProfile: isOwnProfile,
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  group('ProfilePostsTab', () {
    testWidgets('renders posts from userPostsProvider', (tester) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(
          items: [_post('a'), _post('b')],
        ),
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();

      expect(find.text('post a'), findsOneWidget);
      expect(find.text('post b'), findsOneWidget);
      expect(find.text('New post'), findsNothing);
    });

    testWidgets('shows composer entry point on own profile', (tester) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
      );

      await _pump(tester, repo: repo, isOwnProfile: true);
      await tester.pumpAndSettle();

      expect(find.text('New post'), findsOneWidget);
      expect(find.text('No posts yet.'), findsOneWidget);
    });

    testWidgets('load more appends the next page', (tester) async {
      var calls = 0;
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async {
          calls++;
          if (calls == 1) return PostPage(items: [_post('a')], cursor: 'c1');
          expect(cursor, 'c1');
          return PostPage(items: [_post('b')]);
        },
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Load more posts'));
      await tester.pumpAndSettle();

      expect(find.text('post a'), findsOneWidget);
      expect(find.text('post b'), findsOneWidget);
    });

    testWidgets('delete confirmation removes a post', (tester) async {
      final messenger = RecordingMessenger();
      final deleted = <String>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(
          items: [_post('a'), _post('b')],
        ),
        onDelete: (_, rkey) async => deleted.add(rkey),
      );

      await _pump(
        tester,
        repo: repo,
        isOwnProfile: true,
        messenger: messenger,
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(Icons.more_horiz).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete post').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(deleted, ['a']);
      expect(find.text('post a'), findsNothing);
      expect(find.text('post b'), findsOneWidget);
      expect(messenger.calls.last.$2, 'Post deleted.');
    });
  });
}
