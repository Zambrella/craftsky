import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
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
      // Share is icon-only in the action row plus the share icon in
      // the collapsed-state trailing slot.
      expect(find.byIcon(Icons.ios_share_outlined), findsWidgets);
      expect(find.text('Edit profile'), findsNothing);
    });

    testWidgets('tapping Follow dispatches a coming-soon info', (
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

      await tester.tap(find.text('Follow'));
      expect(messenger.calls.length, 1);
      expect(messenger.calls.first.$1, 'info');
      expect(messenger.calls.first.$2, 'Follow coming soon.');
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
