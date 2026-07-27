import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_projects_tab.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

Post _projectPost(String rkey) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: 'project post $rkey',
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
    project: Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#knitting',
        title: 'Project $rkey',
      ),
    ),
  );
}

Future<void> _pump(
  WidgetTester tester, {
  required FakePostRepository repo,
  bool isOwnProfile = false,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: CustomScrollView(
              slivers: [
                ProfileProjectsTab(
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
  group('ProfileProjectsTab', () {
    testWidgets('renders project posts from userProjectsProvider', (
      tester,
    ) async {
      final calls = <({String handleOrDid, String? cursor, int? limit})>[];
      final repo = FakePostRepository(
        onListProjectsByAuthor: (handleOrDid, {cursor, limit}) async {
          calls.add((handleOrDid: handleOrDid, cursor: cursor, limit: limit));
          return PostPage(items: [_projectPost('a'), _projectPost('b')]);
        },
      );

      await _pump(tester, repo: repo);
      await tester.pumpAndSettle();

      expect(calls, [
        (handleOrDid: 'alice.craftsky.social', cursor: null, limit: 10),
      ]);
      expect(find.text('Project a'), findsOneWidget);
      expect(find.text('project post a'), findsOneWidget);
      expect(find.text('Project b'), findsOneWidget);
      expect(find.text('No projects yet.'), findsNothing);
    });

    testWidgets('shows empty state when there are no projects', (tester) async {
      final repo = FakePostRepository(
        onListProjectsByAuthor: (_, {cursor, limit}) async =>
            const PostPage(items: []),
      );

      await _pump(tester, repo: repo);
      await tester.pumpAndSettle();

      expect(find.text('No projects yet.'), findsOneWidget);
    });

    testWidgets('shows safe copy when projects fail to load', (tester) async {
      final repo = FakePostRepository(
        onListProjectsByAuthor: (_, {cursor, limit}) async =>
            throw StateError('project query failed for did:plc:alice'),
      );

      await _pump(tester, repo: repo);
      await tester.pumpAndSettle();

      expect(find.text("This didn't load. Please try again."), findsOneWidget);
      expect(find.textContaining('project query failed'), findsNothing);
      expect(find.textContaining('did:plc:alice'), findsNothing);
    });

    testWidgets('scrolling near the end appends the next page', (tester) async {
      final calls = <({String? cursor, int? limit})>[];
      final repo = FakePostRepository(
        onListProjectsByAuthor: (_, {cursor, limit}) async {
          calls.add((cursor: cursor, limit: limit));
          if (calls.length == 1) {
            return PostPage(
              items: [for (var i = 0; i < 10; i++) _projectPost('a$i')],
              cursor: 'c1',
            );
          }
          expect(cursor, 'c1');
          return PostPage(items: [_projectPost('b')]);
        },
      );

      await _pump(tester, repo: repo);
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('project post a9'),
        500,
        scrollable: find.byType(Scrollable),
      );
      await tester.pumpAndSettle();

      expect(calls, [
        (cursor: null, limit: 10),
        (cursor: 'c1', limit: 10),
      ]);
      expect(find.text('Project a9'), findsOneWidget);
      expect(find.text('Project b'), findsOneWidget);
    });
  });
}
