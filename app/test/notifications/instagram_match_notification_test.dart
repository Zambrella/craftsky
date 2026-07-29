import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as auth_model;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_card.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  setUpAll(initializeMappers);

  InstagramMatchNotification match() =>
      CraftskyNotification.fromMap({
            'id': '00000000-0000-0000-0000-000000000321',
            'type': 'instagramMatch',
            'actor': {
              'available': true,
              'did': 'did:plc:synthetic-match',
              'handle': 'maker.synthetic.invalid',
              'displayName': 'Synthetic Maker',
              'viewerIsFollowing': true,
              'muted': false,
              'blocking': false,
              'blockedBy': false,
            },
            'createdAt': '2026-07-19T12:00:00Z',
            'indexedAt': '2026-07-19T12:04:00Z',
          })
          as InstagramMatchNotification;

  testWidgets('IT-017 renders an actorful match with a follow control', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: NotificationRow(notification: match())),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(ProfileAvatar), findsOneWidget);
    expect(
      find.text(
        'You automatically followed Synthetic Maker from your Instagram '
        'import',
      ),
      findsOneWidget,
    );
    expect(find.text('Unfollow'), findsOneWidget);
    expect(find.textContaining('ready to review'), findsNothing);
  });

  testWidgets('IT-017 opens the matched actor profile', (tester) async {
    GoRouterState? destination;
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) =>
              Scaffold(body: NotificationRow(notification: match())),
        ),
        GoRoute(
          path: '/profile/:handle',
          builder: (_, state) {
            destination = state;
            return const Scaffold(body: Text('Profile'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(
      find.text(
        'You automatically followed Synthetic Maker from your Instagram '
        'import',
      ),
    );
    await tester.pumpAndSettle();

    expect(destination?.uri.path, '/profile/maker.synthetic.invalid');
  });

  testWidgets('TDD-005B actor avatar opens the profile card', (tester) async {
    final repository = FakeProfileRepository(
      onFetch: (_) async => Profile(
        did: 'did:plc:synthetic-match',
        handle: 'maker.synthetic.invalid',
        displayName: 'Synthetic Maker',
        crafts: const ['sewing'],
      ),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: NotificationRow(notification: match())),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(ProfileAvatar));
    await tester.pumpAndSettle();

    expect(find.byType(ProfileCard), findsOneWidget);
  });

  testWidgets(
    'IT-017 retained-owner match cannot unfollow through the active account',
    (tester) async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final registry = auth_model.SessionRegistry.empty()
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
      var aliceUnfollowCalls = 0;
      final aliceRepository = FakeProfileRepository(
        onUnfollow: (_) async {
          aliceUnfollowCalls++;
          return Profile(
            did: 'did:plc:synthetic-match',
            handle: 'maker.synthetic.invalid',
            crafts: const [],
          );
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            profileRepositoryProvider.overrideWithValue(aliceRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: NotificationRow(
                notification: match(),
                owner: registry.leaseFor(bob),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Unfollow'));
      await tester.pumpAndSettle();

      expect(aliceUnfollowCalls, 0);
    },
  );

  testWidgets(
    'IT-017 account switch discards a late notification unfollow failure',
    (tester) async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final registry = auth_model.SessionRegistry.empty()
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
      final bobUnfollow = Completer<Profile>();
      var bobUnfollowCalls = 0;
      final bobRepository = FakeProfileRepository(
        onUnfollow: (_) {
          bobUnfollowCalls++;
          return bobUnfollow.future;
        },
      );
      final messenger = RecordingMessenger();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registry),
            ),
            accountRelationshipRepositoryProvider(
              bob,
            ).overrideWith((ref) async => bobRepository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: MessengerScope(
              messenger: messenger,
              child: Scaffold(
                body: NotificationRow(
                  notification: match(),
                  owner: registry.leaseFor(bob),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Unfollow'));
      await tester.pump();
      expect(bobUnfollowCalls, 1);
      expect(find.text('Follow'), findsOneWidget);

      final container = ProviderScope.containerOf(
        tester.element(find.byType(NotificationRow)),
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(registry.leaseFor(alice)!);
      bobUnfollow.completeError(StateError('late failure'));
      await tester.pumpAndSettle();

      expect(find.text('Follow'), findsOneWidget);
      expect(messenger.calls, isEmpty);
    },
  );
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  auth_model.SessionRegistry registry;

  @override
  Future<auth_model.SessionRegistry> read() async => registry;

  @override
  Future<void> write(auth_model.SessionRegistry registry) async {
    this.registry = registry;
  }
}
