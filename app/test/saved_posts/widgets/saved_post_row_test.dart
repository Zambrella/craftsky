import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('AT-006 saved row keeps navigation and mutations parent-owned', (
    tester,
  ) async {
    var opens = 0;
    var moves = 0;
    var unsaves = 0;
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SavedPostRow(
              account: AccountKey('did:plc:alice'),
              item: _item(),
              onOpen: () => opens++,
              onMove: () => moves++,
              onUnsave: () => unsaves++,
            ),
          ),
        ),
      ),
    );

    expect(find.text('A saved reply'), findsOneWidget);
    expect(find.text('Move'), findsOneWidget);
    expect(find.text('Unsave'), findsOneWidget);
    expect(find.byIcon(Icons.bookmark), findsNothing);
    expect(find.byIcon(Icons.favorite_border), findsNothing);

    await tester.tap(find.text('A saved reply'));
    await tester.tap(find.widgetWithText(TextButton, 'Move'));
    await tester.tap(find.widgetWithText(TextButton, 'Unsave'));
    expect((opens, moves, unsaves), (1, 1, 1));
  });

  testWidgets('AT-012 saved row uses PostSummary with parent-owned metadata', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SavedPostRow(
              account: AccountKey('did:plc:alice'),
              item: _item(),
              onOpen: () {},
              onMove: () {},
              onUnsave: () {},
            ),
          ),
        ),
      ),
    );

    expect(find.byType(PostSummary), findsOneWidget);
    expect(find.byType(RelativeTimeText), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(PostSummary),
        matching: find.text('Move'),
      ),
      findsNothing,
    );
    expect(
      find.descendant(
        of: find.byType(PostSummary),
        matching: find.text('Unsave'),
      ),
      findsNothing,
    );
  });

  testWidgets('AT-013 saved row remains operable at narrow 2x text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: const TextScaler.linear(2)),
            child: child!,
          ),
          home: Scaffold(
            body: SavedPostRow(
              account: AccountKey('did:plc:alice'),
              item: _item(
                text:
                    'A very long Unicode saved reply 🧶 🪡 '
                    'that must remain bounded and readable.',
                handle: 'a-very-long-author-handle.craftsky.social',
              ),
              onOpen: () {},
              onMove: () {},
              onUnsave: () {},
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(tester.takeException(), isNull);
    for (final label in ['Move', 'Unsave']) {
      final button = find.widgetWithText(TextButton, label);
      expect(button, findsOneWidget);
      expect(tester.getSize(button).height, greaterThanOrEqualTo(48));
    }
    expect(find.byType(PostSummary), findsOneWidget);
    semantics.dispose();
  });
}

SavedPostItem _item({
  String text = 'A saved reply',
  String handle = 'author.craftsky.social',
}) => SavedPostItemMapper.fromMap({
  'post': {
    'uri': 'at://did:plc:author/social.craftsky.feed.post/reply',
    'cid': 'bafyreply',
    'rkey': 'reply',
    'text': text,
    'tags': <String>[],
    'likeCount': 0,
    'repostCount': 0,
    'quoteCount': 0,
    'replyCount': 0,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'viewerHasReplied': false,
    'viewerHasSaved': true,
    'viewerSavedFolderId': 'folder-a',
    'createdAt': '2026-07-21T10:00:00.000Z',
    'indexedAt': '2026-07-21T10:00:01.000Z',
    'author': {
      'did': 'did:plc:author',
      'handle': handle,
    },
  },
  'savedAt': '2026-07-21T12:00:00.000Z',
  'folderId': 'folder-a',
});
