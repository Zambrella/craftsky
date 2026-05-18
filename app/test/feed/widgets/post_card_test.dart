import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Post _post({
  String? displayName,
  int likeCount = 0,
  int repostCount = 0,
  int replyCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
  bool viewerHasReplied = false,
  DateTime? createdAt,
}) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafy123',
    rkey: '3lf2abc',
    text: 'Cast on for the Hitchhiker shawl tonight.',
    tags: const [],
    likeCount: likeCount,
    repostCount: repostCount,
    replyCount: replyCount,
    viewerHasLiked: viewerHasLiked,
    viewerHasReposted: viewerHasReposted,
    viewerHasReplied: viewerHasReplied,
    createdAt: createdAt ?? DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: displayName,
    ),
  );
}

Future<void> _pump(WidgetTester tester, Widget child) {
  return tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  group('PostCard', () {
    testWidgets('renders author, handle, body, and relative time', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post(displayName: 'Alice')));

      expect(find.text('Alice'), findsOneWidget);
      expect(find.textContaining('@alice.craftsky.social'), findsOneWidget);
      expect(
        find.text('Cast on for the Hitchhiker shawl tonight.'),
        findsOneWidget,
      );
      expect(find.textContaining('3m'), findsOneWidget);
      expect(find.byIcon(Icons.more_horiz), findsOneWidget);
      expect(find.byIcon(Icons.favorite_border), findsOneWidget);
      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.text('Reply'), findsNothing);
      expect(find.byIcon(Icons.repeat), findsOneWidget);
      expect(find.text('0'), findsNothing);

      final replyIcon = tester.widget<Icon>(
        find.byIcon(Icons.chat_bubble_outline),
      );
      final likeIcon = tester.widget<Icon>(find.byIcon(Icons.favorite_border));
      final repostIcon = tester.widget<Icon>(find.byIcon(Icons.repeat));
      expect(replyIcon.color, BrandColors.ink2);
      expect(likeIcon.color, BrandColors.ink2);
      expect(repostIcon.color, BrandColors.ink2);
    });

    testWidgets('renders engagement counts and selected colours', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            likeCount: 5,
            repostCount: 2,
            replyCount: 3,
            viewerHasLiked: true,
            viewerHasReposted: true,
            viewerHasReplied: true,
          ),
        ),
      );

      expect(find.byIcon(Icons.favorite), findsOneWidget);
      expect(find.text('5'), findsOneWidget);
      expect(find.text('2'), findsOneWidget);
      expect(find.text('3'), findsOneWidget);

      final replyIcon = tester.widget<Icon>(
        find.byIcon(Icons.chat_bubble_outline),
      );
      final likeIcon = tester.widget<Icon>(find.byIcon(Icons.favorite));
      final repostIcon = tester.widget<Icon>(find.byIcon(Icons.repeat));
      final replyCount = tester.widget<Text>(find.text('3'));
      final likeCount = tester.widget<Text>(find.text('5'));
      final repostCount = tester.widget<Text>(find.text('2'));

      expect(replyIcon.color, BrandColors.clay);
      expect(likeIcon.color, BrandColors.red);
      expect(repostIcon.color, BrandColors.moss);
      expect(replyCount.style?.color, BrandColors.clay);
      expect(likeCount.style?.color, BrandColors.red);
      expect(repostCount.style?.color, BrandColors.moss);
    });

    testWidgets('colours reply label when viewer has replied', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(viewerHasReplied: true),
          showReplyCount: false,
          showReplyLabel: true,
        ),
      );

      final replyIcon = tester.widget<Icon>(
        find.byIcon(Icons.chat_bubble_outline),
      );
      final replyLabel = tester.widget<Text>(find.text('Reply'));

      expect(replyIcon.color, BrandColors.clay);
      expect(replyLabel.style?.color, BrandColors.clay);
    });

    testWidgets('formats large engagement counts compactly', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(likeCount: 1234, repostCount: 15000, replyCount: 2000000),
        ),
      );

      expect(find.text('1.2k'), findsOneWidget);
      expect(find.text('15k'), findsOneWidget);
      expect(find.text('2m'), findsOneWidget);
      expect(find.text('1234'), findsNothing);
    });

    testWidgets('can hide reply count while keeping reply action', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(replyCount: 3),
          showReplyCount: false,
          showReplyLabel: true,
        ),
      );

      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.text('3'), findsNothing);
      expect(find.text('Reply'), findsOneWidget);
    });

    testWidgets('can hide reply label while keeping reply count', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(post: _post(replyCount: 3)),
      );

      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.text('3'), findsOneWidget);
      expect(find.text('Reply'), findsNothing);
    });

    testWidgets('flat style does not use the card surface', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(),
          style: PostCardStyle.flat,
          showReplyLabel: true,
        ),
      );

      expect(find.byType(CraftskyCard), findsNothing);
      expect(find.text('Reply'), findsOneWidget);
    });

    testWidgets('uses stronger treatment when highlighted', (tester) async {
      await _pump(tester, PostCard(post: _post(), isHighlighted: true));

      final highlighted = tester.widget<AnimatedContainer>(
        find.byWidgetPredicate(
          (widget) =>
              widget is AnimatedContainer && widget.decoration is BoxDecoration,
        ),
      );
      final decoration = highlighted.decoration! as BoxDecoration;
      final border = decoration.border! as Border;

      expect(decoration.color, BrandColors.sky.withValues(alpha: 0.32));
      expect(border.left.color, BrandColors.cobalt);
      expect(border.left.width, 6);
    });

    testWidgets('shows full timestamp tooltip on relative time', (
      tester,
    ) async {
      final createdAt = DateTime(2026, 5, 15, 14, 30);
      await _pump(tester, PostCard(post: _post(createdAt: createdAt)));

      final tooltip = tester.widget<Tooltip>(find.byType(Tooltip).first);
      expect(tooltip.message, contains('2026'));
      expect(tooltip.message, contains('2:30'));
    });

    testWidgets('lays out interaction actions left-aligned at natural widths', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(post: _post(likeCount: 5, repostCount: 2, replyCount: 3)),
      );

      expect(find.byType(TextButton), findsNWidgets(3));
      final likeRect = tester.getRect(find.byType(TextButton).at(0));
      final replyRect = tester.getRect(find.byType(TextButton).at(1));
      final repostRect = tester.getRect(find.byType(TextButton).at(2));

      expect(likeRect.left, lessThan(replyRect.left));
      expect(replyRect.left, lessThan(repostRect.left));
      expect(replyRect.width, likeRect.width);

      final replyCountRect = tester.getRect(find.text('3'));
      final likeCountRect = tester.getRect(find.text('5'));
      final repostCountRect = tester.getRect(find.text('2'));

      expect(replyCountRect.right, lessThan(replyRect.right));
      expect(likeCountRect.right, lessThan(likeRect.right));
      expect(repostCountRect.right, lessThan(repostRect.right));

      final replyCount = tester.widget<Text>(find.text('3'));
      expect(
        replyCount.style?.fontFeatures,
        contains(const FontFeature.tabularFigures()),
      );
    });

    testWidgets('invokes interaction callbacks', (tester) async {
      var replies = 0;
      var likes = 0;
      var reposts = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(),
          onReply: () => replies++,
          onLike: () => likes++,
          onRepost: () => reposts++,
        ),
      );

      await tester.tap(find.byIcon(Icons.favorite_border));
      await tester.tap(find.byIcon(Icons.chat_bubble_outline));
      await tester.tap(find.byIcon(Icons.repeat));

      expect(replies, 1);
      expect(likes, 1);
      expect(reposts, 1);
    });

    testWidgets('invokes card tap from body', (tester) async {
      var taps = 0;
      await _pump(tester, PostCard(post: _post(), onTap: () => taps++));

      await tester.tap(find.text('Cast on for the Hitchhiker shawl tonight.'));

      expect(taps, 1);
    });

    testWidgets('reply tap does not invoke card tap', (tester) async {
      var replies = 0;
      var taps = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(),
          onTap: () => taps++,
          onReply: () => replies++,
        ),
      );

      await tester.tap(find.byIcon(Icons.chat_bubble_outline));

      expect(replies, 1);
      expect(taps, 0);
    });

    testWidgets('falls back to handle when display name is absent', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post()));

      expect(find.text('alice.craftsky.social'), findsOneWidget);
    });

    testWidgets('opens empty menu when delete callback is absent', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post()));

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.text('Delete post'), findsNothing);
    });

    testWidgets('shows delete action when callback is supplied', (
      tester,
    ) async {
      var tapped = false;
      await _pump(
        tester,
        PostCard(post: _post(), onDelete: () => tapped = true),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete post'));
      expect(tapped, isTrue);
    });

    testWidgets('uses custom delete label when supplied', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(),
          deleteLabel: 'Delete comment',
          onDelete: () {},
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.text('Delete comment'), findsOneWidget);
      expect(find.text('Delete post'), findsNothing);
    });
  });
}
