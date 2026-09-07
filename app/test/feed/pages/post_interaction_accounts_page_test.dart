import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/pages/post_interaction_accounts_page.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
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

import '../fakes/fake_post_repository.dart';

const _did = 'did:plc:alice';
const _rkey = 'root';

void main() {
  for (final testCase in [
    (kind: PostInteractionAccountKind.likes, title: 'Likes'),
    (kind: PostInteractionAccountKind.reposts, title: 'Reposts'),
  ]) {
    testWidgets(
      'AT-003 ${testCase.title} preserves server order and shows '
      'accounts and total',
      (tester) async {
        final repository = FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) async => _accountPage(),
          onListReposts: (did, rkey, {cursor, limit}) async => _accountPage(),
        );

        await _pumpPage(tester, repository, kind: testCase.kind);
        await tester.pumpAndSettle();

        expect(find.text('${testCase.title} (2)'), findsOneWidget);
        expect(find.text('Dana'), findsOneWidget);
        expect(find.text('@dana.craftsky.social'), findsOneWidget);
        expect(find.text('Carol'), findsOneWidget);
        expect(find.text('@carol.craftsky.social'), findsOneWidget);
        expect(
          tester.getTopLeft(find.text('Dana')).dy,
          lessThan(tester.getTopLeft(find.text('Carol')).dy),
        );
      },
    );

    testWidgets(
      'AT-006 ${testCase.title} shows loading then its empty state '
      'and authoritative zero',
      (tester) async {
        final pending = Completer<ProfileAccountPage>();
        final repository = FakePostRepository(
          onListLikes: (did, rkey, {cursor, limit}) => pending.future,
          onListReposts: (did, rkey, {cursor, limit}) => pending.future,
        );

        await _pumpPage(tester, repository, kind: testCase.kind);
        await tester.pump();

        expect(find.byType(StitchProgressIndicator), findsOneWidget);
        expect(find.text(testCase.title), findsOneWidget);
        expect(find.text('${testCase.title} (0)'), findsNothing);

        pending.complete(const ProfileAccountPage(items: [], totalCount: 0));
        await tester.pumpAndSettle();

        expect(find.text('${testCase.title} (0)'), findsOneWidget);
        expect(
          find.text(
            testCase.kind == PostInteractionAccountKind.likes
                ? 'No likes yet.'
                : 'No reposts yet.',
          ),
          findsOneWidget,
        );
      },
    );
  }

  testWidgets('AT-003 tapping an account opens its compact profile', (
    tester,
  ) async {
    final repository = FakePostRepository(
      onListLikes: (did, rkey, {cursor, limit}) async => _accountPage(),
    );
    GoRouterState? destination;
    final router = GoRouter(
      initialLocation: '/',
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => PostInteractionAccountsPage(
            did: Did.parse(_did),
            rkey: RecordKey.parse(_rkey),
            kind: PostInteractionAccountKind.likes,
          ),
        ),
        GoRoute(
          path: '/profile/:handle',
          builder: (_, state) {
            destination = state;
            return const Scaffold(body: Text('Profile'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);

    await _pumpRouter(tester, repository, router);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Dana'));
    await tester.pumpAndSettle();

    expect(destination?.uri.path, '/profile/dana.craftsky.social');
    expect(
      (destination?.extra as ProfilePresentationRequest?)?.startsCompact,
      isTrue,
    );
  });

  testWidgets('AT-006 transient initial failure retries from page one', (
    tester,
  ) async {
    var calls = 0;
    final repository = FakePostRepository(
      onListLikes: (did, rkey, {cursor, limit}) async {
        calls++;
        if (calls == 1) throw const ApiNetworkError('offline');
        return _accountPage();
      },
    );

    await _pumpPage(
      tester,
      repository,
      kind: PostInteractionAccountKind.likes,
    );
    await tester.pumpAndSettle();

    expect(find.text('Retry'), findsOneWidget);
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(calls, 2);
    expect(find.text('Dana'), findsOneWidget);
    expect(find.text('Likes (2)'), findsOneWidget);
  });

  testWidgets('AT-006 post_not_found offers Back without Retry', (
    tester,
  ) async {
    final repository = FakePostRepository(
      onListReposts: (did, rkey, {cursor, limit}) async =>
          throw const ApiBadRequest(
            'post_not_found',
            details: ApiFailureDetails(statusCode: 404),
          ),
    );

    await _pumpPage(
      tester,
      repository,
      kind: PostInteractionAccountKind.reposts,
    );
    await tester.pumpAndSettle();

    expect(find.text('Post unavailable'), findsOneWidget);
    expect(find.text('Back'), findsOneWidget);
    expect(find.text('Retry'), findsNothing);
    expect(find.text('Reposts'), findsOneWidget);
    expect(find.text('Reposts (0)'), findsNothing);
  });

  testWidgets('AT-009 IR-009 direct unavailable route Back returns to Feed', (
    tester,
  ) async {
    final repository = FakePostRepository(
      onListLikes: (did, rkey, {cursor, limit}) async =>
          throw const ApiBadRequest(
            'post_not_found',
            details: ApiFailureDetails(statusCode: 404),
          ),
    );
    final router = GoRouter(
      initialLocation: '/interaction',
      routes: [
        GoRoute(
          path: '/feed',
          builder: (_, _) => const Scaffold(body: Text('Feed destination')),
        ),
        GoRoute(
          path: '/interaction',
          builder: (_, _) => PostInteractionAccountsPage(
            did: Did.parse(_did),
            rkey: RecordKey.parse(_rkey),
            kind: PostInteractionAccountKind.likes,
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

  testWidgets(
    'AT-006 load-more keeps rows through progress and error then retries '
    'the cursor',
    (tester) async {
      final continuation = Completer<ProfileAccountPage>();
      final cursors = <String?>[];
      var continuationCalls = 0;
      final repository = FakePostRepository(
        onListLikes: (did, rkey, {cursor, limit}) {
          cursors.add(cursor);
          if (cursor == null) {
            return Future.value(
              ProfileAccountPage(
                items: [_account('dana', 'Dana')],
                totalCount: 2,
                cursor: 'next',
              ),
            );
          }
          continuationCalls++;
          if (continuationCalls == 1) return continuation.future;
          return Future.value(
            ProfileAccountPage(
              items: [_account('carol', 'Carol')],
              totalCount: 2,
            ),
          );
        },
      );

      await _pumpPage(
        tester,
        repository,
        kind: PostInteractionAccountKind.likes,
      );
      await tester.pumpAndSettle();
      _notifyNearEnd(tester);
      await tester.pump();

      expect(find.text('Dana'), findsOneWidget);
      expect(find.byType(StitchProgressIndicator), findsOneWidget);

      continuation.completeError(const ApiNetworkError('offline'));
      await tester.pumpAndSettle();

      expect(find.text('Dana'), findsOneWidget);
      expect(find.text('Retry'), findsOneWidget);
      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();

      expect(cursors, [null, 'next', 'next']);
      expect(find.text('Dana'), findsOneWidget);
      expect(find.text('Carol'), findsOneWidget);
      expect(find.text('Retry'), findsNothing);
    },
  );
}

Future<void> _pumpPage(
  WidgetTester tester,
  FakePostRepository repository, {
  required PostInteractionAccountKind kind,
}) => tester.pumpWidget(
  ProviderScope(
    overrides: [
      postRepositoryProvider.overrideWithValue(repository),
      secureSessionRegistryStorageProvider.overrideWithValue(
        _MemoryRegistryStorage(),
      ),
    ],
    retry: (_, _) => null,
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: PostInteractionAccountsPage(
        did: Did.parse(_did),
        rkey: RecordKey.parse(_rkey),
        kind: kind,
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
    overrides: [
      postRepositoryProvider.overrideWithValue(repository),
      secureSessionRegistryStorageProvider.overrideWithValue(
        _MemoryRegistryStorage(),
      ),
    ],
    retry: (_, _) => null,
    child: MaterialApp.router(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      routerConfig: router,
    ),
  ),
);

ProfileAccountPage _accountPage() => ProfileAccountPage(
  items: [_account('dana', 'Dana'), _account('carol', 'Carol')],
  totalCount: 2,
);

ProfileAccountSummary _account(String id, String displayName) =>
    ProfileAccountSummary(
      did: 'did:plc:$id',
      handle: '$id.craftsky.social',
      displayName: displayName,
      isCraftskyProfile: true,
    );

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

final class _MemoryRegistryStorage implements SessionRegistryStorage {
  SessionRegistry registry = SessionRegistry.empty();

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry value) async => registry = value;
}
