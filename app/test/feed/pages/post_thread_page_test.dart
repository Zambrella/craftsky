import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/fake_post_repository.dart';

Post _post(
  String rkey,
  String text, {
  int replyCount = 0,
  String handle = 'alice.craftsky.social',
  String? displayName,
}) {
  return Post(
    uri: 'at://did:plc:$rkey/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: text,
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: replyCount,
    viewerHasLiked: false,
    viewerHasReposted: false,
    createdAt: DateTime(2026, 5, 4, 18, 23, 45),
    indexedAt: DateTime(2026, 5, 4, 18, 23, 47),
    author: PostAuthor(
      did: 'did:plc:$rkey',
      handle: handle,
      displayName: displayName,
    ),
  );
}

PostThread _threadFor(String rkey) {
  final root = _post('root', 'root ancestor');
  final parent = _post('parent', 'parent ancestor');
  final replyA = PostThread(
    post: _post(
      'reply-a',
      'first reply',
      handle: 'bobbin.craftsky.social',
      displayName: 'Bobbin Bee',
    ),
    replies: const [],
  );
  final replyB = PostThread(
    post: _post('reply-b', 'second reply', replyCount: 1),
    replies: [
      PostThread(
        post: _post('grandchild', 'nested grandchild'),
        replies: const [],
      ),
    ],
  );
  return PostThread(
    ancestors: [root, parent],
    post: _post(rkey, 'target post'),
    replies: [replyA, replyB],
  );
}

Future<void> _pumpThread(
  WidgetTester tester,
  FakePostRepository repo, {
  Size size = const Size(390, 900),
  double textScaleFactor = 1,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  final router = GoRouter(
    initialLocation: '/posts/did:plc:alice/target',
    routes: [
      GoRoute(
        path: RouteLocations.postThread,
        builder: (context, state) => PostThreadPage(
          did: state.pathParameters['did']!,
          rkey: state.pathParameters['rkey']!,
        ),
      ),
    ],
  );
  await tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        routerConfig: router,
        builder: (context, child) {
          final thread = FormFactorWidget(
            child: child ?? const SizedBox.shrink(),
          );
          return MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: TextScaler.linear(textScaleFactor),
            ),
            child: thread,
          );
        },
      ),
    ),
  );
}

void _expectTextJustBelowAppBar(WidgetTester tester, String text) {
  final appBarBottom = tester.getBottomLeft(find.byType(AppBar)).dy;
  final textTop = tester.getTopLeft(find.text(text)).dy;

  expect(textTop, greaterThanOrEqualTo(appBarBottom));
  expect(textTop, lessThan(appBarBottom + 160));
}

