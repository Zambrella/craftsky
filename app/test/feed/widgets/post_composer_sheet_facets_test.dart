import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  group('PostComposerSheet facets', () {
    testWidgets('AT-001 submits generated mention, link, and tag facets', (
      tester,
    ) async {
      List<Map<String, dynamic>>? capturedFacets;
      final repo = FakePostRepository(
        onCreateWithFacets:
            ({required text, reply, project, images, facets}) async {
              capturedFacets = facets;
              return _post(text);
            },
      );

      await _openComposer(
        tester,
        overrides: [
          postRepositoryProvider.overrideWithValue(repo),
          accountSuggestionRepositoryProvider.overrideWithValue(
            const MockAccountSuggestionRepository(
              accounts: [
                AccountSuggestion(
                  did: 'did:plc:alice',
                  handle: 'alice.craftsky.social',
                  displayName: 'Alice',
                  avatar: null,
                  isCraftskyProfile: true,
                  viewerIsFollowing: true,
                ),
              ],
            ),
          ),
        ],
      );

      await tester.enterText(
        find.byType(TextField).first,
        '🧶 Hi @alice.craftsky.social see craftsky.social, #SockKAL',
      );
      await _pumpUntilPostEnabled(tester);
      await tester.tap(find.widgetWithText(TextButton, 'Post'));
      await tester.pumpAndSettle();

      expect(capturedFacets, isNotNull);
      final facets = capturedFacets!;
      expect(
        facets
            .expand((facet) => facet['features']! as List<dynamic>)
            .map(
              (feature) => (feature as Map<String, dynamic>)[r'$type'],
            ),
        containsAll([
          'app.bsky.richtext.facet#mention',
          'app.bsky.richtext.facet#link',
          'app.bsky.richtext.facet#tag',
        ]),
      );
      expect(
        facets.expand((facet) => facet['features']! as List<dynamic>),
        containsAll([
          {r'$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:alice'},
          {
            r'$type': 'app.bsky.richtext.facet#link',
            'uri': 'https://craftsky.social',
          },
          {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'SockKAL'},
        ]),
      );
    });

    testWidgets('submits quote target through create provider', (
      tester,
    ) async {
      final quoteTarget = _post('timeline post target');
      final repo = FakePostRepository(
        onCreate: ({required text, reply, images}) async => _post(text),
      );

      await _openComposer(
        tester,
        quoteTarget: quoteTarget,
        overrides: [postRepositoryProvider.overrideWithValue(repo)],
      );

      expect(find.text('timeline post target'), findsOneWidget);

      await tester.enterText(find.byType(TextField).first, 'quote commentary');
      await _pumpUntilPostEnabled(tester);
      await tester.tap(find.widgetWithText(TextButton, 'Post'));
      await tester.pumpAndSettle();

      expect(repo.lastCreateQuote?.uri, quoteTarget.uri);
      expect(repo.lastCreateQuote?.cid, quoteTarget.cid);
    });
  });
}

Future<void> _openComposer(
  WidgetTester tester, {
  Post? quoteTarget,
  List<dynamic> overrides = const [],
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from(overrides),
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: ElevatedButton(
                onPressed: () {
                  unawaited(
                    Navigator.of(context).push<Post?>(
                      MaterialPageRoute<Post?>(
                        fullscreenDialog: true,
                        builder: (_) => PostComposerSheet(
                          composerId: 'facet-composer',
                          quoteTarget: quoteTarget,
                        ),
                      ),
                    ),
                  );
                },
                child: const Text('Open composer'),
              ),
            ),
          ),
        ),
      ),
    ),
  );

  await tester.tap(find.text('Open composer'));
  await tester.pumpAndSettle();
}

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i += 1) {
    await tester.pump(const Duration(milliseconds: 20));
    final buttons = find.widgetWithText(TextButton, 'Post').evaluate();
    if (buttons.isEmpty) continue;
    final button = tester.widget<TextButton>(
      find.widgetWithText(TextButton, 'Post'),
    );
    if (button.onPressed != null) return;
  }
  fail('Timed out waiting for Post button to be enabled');
}

Post _post(String text) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafy123',
    rkey: '3lf2abc',
    text: text,
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    createdAt: DateTime(2026, 5, 22, 12),
    indexedAt: DateTime(2026, 5, 22, 12, 1),
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
  );
}
