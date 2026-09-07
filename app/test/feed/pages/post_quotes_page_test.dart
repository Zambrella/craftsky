import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/pages/post_quotes_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../../fakes/auth_session_fakes.dart';
import '../fakes/fake_post_repository.dart';

const _subjectDid = 'did:plc:alice';
const _subjectRkey = 'root';

void main() {
  testWidgets(
    'AT-004 renders repeated authors as normal post cards in server order '
    'with a plain title',
    (tester) async {
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) async => PostPage(
          items: [
            _quote('dana-one', 'Dana'),
            _quote('dana-two', 'Dana'),
            _quote('carol', 'Carol'),
          ],
        ),
      );

      await _pumpPage(tester, repository);
      await tester.pumpAndSettle();

      expect(find.widgetWithText(AppBar, 'Quotes'), findsOneWidget);
      expect(find.textContaining('Quotes ('), findsNothing);
      expect(find.byType(PostCard), findsNWidgets(3));
      expect(find.text('Dana'), findsNWidgets(2));
      expect(find.text('Quote dana-one'), findsOneWidget);
      expect(find.text('Quote dana-two'), findsOneWidget);
      expect(find.text('Quote carol'), findsOneWidget);
      expect(
        tester.getTopLeft(find.text('Quote dana-one')).dy,
        lessThan(tester.getTopLeft(find.text('Quote dana-two')).dy),
      );
      expect(
        tester.getTopLeft(find.text('Quote dana-two')).dy,
        lessThan(tester.getTopLeft(find.text('Quote carol')).dy),
      );

      for (final card in tester.widgetList<PostCard>(find.byType(PostCard))) {
        expect(card.onTap, isNotNull);
        expect(card.onReply, isNotNull);
        expect(card.onLike, isNotNull);
        expect(card.onRepost, isNotNull);
        expect(card.onQuote, isNotNull);
      }
    },
  );

  testWidgets(
    'AT-004 preserves normal author, quote preview, and post navigation',
    (
      tester,
    ) async {
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) async => PostPage(
          items: [_quote('dana-one', 'Dana')],
        ),
      );
      final destinations = <Uri>[];
      final extras = <Object?>[];
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) => PostQuotesPage(
              did: Did.parse(_subjectDid),
              rkey: RecordKey.parse(_subjectRkey),
            ),
          ),
          GoRoute(
            path: '/profile/:handle',
            builder: (_, state) {
              destinations.add(state.uri);
              extras.add(state.extra);
              return const Scaffold(body: Text('Profile destination'));
            },
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (_, state) {
              destinations.add(state.uri);
              return const Scaffold(body: Text('Post destination'));
            },
          ),
        ],
      );
      addTearDown(router.dispose);

      await _pumpRouter(tester, repository, router);
      await tester.pumpAndSettle();

      _tapGestureForText(tester, 'Dana');
      await tester.pumpAndSettle();
      expect(destinations.last.path, '/profile/dana.craftsky.social');
      expect(
        (extras.last! as ProfilePresentationRequest).startsCompact,
        isTrue,
      );
      router.pop();
      await tester.pumpAndSettle();

      _tapGestureForText(tester, 'Alice');
      await tester.pumpAndSettle();
      expect(destinations.last.path, '/profile/alice.craftsky.social');
      router.pop();
      await tester.pumpAndSettle();

      await tester.tap(find.text('Original post'));
      await tester.pumpAndSettle();
      expect(destinations.last.path, '/posts/did%3Aplc%3Aalice/original');
      router.pop();
      await tester.pumpAndSettle();

      tester.widget<PostCard>(find.byType(PostCard)).onTap!();
      await tester.pumpAndSettle();
      expect(destinations.last.path, '/posts/did%3Aplc%3Adana/dana-one');
    },
  );

  testWidgets(
    'AT-004 successful card mutations replace and deletion removes '
    'provider items',
    (tester) async {
      final original = _quote('dana-one', 'Dana');
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) async =>
            PostPage(items: [original]),
        onLike: (did, rkey) async => _interaction(original),
        onRepost: (did, rkey) async => _interaction(original),
        onDelete: (did, rkey) async {},
      );

      await _pumpPage(
        tester,
        repository,
        signedInDid: 'did:plc:dana',
      );
      await tester.pumpAndSettle();

      var card = tester.widget<PostCard>(find.byType(PostCard));
      card.onLike!();
      await tester.pumpAndSettle();
      card = tester.widget<PostCard>(find.byType(PostCard));
      expect(card.post.viewerHasLiked, isTrue);
      expect(card.post.likeCount, 1);

      card.onRepost!();
      await tester.pumpAndSettle();
      card = tester.widget<PostCard>(find.byType(PostCard));
      expect(card.post.viewerHasReposted, isTrue);
      expect(card.post.repostCount, 1);

      card.onDelete!();
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(find.byType(PostCard), findsNothing);
      expect(find.text('No quotes yet.'), findsOneWidget);
    },
  );

  testWidgets(
    'REG-005 IR-004 keeps placeholders bounded and reveals muted quotes',
    (tester) async {
      final muted = _quote('muted', 'Muted').copyWith(
        availability: 'muted',
        relationship: const ContentRelationship(
          state: 'muted',
          revealable: true,
        ),
      );
      final unavailable = _quote('unavailable', 'Dana').copyWith(
        quoteView: const QuoteView(state: 'unavailable'),
      );
      final oneLevel = _quote('one-level', 'Carol');
      var revealCalls = 0;
      final repository = FakePostRepository(
        onListQuotes: (did, rkey, {cursor, limit}) async => PostPage(
          items: [muted, unavailable, oneLevel],
        ),
        onFetch: (did, rkey) async {
          revealCalls++;
          expect(did.toString(), 'did:plc:muted');
          expect(rkey.toString(), 'muted');
          return _quote('muted', 'Muted');
        },
      );

      await _pumpPage(tester, repository);
      await tester.pumpAndSettle();

      expect(find.byType(PostCard), findsNWidgets(3));
      expect(find.text('Post from a muted account'), findsOneWidget);
      expect(find.text('Quoted post unavailable'), findsOneWidget);
      expect(find.text('Original post'), findsOneWidget);
      expect(find.textContaining('unavailable.invalid'), findsNothing);

      final cards = tester.widgetList<PostCard>(find.byType(PostCard)).toList();
      expect(cards[0].post.relationship?.revealable, isTrue);
      expect(cards[1].post.quoteView?.post, isNull);
      expect(cards[2].post.quoteView?.post?.text, 'Original post');

      expect(find.text('Show post'), findsOneWidget);
      await tester.tap(find.text('Show post'));
      await tester.pumpAndSettle();

      expect(revealCalls, 1);
      expect(find.text('Quote muted'), findsOneWidget);
      expect(find.text('Post from a muted account'), findsNothing);
    },
  );

  testWidgets('AT-006 Quotes shows loading then its empty state', (
    tester,
  ) async {
    final pending = Completer<PostPage>();
    final repository = FakePostRepository(
      onListQuotes: (did, rkey, {cursor, limit}) => pending.future,
    );
    await _pumpPage(tester, repository);
    await tester.pump();

    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.text('Quotes'), findsOneWidget);

    pending.complete(const PostPage(items: []));
    await tester.pumpAndSettle();
    expect(find.text('No quotes yet.'), findsOneWidget);
  });

  testWidgets('AT-006 Quotes retries transient failure from page one', (
    tester,
  ) async {
    var calls = 0;
    final repository = FakePostRepository(
      onListQuotes: (did, rkey, {cursor, limit}) async {
        calls++;
        if (calls == 1) throw const ApiNetworkError('offline');
        return PostPage(items: [_quote('recovered', 'Dana')]);
      },
    );
    await _pumpPage(tester, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(calls, 2);
    expect(find.text('Quote recovered'), findsOneWidget);
  });

  testWidgets('AT-006 Quotes post_not_found offers Back without Retry', (
    tester,
  ) async {
    final repository = FakePostRepository(
      onListQuotes: (did, rkey, {cursor, limit}) async =>
          throw const ApiBadRequest(
            'post_not_found',
            details: ApiFailureDetails(statusCode: 404),
          ),
    );
    await _pumpPage(tester, repository);
    await tester.pumpAndSettle();

    expect(find.text('Post unavailable'), findsOneWidget);
    expect(find.text('Back'), findsOneWidget);
    expect(find.text('Retry'), findsNothing);
  });

  testWidgets('AT-009 IR-009 direct unavailable Quotes Back returns to Feed', (
    tester,
  ) async {
    final repository = FakePostRepository(
      onListQuotes: (did, rkey, {cursor, limit}) async =>
          throw const ApiBadRequest(
            'post_not_found',
            details: ApiFailureDetails(statusCode: 404),
          ),
    );
    final router = GoRouter(
      initialLocation: '/quotes',
      routes: [
        GoRoute(
          path: '/feed',
          builder: (_, _) => const Scaffold(body: Text('Feed destination')),
        ),
        GoRoute(
          path: '/quotes',
          builder: (_, _) => PostQuotesPage(
            did: Did.parse(_subjectDid),
            rkey: RecordKey.parse(_subjectRkey),
          ),
        ),
      ],
    );
    addTearDown(router.dispose);
    await _pumpRouter(tester, repository, router);
    await tester.pumpAndSettle();

    await tester.tap(find.text('Back'));
    await tester.pumpAndSettle();

    expect(find.text('Feed destination'), findsOneWidget);
  });

  testWidgets('AT-006 Quotes retains rows and retries the same cursor', (
    tester,
  ) async {
    final continuation = Completer<PostPage>();
    final cursors = <String?>[];
    var continuationCalls = 0;
    final repository = FakePostRepository(
      onListQuotes: (did, rkey, {cursor, limit}) {
        cursors.add(cursor);
        if (cursor == null) {
          return Future.value(
            PostPage(items: [_quote('first', 'Dana')], cursor: 'next'),
          );
        }
        continuationCalls++;
        if (continuationCalls == 1) return continuation.future;
        return Future.value(PostPage(items: [_quote('second', 'Carol')]));
      },
    );
    await _pumpPage(tester, repository);
    await tester.pumpAndSettle();
    _notifyNearEnd(tester);
    await tester.pump();
    expect(find.text('Quote first'), findsOneWidget);

    continuation.completeError(const ApiNetworkError('offline'));
    await tester.pumpAndSettle();
    expect(find.text('Quote first'), findsOneWidget);
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(cursors, [null, 'next', 'next']);
    expect(find.text('Quote first'), findsOneWidget);
    expect(find.text('Quote second'), findsOneWidget);
  });
}

