import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Post _post({String? displayName}) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafy123',
    rkey: '3lf2abc',
    text: 'Cast on for the Hitchhiker shawl tonight.',
    tags: const [],
    createdAt: DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: displayName,
    ),
  );
}

Future<void> _pump(
  WidgetTester tester,
  Widget child,
) {
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
      expect(find.byIcon(Icons.more_horiz), findsNothing);
      expect(find.byIcon(Icons.favorite_border), findsOneWidget);
      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.byIcon(Icons.repeat), findsOneWidget);
    });

    testWidgets('falls back to handle when display name is absent', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post()));

      expect(find.text('alice.craftsky.social'), findsOneWidget);
    });

    testWidgets('shows delete action only when callback is supplied', (
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
  });
}
