import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/pages/post_interaction_accounts_page.dart';
import 'package:craftsky_app/feed/pages/post_quotes_page.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';

const _did = 'did:plc:alice';
const _rkey = 'root';

ActiveAccountInitialization _completedInitialization() =>
    ActiveAccountInitialization(
      lease: ActiveAccountLease(
        session: AccountSessionLease(
          account: AccountKey('did:plc:test'),
          sessionGeneration: 1,
        ),
        activationGeneration: 1,
      ),
      languagePreferences: const LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      ),
      onboardingComplete: true,
    );

Post _post() => Post(
  uri: 'at://$_did/social.craftsky.feed.post/$_rkey',
  cid: 'bafy-root',
  rkey: _rkey,
  text: 'Originating thread state',
  tags: const [],
  likeCount: 1,
  repostCount: 1,
  replyCount: 0,
  quoteCount: 1,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  createdAt: DateTime.utc(2026, 9, 6),
  indexedAt: DateTime.utc(2026, 9, 6),
  author: PostAuthor(did: _did, handle: 'alice.test'),
);

FakePostRepository _repository({void Function()? onThreadLoad}) =>
    FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        onThreadLoad?.call();
        return PostCommentSection(
          post: _post(),
          comments: const CommentPage(items: []),
          sort: CommentSort.oldest,
        );
      },
      onListLikes: (did, rkey, {cursor, limit}) async =>
          const ProfileAccountPage(items: [], totalCount: 0),
      onListReposts: (did, rkey, {cursor, limit}) async =>
          const ProfileAccountPage(items: [], totalCount: 0),
      onListQuotes: (did, rkey, {cursor, limit}) async =>
          const PostPage(items: []),
    );

FakePostRepository _repositoryWithResponse({void Function()? onThreadLoad}) {
  final root = _post();
  final response = Post(
    uri: 'at://did:plc:bob/social.craftsky.feed.post/comment',
    cid: 'bafy-comment',
    rkey: 'comment',
    text: 'Liked response',
    tags: const [],
    likeCount: 2,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    createdAt: DateTime.utc(2026, 9, 6),
    indexedAt: DateTime.utc(2026, 9, 6),
    author: PostAuthor(did: 'did:plc:bob', handle: 'bob.test'),
    reply: PostReply(
      root: PostRef(uri: root.uri, cid: root.cid),
      parent: PostRef(uri: root.uri, cid: root.cid),
    ),
  );
  return FakePostRepository(
    onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
      onThreadLoad?.call();
      return PostCommentSection(
        post: root,
        comments: CommentPage(
          items: [
            CommentItem(
              post: response,
              placement: CommentPlacement.normal,
              replies: const ReplyPage(loaded: false, items: []),
            ),
          ],
        ),
        sort: CommentSort.oldest,
      );
    },
    onListLikes: (did, rkey, {cursor, limit}) async =>
        const ProfileAccountPage(items: [], totalCount: 0),
    onListReposts: (did, rkey, {cursor, limit}) async =>
        const ProfileAccountPage(items: [], totalCount: 0),
    onListQuotes: (did, rkey, {cursor, limit}) async =>
        const PostPage(items: []),
  );
}

ProviderContainer _container({
  required bool signedIn,
  FakePostRepository? repo,
}) {
  return ProviderContainer.test(
    overrides: [
      authSessionProvider.overrideWith(
        signedIn ? SignedInAuthSession.new : SignedOutAuthSession.new,
      ),
      activeAccountInitializationProvider.overrideWith(
        (ref) => _completedInitialization(),
      ),
      secureSessionRegistryStorageProvider.overrideWithValue(
        _MemoryRegistryStorage(),
      ),
      postRepositoryProvider.overrideWithValue(repo ?? _repository()),
    ],
    retry: (_, _) => null,
  );
}

final class _MemoryRegistryStorage implements SessionRegistryStorage {
  SessionRegistry registry = SessionRegistry.empty().upsertAndActivate(
    token: 'token-test',
    did: 'did:plc:test',
    handle: 'test.craftsky.social',
  );

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry value) async => registry = value;
}

