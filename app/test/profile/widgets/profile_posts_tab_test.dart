import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/external_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_posts_tab.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

final class _ProfilePinRegistryStorage implements SessionRegistryStorage {
  _ProfilePinRegistryStorage()
    : value = SessionRegistry.empty().upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
      );

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

Post _post(String rkey, {PostExternal? external}) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: 'post $rkey',
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    external: external,
    createdAt: DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
    ),
  );
}

Future<void> _pump(
  WidgetTester tester, {
  required FakePostRepository repo,
  required bool isOwnProfile,
  RecordingMessenger? messenger,
  List<dynamic> overrides = const [],
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: List.from([
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postRepositoryProvider.overrideWithValue(repo),
        ...overrides,
      ]),
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: CustomScrollView(
              slivers: [
                ProfilePostsTab(
                  handle: 'alice.craftsky.social',
                  isOwnProfile: isOwnProfile,
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  group('ProfilePostsTab', () {
    testWidgets('IT-014 renders a full external card on profile posts', (
      tester,
    ) async {
      Uri? launched;
      final post = _post(
        'external',
        external: const PostExternal(
          uri: 'https://example.com/profile-pattern?token=final#section',
          title: 'Profile pattern',
          description: 'Profile description',
        ),
      );
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(items: [post]),
      );

      await _pump(
        tester,
        repo: repo,
        isOwnProfile: false,
        overrides: [
          externalCardLauncherProvider.overrideWithValue((uri) async {
            launched = uri;
            return true;
          }),
        ],
      );
      await tester.pumpAndSettle();

      expect(find.byType(ExternalCard), findsOneWidget);
      expect(find.text('Profile pattern'), findsOneWidget);
      expect(find.text('Profile description'), findsOneWidget);
      expect(find.text('example.com'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.tap(find.byType(ExternalCard));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();
      expect(
        launched.toString(),
        'https://example.com/profile-pattern?token=final#section',
      );
    });

    testWidgets('AT-001 pins an own standard post from the profile menu', (
      tester,
    ) async {
      final pinnedTargets = <String>[];
      final messenger = RecordingMessenger();
      final post = _post('standard');
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(items: [post]),
        onProfilePins: () async => const ProfilePinState(),
        onPin: (did, rkey) async {
          pinnedTargets.add('$did/$rkey');
          return ProfilePinState(standardPostUri: post.uri.value);
        },
      );

      await _pump(
        tester,
        repo: repo,
        isOwnProfile: true,
        messenger: messenger,
        overrides: [
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _ProfilePinRegistryStorage(),
          ),
        ],
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.more));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Pin post'));
      await tester.pumpAndSettle();

      expect(pinnedTargets, ['did:plc:alice/standard']);
      expect(messenger.calls, [('info', 'Post pinned', null)]);
    });

    testWidgets('UIP-001 uses the themed message for unpin confirmation', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      final post = _post('standard');
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(items: [post]),
        onProfilePins: () async => ProfilePinState(
          standardPostUri: post.uri.value,
        ),
        onUnpin: (_, _) async => const ProfilePinState(),
      );

      await _pump(
        tester,
        repo: repo,
        isOwnProfile: true,
        messenger: messenger,
        overrides: [
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _ProfilePinRegistryStorage(),
          ),
        ],
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.more));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Unpin post'));
      await tester.pumpAndSettle();

      expect(messenger.calls, [('info', 'Post unpinned', null)]);
    });

    testWidgets('UIP-001 uses the themed error message for pin failure', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      final post = _post('standard');
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(items: [post]),
        onProfilePins: () async => const ProfilePinState(),
        onPin: (_, _) async => throw StateError('network failed'),
      );

      await _pump(
        tester,
        repo: repo,
        isOwnProfile: true,
        messenger: messenger,
        overrides: [
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _ProfilePinRegistryStorage(),
          ),
        ],
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.more));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Pin post'));
      await tester.pumpAndSettle();

      expect(
        messenger.calls,
        [('error', 'Couldn’t pin post. Try again.', null)],
      );
    });

    testWidgets('AT-005 annotates only page-one pinned metadata', (
      tester,
    ) async {
      final pinned = _post('pinned');
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(
          items: [pinned, _post('ordinary')],
          pinnedPostUri: pinned.uri.value,
        ),
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();

      expect(find.text('Pinned post'), findsOneWidget);
      expect(find.byIcon(CraftskyIcons.pin), findsOneWidget);
    });

    testWidgets('renders posts from userPostsProvider', (tester) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async =>
            PostPage(items: [_post('a'), _post('b')]),
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();

      expect(find.text('post a'), findsOneWidget);
      expect(find.text('post b'), findsOneWidget);
      expect(find.text('New post'), findsNothing);
    });

    testWidgets('does not show a top-level composer on own profile', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
      );

      await _pump(tester, repo: repo, isOwnProfile: true);
      await tester.pumpAndSettle();

      expect(find.text('New post'), findsNothing);
      expect(find.text('No posts yet.'), findsOneWidget);
    });

    testWidgets('scrolling near the end appends the next page', (tester) async {
      final calls = <({String? cursor, int? limit})>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async {
          calls.add((cursor: cursor, limit: limit));
          if (calls.length == 1) {
            return PostPage(
              items: [for (var i = 0; i < 10; i++) _post('a$i')],
              cursor: 'c1',
            );
          }
          expect(cursor, 'c1');
          return PostPage(items: [_post('b')]);
        },
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('post a9'),
        500,
        scrollable: find.byType(Scrollable),
      );
      await tester.pumpAndSettle();

      expect(calls, [
        (cursor: null, limit: 10),
        (cursor: 'c1', limit: 10),
      ]);
      expect(find.text('post a9'), findsOneWidget);
      expect(find.text('post b'), findsOneWidget);
      expect(find.text('Load more posts'), findsNothing);
    });

    testWidgets('wires reply composer, like, and repost actions', (
      tester,
    ) async {
      final calls = <String>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async =>
            PostPage(items: [_post('a')]),
        onLike: (did, rkey) async {
          calls.add('like:$did/$rkey');
          final post = _post(rkey);
          return InteractionWriteResponse(
            uri: 'at://did:plc:viewer/social.craftsky.feed.like/like1',
            cid: 'bafy_like',
            rkey: 'like1',
            subject: PostRef(uri: post.uri, cid: post.cid),
            createdAt: DateTime.now(),
          );
        },
        onRepost: (did, rkey) async {
          calls.add('repost:$did/$rkey');
          final post = _post(rkey);
          return InteractionWriteResponse(
            uri: 'at://did:plc:viewer/social.craftsky.feed.repost/repost1',
            cid: 'bafy_repost',
            rkey: 'repost1',
            subject: PostRef(uri: post.uri, cid: post.cid),
            createdAt: DateTime.now(),
          );
        },
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.comment));
      await tester.pumpAndSettle();
      expect(find.text('Comment'), findsWidgets);
      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.like));
      await tester.pump();
      await tester.tap(find.byIcon(CraftskyIconsBold.repost));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Repost'));
      await tester.pumpAndSettle();

      expect(calls, [
        'like:did:plc:alice/a',
        'repost:did:plc:alice/a',
      ]);
    });

    testWidgets('reply create opens thread focused on the new comment', (
      tester,
    ) async {
      GoRouterState? threadState;
      final root = _post('root');
      final created = _post('created');
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(items: [root]),
        onCreate: ({required text, reply, images}) async => created,
      );
      final router = GoRouter(
        initialLocation: '/',
        routes: [
          GoRoute(
            path: '/',
            builder: (context, state) => const Scaffold(
              body: CustomScrollView(
                slivers: [
                  ProfilePostsTab(
                    handle: 'alice.craftsky.social',
                    isOwnProfile: false,
                  ),
                ],
              ),
            ),
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (context, state) {
              threadState = state;
              return const Scaffold(body: Text('Thread route'));
            },
          ),
        ],
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            postRepositoryProvider.overrideWithValue(repo),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              routerConfig: router,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(CraftskyIconsBold.comment));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'new comment');
      await tester.pump();
      await tester.tap(find.widgetWithText(ChunkyButton, 'Comment'));
      await tester.pumpAndSettle();

      expect(find.text('Thread route'), findsOneWidget);
      expect(threadState?.pathParameters['did'], root.author.did);
      expect(threadState?.pathParameters['rkey'], root.rkey);
      expect(threadState?.uri.queryParameters['focus'], created.uri);
      expect(threadState?.extra, isA<Post>());
      expect((threadState!.extra! as Post).uri, created.uri);
      expect((threadState!.extra! as Post).reply?.root.uri, root.uri);
    });

    testWidgets('delete confirmation removes a post', (tester) async {
      final messenger = RecordingMessenger();
      final deleted = <String>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async =>
            PostPage(items: [_post('a'), _post('b')]),
        onDelete: (_, rkey) async => deleted.add(rkey),
      );

      await _pump(tester, repo: repo, isOwnProfile: true, messenger: messenger);
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.more).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete post').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(deleted, ['a']);
      expect(find.text('post a'), findsNothing);
      expect(find.text('post b'), findsOneWidget);
      expect(messenger.calls.last.$2, 'Post deleted.');
    });
  });
}