void main() {
  final l10n = AppLocalizationsEn();

  group('PostThreadPage', () {
    testWidgets('anchors top-level post below app bar with replies below', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => PostThread(
          post: _post(rkey, 'top-level target'),
          replies: [
            PostThread(
              post: _post('reply-a', 'first reply'),
              replies: const [],
            ),
          ],
        ),
      );

      await _pumpThread(tester, repo, size: const Size(390, 900));
      await tester.pumpAndSettle();

      final targetTop = tester.getTopLeft(find.text('top-level target')).dy;

      _expectTextJustBelowAppBar(tester, 'top-level target');
      expect(
        tester.getTopLeft(find.text('first reply')).dy,
        greaterThan(targetTop),
      );
    });

    testWidgets('anchors selected reply below app bar when opened', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo, size: const Size(390, 900));
      await tester.pumpAndSettle();

      _expectTextJustBelowAppBar(tester, 'target post');
      expect(find.text('first reply'), findsOneWidget);

      await tester.drag(find.byType(Scrollable), const Offset(0, 300));
      await tester.pumpAndSettle();

      expect(
        tester.getTopLeft(find.text('root ancestor')).dy,
        lessThan(tester.getTopLeft(find.text('parent ancestor')).dy),
      );
      expect(
        tester.getTopLeft(find.text('parent ancestor')).dy,
        lessThan(tester.getTopLeft(find.text('target post')).dy),
      );
    });

    testWidgets('tapping selected post does not push same thread route', (
      tester,
    ) async {
      final calls = <String>[];
      final repo = FakePostRepository(
        onThread: (_, rkey) async {
          calls.add(rkey);
          return _threadFor(rkey);
        },
      );

      await _pumpThread(tester, repo, size: const Size(390, 900));
      await tester.pumpAndSettle();

      await tester.tap(find.text('target post'));
      await tester.pumpAndSettle();

      expect(calls, ['target']);
    });

    testWidgets('tapping non-selected reply navigates to that reply thread', (
      tester,
    ) async {
      final calls = <String>[];
      final repo = FakePostRepository(
        onThread: (_, rkey) async {
          calls.add(rkey);
          return _threadFor(rkey);
        },
      );

      await _pumpThread(tester, repo, size: const Size(390, 900));
      await tester.pumpAndSettle();

      await tester.tap(find.text('first reply'));
      await tester.pumpAndSettle();

      expect(calls, ['target', 'reply-a']);
    });

    testWidgets('tapping non-selected ancestor navigates to ancestor thread', (
      tester,
    ) async {
      final calls = <String>[];
      final repo = FakePostRepository(
        onThread: (_, rkey) async {
          calls.add(rkey);
          return _threadFor(rkey);
        },
      );

      await _pumpThread(tester, repo, size: const Size(390, 900));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(Scrollable), const Offset(0, 300));
      await tester.pumpAndSettle();

      await tester.tap(find.text('parent ancestor'));
      await tester.pumpAndSettle();

      expect(calls, ['target', 'parent']);
    });

    testWidgets('shows focused thread content', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();

      expect(find.text('target post'), findsOneWidget);
      await tester.drag(find.byType(Scrollable), const Offset(0, 300));
      await tester.pumpAndSettle();
      expect(find.text('root ancestor'), findsOneWidget);
      expect(find.text('parent ancestor'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('first reply'),
        250,
        scrollable: find.byType(Scrollable),
      );
      expect(find.text('first reply'), findsOneWidget);
      expect(find.text('second reply'), findsOneWidget);
      expect(find.text('nested grandchild'), findsNothing);
      expect(
        find.widgetWithText(TextButton, l10n.postThreadShowMoreReplies),
        findsNothing,
      );
      expect(find.text(l10n.postThreadContinueThread), findsOneWidget);
    });

    testWidgets('does not show a continuation row for anchor direct replies', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();

      expect(
        find.widgetWithText(TextButton, l10n.postThreadShowMoreReplies),
        findsNothing,
      );
    });

    testWidgets('shows empty reply state', (tester) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => PostThread(
          post: _post(rkey, 'target post'),
          replies: const [],
        ),
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();

      final targetTop = tester.getTopLeft(find.text('target post')).dy;

      _expectTextJustBelowAppBar(tester, 'target post');
      expect(find.text(l10n.postThreadEmptyReplies), findsOneWidget);
      expect(
        tester.getTopLeft(find.text(l10n.postThreadEmptyReplies)).dy,
        greaterThan(targetTop),
      );
    });

    testWidgets('uses sticky reply prompt on small screens', (tester) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo);
      await tester.pumpAndSettle();

      expect(
        find.byKey(const ValueKey('threadStickyReplyPrompt')),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('threadInlineReplyPrompt')),
        findsNothing,
      );
    });

    testWidgets('uses inline reply prompt on large screens', (tester) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo, size: const Size(1024, 900));
      await tester.pumpAndSettle();

      expect(
        find.byKey(const ValueKey('threadInlineReplyPrompt')),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('threadStickyReplyPrompt')),
        findsNothing,
      );
    });

    testWidgets('shows retry on load error', (tester) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => throw Exception('boom $rkey'),
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();

      expect(find.text(l10n.retryButton), findsOneWidget);
    });

    testWidgets('labels continuation rows with target author context', (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();

      final continuation = find.bySemanticsLabel(
        l10n.postThreadContinueThreadFromAuthor('@alice.craftsky.social'),
      );

      expect(continuation, findsOneWidget);
      semantics.dispose();
    });

    testWidgets('labels reply actions with target author context', (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('first reply'),
        250,
        scrollable: find.byType(Scrollable),
      );

      expect(
        find.bySemanticsLabel(
          l10n.postThreadReplyToAuthor(
            'Bobbin Bee (@bobbin.craftsky.social)',
          ),
        ),
        findsOneWidget,
      );
      semantics.dispose();
    });

    testWidgets('avoids overflow at narrow width and text scale 2', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onThread: (_, rkey) async => _threadFor(rkey),
      );

      await _pumpThread(
        tester,
        repo,
        size: const Size(390, 1300),
        textScaleFactor: 2,
      );
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      await tester.scrollUntilVisible(
        find.text(l10n.postThreadContinueThread),
        250,
        scrollable: find.byType(Scrollable),
      );
      expect(find.text(l10n.postThreadContinueThread), findsOneWidget);
    });

    testWidgets('continuation row re-anchors to the reply', (tester) async {
      final calls = <String>[];
      final repo = FakePostRepository(
        onThread: (_, rkey) async {
          calls.add(rkey);
          return _threadFor(rkey);
        },
      );

      await _pumpThread(tester, repo, size: const Size(390, 1300));
      await tester.pumpAndSettle();
      final continuation = find.widgetWithText(
        TextButton,
        l10n.postThreadContinueThread,
      );
      expect(continuation, findsOneWidget);
      await tester.scrollUntilVisible(
        continuation,
        250,
        scrollable: find.byType(Scrollable),
      );
      await tester.ensureVisible(continuation);
      await tester.tap(continuation);
      await tester.pumpAndSettle();

      expect(calls, contains('reply-b'));
    });
  });
}
