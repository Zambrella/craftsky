import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/settings/pages/follow_list_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

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
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FollowListPage(kind: FollowListKind.following),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Following (0)'), findsOneWidget);
    expect(find.text('You are not following anyone'), findsOneWidget);
  });
}