Post _quote(String rkey, String displayName) => Post(
  uri:
      'at://did:plc:${displayName.toLowerCase()}/social.craftsky.feed.post/$rkey',
  cid: 'bafy$rkey',
  rkey: rkey,
  text: 'Quote $rkey',
  tags: const [],
  createdAt: DateTime.utc(2026, 9, 6),
  indexedAt: DateTime.utc(2026, 9, 6),
  author: PostAuthor(
    did: 'did:plc:${displayName.toLowerCase()}',
    handle: '${displayName.toLowerCase()}.craftsky.social',
    displayName: displayName,
  ),
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  quoteView: QuoteView(
    state: 'visible',
    post: QuotePreviewPost(
      uri: 'at://did:plc:alice/social.craftsky.feed.post/original',
      cid: 'bafyoriginal',
      text: 'Original post',
      author: PostAuthor(
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
        displayName: 'Alice',
      ),
      createdAt: DateTime.utc(2026, 9, 5),
    ),
  ),
);

InteractionWriteResponse _interaction(Post post) => InteractionWriteResponse(
  uri: 'at://did:plc:test/social.craftsky.feed.like/interaction',
  cid: 'bafyinteraction',
  rkey: 'interaction',
  subject: PostRef(uri: post.uri.value, cid: post.cid.value),
  createdAt: DateTime.utc(2026, 9, 6),
);

