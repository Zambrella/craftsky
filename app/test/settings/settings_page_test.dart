import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  testWidgets(
    'SettingsPage renders title, ClearImageCacheTile, and SignOutTile',
    (tester) async {
      await tester.pumpWidget(
        const ProviderScope(
          child: MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: SettingsPage(),
          ),
        ),
      );
      expect(find.text('Settings'), findsWidgets);
      expect(find.text('Followers'), findsOneWidget);
      expect(find.text('Following'), findsOneWidget);
      expect(find.text('Find people from Instagram'), findsOneWidget);
      expect(find.text('Saved posts'), findsOneWidget);
      expect(find.text('Languages'), findsOneWidget);
      expect(find.textContaining(RegExp(r'\d+ followers')), findsNothing);
      expect(find.textContaining(RegExp(r'\d+ following')), findsNothing);
      expect(find.byType(ClearImageCacheTile), findsOneWidget);
      expect(find.byType(SignOutTile), findsOneWidget);
    },
  );

  testWidgets('Instagram settings entry opens the typed migration location', (
    tester,
  ) async {
    final router = GoRouter(
      initialLocation: '/settings',
      routes: [
        GoRoute(
          path: '/settings',
          builder: (_, _) => const SettingsPage(),
        ),
        GoRoute(
          path: '/profile/settings/instagram',
          builder: (_, _) => const Scaffold(body: Text('Instagram route')),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Find people from Instagram'));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/profile/settings/instagram');
    expect(find.text('Instagram route'), findsOneWidget);
  });

  testWidgets(
    'AT-006 shows active-account attention count and typed scheduled route',
    (tester) async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'alice-token',
            did: alice.did.value,
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'bob-token',
            did: bob.did.value,
            handle: 'bob.test',
          );
      final aliceRegistry = registry.activate(registry.leaseFor(alice)!);
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(aliceRegistry),
          ),
          accountScheduledPostRepositoryProvider(alice).overrideWith(
            (ref) async => _SettingsRepository([
              _scheduledItem('alice-scheduled', ScheduledPostStatus.scheduled),
              _scheduledItem(
                'alice-attention-1',
                ScheduledPostStatus.needsAttention,
              ),
              _scheduledItem(
                'alice-attention-2',
                ScheduledPostStatus.needsAttention,
              ),
            ]),
          ),
          accountScheduledPostRepositoryProvider(
            bob,
          ).overrideWith((ref) async => const _SettingsRepository([])),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);

      final router = GoRouter(
        initialLocation: '/settings',
        routes: [
          GoRoute(
            path: '/settings',
            builder: (_, _) => const SettingsPage(),
          ),
          GoRoute(
            path: '/profile/settings/scheduled',
            builder: (_, _) => const Scaffold(
              body: Text('Scheduled posts route'),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();

      Finder scheduledTile() => find.ancestor(
        of: find.text('Scheduled posts'),
        matching: find.byType(ListTile),
      );

      expect(scheduledTile(), findsOneWidget);
      expect(
        find.descendant(of: scheduledTile(), matching: find.text('2')),
        findsOneWidget,
      );

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceRegistry.leaseFor(bob)!);
      await tester.pumpAndSettle();

      expect(scheduledTile(), findsOneWidget);
      expect(
        find.descendant(of: scheduledTile(), matching: find.byType(Badge)),
        findsNothing,
      );

      await tester.tap(find.text('Scheduled posts'));
      await tester.pumpAndSettle();

      expect(router.state.uri.path, '/profile/settings/scheduled');
      expect(find.text('Scheduled posts route'), findsOneWidget);
    },
  );
}

final class _SettingsRepository implements ScheduledPostRepository {
  const _SettingsRepository(this.items);

  final List<ScheduledPostSummary> items;

  @override
  Future<List<ScheduledPostSummary>> list() async => items;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

ScheduledPostSummary _scheduledItem(String id, ScheduledPostStatus status) =>
    ScheduledPostSummary(
      id: id,
      kind: ScheduledPostKind.standard,
      status: status,
      text: '$id preview',
      scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
    );
