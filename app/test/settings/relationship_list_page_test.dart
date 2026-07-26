import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/settings/pages/relationship_list_page.dart';
import 'package:craftsky_app/settings/providers/relationship_list_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../profile/fakes/fake_profile_repository.dart';

Widget _app(FakeProfileRepository repo, RelationshipListKind kind) =>
    ProviderScope(
      overrides: [profileRepositoryProvider.overrideWithValue(repo)],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: RelationshipListPage(kind: kind),
      ),
    );

void main() {
  test('provider owns pagination and row mutation state', () async {
    final unmute = Completer<ProfileRelationship>();
    final cursors = <String?>[];
    final bob = ProfileAccountSummary(
      did: 'did:plc:bob',
      handle: 'bob.craftsky.social',
      isCraftskyProfile: true,
      muted: true,
    );
    final carol = ProfileAccountSummary(
      did: 'did:plc:carol',
      handle: 'carol.craftsky.social',
      isCraftskyProfile: true,
      muted: true,
    );
    final repo = FakeProfileRepository(
      onListMutedProfiles: ({limit, cursor}) async {
        cursors.add(cursor);
        return cursor == null
            ? ProfileAccountPage(
                items: [bob],
                cursor: 'next',
                totalCount: 2,
              )
            : ProfileAccountPage(items: [carol], totalCount: 2);
      },
      onUnmute: (_) => unmute.future,
    );
    final provider = relationshipListProvider(RelationshipListKind.muted);
    final container = ProviderContainer.test(
      overrides: [profileRepositoryProvider.overrideWithValue(repo)],
    );
    addTearDown(container.dispose);

    expect((await container.read(provider.future)).items, [bob]);
    await container.read(provider.notifier).loadMore();
    expect(container.read(provider).requireValue.items, [bob, carol]);
    expect(cursors, [null, 'next']);

    final pending = container.read(provider.notifier).reverse(bob);
    expect(
      container.read(provider).requireValue.mutatingDids,
      {'did:plc:bob'},
    );
    unmute.complete(const ProfileRelationship());
    await pending;

    final state = container.read(provider).requireValue;
    expect(state.items, [carol]);
    expect(state.mutatingDids, isEmpty);
  });

  testWidgets('muted accounts empty state is localized', (tester) async {
    final repo = FakeProfileRepository(
      onListMutedProfiles: ({limit, cursor}) async =>
          const ProfileAccountPage(items: [], totalCount: 0),
    );
    await tester.pumpWidget(_app(repo, RelationshipListKind.muted));
    await tester.pumpAndSettle();

    expect(find.text('Muted accounts'), findsOneWidget);
    expect(find.text('You have not muted any accounts.'), findsOneWidget);
  });

  testWidgets('row-level unmute removes the account after success', (
    tester,
  ) async {
    var unmuteCalls = 0;
    final repo = FakeProfileRepository(
      onListMutedProfiles: ({limit, cursor}) async => ProfileAccountPage(
        totalCount: 1,
        items: [
          ProfileAccountSummary(
            did: 'did:plc:bob',
            handle: 'bob.craftsky.social',
            isCraftskyProfile: true,
            muted: true,
          ),
        ],
      ),
      onUnmute: (_) async {
        unmuteCalls++;
        return const ProfileRelationship();
      },
    );
    await tester.pumpWidget(_app(repo, RelationshipListKind.muted));
    await tester.pumpAndSettle();

    expect(find.text('@bob.craftsky.social'), findsOneWidget);
    await tester.tap(find.text('Unmute'));
    await tester.pumpAndSettle();

    expect(unmuteCalls, 1);
    expect(find.text('@bob.craftsky.social'), findsNothing);
    expect(find.text('You have not muted any accounts.'), findsOneWidget);
  });

  testWidgets('load error exposes a retry action', (tester) async {
    var calls = 0;
    final repo = FakeProfileRepository(
      onListBlockedProfiles: ({limit, cursor}) async {
        calls++;
        if (calls == 1) throw StateError('nope');
        return const ProfileAccountPage(items: [], totalCount: 0);
      },
    );
    await tester.pumpWidget(_app(repo, RelationshipListKind.blocked));
    await tester.pumpAndSettle();

    expect(find.text('Could not load blocked accounts.'), findsOneWidget);
    await tester.tap(find.text('Try again'));
    await tester.pumpAndSettle();
    expect(calls, 2);
    expect(find.text('You have not blocked any accounts.'), findsOneWidget);
  });

  testWidgets('row-level unblock confirms restoration consequences', (
    tester,
  ) async {
    var unblockCalls = 0;
    final repo = FakeProfileRepository(
      onListBlockedProfiles: ({limit, cursor}) async => ProfileAccountPage(
        totalCount: 1,
        items: [
          ProfileAccountSummary(
            did: 'did:plc:bob',
            handle: 'bob.craftsky.social',
            isCraftskyProfile: true,
            blocking: true,
          ),
        ],
      ),
      onUnblock: (_) async {
        unblockCalls++;
        return const ProfileRelationship();
      },
    );
    await tester.pumpWidget(_app(repo, RelationshipListKind.blocked));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Unblock'));
    await tester.pumpAndSettle();
    expect(find.text('Unblock this account?'), findsOneWidget);
    expect(
      find.text("You may see and interact with each other's content again."),
      findsOneWidget,
    );
    expect(unblockCalls, 0);

    await tester.tap(find.widgetWithText(TextButton, 'Unblock account'));
    await tester.pumpAndSettle();
    expect(unblockCalls, 1);
    expect(find.text('@bob.craftsky.social'), findsNothing);
  });
}
