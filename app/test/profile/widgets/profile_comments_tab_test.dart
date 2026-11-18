import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_comments_tab.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../feed/fakes/fake_post_repository.dart';
import 'profile_tab_test_harness.dart';

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
    sponsored: false,
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
  return pumpProfileTab(
    tester,
    sliver: ProfileCommentsTab(
      did: Did.parse('did:plc:alice'),
      isOwnProfile: false,
    ),
    repository: repo,
  );
}

void main() {
  group('ProfileCommentsTab', () {
    for (final disposeTab in [false, true]) {
      testWidgets(
        disposeTab
            ? 'late reply completion does not use a disposed tab ref'
            : 'reply completion refreshes a mounted tab',
        (tester) async {
          tester.view.physicalSize = const Size(1200, 900);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.resetPhysicalSize);
          addTearDown(tester.view.resetDevicePixelRatio);
          final showTab = ValueNotifier(true);
          addTearDown(showTab.dispose);
          final reply = Completer<Post?>();
          var openedComposer = false;
          final post = _comment('original');
          var loads = 0;
          await pumpProfileTab(
            tester,
            repository: FakePostRepository(
              onListCommentsByAuthor: (_, {cursor, limit}) async {
                loads++;
                return PostPage(items: [post]);
              },
            ),
            sliver: SliverFillRemaining(
              child: _HeldComposerNavigator(
                result: reply.future,
                onOpen: () => openedComposer = true,
                child: Scaffold(
                  body: ValueListenableBuilder(
                    valueListenable: showTab,
                    builder: (_, visible, _) => CustomScrollView(
                      slivers: [
                        if (visible)
                          ProfileCommentsTab(
                            did: Did.parse('did:plc:alice'),
                            isOwnProfile: false,
                          )
                        else
                          const SliverToBoxAdapter(),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          tester.widget<PostCard>(find.byType(PostCard)).onReply!();
          expect(openedComposer, isTrue);
          if (disposeTab) {
            showTab.value = false;
            await tester.pump();
            expect(find.byType(ProfileCommentsTab), findsNothing);
          }

          reply.complete(_comment('published-reply'));
          await tester.pumpAndSettle();

          expect(tester.takeException(), isNull);
          expect(loads, disposeTab ? 1 : 2);
        },
      );
    }

    for (final isReply in [false, true]) {
      for (final likeCount in [0, 2]) {
        testWidgets(
          '${isReply ? 'reply' : 'comment'} menu with $likeCount likes',
          (tester) async {
            final root = PostRef(
              uri: 'at://did:plc:bob/social.craftsky.feed.post/root',
              cid: 'bafyroot',
            );
            final post = _comment('response').copyWith(
              likeCount: likeCount,
              reply: PostReply(
                root: root,
                parent: isReply
                    ? PostRef(
                        uri:
                            'at://did:plc:bob/social.craftsky.feed.post/parent',
                        cid: 'bafyparent',
                      )
                    : root,
              ),
            );
            await _pump(
              tester,
              repo: FakePostRepository(
                onListCommentsByAuthor: (_, {cursor, limit}) async =>
                    PostPage(items: [post]),
              ),
            );
            await tester.pumpAndSettle();
            final menu = tester.widget<CraftskyContextMenuButton>(
              find.byType(CraftskyContextMenuButton).first,
            );
            final items = menu.groups.expand((group) => group.items);
            if (likeCount > 0) {
              expect(menu.groups.first.items.single.text, 'View likes');
              expect(menu.groups.first.items.single.onPressed, isNotNull);
            } else {
              expect(
                items.map((item) => item.text),
                isNot(contains('View likes')),
              );
            }
          },
        );
      }
    }

    testWidgets('shows comment skeletons during the initial load', (
      tester,
    ) async {
      final pending = Completer<PostPage>();
      final repo = FakePostRepository(
        onListCommentsByAuthor: (_, {cursor, limit}) => pending.future,
      );

      await _pump(tester, repo: repo);
      await tester.pump();

      expect(find.byType(CraftskySkeletonSliverList), findsOneWidget);
      expect(find.byType(CommentRowSkeleton), findsWidgets);
    });

    testWidgets('scrolling near the end appends the next page', (tester) async {
      final calls = <ProfilePageRequest>[];
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

      await expectProfileTabInfiniteScroll(
        tester,
        sliver: ProfileCommentsTab(
          did: Did.parse('did:plc:alice'),
          isOwnProfile: false,
        ),
        repository: repo,
        requests: calls,
        lastInitialItem: find.text('comment a9'),
        appendedItem: find.text('comment b'),
      );
      expect(find.text('Load more comments'), findsNothing);
    });
  });
}

// Hold the modal's result without building the composer. The regression is the
// originating tab's lifetime while an independent route operation is pending.
class _HeldComposerNavigator extends Navigator {
  new({
    required this.result,
    required this.onOpen,
    required Widget child,
  }) : super(
         onGenerateRoute: (_) => MaterialPageRoute<void>(builder: (_) => child),
       );

  final Future<Post?> result;
  final VoidCallback onOpen;

  @override
  NavigatorState createState() => _HeldComposerNavigatorState();
}

class _HeldComposerNavigatorState extends NavigatorState {
  @override
  Future<T?> push<T extends Object?>(Route<T> route) {
    final host = widget as _HeldComposerNavigator;
    if (route.settings.name != 'post-composer') return super.push(route);
    host.onOpen();
    return host.result.then((value) => value as T?);
  }
}
