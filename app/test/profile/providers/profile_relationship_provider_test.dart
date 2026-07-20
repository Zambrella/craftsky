import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  setUpAll(initializeMappers);

  final alice = AccountKey('did:plc:alice');
  const subject = 'bob.craftsky.social';

  test('UT-011 applies optimistic state and rolls back on failure', () async {
    final completer = Completer<ProfileRelationship>();
    final repo = FakeProfileRepository(onMute: (_) => completer.future);
    final provider = profileRelationshipProvider(alice, subject);
    final container = ProviderContainer.test(
      overrides: [
        accountRelationshipRepositoryProvider(
          alice,
        ).overrideWith((ref) async => repo),
      ],
    );
    addTearDown(container.dispose);

    container.read(provider.notifier).seed(const ProfileRelationship());
    final pending = container
        .read(provider.notifier)
        .mutate(ProfileRelationshipAction.mute);

    expect(container.read(provider).muted, isTrue);
    expect(
      container.read(provider).pendingAction,
      ProfileRelationshipAction.mute,
    );

    await Future<void>.delayed(Duration.zero);
    completer.completeError(StateError('failed'));
    await pending;

    expect(container.read(provider).muted, isFalse);
    expect(container.read(provider).pendingAction, isNull);
    expect(container.read(provider).lastError, isA<StateError>());
  });

  test(
    'UT-011 confirmed block overlay wins until Tap state catches up',
    () async {
      final repo = FakeProfileRepository(
        onBlock: (_) async => const ProfileRelationship(
          blocking: true,
          uri: 'at://did:plc:alice/app.bsky.graph.block/3abc',
          rkey: '3abc',
        ),
      );
      final provider = profileRelationshipProvider(alice, subject);
      final container = ProviderContainer.test(
        overrides: [
          accountRelationshipRepositoryProvider(
            alice,
          ).overrideWith((ref) async => repo),
        ],
      );
      addTearDown(container.dispose);

      final notifier = container.read(provider.notifier)
        ..seed(const ProfileRelationship());
      await notifier.mutate(ProfileRelationshipAction.block);
      expect(container.read(provider).blocking, isTrue);
      expect(container.read(provider).confirmedOverlay, isTrue);

      notifier.seed(const ProfileRelationship());
      expect(container.read(provider).blocking, isTrue);

      notifier.seed(const ProfileRelationship(blocking: true));
      expect(container.read(provider).blocking, isTrue);
      expect(container.read(provider).confirmedOverlay, isFalse);
    },
  );

  test('UT-011 relationship cache is isolated by account key', () async {
    final bob = AccountKey('did:plc:bob');
    final aliceProvider = profileRelationshipProvider(alice, subject);
    final bobProvider = profileRelationshipProvider(bob, subject);
    final container = ProviderContainer.test();
    addTearDown(container.dispose);

    container
        .read(aliceProvider.notifier)
        .seed(const ProfileRelationship(muted: true));
    container
        .read(bobProvider.notifier)
        .seed(const ProfileRelationship(blockedBy: true));

    expect(container.read(aliceProvider).kind, ProfileRelationshipKind.muted);
    expect(
      container.read(bobProvider).kind,
      ProfileRelationshipKind.blockedBy,
    );
  });

  test('IT-009 confirmed overlays schedule bounded reconciliation', () async {
    Duration? scheduledDelay;
    void Function()? reconcile;
    var diagnostics = 0;
    final repo = FakeProfileRepository(
      onMute: (_) async => const ProfileRelationship(muted: true),
    );
    final countRepository = _CountingNewnessRepository(2);
    final provider = profileRelationshipProvider(alice, 'did:plc:bob');
    final container = ProviderContainer.test(
      overrides: [
        accountRelationshipRepositoryProvider(
          alice,
        ).overrideWith((ref) async => repo),
        relationshipReconciliationSchedulerProvider.overrideWithValue((
          delay,
          callback,
        ) {
          scheduledDelay = delay;
          reconcile = callback;
          return () {};
        }),
        relationshipReconciliationDiagnosticProvider.overrideWithValue(
          () => diagnostics++,
        ),
        accountNotificationNewnessRepositoryProvider(
          alice,
        ).overrideWith((ref) async => countRepository),
      ],
    );
    addTearDown(container.dispose);

    container
        .read(provider.notifier)
        .seed(const ProfileRelationship(initialized: true));
    expect(
      await container.read(accountNotificationNewCountProvider(alice).future),
      2,
    );
    await container
        .read(provider.notifier)
        .mutate(ProfileRelationshipAction.mute);

    expect(
      await container.read(accountNotificationNewCountProvider(alice).future),
      2,
    );
    expect(countRepository.countCalls, 2);

    expect(scheduledDelay, isNotNull);
    expect(scheduledDelay, lessThanOrEqualTo(const Duration(seconds: 5)));
    expect(diagnostics, 0);

    reconcile!();

    expect(diagnostics, 1);
    expect(container.read(provider).muted, isTrue);
    expect(container.read(provider).confirmedOverlay, isTrue);
    expect(
      await container.read(accountNotificationNewCountProvider(alice).future),
      2,
    );
    expect(countRepository.countCalls, 3);
  });

  test(
    'AT-006 pending mute suppresses loaded notifications and badge',
    () async {
      final mutation = Completer<ProfileRelationship>();
      final repo = FakeProfileRepository(onMute: (_) => mutation.future);
      final notificationRepository = _StaticNotificationRepository(
        NotificationPage(
          items: [_followFrom('did:plc:bob'), _followFrom('did:plc:carol')],
        ),
      );
      final container = ProviderContainer.test(
        overrides: [
          accountRelationshipRepositoryProvider(
            alice,
          ).overrideWith((ref) async => repo),
          notificationRepositoryProvider.overrideWithValue(
            notificationRepository,
          ),
          notificationNewnessRepositoryProvider.overrideWithValue(
            const _StaticNewnessRepository(2),
          ),
        ],
      );
      addTearDown(container.dispose);
      await container.read(notificationsProvider.future);
      await container.read(notificationNewCountProvider.future);
      final provider = profileRelationshipProvider(alice, 'did:plc:bob');
      container
          .read(provider.notifier)
          .seed(const ProfileRelationship(initialized: true));

      final pending = container
          .read(provider.notifier)
          .mutate(ProfileRelationshipAction.mute);

      expect(
        container
            .read(notificationsProvider)
            .requireValue
            .items
            .single
            .actor
            .did,
        'did:plc:carol',
      );
      expect(container.read(notificationNewCountProvider).requireValue, 1);

      mutation.complete(const ProfileRelationship(muted: true));
      await pending;
    },
  );
}

CraftskyNotification _followFrom(String did) => CraftskyNotification.fromMap({
  'id': 'notification-$did',
  'uri': 'at://$did/app.bsky.graph.follow/follow',
  'cid': 'bafy-follow',
  'rkey': 'follow',
  'type': 'follow',
  'actor': {'did': did, 'handle': 'actor.craftsky.social'},
  'createdAt': '2026-07-19T12:00:00Z',
  'indexedAt': '2026-07-19T12:00:01Z',
});

final class _StaticNotificationRepository implements NotificationRepository {
  const _StaticNotificationRepository(this.page);

  final NotificationPage page;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) async => page;
}

final class _StaticNewnessRepository implements NotificationNewnessRepository {
  const _StaticNewnessRepository(this.value);

  final int value;

  @override
  Future<int> count() async => value;

  @override
  Future<void> markSeen() async {}
}

final class _CountingNewnessRepository
    implements NotificationNewnessRepository {
  _CountingNewnessRepository(this.value);

  final int value;
  int countCalls = 0;

  @override
  Future<int> count() async {
    countCalls++;
    return value;
  }

  @override
  Future<void> markSeen() async {}
}
