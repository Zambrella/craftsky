import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('AT-006 keeps Upcoming and History traversals independent', () async {
    final repository = _Repository({
      OwnerEventFilter.upcoming: [
        BusinessEventPage(
          items: [_event('near'), _event('far')],
          cursor: 'upcoming opaque +/%',
        ),
      ],
      OwnerEventFilter.history: [
        BusinessEventPage(
          items: [_event('recent', status: 'cancelled')],
          cursor: 'history opaque +/%',
        ),
      ],
    });
    final container = _container(repository);
    addTearDown(container.dispose);

    final upcoming = await container.read(
      ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
    );
    final history = await container.read(
      ownerBusinessEventsProvider(OwnerEventFilter.history).future,
    );

    expect(upcoming.items.map((event) => event.rkey.toString()), [
      'near',
      'far',
    ]);
    expect(upcoming.cursor, 'upcoming opaque +/%');
    expect(history.items.single.rkey.toString(), 'recent');
    expect(history.cursor, 'history opaque +/%');
    expect(repository.calls, [
      (filter: OwnerEventFilter.upcoming, cursor: null),
      (filter: OwnerEventFilter.history, cursor: null),
    ]);
  });

  test('AT-006 appends in server order and dedupes only by identity', () async {
    final unusual = _event('unusual', status: 'cancelled');
    final repository = _Repository({
      OwnerEventFilter.upcoming: [
        BusinessEventPage(
          items: [_event('first'), unusual],
          cursor: 'opaque-next',
        ),
        BusinessEventPage(
          items: [
            unusual.copyWith(name: 'duplicate'),
            _event('third'),
          ],
        ),
      ],
    });
    final container = _container(repository);
    addTearDown(container.dispose);
    await container.read(
      ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
    );

    await container
        .read(
          ownerBusinessEventsProvider(OwnerEventFilter.upcoming).notifier,
        )
        .loadMore();
    final state = container
        .read(ownerBusinessEventsProvider(OwnerEventFilter.upcoming))
        .value!;

    expect(state.items.map((event) => event.rkey.toString()), [
      'first',
      'unusual',
      'third',
    ]);
    expect(state.items[1], same(unusual));
    expect(repository.calls.last.cursor, 'opaque-next');
  });

  test('AT-006 retains confirmed rows after incremental failure', () async {
    final repository = _Repository({
      OwnerEventFilter.history: [
        BusinessEventPage(items: [_event('kept')], cursor: 'next'),
        StateError('offline'),
      ],
    });
    final container = _container(repository);
    addTearDown(container.dispose);
    await container.read(
      ownerBusinessEventsProvider(OwnerEventFilter.history).future,
    );

    await container
        .read(ownerBusinessEventsProvider(OwnerEventFilter.history).notifier)
        .loadMore();
    final state = container
        .read(ownerBusinessEventsProvider(OwnerEventFilter.history))
        .value!;

    expect(state.items.single.rkey.toString(), 'kept');
    expect(state.cursor, 'next');
    expect(state.incrementalError, isA<StateError>());
  });

  test(
    'AT-006 invalid_cursor restarts only that view once cursorless',
    () async {
      final repository = _Repository({
        OwnerEventFilter.upcoming: [
          BusinessEventPage(items: [_event('upcoming')]),
        ],
        OwnerEventFilter.history: [
          BusinessEventPage(items: [_event('old')], cursor: 'expired'),
          const ApiBadRequest('invalid_cursor'),
          BusinessEventPage(items: [_event('restarted')]),
        ],
      });
      final container = _container(repository);
      addTearDown(container.dispose);
      await container.read(
        ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
      );
      await container.read(
        ownerBusinessEventsProvider(OwnerEventFilter.history).future,
      );

      await container
          .read(ownerBusinessEventsProvider(OwnerEventFilter.history).notifier)
          .loadMore();

      final history = container
          .read(ownerBusinessEventsProvider(OwnerEventFilter.history))
          .value!;
      final upcoming = container
          .read(ownerBusinessEventsProvider(OwnerEventFilter.upcoming))
          .value!;
      expect(history.items.single.rkey.toString(), 'restarted');
      expect(history.refreshGeneration, 1);
      expect(upcoming.items.single.rkey.toString(), 'upcoming');
      expect(repository.calls, [
        (filter: OwnerEventFilter.upcoming, cursor: null),
        (filter: OwnerEventFilter.history, cursor: null),
        (filter: OwnerEventFilter.history, cursor: 'expired'),
        (filter: OwnerEventFilter.history, cursor: null),
      ]);
    },
  );

  test('AT-006 refresh replaces only its view with a new generation', () async {
    final repository = _Repository({
      OwnerEventFilter.upcoming: [
        BusinessEventPage(items: [_event('before')], cursor: 'old-cursor'),
        BusinessEventPage(items: [_event('after')], cursor: 'new-cursor'),
      ],
    });
    final container = _container(repository);
    addTearDown(container.dispose);
    await container.read(
      ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
    );

    await container
        .read(
          ownerBusinessEventsProvider(OwnerEventFilter.upcoming).notifier,
        )
        .refresh();
    final state = container
        .read(ownerBusinessEventsProvider(OwnerEventFilter.upcoming))
        .value!;

    expect(state.items.single.rkey.toString(), 'after');
    expect(state.cursor, 'new-cursor');
    expect(state.refreshGeneration, 1);
  });

  test(
    'IT-013 a pre-mutation owner refresh cannot publish stale rows or cursor',
    () async {
      final staleRefresh = Completer<BusinessEventPage>();
      final before = _event('event').copyWith(name: 'Before');
      final accepted = before.copyWith(
        cid: Cid.parse('bafy-accepted'),
        name: 'Accepted',
      );
      final repository = _Repository({
        OwnerEventFilter.upcoming: [
          BusinessEventPage(items: [before], cursor: 'current-cursor'),
          staleRefresh,
        ],
      });
      final container = _container(repository);
      addTearDown(container.dispose);
      final provider = ownerBusinessEventsProvider(OwnerEventFilter.upcoming);
      final subscription = container.listen(
        provider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(provider.future);

      final refresh = container.read(provider.notifier).refresh();
      await Future<void>.delayed(Duration.zero);
      final lease = AccountSessionLease(
        account: AccountKey('did:plc:owner'),
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        lease.account,
        before.did,
        before.rkey,
      );
      final overlays = container.read(
        businessProjectionOverlayProvider.notifier,
      );
      final generation = overlays.beginMutation(key, lease);
      overlays.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: before.cid,
        acceptedCid: accepted.cid,
        acceptedView: accepted,
      );

      staleRefresh.complete(
        BusinessEventPage(
          items: [before.copyWith(name: 'Stale')],
          cursor: 'stale-cursor',
        ),
      );
      await refresh;

      final state = container.read(provider).requireValue;
      expect(state.items.single.name, 'Accepted');
      expect(state.cursor, 'current-cursor');
      expect(state.isRefreshing, isFalse);
      expect(container.read(businessProjectionOverlayProvider), contains(key));
    },
  );

  test(
    'IT-010 IT-013 refresh retains rows and cursor when reconciliation is '
    'stale',
    () async {
      final staleRefresh = Completer<BusinessEventPage>();
      final confirmed = _event('event').copyWith(name: 'Confirmed');
      final accepted = confirmed.copyWith(
        cid: Cid.parse('bafy-accepted'),
        name: 'Accepted',
      );
      final repository = _Repository({
        OwnerEventFilter.upcoming: [
          BusinessEventPage(items: [confirmed], cursor: 'current-cursor'),
          staleRefresh,
        ],
      });
      final container = _container(repository);
      addTearDown(container.dispose);
      final provider = ownerBusinessEventsProvider(OwnerEventFilter.upcoming);
      final subscription = container.listen(
        provider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(provider.future);

      final lease = AccountSessionLease(
        account: AccountKey('did:plc:owner'),
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        lease.account,
        confirmed.did,
        confirmed.rkey,
      );
      final overlays = container.read(
        businessProjectionOverlayProvider.notifier,
      );
      final generation = overlays.beginMutation(key, lease);
      overlays.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: confirmed.cid,
        acceptedCid: accepted.cid,
        acceptedView: accepted,
      );

      final refresh = container.read(provider.notifier).refresh();
      await Future<void>.delayed(Duration.zero);
      overlays.reconcile<BusinessEvent>(
        key: key,
        fence: overlays.captureRead(lease),
        authoritativeCid: accepted.cid,
        authoritativeView: accepted,
      );
      staleRefresh.complete(
        BusinessEventPage(
          items: [confirmed.copyWith(name: 'Stale response')],
          cursor: 'stale-cursor',
        ),
      );
      await refresh;

      final state = container.read(provider).requireValue;
      expect(state.items, [same(confirmed)]);
      expect(state.cursor, 'current-cursor');
      expect(state.isRefreshing, isFalse);
      expect(container.read(businessProjectionOverlayProvider), isEmpty);
    },
  );
}

