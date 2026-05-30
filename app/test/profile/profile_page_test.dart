import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/moderation_metadata.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/image_cache_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';
import 'fakes/fake_profile_repository.dart';

final _emptyPostRepository = FakePostRepository(
  onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
  onListCommentsByAuthor: (_, {cursor, limit}) async =>
      const PostPage(items: []),
);

void main() {
  group('ProfilePage', () {
    testWidgets('signed-in self profile renders identity + edit actions', (
      tester,
    ) async {
      final profile = Profile(
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
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
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
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        viewerIsFollowing: false,
        isCraftskyProfile: true,
        followingCount: 7,
        followerCount: 9,
        mutualFollowerCount: 12,
        postsLast7Days: 2,
        postCount: 5,
        projectCount: 0,
        createdAt: DateTime.now().subtract(const Duration(days: 30)),
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Follow'), findsOneWidget);
      expect(find.text('Unfollow'), findsNothing);
      // Share is icon-only in the action row plus the share icon in
      // the collapsed-state trailing slot.
      expect(find.byIcon(Icons.ios_share_outlined), findsWidgets);
      expect(find.text('Edit profile'), findsNothing);
      expect(find.text('following'), findsNothing);
      expect(find.text('followers'), findsNothing);
      await tester.ensureVisible(find.text('2 posts'));
      expect(find.text('2 posts'), findsOneWidget);
      expect(find.text('7 days'), findsOneWidget);
      expect(find.text('0'), findsOneWidget);
      expect(find.text('projects'), findsOneWidget);
      expect(find.text('12 mutual followers'), findsOneWidget);
      expect(find.text('Non Craftsky profile'), findsNothing);
    });

    testWidgets('warned profile shows generic warning copy', (tester) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        moderation: const ModerationMetadata(warningKind: 'profile'),
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text('This profile may not follow Craftsky community guidelines.'),
        findsOneWidget,
      );
      expect(find.textContaining('raw unsafe reason fixture'), findsNothing);
    });

    testWidgets('visitor profile report action submits through report sheet', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
      );
      String? submittedTarget;
      ReportSubmission? submitted;
      final messenger = RecordingMessenger();
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
        onReport: (handleOrDid, submission) async {
          submittedTarget = handleOrDid;
          submitted = submission;
          return const ReportResult(reportId: 'report-1', status: 'accepted');
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProfilePage(handle: 'alice.bsky.social'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('Report profile'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Spam'));
      await tester.pump();
      await tester.ensureVisible(find.widgetWithText(FilledButton, 'Submit'));
      await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
      await tester.pumpAndSettle();

      expect(submittedTarget, 'alice.bsky.social');
      expect(submitted?.reasonType, 'spam');
      expect(find.text('Report profile'), findsNothing);
      expect(
        messenger.calls,
        contains(('info', 'Thanks — your report was submitted.', null)),
      );
    });

    testWidgets('tapping mutual followers opens bottom sheet list', (
      tester,
    ) async {
      final cursors = <String?>[];
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'bob.bsky.social',
        displayName: 'Bob',
        crafts: [],
        viewerIsFollowing: false,
        isCraftskyProfile: true,
        mutualFollowerCount: 12,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
        onListMutualFollowers: (_, {cursor, limit}) async {
          cursors.add(cursor);
          if (cursor == 'next-mutuals') {
            return ProfileAccountPage(
              totalCount: 12,
              items: [
                ProfileAccountSummary(
                  did: 'did:plc:dana',
                  handle: 'dana.craftsky.social',
                  displayName: 'Dana',
                  isCraftskyProfile: true,
                ),
              ],
            );
          }
          return ProfileAccountPage(
            totalCount: 12,
            cursor: 'next-mutuals',
            items: [
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
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'bob.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('12 mutual followers'));
      await tester.pumpAndSettle();

      expect(find.text('Mutual followers'), findsOneWidget);
      expect(find.text('Carol'), findsOneWidget);
      expect(find.text('@carol.craftsky.social'), findsOneWidget);
      expect(cursors, [isNull]);
      expect(find.text('Load more'), findsOneWidget);

      await tester.tap(find.text('Load more'));
      await tester.pumpAndSettle();

      expect(cursors, [isNull, 'next-mutuals']);
      expect(find.text('Dana'), findsOneWidget);
      expect(find.text('@dana.craftsky.social'), findsOneWidget);
      expect(find.text('Load more'), findsNothing);
    });

    testWidgets('visitor profile renders Unfollow when already following', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        viewerIsFollowing: true,
        isCraftskyProfile: true,
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Unfollow'), findsOneWidget);
      expect(find.text('Follow'), findsNothing);
    });

    testWidgets('non-Craftsky profile shows marker and unknown counts', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'carol.bsky.social',
        displayName: 'Carol',
        crafts: [],
        viewerIsFollowing: false,
        isCraftskyProfile: false,
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'carol.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Non Craftsky profile'), findsOneWidget);
      expect(find.text('342'), findsNothing);
      expect(find.text('1200'), findsNothing);
      expect(find.textContaining('Joined'), findsNothing);
      expect(find.text('followers'), findsNothing);
      expect(find.text('following'), findsNothing);
    });

    testWidgets('tapping Follow updates profile from repository response', (
      tester,
    ) async {
      var followCalls = 0;
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        viewerIsFollowing: false,
        isCraftskyProfile: true,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
        onFollow: (_) async {
          followCalls++;
          return profile.copyWith(viewerIsFollowing: true);
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Follow'));
      await tester.pumpAndSettle();

      expect(followCalls, 1);
      expect(find.text('Unfollow'), findsOneWidget);
      expect(find.text('Follow'), findsNothing);
    });

    testWidgets('tapping Unfollow updates profile from repository response', (
      tester,
    ) async {
      var unfollowCalls = 0;
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        viewerIsFollowing: true,
        isCraftskyProfile: true,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
        onUnfollow: (_) async {
          unfollowCalls++;
          return profile.copyWith(viewerIsFollowing: false);
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Unfollow'));
      await tester.pumpAndSettle();

      expect(unfollowCalls, 1);
      expect(find.text('Follow'), findsOneWidget);
      expect(find.text('Unfollow'), findsNothing);
    });

    testWidgets('follow button is not re-entrant while request is in flight', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        viewerIsFollowing: false,
        isCraftskyProfile: true,
      );
      var followCalls = 0;
      final completer = Completer<Profile>();
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
        onFollow: (_) {
          followCalls++;
          return completer.future;
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'alice.bsky.social'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Follow'));
      await tester.pump();
      await tester.tap(find.text('Unfollow'));
      await tester.pump();

      expect(followCalls, 1);

      completer.complete(profile.copyWith(viewerIsFollowing: true));
      await tester.pumpAndSettle();
    });

    testWidgets('failed follow restores previous state and shows error', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
        viewerIsFollowing: false,
        isCraftskyProfile: true,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
        onFollow: (_) async => throw Exception('boom'),
      );
      final messenger = RecordingMessenger();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProfilePage(handle: 'alice.bsky.social'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Follow'));
      await tester.pumpAndSettle();

      expect(find.text('Follow'), findsOneWidget);
      expect(find.text('Unfollow'), findsNothing);
      expect(messenger.calls.length, 1);
      expect(messenger.calls.first.$1, 'error');
      expect(messenger.calls.first.$2, 'Could not update follow state.');
    });

    testWidgets('tapping Share dispatches a coming-soon info', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:other',
        handle: 'alice.bsky.social',
        displayName: 'Alice',
        crafts: [],
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);
      final messenger = RecordingMessenger();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
          ],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProfilePage(handle: 'alice.bsky.social'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.ios_share_outlined).first);
      expect(messenger.calls.length, 1);
      expect(messenger.calls.first.$1, 'info');
      expect(messenger.calls.first.$2, 'Share coming soon.');
    });

    testWidgets('tapping profile avatar opens the fullscreen image gallery', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:test',
        handle: 'test.bsky.social',
        displayName: 'Test User',
        avatar: 'https://example.test/avatar.jpg',
        crafts: [],
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);
      final fakeCache = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
            profileImageCacheManagerProvider.overrideWith((ref) => fakeCache),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('profile-avatar-viewer-target')));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(PostImageGallery), findsOneWidget);
      expect(find.text('Test User profile picture'), findsOneWidget);
      expect(find.byType(CloseButton), findsOneWidget);
    });

    testWidgets('tapping profile banner opens the fullscreen image gallery', (
      tester,
    ) async {
      final profile = Profile(
        did: 'did:plc:test',
        handle: 'test.bsky.social',
        displayName: 'Test User',
        banner: 'https://example.test/banner.jpg',
        crafts: [],
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);
      final fakeCache = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            profileRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_emptyPostRepository),
            profileImageCacheManagerProvider.overrideWith((ref) => fakeCache),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('profile-banner-viewer-target')));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(PostImageGallery), findsOneWidget);
      expect(find.text('Test User profile banner'), findsOneWidget);
      expect(find.byType(CloseButton), findsOneWidget);
    });
  });
}
