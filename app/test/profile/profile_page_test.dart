import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import 'fakes/fake_profile_repository.dart';

void main() {
  group('ProfilePage', () {
    testWidgets('signed-in self profile renders identity + edit actions', (
      tester,
    ) async {
      const profile = Profile(
        did: 'did:plc:test',
        handle: 'test.bsky.social',
        displayName: 'Test User',
        description: 'Sewist in Bristol',
        crafts: ['sewing', 'quilting'],
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            home: const ProfilePage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // 'Test User' appears in the (faded) collapsed app-bar title
      // as well as the identity block — both are in the tree at all
      // times, with opacity tied to scroll.
      expect(find.text('Test User'), findsWidgets);
      expect(find.text('Sewist in Bristol'), findsOneWidget);
      expect(find.text('Edit profile'), findsOneWidget);
      // Settings is icon-only in the action row plus the cog in the
      // collapsed-state trailing slot — assert by icon, not text.
      expect(find.byIcon(Icons.settings_outlined), findsWidgets);
    });

    testWidgets('visitor profile renders Follow + Share actions', (
      tester,
    ) async {
      const profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Follow'), findsOneWidget);
      // Share is icon-only in the action row plus the share icon in
      // the collapsed-state trailing slot.
      expect(find.byIcon(Icons.ios_share_outlined), findsWidgets);
      expect(find.text('Edit profile'), findsNothing);
    });
  });
}
