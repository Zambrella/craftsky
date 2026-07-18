import 'dart:async';
import 'dart:ui' show SemanticsAction, Tristate;

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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

final class _ZeroCountRepository implements NotificationNewnessRepository {
  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}

final class _CountingCountRepository implements NotificationNewnessRepository {
  int countCalls = 0;

  @override
  Future<int> count() async {
    countCalls++;
    return 0;
  }

  @override
  Future<void> markSeen() async {}
}

Future<ProviderContainer> _pumpShell(
  WidgetTester tester, {
  required SessionRegistry registry,
  required Size size,
  Profile? activeProfile,
  ProfileRepository? profileRepository,
  NotificationNewnessRepository Function(AccountKey account)? countRepository,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.resetDevicePixelRatio);
  addTearDown(tester.view.resetPhysicalSize);
  final initialActive = registry.activeDid == null
      ? null
      : registry.sessions[registry.activeDid];
  final container = ProviderContainer.test(
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(
        _RegistryStorage(registry),
      ),
      sessionValidationLauncherProvider.overrideWithValue((_) async {}),
      onboardingStatusProvider.overrideWith2(
        (_) => CompletedOnboardingStatus(),
      ),
      accountNotificationNewnessRepositoryProvider.overrideWith(
        (ref, account) async =>
            countRepository?.call(account) ?? _ZeroCountRepository(),
      ),
      postRepositoryProvider.overrideWithValue(
        FakePostRepository(
          onListTimeline: ({cursor, limit}) async =>
              const TimelinePage(items: []),
        ),
      ),
      profileRepositoryProvider.overrideWithValue(
        profileRepository ??
            FakeProfileRepository(
              onFetch: (id) async =>
                  activeProfile ??
                  Profile(
                    did: initialActive?.did.value ?? 'did:plc:alice',
                    handle: initialActive?.handle.value ?? 'alice.test',
                    displayName: initialActive?.cachedDisplayName,
                    avatar: initialActive?.cachedAvatarUrl,
                    crafts: const [],
                  ),
            ),
      ),
    ],
  );
  await container.read(authSessionProvider.future);
  final routerSubscription = container.listen(
    goRouterProvider,
    (_, _) {},
    fireImmediately: true,
  );
  addTearDown(routerSubscription.close);
  final router = routerSubscription.read()..go(RouteLocations.feed);
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
  return container;
}