void _tapGestureForText(WidgetTester tester, String text) {
  final target = find.ancestor(
    of: find.text(text),
    matching: find.byType(GestureDetector),
  );
  tester.widget<GestureDetector>(target.first).onTap!();
}

void _notifyNearEnd(WidgetTester tester) {
  final context = tester.element(
    find.descendant(
      of: find.byType(AutoPaginatedListView),
      matching: find.byType(ListView),
    ),
  );
  ScrollUpdateNotification(
    metrics: FixedScrollMetrics(
      minScrollExtent: 0,
      maxScrollExtent: 100,
      pixels: 100,
      viewportDimension: 600,
      axisDirection: AxisDirection.down,
      devicePixelRatio: 1,
    ),
    context: context,
    scrollDelta: 1,
  ).dispatch(context);
}

Future<void> _pumpPage(
  WidgetTester tester,
  FakePostRepository repository, {
  String signedInDid = 'did:plc:test',
}) => tester.pumpWidget(
  ProviderScope(
    overrides: List.from(_overrides(repository, signedInDid)),
    retry: (_, _) => null,
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: PostQuotesPage(
        did: Did.parse(_subjectDid),
        rkey: RecordKey.parse(_subjectRkey),
      ),
    ),
  ),
);

Future<void> _pumpRouter(
  WidgetTester tester,
  FakePostRepository repository,
  GoRouter router,
) => tester.pumpWidget(
  ProviderScope(
    overrides: List.from(_overrides(repository, 'did:plc:test')),
    retry: (_, _) => null,
    child: MaterialApp.router(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      routerConfig: router,
    ),
  ),
);

List<dynamic> _overrides(
  FakePostRepository repository,
  String signedInDid,
) => [
  postRepositoryProvider.overrideWithValue(repository),
  authSessionProvider.overrideWith(() => SignedInAuthSession(did: signedInDid)),
  activeLanguagePreferencesProvider.overrideWith(
    (ref) => const LanguagePreferences(
      primaryLanguage: 'en',
      contentLanguages: ['en'],
    ),
  ),
  secureSessionRegistryStorageProvider.overrideWithValue(
    _MemoryRegistryStorage(signedInDid),
  ),
];

final class _MemoryRegistryStorage implements SessionRegistryStorage {
  _MemoryRegistryStorage(String did)
    : registry = SessionRegistry.empty().upsertAndActivate(
        token: 'token-$did',
        did: did,
        handle: 'test.craftsky.social',
      );

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry value) async => registry = value;
}
