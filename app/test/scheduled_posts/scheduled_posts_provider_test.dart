import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-017 sorts, replaces on refresh and isolates account families',
    () async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceRepository = _QueuedRepository([
        [_summary('later', 14), _summary('earlier', 12)],
        [_summary('replacement', 13), _summary('replacement', 13)],
      ]);
      final bobRepository = _QueuedRepository([
        [_summary('bob-only', 15)],
      ]);
      final container = ProviderContainer.test(
        overrides: [
          accountScheduledPostRepositoryProvider(
            alice,
          ).overrideWith((ref) async => aliceRepository),
          accountScheduledPostRepositoryProvider(
            bob,
          ).overrideWith((ref) async => bobRepository),
        ],
      );
      addTearDown(container.dispose);
      final aliceSubscription = container.listen(
        scheduledPostsProvider(alice),
        (_, _) {},
      );
      final bobSubscription = container.listen(
        scheduledPostsProvider(bob),
        (_, _) {},
      );
      addTearDown(aliceSubscription.close);
      addTearDown(bobSubscription.close);

      final initialAlice = await container.read(
        scheduledPostsProvider(alice).future,
      );
      expect(initialAlice.items.map((item) => item.id), ['earlier', 'later']);

      await container.read(scheduledPostsProvider(alice).notifier).refresh();
      expect(
        container
            .read(scheduledPostsProvider(alice))
            .requireValue
            .items
            .map((item) => item.id),
        ['replacement'],
      );

      final bobState = await container.read(scheduledPostsProvider(bob).future);
      expect(bobState.items.map((item) => item.id), ['bob-only']);
      expect(
        container
            .read(scheduledPostsProvider(alice))
            .requireValue
            .items
            .map((item) => item.id),
        ['replacement'],
      );
    },
  );

  test(
    'IT-021 discards a delayed list result after the active account changes',
    () async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceRegistry = registry.activate(registry.leaseFor(alice)!);
      final aliceResponse = Completer<List<ScheduledPostSummary>>();
      final aliceRepository = _CompleterRepository(aliceResponse);
      final bobRepository = _QueuedRepository([
        [_summary('bob-only', 15)],
      ]);
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(aliceRegistry),
          ),
          accountScheduledPostRepositoryProvider(
            alice,
          ).overrideWith((ref) async => aliceRepository),
          accountScheduledPostRepositoryProvider(
            bob,
          ).overrideWith((ref) async => bobRepository),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);

      final aliceLoad = container.read(scheduledPostsProvider(alice).future);
      await aliceRepository.started.future;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceRegistry.leaseFor(bob)!);
      aliceResponse.complete([_summary('alice-private', 12)]);

      await expectLater(aliceLoad, throwsStateError);
      expect(container.read(scheduledPostsProvider(alice)).hasValue, isFalse);
      final bobState = await container.read(scheduledPostsProvider(bob).future);
      expect(bobState.items.map((item) => item.id), ['bob-only']);
    },
  );
}

final class _QueuedRepository implements ScheduledPostRepository {
  _QueuedRepository(this.responses);

  final List<List<ScheduledPostSummary>> responses;
  int calls = 0;

  @override
  Future<List<ScheduledPostSummary>> list() async {
    final response = responses[calls];
    calls++;
    return response;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _CompleterRepository implements ScheduledPostRepository {
  _CompleterRepository(this.response);

  final Completer<List<ScheduledPostSummary>> response;
  final started = Completer<void>();

  @override
  Future<List<ScheduledPostSummary>> list() {
    if (!started.isCompleted) started.complete();
    return response.future;
  }

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

ScheduledPostSummary _summary(String id, int hour) {
  return ScheduledPostSummary(
    id: id,
    kind: ScheduledPostKind.standard,
    status: ScheduledPostStatus.scheduled,
    text: id,
    scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 1, hour)),
  );
}
