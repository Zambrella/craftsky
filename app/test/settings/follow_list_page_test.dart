import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/settings/pages/follow_list_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../profile/fakes/fake_profile_repository.dart';

void main() {
  testWidgets('followers page shows count and preserves repository order', (
    tester,
  ) async {
    final repo = FakeProfileRepository(
      onListFollowersMe: ({cursor, limit}) async => ProfileAccountPage(
        totalCount: 3,
        items: [
          ProfileAccountSummary(
            did: 'did:plc:dana',
            handle: 'dana.craftsky.social',
            displayName: 'Dana',
            isCraftskyProfile: true,
          ),
          ProfileAccountSummary(
            did: 'did:plc:carol',
            handle: 'carol.craftsky.social',
            displayName: 'Carol',
            isCraftskyProfile: true,
          ),
        ],
      ),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FollowListPage(kind: FollowListKind.followers),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Followers (3)'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Dana')).dy,
      lessThan(tester.getTopLeft(find.text('Carol')).dy),
    );
  });

  testWidgets('following page shows empty copy', (tester) async {
    final repo = FakeProfileRepository(
      onListFollowingMe: ({cursor, limit}) async =>
          const ProfileAccountPage(totalCount: 0, items: []),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const FollowListPage(kind: FollowListKind.following),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Following (0)'), findsOneWidget);
    expect(find.text('You are not following anyone'), findsOneWidget);
  });

  testWidgets('following page refreshes from empty', (tester) async {
    var calls = 0;
    final repo = FakeProfileRepository(
      onListFollowingMe: ({cursor, limit}) async {
        calls++;
        return ProfileAccountPage(
          totalCount: calls == 1 ? 0 : 1,
          items: calls == 1
              ? const []
              : [
                  ProfileAccountSummary(
                    did: 'did:plc:dana',
                    handle: 'dana.craftsky.social',
                    displayName: 'Dana',
                    isCraftskyProfile: true,
                  ),
                ],
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const FollowListPage(kind: FollowListKind.following),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pumpAndSettle();

    expect(calls, 2);
    expect(find.text('Following (1)'), findsOneWidget);
    expect(find.text('Dana'), findsOneWidget);
  });

  testWidgets('followers page loads and appends cursor pages', (tester) async {
    final cursors = <String?>[];
    final repo = FakeProfileRepository(
      onListFollowersMe: ({cursor, limit}) async {
        cursors.add(cursor);
        if (cursor == 'next-followers') {
          return ProfileAccountPage(
            totalCount: 3,
            items: [
              ProfileAccountSummary(
                did: 'did:plc:bob',
                handle: 'bob.craftsky.social',
                displayName: 'Bob',
                isCraftskyProfile: true,
              ),
            ],
          );
        }
        return ProfileAccountPage(
          totalCount: 3,
          cursor: 'next-followers',
          items: [
            ProfileAccountSummary(
              did: 'did:plc:dana',
              handle: 'dana.craftsky.social',
              displayName: 'Dana',
              isCraftskyProfile: true,
            ),
            ProfileAccountSummary(
              did: 'did:plc:carol',
              handle: 'carol.craftsky.social',
              displayName: 'Carol',
              isCraftskyProfile: true,
            ),
          ],
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FollowListPage(kind: FollowListKind.followers),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(cursors, [isNull]);
    expect(find.text('Dana'), findsOneWidget);
    expect(find.text('Carol'), findsOneWidget);
    expect(find.text('Load more'), findsOneWidget);

    await tester.tap(find.text('Load more'));
    await tester.pumpAndSettle();

    expect(cursors, [isNull, 'next-followers']);
    expect(find.text('Bob'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Carol')).dy,
      lessThan(tester.getTopLeft(find.text('Bob')).dy),
    );
    expect(find.text('Load more'), findsNothing);
  });

  testWidgets('refresh ignores an older in-flight pagination result', (
    tester,
  ) async {
    final loadMore = Completer<ProfileAccountPage>();
    var firstPageCalls = 0;
    final repo = FakeProfileRepository(
      onListFollowersMe: ({cursor, limit}) async {
        if (cursor != null) return loadMore.future;
        firstPageCalls++;
        return ProfileAccountPage(
          totalCount: 1,
          cursor: firstPageCalls == 1 ? 'next' : null,
          items: [
            ProfileAccountSummary(
              did: firstPageCalls == 1
                  ? 'did:plc:initial'
                  : 'did:plc:refreshed',
              handle: firstPageCalls == 1 ? 'initial.test' : 'refreshed.test',
              displayName: firstPageCalls == 1 ? 'Initial' : 'Refreshed',
              isCraftskyProfile: true,
            ),
          ],
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FollowListPage(kind: FollowListKind.followers),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Load more'));
    await tester.pump();

    final refresh = tester
        .widget<RefreshIndicator>(find.byType(RefreshIndicator))
        .onRefresh();
    await refresh;
    await tester.pump();
    loadMore.complete(
      ProfileAccountPage(
        totalCount: 2,
        items: [
          ProfileAccountSummary(
            did: 'did:plc:stale',
            handle: 'stale.test',
            displayName: 'Stale',
            isCraftskyProfile: true,
          ),
        ],
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Refreshed'), findsOneWidget);
    expect(find.text('Initial'), findsNothing);
    expect(find.text('Stale'), findsNothing);
  });

  testWidgets('TDD-005C tapping an account opens the compact profile route', (
    tester,
  ) async {
    final repo = FakeProfileRepository(
      onListFollowingMe: ({cursor, limit}) async => ProfileAccountPage(
        totalCount: 1,
        items: [
          ProfileAccountSummary(
            did: 'did:plc:dana',
            handle: 'dana.craftsky.social',
            displayName: 'Dana',
            isCraftskyProfile: true,
          ),
        ],
      ),
    );
    GoRouterState? destination;
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) =>
              const FollowListPage(kind: FollowListKind.following),
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

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Dana'));
    await tester.pumpAndSettle();

    expect(destination?.uri.path, '/profile/dana.craftsky.social');
    expect(
      (destination?.extra as ProfilePresentationRequest?)?.startsCompact,
      isTrue,
    );
  });
}
