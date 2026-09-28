import 'dart:ui' show SemanticsAction;

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_interaction_summary.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UT-001 omits the summary when every count is zero', (
    tester,
  ) async {
    await _pumpSummary(tester, post: _post());

    final summary = find.byType(PostInteractionSummary);
    expect(summary, findsOneWidget);
    expect(
      find.descendant(of: summary, matching: find.byType(SizedBox)),
      findsOneWidget,
    );
    expect(find.byType(Wrap), findsNothing);
    expect(find.textContaining(RegExp('^0 ')), findsNothing);
    expect(tester.getSize(find.byType(SizedBox)), Size.zero);
  });

  for (final testCase in [
    (name: 'likes', counts: (1, 0, 0), label: '1 Like'),
    (name: 'reposts', counts: (0, 1, 0), label: '1 Repost'),
    (name: 'quotes', counts: (0, 0, 1), label: '1 Quote'),
  ]) {
    testWidgets('UT-001 shows the ${testCase.name} singleton', (tester) async {
      final (likes, reposts, quotes) = testCase.counts;

      await _pumpSummary(
        tester,
        post: _post(
          likeCount: likes,
          repostCount: reposts,
          quoteCount: quotes,
        ),
      );

      expect(find.text(testCase.label), findsOneWidget);
      expect(find.byType(TextButton), findsOneWidget);
    });
  }

  testWidgets('UT-001 omits zeros from a mixed summary', (tester) async {
    await _pumpSummary(
      tester,
      post: _post(repostCount: 2, quoteCount: 1),
    );

    expect(find.text('2 Reposts'), findsOneWidget);
    expect(find.text('1 Quote'), findsOneWidget);
    expect(find.textContaining('Like'), findsNothing);
  });

  testWidgets(
    'UT-001 uses exact localized integers, fixed order, Wrap, '
    'and no separators',
    (tester) async {
      await _pumpSummary(
        tester,
        post: _post(likeCount: 3, repostCount: 2, quoteCount: 1),
      );

      final wrap = tester.widget<Wrap>(find.byType(Wrap));
      expect(wrap.alignment, WrapAlignment.start);
      expect(
        wrap.children
            .map((child) => ((child as Semantics).child! as TextButton).child)
            .whereType<Text>()
            .map((text) => text.data),
        ['3 Likes', '2 Reposts', '1 Quote'],
      );
      expect(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text && RegExp(r'^[,.;|]$').hasMatch(widget.data ?? ''),
        ),
        findsNothing,
      );
    },
  );

  testWidgets('UT-001 invokes each callback independently', (tester) async {
    var likes = 0;
    var reposts = 0;
    var quotes = 0;
    await _pumpSummary(
      tester,
      post: _post(likeCount: 3, repostCount: 2, quoteCount: 1),
      onLikes: () => likes++,
      onReposts: () => reposts++,
      onQuotes: () => quotes++,
    );

    await tester.tap(find.text('3 Likes'));
    expect((likes, reposts, quotes), (1, 0, 0));
    await tester.tap(find.text('2 Reposts'));
    expect((likes, reposts, quotes), (1, 1, 0));
    await tester.tap(find.text('1 Quote'));
    expect((likes, reposts, quotes), (1, 1, 1));
  });

  testWidgets('UT-001 derives labels from the Post on every build', (
    tester,
  ) async {
    await _pumpSummary(tester, post: _post(likeCount: 1));
    expect(find.text('1 Like'), findsOneWidget);

    await _pumpSummary(tester, post: _post(likeCount: 4));
    expect(find.text('1 Like'), findsNothing);
    expect(find.text('4 Likes'), findsOneWidget);
  });

  testWidgets('UT-001 exposes minimum touch and destination semantics', (
    tester,
  ) async {
    await _pumpSummary(
      tester,
      post: _post(likeCount: 12, repostCount: 3, quoteCount: 2),
    );

    for (final entry in [
      (label: '12 Likes', destination: 'Likes'),
      (label: '3 Reposts', destination: 'Reposts'),
      (label: '2 Quotes', destination: 'Quotes'),
    ]) {
      final button = find.widgetWithText(TextButton, entry.label);
      final size = tester.getSize(button);
      final semantics = tester.getSemantics(button).getSemanticsData();
      expect(size.width, greaterThanOrEqualTo(48));
      expect(size.height, greaterThanOrEqualTo(48));
      expect(semantics.label, entry.label);
      expect(semantics.hint, 'Open ${entry.destination} for this post');
      expect(semantics.flagsCollection.isButton, isTrue);
      expect(semantics.hasAction(SemanticsAction.tap), isTrue);
    }
  });
}

Future<void> _pumpSummary(
  WidgetTester tester, {
  required Post post,
  VoidCallback? onLikes,
  VoidCallback? onReposts,
  VoidCallback? onQuotes,
}) {
  return tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        body: PostInteractionSummary(
          post: post,
          onLikes: onLikes ?? () {},
          onReposts: onReposts ?? () {},
          onQuotes: onQuotes ?? () {},
        ),
      ),
    ),
  );
}

Post _post({int likeCount = 0, int repostCount = 0, int quoteCount = 0}) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
    cid: 'bafyroot',
    rkey: 'root',
    text: 'A post',
    tags: const [],
    createdAt: DateTime.utc(2026, 9, 6),
    indexedAt: DateTime.utc(2026, 9, 6),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
    ),
    likeCount: likeCount,
    repostCount: repostCount,
    quoteCount: quoteCount,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    sponsored: false,
  );
}
