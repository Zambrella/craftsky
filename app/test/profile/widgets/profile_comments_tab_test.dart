import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_comments_tab.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

Post _comment(String rkey) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: 'comment $rkey',
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    createdAt: DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
    ),
  );
}

Future<void> _pump(WidgetTester tester, {required FakePostRepository repo}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(
            body: CustomScrollView(
              slivers: [
                ProfileCommentsTab(
                  handle: 'alice.craftsky.social',
                  isOwnProfile: false,
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
  group('ProfileCommentsTab', () {
    testWidgets('scrolling near the end appends the next page', (tester) async {
      final calls = <({String? cursor, int? limit})>[];
      final repo = FakePostRepository(
        onListCommentsByAuthor: (_, {cursor, limit}) async {
          calls.add((cursor: cursor, limit: limit));
          if (calls.length == 1) {
            return PostPage(
              items: [for (var i = 0; i < 10; i++) _comment('a$i')],
              cursor: 'c1',
            );
          }
          expect(cursor, 'c1');
          return PostPage(items: [_comment('b')]);
        },
      );

      await _pump(tester, repo: repo);
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('comment a9'),
        500,
        scrollable: find.byType(Scrollable),
      );
      await tester.pumpAndSettle();

      expect(calls, [
        (cursor: null, limit: 10),
        (cursor: 'c1', limit: 10),
      ]);
      expect(find.text('comment a9'), findsOneWidget);
      expect(find.text('comment b'), findsOneWidget);
      expect(find.text('Load more comments'), findsNothing);
    });
  });
}