Future<void> _pumpRouter(
  WidgetTester tester,
  ProviderContainer container, {
  required String initialLocation,
  required Size size,
}) async {
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;

  final subscription = container.listen(goRouterProvider, (_, _) {});
  addTearDown(subscription.close);
  final router = subscription.read()..go(initialLocation);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        routerConfig: router,
        builder: (context, child) =>
            FormFactorWidget(child: child ?? const SizedBox.shrink()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void _expectDestination(String suffix, WidgetTester tester) {
  switch (suffix) {
    case 'likes':
      final page = tester.widget<PostInteractionAccountsPage>(
        find.byType(PostInteractionAccountsPage),
      );
      expect(page.did, Did.parse(_did));
      expect(page.rkey, RecordKey.parse(_rkey));
      expect(page.kind, PostInteractionAccountKind.likes);
    case 'reposts':
      final page = tester.widget<PostInteractionAccountsPage>(
        find.byType(PostInteractionAccountsPage),
      );
      expect(page.did, Did.parse(_did));
      expect(page.rkey, RecordKey.parse(_rkey));
      expect(page.kind, PostInteractionAccountKind.reposts);
    case 'quotes':
      final page = tester.widget<PostQuotesPage>(find.byType(PostQuotesPage));
      expect(page.did, Did.parse(_did));
      expect(page.rkey, RecordKey.parse(_rkey));
  }
}

void main() {
  group('AT-009 interaction detail routes', () {
    for (final suffix in ['likes', 'reposts', 'quotes']) {
      testWidgets('authenticated direct $suffix parses DID and rkey', (
        tester,
      ) async {
        await _pumpRouter(
          tester,
          _container(signedIn: true),
          initialLocation: '/posts/$_did/$_rkey/$suffix',
          size: const Size(390, 844),
        );

        _expectDestination(suffix, tester);
        expect(find.byType(NavigationRail), findsNothing);
        expect(find.byType(NavigationBar), findsNothing);
      });

      testWidgets('signed-out direct $suffix redirects to welcome', (
        tester,
      ) async {
        await _pumpRouter(
          tester,
          _container(signedIn: false),
          initialLocation: '/posts/$_did/$_rkey/$suffix',
          size: const Size(390, 844),
        );

        expect(find.byType(WelcomePage), findsOneWidget);
        expect(find.byType(PostInteractionAccountsPage), findsNothing);
        expect(find.byType(PostQuotesPage), findsNothing);
      });
    }

    testWidgets('large direct route retains authenticated shell rail', (
      tester,
    ) async {
      await _pumpRouter(
        tester,
        _container(signedIn: true),
        initialLocation: '/posts/$_did/$_rkey/likes',
        size: const Size(1200, 800),
      );

      _expectDestination('likes', tester);
      expect(find.byType(NavigationRail), findsOneWidget);
      expect(find.byKey(const Key('large-shell-content')), findsOneWidget);
    });

    testWidgets('pushed route pops to existing originating thread state', (
      tester,
    ) async {
      var threadLoads = 0;
      final container = _container(
        signedIn: true,
        repo: _repository(onThreadLoad: () => threadLoads++),
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '/posts/$_did/$_rkey',
        size: const Size(390, 844),
      );
      expect(find.text('Originating thread state'), findsOneWidget);
      expect(threadLoads, 1);

      final context = tester.element(find.byType(PostThreadPage));
      final routeResult = const PostLikesRoute(
        did: _did,
        rkey: _rkey,
      ).push<void>(context);
      await tester.pumpAndSettle();
      expect(find.byType(PostInteractionAccountsPage), findsOneWidget);

      container.read(goRouterProvider).pop();
      await tester.pumpAndSettle();
      await routeResult;

      expect(find.byType(PostThreadPage), findsOneWidget);
      expect(find.text('Originating thread state'), findsOneWidget);
      expect(threadLoads, 1);
    });

    for (final size in [const Size(390, 844), const Size(1200, 800)]) {
      for (final entry in ['root summary', 'response menu']) {
        testWidgets(
          'REG-006 $entry preserves ${size.width == 390 ? 'compact' : 'large'} '
          'thread shell and back state',
          (tester) async {
            var threadLoads = 0;
            final container = _container(
              signedIn: true,
              repo: _repositoryWithResponse(
                onThreadLoad: () => threadLoads++,
              ),
            );
            await _pumpRouter(
              tester,
              container,
              initialLocation: '/posts/$_did/$_rkey',
              size: size,
            );

            if (entry == 'root summary') {
              await tester.tap(find.text('1 Like'));
            } else {
              final responseCard = find.byWidgetPredicate(
                (widget) =>
                    widget is PostCard &&
                    widget.post.rkey.toString() == 'comment',
              );
              await tester.ensureVisible(responseCard);
              await tester.tap(
                find.descendant(
                  of: responseCard,
                  matching: find.byIcon(CraftskyIconsBold.more),
                ),
              );
              await tester.pumpAndSettle();
              await tester.tap(find.text('View likes'));
            }
            await tester.pumpAndSettle();

            expect(find.byType(PostInteractionAccountsPage), findsOneWidget);
            expect(
              find.byType(NavigationRail),
              size.width == 390 ? findsNothing : findsOneWidget,
            );

            container.read(goRouterProvider).pop();
            await tester.pumpAndSettle();

            expect(find.byType(PostThreadPage), findsOneWidget);
            expect(find.text('Originating thread state'), findsOneWidget);
            expect(find.text('Liked response'), findsOneWidget);
            expect(threadLoads, 1);
            expect(
              find.byType(NavigationRail),
              size.width == 390 ? findsNothing : findsOneWidget,
            );
          },
        );
      }
    }
  });
}