ProviderContainer _container(_Repository repository) => ProviderContainer(
  overrides: [businessRepositoryProvider.overrideWithValue(repository)],
  retry: (_, _) => null,
);

BusinessEvent _event(String rkey, {String status = 'scheduled'}) =>
    BusinessEvent(
      did: 'did:plc:owner',
      rkey: rkey,
      uri: 'at://did:plc:owner/social.craftsky.business.event/$rkey',
      cid: 'bafy-$rkey',
      name: 'Event $rkey',
      startsAt: DateTime.utc(2026, 9, 5, 9),
      endsAt: DateTime.utc(2026, 9, 5, 17),
      roles: const [],
      status: BusinessOpenValue(value: status, known: status == 'scheduled'),
      isAllDay: false,
      createdAt: DateTime.utc(2026, 8, 30),
      past: false,
      publicSuppressionReasons: const [],
      upcomingExclusionReasons: const [],
    );

final class _Repository extends Fake implements BusinessRepository {
  _Repository(this.results);

  final Map<OwnerEventFilter, List<Object>> results;
  final calls = <({OwnerEventFilter filter, String? cursor})>[];
  final indices = <OwnerEventFilter, int>{};

  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) async {
    calls.add((filter: filter, cursor: cursor));
    final index = indices.update(
      filter,
      (value) => value + 1,
      ifAbsent: () => 0,
    );
    final result = results[filter]![index];
    if (result is BusinessEventPage) return result;
    if (result is Completer<BusinessEventPage>) return result.future;
    if (result is Exception) throw result;
    if (result is Error) throw result;
    throw StateError('Unexpected fake result');
  }
}