void main() {
  testWidgets(
    'initial signed-in shell hydrates the active Profile avatar',
    (tester) async {
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final container = await _pumpShell(
        tester,
        registry: registry,
        size: const Size(500, 800),
        activeProfile: Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          avatar: 'https://example.test/alice.jpg',
          crafts: const [],
        ),
      );

      expect(
        tester.widget<AccountAvatar>(find.byType(AccountAvatar)).avatarUrl,
        'https://example.test/alice.jpg',
      );
      expect(
        container
            .read(sessionRegistryProvider)
            .requireValue
            .sessions[AccountKey('did:plc:alice').did]
            ?.cachedAvatarUrl,
        'https://example.test/alice.jpg',
      );
    },
  );

  testWidgets(
    'IT-011 pending profile refresh never shows the previous account avatar',
    (tester) async {
      final bobProfile = Completer<Profile>();
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
            cachedAvatarUrl: 'https://example.test/bob-cached.jpg',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
            cachedAvatarUrl: 'https://example.test/alice.jpg',
          );
      final container = await _pumpShell(
        tester,
        registry: registry,
        size: const Size(500, 800),
        profileRepository: FakeProfileRepository(
          onFetch: (id) async => id == 'bob.test'
              ? bobProfile.future
              : Profile(
                  did: 'did:plc:alice',
                  handle: 'alice.test',
                  avatar: 'https://example.test/alice.jpg',
                  crafts: const [],
                ),
        ),
      );
      expect(
        tester.widget<AccountAvatar>(find.byType(AccountAvatar)).avatarUrl,
        'https://example.test/alice.jpg',
      );
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;

      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      await tester.pump();

      expect(
        tester.widget<AccountAvatar>(find.byType(AccountAvatar)).avatarUrl,
        'https://example.test/bob-cached.jpg',
      );

      bobProfile.complete(
        Profile(
          did: 'did:plc:bob',
          handle: 'bob.test',
          avatar: 'https://example.test/bob-fresh.jpg',
          crafts: const [],
        ),
      );
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'SIM-UT-004 switcher never loads inactive notification counts',
    (tester) async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final repositories = {
        alice: _CountingCountRepository(),
        bob: _CountingCountRepository(),
      };
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: bob.did.value,
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: alice.did.value,
            handle: 'alice.test',
          );
      await _pumpShell(
        tester,
        registry: registry,
        size: const Size(500, 800),
        countRepository: (account) => repositories[account]!,
      );

      expect(repositories[alice]!.countCalls, 1);
      expect(repositories[bob]!.countCalls, 0);

      await tester.longPress(find.byTooltip('Switch account').first);
      await tester.pumpAndSettle();

      expect(repositories[bob]!.countCalls, 0);
    },
  );

  testWidgets(
    'SIM-UT-004 inactive switcher rows do not show unread badges',
    (
      tester,
    ) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'b-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
            cachedDisplayName: 'Bob',
          )
          .upsertAndActivate(
            token: 'a-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final state = AccountSwitcherState.fromRegistry(registry);
      final selected = <String>[];
      var adds = 0;

      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: AccountSwitcherContent(
              state: state,
              onSelect: (lease) => selected.add(lease.account.did.value),
              onAddAccount: () => adds++,
            ),
          ),
        ),
      );

      expect(find.text('alice.test'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('@bob.test'), findsOneWidget);
      expect(find.text('7'), findsNothing);
      expect(find.text('Add account'), findsOneWidget);
      expect(find.textContaining('Sign out'), findsNothing);
      expect(find.byIcon(Icons.delete_outline), findsNothing);

      await tester.tap(find.text('Bob'));
      await tester.tap(find.text('Add account'));
      expect(selected, ['did:plc:bob']);
      expect(adds, 1);
    },
  );

  testWidgets('UT-018 current semantics and five-account helper are explicit', (
    tester,
  ) async {
    var registry = SessionRegistry.empty();
    for (var index = 0; index < SessionRegistry.maxRetainedAccounts; index++) {
      registry = registry.upsertAndActivate(
        token: 'token-$index',
        did: 'did:plc:a$index',
        handle: 'a$index.test',
      );
    }

    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: AccountSwitcherContent(
            state: AccountSwitcherState.fromRegistry(registry),
            onSelect: (_) => fail('current account must not select'),
            onAddAccount: () => fail('disabled Add must not run'),
          ),
        ),
      ),
    );

    expect(find.text('Maximum of 5 accounts'), findsOneWidget);
    final add = tester.widget<ListTile>(
      find.widgetWithText(ListTile, 'Add account'),
    );
    expect(add.enabled, isFalse);
    expect(
      tester
          .getSemantics(find.widgetWithText(ListTile, 'a4.test'))
          .flagsCollection
          .isSelected,
      Tristate.isTrue,
    );
  });

  testWidgets('SIM-IT-005 activation loads and disables inside switcher', (
    tester,
  ) async {
    final registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;

    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: AccountSwitcherContent(
            state: AccountSwitcherState.fromRegistry(registry),
            activating: bob,
            onSelect: (_) => fail('busy switcher must not select'),
            onAddAccount: () => fail('busy switcher must not add'),
          ),
        ),
      ),
    );

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(
      tester
          .widget<ListTile>(find.widgetWithText(ListTile, 'bob.test'))
          .enabled,
      isFalse,
    );
    expect(
      tester
          .widget<ListTile>(find.widgetWithText(ListTile, 'Add account'))
          .enabled,
      isFalse,
    );
  });

  testWidgets(
    'UT-018 IT-011 compact failed avatar falls back and long-press switches',
    (tester) async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
            cachedAvatarUrl: 'https://example.test/alice.jpg',
          );
      final container = await _pumpShell(
        tester,
        registry: registry,
        size: const Size(500, 800),
      );

      expect(
        tester.widget<AccountAvatar>(find.byType(AccountAvatar)).selected,
        isFalse,
      );
      expect(
        find.descendant(
          of: find.byType(AccountAvatar),
          matching: find.byIcon(Icons.person),
        ),
        findsOneWidget,
      );

      await tester.longPress(find.byTooltip('Switch account').first);
      await tester.pumpAndSettle();
      expect(find.text('bob.test'), findsOneWidget);
      expect(find.byType(BottomSheet), findsOneWidget);

      await tester.tap(find.text('bob.test'));
      await tester.pumpAndSettle();
      expect(
        container.read(sessionRegistryProvider).requireValue.activeDid,
        'did:plc:bob',
      );
      expect(find.textContaining('Sign out all'), findsNothing);
    },
  );

  testWidgets('IT-011 large Profile trigger opens an anchored account menu', (
    tester,
  ) async {
    final registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
          cachedAvatarUrl: 'https://example.test/alice.jpg',
        );
    final container = await _pumpShell(
      tester,
      registry: registry,
      size: const Size(1000, 800),
    );

    expect(find.byType(NavigationRail), findsOneWidget);
    expect(find.byType(AccountAvatar), findsOneWidget);
    expect(
      tester.widget<AccountAvatar>(find.byType(AccountAvatar)).avatarUrl,
      'https://example.test/alice.jpg',
    );
    expect(find.byIcon(Icons.switch_account), findsNothing);

    final profileAvatar = find.byType(AccountAvatar);
    var avatar = tester.widget<AccountAvatar>(profileAvatar);
    expect(avatar.selected, isFalse);
    await tester.tap(profileAvatar);
    await tester.pumpAndSettle();
    expect(
      tester.widget<NavigationRail>(find.byType(NavigationRail)).selectedIndex,
      4,
    );
    expect(find.byType(AccountSwitcherContent), findsNothing);
    avatar = tester.widget<AccountAvatar>(profileAvatar);
    expect(avatar.selected, isTrue);

    final semantics = tester.getSemantics(profileAvatar);
    expect(
      semantics.getSemanticsData().hasAction(SemanticsAction.longPress),
      isTrue,
    );
    tester
        .widget<FocusableActionDetector>(
          find.ancestor(
            of: profileAvatar,
            matching: find.byType(FocusableActionDetector),
          ),
        )
        .focusNode!
        .requestFocus();
    await tester.pump();
    await tester.sendKeyDownEvent(LogicalKeyboardKey.alt);
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
    await tester.sendKeyUpEvent(LogicalKeyboardKey.alt);
    await tester.pumpAndSettle();

    expect(find.byType(AccountSwitcherContent), findsOneWidget);
    expect(find.text('bob.test'), findsOneWidget);
    expect(find.text('Add account'), findsOneWidget);
    await tester.tap(find.text('bob.test'));
    await tester.pumpAndSettle();
    expect(
      container.read(sessionRegistryProvider).requireValue.activeDid,
      'did:plc:bob',
    );
  });

  testWidgets('UT-018 large missing avatar keeps selected person fallback', (
    tester,
  ) async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    await _pumpShell(
      tester,
      registry: registry,
      size: const Size(1000, 800),
    );

    var avatar = tester.widget<AccountAvatar>(find.byType(AccountAvatar));
    expect(avatar.avatarUrl, isNull);
    expect(avatar.selected, isFalse);
    expect(
      find.descendant(
        of: find.byType(AccountAvatar),
        matching: find.byIcon(Icons.person),
      ),
      findsOneWidget,
    );

    await tester.tap(find.byType(AccountAvatar));
    await tester.pumpAndSettle();

    avatar = tester.widget<AccountAvatar>(find.byType(AccountAvatar));
    expect(avatar.selected, isTrue);
    expect(
      tester.widget<NavigationRail>(find.byType(NavigationRail)).selectedIndex,
      4,
    );
    expect(
      tester
          .getSemantics(find.byType(AccountAvatar))
          .flagsCollection
          .isSelected,
      Tristate.isTrue,
    );
    expect(find.byType(AccountSwitcherContent), findsNothing);
    expect(
      find.descendant(
        of: find.byType(AccountAvatar),
        matching: find.byIcon(Icons.person),
      ),
      findsOneWidget,
    );
  });
}
