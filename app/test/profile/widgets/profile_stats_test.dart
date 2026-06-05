import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('formatJoinedAge', () {
    test('formats account age as a compact value', () {
      final joined = DateTime.utc(2025, 5, 27, 12);
      final now = DateTime.utc(2026, 5, 27, 12);

      expect(formatJoinedAge(joined, now: now), '1y');
    });
  });

  group('ProfileStats', () {
    testWidgets('renders summary stats and hides follower metrics', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
        crafts: [],
        createdAt: DateTime.now().subtract(const Duration(days: 370)),
        followerCount: 9,
        followingCount: 7,
        postsLast7Days: 2,
        postCount: 5,
        projectCount: 0,
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: ProfileStats(profile: profile)),
        ),
      );

      expect(find.text('1y'), findsOneWidget);
      expect(find.text('here'), findsOneWidget);
      expect(find.text('2 posts'), findsOneWidget);
      expect(find.text('7 days'), findsOneWidget);
      expect(find.text('5'), findsNothing);
      expect(find.text('posts'), findsNothing);
      expect(find.text('0'), findsOneWidget);
      expect(find.text('projects'), findsOneWidget);
      expect(find.text('followers'), findsNothing);
      expect(find.text('following'), findsNothing);
      expect(find.text('9'), findsNothing);
      expect(find.text('7'), findsNothing);
    });

    testWidgets('hides account age for non-Craftsky profiles', (tester) async {
      final profile = Profile(
        did: 'did:plc:carol',
        handle: 'carol.bsky.social',
        crafts: [],
        createdAt: DateTime.now().subtract(const Duration(days: 370)),
        isCraftskyProfile: false,
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: ProfileStats(profile: profile)),
        ),
      );

      expect(find.text('here'), findsNothing);
    });
  });
}
