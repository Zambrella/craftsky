import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/handoff_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _HandoffApi implements HandoffApiClient {
  @override
  Future<WhoAmI> whoami() async =>
      WhoAmI(did: 'did:plc:bob', handle: 'bob.test');
}

final class _ZeroCountRepository implements NotificationNewnessRepository {
  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}

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
  viewerHasSaved: false,
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
  bool settle = true,
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
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
  }
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
                  const TimelinePage(items: []),
            ),
          ),
        ],
      );
      await _pumpRouter(tester, container);
      expect(find.byType(FeedPage), findsOneWidget);
    });

    testWidgets('SignedIn + onboarded can open Add account', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith2(
            (_) => CompletedOnboardingStatus(),
          ),
        ],
      );

      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.addAccount,
      );

      expect(find.byType(SignInPage), findsOneWidget);
      expect(find.text('Add account'), findsOneWidget);
      expect(
        find.text(
          'Sign in to another account. '
          'Your current account stays signed in.',
        ),
        findsOneWidget,
      );
    });

    testWidgets(
      'SignedIn + onboarded keeps Add account callback on AuthCompletePage',
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
                    const TimelinePage(items: []),
              ),
            ),
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

    testWidgets(
      'Add account callback retains A, activates B, and returns Home',
      (
        tester,
      ) async {
        final storage = _RegistryStorage(
          SessionRegistry.empty().upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          ),
        );
        final container = ProviderContainer.test(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(storage),
            sessionValidationLauncherProvider.overrideWithValue((_) async {}),
            handoffApiClientProvider.overrideWith((ref, key) => _HandoffApi()),
            deviceIdProvider.overrideWith((ref) async => 'test-device'),
            accountStateInvalidatorProvider.overrideWithValue(() async {}),
            accountNotificationNewnessRepositoryProvider.overrideWith(
              (ref, account) async => _ZeroCountRepository(),
            ),
            onboardingStatusProvider.overrideWith2(
              (_) => CompletedOnboardingStatus(),
            ),
            postRepositoryProvider.overrideWithValue(
              FakePostRepository(
                onListTimeline: ({cursor, limit}) async =>
                    const TimelinePage(items: []),
              ),
            ),
            profileRepositoryProvider.overrideWithValue(
              FakeProfileRepository(
                onFetch: (id) async => Profile(
                  did: 'did:plc:bob',
                  handle: 'bob.test',
                  crafts: const [],
                ),
              ),
            ),
          ],
        );
        await container.read(authSessionProvider.future);
        container.read(pendingAuthProvider.notifier).start('bob.test');

        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=token-b',
          settle: false,
        );
        for (var index = 0; index < 10; index++) {
          await tester.pump(const Duration(milliseconds: 10));
        }

        final registry = container.read(sessionRegistryProvider).requireValue;
        expect(registry.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
        expect(registry.activeDid?.value, 'did:plc:bob');
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
      'SignedIn + !onboarded keeps callback on AuthCompletePage',
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
        expect(find.byType(AuthCompletePage), findsOneWidget);
      },
    );
  });
}
