import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart'
    hide PostCommentSection;
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/fake_post_repository.dart';

PostCommentSection _section(String text) => PostCommentSection(
  post: Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
    cid: 'bafyroot',
    rkey: 'root',
    text: text,
    tags: const [],
    createdAt: DateTime.utc(2026, 7, 16),
    indexedAt: DateTime.utc(2026, 7, 16),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
    ),
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
  ),
  sort: CommentSort.oldest,
  comments: const CommentPage(items: []),
);

void main() {
  testWidgets('notification destination 404 shows permanent recovery actions', (
    tester,
  ) async {
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          throw const ApiBadRequest(
            'post_not_found',
            details: ApiFailureDetails(statusCode: 404),
          ),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [postRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(
              did: Did.parse('did:plc:alice'),
              rkey: RecordKey.parse('root'),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Post'), findsOneWidget);
    expect(find.text('This is no longer available'), findsOneWidget);
    expect(
      find.text('This post or profile may have been deleted or hidden.'),
      findsOneWidget,
    );
    expect(find.widgetWithText(TextButton, 'Back'), findsOneWidget);
    expect(
      find.widgetWithText(TextButton, 'View notifications'),
      findsOneWidget,
    );
    expect(find.text('Retry'), findsNothing);
  });

  testWidgets('permanent refresh error hides previously loaded post content', (
    tester,
  ) async {
    final did = Did.parse('did:plc:alice');
    final rkey = RecordKey.parse('root');
    var calls = 0;
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        if (calls++ == 0) {
          return _section('previously loaded private content');
        }
        throw const ApiBadRequest(
          'post_not_found',
          details: ApiFailureDetails(statusCode: 404),
        );
      },
    );
    final container = ProviderContainer(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      retry: (_, _) => null,
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(did: did, rkey: rkey),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('previously loaded private content'), findsOneWidget);

    container.invalidate(postCommentSectionProvider(did, rkey));
    await tester.pumpAndSettle();

    expect(calls, 2);
    final state = container.read(
      postCommentSectionProvider(did, rkey),
    );
    expect(state.hasError, isTrue);
    expect(state.error, isA<ApiBadRequest>());
    expect(find.text('This is no longer available'), findsOneWidget);
    expect(find.text('previously loaded private content'), findsNothing);
  });

  testWidgets('transient refresh error keeps destination Retry available', (
    tester,
  ) async {
    final did = Did.parse('did:plc:alice');
    final rkey = RecordKey.parse('root');
    var calls = 0;
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        if (calls++ == 0) return _section('authenticated cached post');
        throw const ApiNetworkError('offline');
      },
    );
    final container = ProviderContainer(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      retry: (_, _) => null,
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(did: did, rkey: rkey),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('authenticated cached post'), findsOneWidget);

    container.invalidate(postCommentSectionProvider(did, rkey));
    await tester.pumpAndSettle();

    expect(calls, 2);
    expect(find.text('authenticated cached post'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
  });

  testWidgets(
    'permanent recovery actions use back stack and notifications route',
    (
      tester,
    ) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            throw const ApiBadRequest(
              'post_not_found',
              details: ApiFailureDetails(statusCode: 404),
            ),
      );
      final router = GoRouter(
        initialLocation: '/feed',
        routes: [
          GoRoute(
            path: '/feed',
            builder: (_, _) => const Scaffold(body: Text('Feed destination')),
          ),
          GoRoute(
            path: '/notifications',
            builder: (_, _) =>
                const Scaffold(body: Text('Notifications destination')),
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (_, state) => FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse(state.pathParameters['did']!),
                rkey: RecordKey.parse(state.pathParameters['rkey']!),
              ),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            routerConfig: router,
          ),
        ),
      );
      unawaited(router.push<void>('/posts/did:plc:alice/root'));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(TextButton, 'Back'));
      await tester.pumpAndSettle();
      expect(find.text('Feed destination'), findsOneWidget);

      unawaited(router.push<void>('/posts/did:plc:alice/root'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(TextButton, 'View notifications'));
      await tester.pumpAndSettle();
      expect(find.text('Notifications destination'), findsOneWidget);
    },
  );

  testWidgets(
    'transient destination failure retries in place and then renders',
    (
      tester,
    ) async {
      var calls = 0;
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
          if (calls++ == 0) throw const ApiNetworkError('offline');
          return _section('loaded after destination retry');
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          retry: (_, _) => null,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse('did:plc:alice'),
                rkey: RecordKey.parse('root'),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text("That didn't load"), findsOneWidget);
      expect(find.text('Check your connection and try again.'), findsOneWidget);
      expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
      expect(calls, 1);

      await tester.tap(find.widgetWithText(TextButton, 'Retry'));
      await tester.pumpAndSettle();

      expect(calls, 2);
      expect(find.text('loaded after destination retry'), findsOneWidget);
      expect(find.text('This is no longer available'), findsNothing);
    },
  );

  testWidgets(
    'authentication loss exposes no notification-specific error state',
    (
      tester,
    ) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            throw const ApiUnauthorized(
              details: ApiFailureDetails(statusCode: 401),
            ),
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          retry: (_, _) => null,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse('did:plc:alice'),
                rkey: RecordKey.parse('root'),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Post'), findsOneWidget);
      expect(find.text('This is no longer available'), findsNothing);
      expect(find.text("That didn't load"), findsNothing);
      expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
    },
  );
}
