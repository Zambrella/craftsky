import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_block_profile_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  final bobDid = Did.parse('did:plc:bob');
  final fallbackScope = blockProfileMutationScopeFor(
    AccountSessionLease(
      account: AccountKey(bobDid.toString()),
      sessionGeneration: 0,
    ),
    bobDid,
  );

  test('block becomes visible only after definite acceptance', () async {
    final accepted = Completer<ProfileRelationship>();
    final repo = FakeProfileRepository(
      onBlock: (_, _) => accepted.future,
    );
    final container = ProviderContainer.test(
      overrides: [profileRepositoryProvider.overrideWithValue(repo)],
    );

    final pending = container
        .read(toggleBlockProfileProvider.notifier)
        .toggle(targetDid: bobDid, isBlocking: false);

    expect(container.read(toggleBlockProfileProvider).isLoading, isTrue);
    expect(
      applyBlockRelationshipOverlayForScope(
        container.read(pdsRecordOperationControllerProvider),
        fallbackScope,
        const ProfileRelationship(initialized: true),
      ).blocking,
      isFalse,
    );

    accepted.complete(const ProfileRelationship(blocking: true));
    await pending;

    expect(
      applyBlockRelationshipOverlayForScope(
        container.read(pdsRecordOperationControllerProvider),
        fallbackScope,
        const ProfileRelationship(initialized: true),
      ).blocking,
      isTrue,
    );
  });

  test('ambiguous block retries with one canonical operation key', () async {
    var calls = 0;
    final repo = FakeProfileRepository(
      onBlock: (_, _) async {
        calls++;
        if (calls < 3) {
          throw const PdsMutationAmbiguousException(retryAfterSeconds: 2);
        }
        return const ProfileRelationship(blocking: true);
      },
    );
    final container = ProviderContainer.test(
      overrides: [
        profileRepositoryProvider.overrideWithValue(repo),
        pdsMutationDelayProvider.overrideWithValue((_) async {}),
        pdsMutationJitterProvider.overrideWithValue((_) => 0),
      ],
    );

    await container
        .read(toggleBlockProfileProvider.notifier)
        .toggle(targetDid: bobDid, isBlocking: false);

    expect(calls, 3);
    expect(repo.blockOperationKeys.toSet(), hasLength(1));
    expect(
      isCanonicalPdsMutationOperationKey(repo.blockOperationKeys.first),
      isTrue,
    );
  });

  test(
    'loaded notifications are suppressed only after block acceptance',
    () async {
      final accepted = Completer<ProfileRelationship>();
      final repo = FakeProfileRepository(onBlock: (_, _) => accepted.future);
      final notifications = _StaticNotificationRepository(
        NotificationPage(
          items: [_followFrom('did:plc:bob'), _followFrom('did:plc:carol')],
        ),
      );
      final container = ProviderContainer.test(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repo),
          notificationRepositoryProvider.overrideWithValue(notifications),
          pdsRecordOperationControllerProvider.overrideWithValue(
            PdsRecordOperationController(schedule: (_, _) {}),
          ),
        ],
      );
      await container.read(notificationsProvider.future);

      final pending = container
          .read(toggleBlockProfileProvider.notifier)
          .toggle(targetDid: bobDid, isBlocking: false);
      expect(
        container.read(notificationsProvider).requireValue.items,
        hasLength(2),
      );

      accepted.complete(const ProfileRelationship(blocking: true));
      await pending;
      final visible = await container.read(notificationsProvider.future);
      expect(visible.items, hasLength(1));
      expect(
        (visible.items.single as SocialNotification).actor.did.toString(),
        'did:plc:carol',
      );
    },
  );

  test('unblock masks stale active state until agreement or expiry', () async {
    var now = DateTime.utc(2026);
    final controller = PdsRecordOperationController(now: () => now);
    final repo = FakeProfileRepository(onUnblock: (_, _) async {});
    final container = ProviderContainer.test(
      overrides: [
        profileRepositoryProvider.overrideWithValue(repo),
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );

    await container
        .read(toggleBlockProfileProvider.notifier)
        .toggle(targetDid: bobDid, isBlocking: true);

    final stale = applyBlockRelationshipOverlayForScope(
      controller,
      fallbackScope,
      const ProfileRelationship(blocking: true, initialized: true),
    );
    expect(stale.blocking, isFalse);

    now = now.add(const Duration(seconds: 31));
    controller.advanceTime();
    final expired = applyBlockRelationshipOverlayForScope(
      controller,
      fallbackScope,
      const ProfileRelationship(blocking: true, initialized: true),
    );
    expect(expired.blocking, isTrue);
  });

  test('late block acceptance is fenced after an account switch', () async {
    final pending = Completer<ProfileRelationship>();
    final registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final controller = PdsRecordOperationController();
    final repo = FakeProfileRepository(onBlock: (_, _) => pending.future);
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        profileRepositoryProvider.overrideWithValue(repo),
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    final loaded = await container.read(sessionRegistryProvider.future);
    final aliceLease = loaded.activeLease!.session;

    final mutation = container
        .read(toggleBlockProfileProvider.notifier)
        .toggle(targetDid: bobDid, isBlocking: false);
    await container
        .read(sessionRegistryProvider.notifier)
        .activate(
          loaded.leaseFor(AccountKey('did:plc:bob'))!,
        );
    pending.complete(const ProfileRelationship(blocking: true));
    await mutation;

    expect(
      controller.overlayFor(blockProfileMutationScopeFor(aliceLease, bobDid)),
      isNull,
    );
  });
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

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}
