import 'dart:async';

import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/pending_handoff.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/handoff_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

ActiveAccountInitialization _initialization(bool completed) =>
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
      onboardingComplete: completed,
    );

SessionRegistry _activeTestRegistry() =>
    SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:test',
      handle: 'test.test',
    );

final class _LoadingOnboardingFlow extends OnboardingFlow {
  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) =>
      Completer<OnboardingFlowState>().future;
}

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
  Future<PendingHandoff> exchange({required String code}) async =>
      PendingHandoff(
        token: 'token-b',
        did: 'did:plc:bob',
        handle: 'bob.test',
        receiptId: '00000000-0000-4000-8000-000000000821',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      );

  @override
  Future<void> confirm({
    required String token,
    required String receiptId,
  }) async {}
}

final class _ZeroCountRepository implements NotificationNewnessRepository {
  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}

final class _CompletionAuthController extends AuthController {
  _CompletionAuthController({
    required this.onComplete,
    required this.onStartRegistration,
  });

  final Future<void> Function(String code) onComplete;
  final Future<void> Function() onStartRegistration;

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> completeFromDeepLink(String code) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() => onComplete(code));
  }

  @override
  Future<void> startRegistration() async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(onStartRegistration);
  }
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
  Size? size,
}) async {
  if (size case final value?) {
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = value;
  }
  // Drive the router to a specific initial location before pumping
  // the app, so deep-link-style tests can start on /auth/complete.
  final routerSubscription = container.listen(goRouterProvider, (_, _) {});
  addTearDown(routerSubscription.close);
  final router = routerSubscription.read()..go(initialLocation);
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
      'SignedOut + legacy token callback stays on page but fails closed',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedOutAuthSession.new),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation:
              '${RouteLocations.authComplete}?token=not-a-credential',
        );
        expect(find.byType(AuthCompletePage), findsOneWidget);
        expect(find.textContaining('sign-in link expired'), findsOneWidget);
      },
    );

    const registrationFailureMessages = {
      RegistrationFailure.canceled: 'Account creation was canceled.',
      RegistrationFailure.providerUnavailable:
          'Bluesky is temporarily unavailable. Please try again.',
      RegistrationFailure.registrationIncomplete:
          "We couldn't verify or complete account creation.",
    };

    test('AT-006 exposes exactly three bounded registration outcomes', () {
      expect(RegistrationFailure.values, hasLength(3));
      expect(registrationFailureMessages.keys, RegistrationFailure.values);
    });

    for (final MapEntry(key: failure, value: message)
        in registrationFailureMessages.entries) {
      testWidgets('AT-006 routes ${failure.name} to its safe outcome', (
        tester,
      ) async {
        final exchangedCodes = <String>[];
        var registrationStarts = 0;
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedOutAuthSession.new),
            authControllerProvider.overrideWith(
              () => _CompletionAuthController(
                onComplete: (code) async => exchangedCodes.add(code),
                onStartRegistration: () async => registrationStarts++,
              ),
            ),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: AuthCompleteRoute(
            code: 'must-not-exchange',
            error: failure.name,
          ).location,
        );

        expect(find.byType(AuthCompletePage), findsOneWidget);
        expect(find.text(message), findsOneWidget);
        expect(find.textContaining('issuer'), findsNothing);
        expect(find.textContaining('DID'), findsNothing);
        expect(find.textContaining('token'), findsNothing);
        expect(find.textContaining('lifecycle'), findsNothing);
        expect(exchangedCodes, isEmpty);
        expect(registrationStarts, 0);

        await tester.tap(find.widgetWithText(TextButton, 'Retry'));
        await tester.pumpAndSettle();

        expect(registrationStarts, 1);
        expect(exchangedCodes, isEmpty);
        expect(find.byType(AuthCompletePage), findsOneWidget);
        expect(find.byType(FeedPage), findsNothing);
      });
    }

    testWidgets(
      'AT-007 unknown Flutter callback never exchanges or navigates to Feed',
      (tester) async {
        final exchangedCodes = <String>[];
        var registrationStarts = 0;
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedOutAuthSession.new),
            authControllerProvider.overrideWith(
              () => _CompletionAuthController(
                onComplete: (code) async => exchangedCodes.add(code),
                onStartRegistration: () async => registrationStarts++,
              ),
            ),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: const AuthCompleteRoute(
            code: 'must-not-exchange',
            error: 'issuer-did-token-lifecycle-detail',
          ).location,
        );

        expect(find.byType(AuthCompletePage), findsOneWidget);
        expect(
          find.text("Couldn't complete sign-in. Please sign in again."),
          findsOneWidget,
        );
        expect(find.textContaining('issuer-did-token-lifecycle'), findsNothing);
        expect(find.text('Retry'), findsNothing);
        expect(exchangedCodes, isEmpty);
        expect(registrationStarts, 0);
        expect(find.byType(FeedPage), findsNothing);
      },
    );

    testWidgets('SignedIn + not onboarded → OnboardingPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(_activeTestRegistry()),
          ),
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(false),
          ),
          onboardingFlowProvider.overrideWith2(
            (_) => _LoadingOnboardingFlow(),
          ),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.feed,
        settle: false,
      );
      expect(find.byType(OnboardingPage), findsOneWidget);
    });

    testWidgets('AT-015 stale initialization cannot redirect active lease', (
      tester,
    ) async {
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final aliceSession = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final bobSession = registry.leaseFor(AccountKey('did:plc:bob'))!;
      registry = registry.activate(aliceSession);
      final aliceLease = registry.activeLease!;
      registry = registry.activate(bobSession);
      final stale = ActiveAccountInitialization(
        lease: aliceLease,
        languagePreferences: const LanguagePreferences(
          primaryLanguage: 'en',
          contentLanguages: ['en'],
        ),
        onboardingComplete: false,
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:bob'),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeAccountInitializationProvider.overrideWith((ref) => stale),
          onboardingFlowProvider.overrideWith2(
            (_) => _LoadingOnboardingFlow(),
          ),
        ],
      );

      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.addAccount,
        settle: false,
      );

      expect(find.byType(SignInPage), findsOneWidget);
      expect(find.byType(OnboardingPage), findsNothing);
    });

    testWidgets('SignedIn + onboarded + /welcome → FeedPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(_activeTestRegistry()),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
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
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
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
            activeAccountInitializationProvider.overrideWith(
              (ref) => _initialization(true),
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
          initialLocation: RouteLocations.authComplete,
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
            handoffApiClientProvider.overrideWithValue(_HandoffApi()),
            deviceIdProvider.overrideWith((ref) async => 'test-device'),
            accountStateInvalidatorProvider.overrideWithValue(() async {}),
            accountNotificationNewnessRepositoryProvider.overrideWith(
              (ref, account) async => _ZeroCountRepository(),
            ),
            activeAccountInitializationProvider.overrideWith(
              (ref) => _initialization(true),
            ),
            postRepositoryProvider.overrideWithValue(
              FakePostRepository(
                onListTimeline: ({cursor, limit}) async =>
                    const TimelinePage(items: []),
              ),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
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
        container.read(pendingAuthProvider.notifier).startSignIn('bob.test');

        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?code=browser-code',
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
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
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

    testWidgets('large post details keep the navigation rail visible', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(did, rkey),
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '/posts/did:plc:alice/root',
        size: const Size(1600, 800),
      );

      expect(find.byType(PostThreadPage), findsOneWidget);
      final rail = tester.widget<NavigationRail>(find.byType(NavigationRail));
      expect(rail.selectedIndex, 0);
      expect(
        tester.getRect(find.byKey(const Key('large-shell-content'))).width,
        800,
      );
    });

    testWidgets('pushing post details does not cover the large rail', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            const TimelinePage(items: []),
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(did, rkey),
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
        retry: (_, _) => null,
      );
      final routerSubscription = container.listen(
        goRouterProvider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(routerSubscription.close);

      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.feed,
        size: const Size(1200, 800),
      );

      final router = routerSubscription.read();
      router.push<void>('/posts/did:plc:alice/root').ignore();
      await tester.pumpAndSettle();

      expect(router.state.matchedLocation, '/posts/did:plc:alice/root');
      expect(router.state.error, isNull);
      expect(find.byType(PostThreadPage), findsOneWidget);
      expect(find.byType(NavigationRail), findsOneWidget);
    });

    testWidgets('compact post details remain full-screen', (tester) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(did, rkey),
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
      );

      await _pumpRouter(
        tester,
        container,
        initialLocation: '/posts/did:plc:alice/root',
        size: const Size(500, 800),
      );

      expect(find.byType(PostThreadPage), findsOneWidget);
      expect(find.byType(NavigationRail), findsNothing);
      expect(find.byType(NavigationBar), findsNothing);
    });

    testWidgets('large other-user profiles keep the navigation rail visible', (
      tester,
    ) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
          ),
          profileRepositoryProvider.overrideWithValue(
            FakeProfileRepository(
              onFetch: (_) async => Profile(
                did: 'did:plc:alice',
                handle: 'alice.test',
                crafts: [],
              ),
            ),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListByAuthor: (_, {cursor, limit}) async =>
                  const PostPage(items: []),
            ),
          ),
        ],
        retry: (_, _) => null,
      );

      await _pumpRouter(
        tester,
        container,
        initialLocation: '/profile/alice.test',
        size: const Size(1200, 800),
      );

      final rail = tester.widget<NavigationRail>(find.byType(NavigationRail));
      expect(rail.selectedIndex, 4);
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
          activeAccountInitializationProvider.overrideWith(
            (ref) => _initialization(true),
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
            activeAccountInitializationProvider.overrideWith(
              (ref) => _initialization(false),
            ),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: RouteLocations.authComplete,
        );
        expect(find.byType(AuthCompletePage), findsOneWidget);
      },
    );
  });
}
