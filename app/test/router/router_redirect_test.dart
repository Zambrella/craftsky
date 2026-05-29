import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';

Post _post(String did, String rkey) => Post(
  uri: 'at://$did/social.craftsky.feed.post/$rkey',
  cid: 'bafy_$rkey',
  rkey: rkey,
  text: '$did/$rkey',
  tags: const [],
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  createdAt: DateTime(2026, 5, 4, 18, 23, 45),
  indexedAt: DateTime(2026, 5, 4, 18, 23, 47),
  author: PostAuthor(did: did, handle: 'alice.craftsky.social'),
);

PostCommentSection _section(String did, String rkey) => PostCommentSection(
  post: _post(did, rkey),
  comments: const CommentPage(items: []),
  sort: CommentSort.oldest,
);

Future<void> _pumpRouter(
  WidgetTester tester,
  ProviderContainer container, {
  String initialLocation = RouteLocations.welcome,
}) async {
  // Drive the router to a specific initial location before pumping
  // the app, so deep-link-style tests can start on /auth/complete.
  final router = container.read(goRouterProvider)..go(initialLocation);
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

void main() {
  group('router redirect', () {
    testWidgets('SignedOut + /feed → WelcomePage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.feed,
      );
      expect(find.byType(WelcomePage), findsOneWidget);
    });

    testWidgets(
      'SignedOut + /auth/complete stays on AuthCompletePage',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedOutAuthSession.new),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=t',
        );
        expect(find.byType(AuthCompletePage), findsOneWidget);
      },
    );

    testWidgets('SignedIn + not onboarded → OnboardingPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith2(
            (_) => PendingOnboardingStatus(),
          ),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.feed,
      );
      expect(find.byType(OnboardingPage), findsOneWidget);
    });

    testWidgets('SignedIn + onboarded + /welcome → FeedPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith2(
            (_) => CompletedOnboardingStatus(),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async =>
                  const PostPage(items: []),
            ),
          ),
        ],
      );
      await _pumpRouter(tester, container);
      expect(find.byType(FeedPage), findsOneWidget);
    });

    testWidgets(
      'SignedIn + onboarded + /auth/complete → FeedPage',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            onboardingStatusProvider.overrideWith2(
              (_) => CompletedOnboardingStatus(),
            ),
            postRepositoryProvider.overrideWithValue(
              FakePostRepository(
                onListTimeline: ({cursor, limit}) async =>
                    const PostPage(items: []),
              ),
            ),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=t',
        );
        expect(find.byType(FeedPage), findsOneWidget);
      },
    );

    testWidgets('SignedIn + onboarded + /posts/:did/:rkey → PostThreadPage', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(did, rkey),
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith2(
            (_) => CompletedOnboardingStatus(),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '/posts/did:plc:alice/root',
      );

      expect(find.byType(PostThreadPage), findsOneWidget);
      expect(find.text('did:plc:alice/root'), findsOneWidget);
    });

    testWidgets('post route decodes focus query parameter', (tester) async {
      const focus = 'at://did:plc:bob/social.craftsky.feed.post/reply1';
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(did, rkey),
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith2(
            (_) => CompletedOnboardingStatus(),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation:
            '/posts/did:plc:alice/root?focus=${Uri.encodeQueryComponent(focus)}',
      );

      final page = tester.widget<PostThreadPage>(find.byType(PostThreadPage));
      expect(page.focus, focus);
    });

    testWidgets('SignedOut + /posts/:did/:rkey → WelcomePage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [authSessionProvider.overrideWith(SignedOutAuthSession.new)],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '/posts/did:plc:alice/root',
      );

      expect(find.byType(WelcomePage), findsOneWidget);
    });

    testWidgets(
      'SignedIn + !onboarded + /auth/complete → OnboardingPage',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            onboardingStatusProvider.overrideWith2(
              (_) => PendingOnboardingStatus(),
            ),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=t',
        );
        expect(find.byType(OnboardingPage), findsOneWidget);
      },
    );
  });
}
