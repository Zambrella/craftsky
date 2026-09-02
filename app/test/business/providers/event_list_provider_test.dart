import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test(
    'UT-010 exposes stable initial loading and retryable initial error',
    () async {
      final firstRequest = Completer<BusinessEventPage>();
      final repository = _EventRepository([
        firstRequest,
        BusinessEventPage(items: [_event('recovered')]),
      ]);
      final target = _target();
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      final subscription = container.listen(
        profileBusinessEventsProvider(target),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);

      expect(
        container.read(profileBusinessEventsProvider(target)).isLoading,
        isTrue,
      );
      firstRequest.completeError(StateError('initial load failed'));
      await expectLater(
        container.read(profileBusinessEventsProvider(target).future),
        throwsStateError,
      );
      expect(
        container.read(profileBusinessEventsProvider(target)).hasError,
        isTrue,
      );

      await container
          .read(profileBusinessEventsProvider(target).notifier)
          .retryInitial();
      final recovered = container
          .read(profileBusinessEventsProvider(target))
          .value!;
      expect(recovered.items.single.rkey.toString(), 'recovered');
    },
  );

  test(
    'UT-010 preserves order and rows after incremental failure',
    () async {
      final repository = _EventRepository([
        BusinessEventPage(
          items: [_event('first'), _event('second')],
          cursor: 'opaque:first page +/%',
        ),
        BusinessEventPage(
          items: [_event('second'), _event('third')],
          cursor: 'opaque:second-page',
        ),
        StateError('next page failed'),
      ]);
      final target = ProfileBusinessEventsTarget(
        account: AccountKey('did:plc:viewer'),
        owner: AtIdentifier.parse('business.example'),
      );
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);

      final initial = await container.read(
        profileBusinessEventsProvider(target).future,
      );
      expect(initial.items.map((event) => event.rkey.toString()), [
        'first',
        'second',
      ]);

      await container
          .read(profileBusinessEventsProvider(target).notifier)
          .loadMore();
      var state = container.read(profileBusinessEventsProvider(target)).value!;
      expect(state.items.map((event) => event.rkey.toString()), [
        'first',
        'second',
        'third',
      ]);
      expect(state.cursor, 'opaque:second-page');

      await container
          .read(profileBusinessEventsProvider(target).notifier)
          .loadMore();
      state = container.read(profileBusinessEventsProvider(target)).value!;
      expect(state.items.map((event) => event.rkey.toString()), [
        'first',
        'second',
        'third',
      ]);
      expect(state.cursor, 'opaque:second-page');
      expect(state.incrementalError, isNotNull);
      expect(repository.cursors, [
        null,
        'opaque:first page +/%',
        'opaque:second-page',
      ]);
    },
  );

  test(
    'UT-010 refresh failure retains rows and public rows are never filtered',
    () async {
      final cancelled = _event('cancelled', status: 'cancelled', past: true);
      final repository = _EventRepository([
        BusinessEventPage(items: [cancelled], cursor: 'opaque:retained'),
        StateError('refresh failed'),
      ]);
      final target = _target();
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);

      await container.read(profileBusinessEventsProvider(target).future);
      await container
          .read(profileBusinessEventsProvider(target).notifier)
          .refresh();
      final state = container
          .read(profileBusinessEventsProvider(target))
          .value!;

      expect(state.items, [same(cancelled)]);
      expect(state.cursor, 'opaque:retained');
      expect(state.refreshError, isNotNull);
      expect(repository.cursors, [null, null]);
    },
  );

  test('UT-010 null cursor is a stable end state', () async {
    final repository = _EventRepository([
      BusinessEventPage(items: [_event('only')]),
    ]);
    final target = _target();
    final container = ProviderContainer(
      overrides: [businessRepositoryProvider.overrideWithValue(repository)],
    );
    addTearDown(container.dispose);

    final initial = await container.read(
      profileBusinessEventsProvider(target).future,
    );
    expect(initial.hasMore, isFalse);
    await container
        .read(profileBusinessEventsProvider(target).notifier)
        .loadMore();
    expect(repository.cursors, [null]);
  });

  test('UT-010 provider identity includes viewer account and target', () {
    final owner = AtIdentifier.parse('business.example');
    final first = ProfileBusinessEventsTarget(
      account: AccountKey('did:plc:first'),
      owner: owner,
    );
    final second = ProfileBusinessEventsTarget(
      account: AccountKey('did:plc:second'),
      owner: owner,
    );
    final otherTarget = ProfileBusinessEventsTarget(
      account: first.account,
      owner: AtIdentifier.parse('other.example'),
    );

    expect(first, isNot(second));
    expect(first, isNot(otherTarget));
  });

  test(
    'IT-013 a pre-mutation list refresh cannot publish stale rows or cursor',
    () async {
      final staleRefresh = Completer<BusinessEventPage>();
      final before = _event('event').copyWith(name: 'Before');
      final accepted = before.copyWith(
        cid: Cid.parse('bafy-accepted'),
        name: 'Accepted',
      );
      final repository = _EventRepository([
        BusinessEventPage(items: [before], cursor: 'current-cursor'),
        staleRefresh,
      ]);
      final target = ProfileBusinessEventsTarget(
        account: AccountKey('did:plc:viewer'),
        owner: AtIdentifier.parse('did:plc:business'),
      );
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      final subscription = container.listen(
        profileBusinessEventsProvider(target),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(profileBusinessEventsProvider(target).future);

      final refresh = container
          .read(profileBusinessEventsProvider(target).notifier)
          .refresh();
      await Future<void>.delayed(Duration.zero);
      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        target.account,
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

      final state = container
          .read(profileBusinessEventsProvider(target))
          .requireValue;
      expect(state.items.single.name, 'Accepted');
      expect(state.cursor, 'current-cursor');
      expect(state.isRefreshing, isFalse);
      expect(container.read(businessProjectionOverlayProvider), contains(key));
    },
  );

  test(
    'IT-010 IT-013 load more retains rows and cursor when reconciliation is '
    'stale',
    () async {
      final stalePage = Completer<BusinessEventPage>();
      final confirmed = _event('event').copyWith(name: 'Confirmed');
      final accepted = confirmed.copyWith(
        cid: Cid.parse('bafy-accepted'),
        name: 'Accepted',
      );
      final repository = _EventRepository([
        BusinessEventPage(items: [confirmed], cursor: 'current-cursor'),
        stalePage,
      ]);
      final target = ProfileBusinessEventsTarget(
        account: AccountKey('did:plc:viewer'),
        owner: AtIdentifier.parse('did:plc:business'),
      );
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      final provider = profileBusinessEventsProvider(target);
      final subscription = container.listen(
        provider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(subscription.close);
      await container.read(provider.future);

      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        target.account,
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

      final loadMore = container.read(provider.notifier).loadMore();
      await Future<void>.delayed(Duration.zero);
      overlays.reconcile<BusinessEvent>(
        key: key,
        fence: overlays.captureRead(lease),
        authoritativeCid: accepted.cid,
        authoritativeView: accepted,
      );
      stalePage.complete(
        BusinessEventPage(
          items: [confirmed.copyWith(name: 'Stale response')],
          cursor: 'stale-cursor',
        ),
      );
      await loadMore;

      final state = container.read(provider).requireValue;
      expect(state.items, [same(confirmed)]);
      expect(state.cursor, 'current-cursor');
      expect(state.isLoadingMore, isFalse);
      expect(container.read(businessProjectionOverlayProvider), isEmpty);
    },
  );
}

ProfileBusinessEventsTarget _target() => ProfileBusinessEventsTarget(
  account: AccountKey('did:plc:viewer'),
  owner: AtIdentifier.parse('business.example'),
);

BusinessEvent _event(
  String rkey, {
  String status = 'scheduled',
  bool past = false,
}) => BusinessEvent(
  did: 'did:plc:business',
  rkey: rkey,
  uri: 'at://did:plc:business/social.craftsky.business.event/$rkey',
  cid: 'bafy-$rkey',
  name: 'Event $rkey',
  startsAt: DateTime.utc(2026, 9, 5, 9),
  endsAt: DateTime.utc(2026, 9, 5, 17),
  roles: const [],
  status: BusinessOpenValue(value: status, known: true),
  isAllDay: false,
  createdAt: DateTime.utc(2026, 8, 30),
  past: past,
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);

final class _EventRepository extends Fake implements BusinessRepository {
  _EventRepository(this.results);

  final List<Object> results;
  final List<String?> cursors = [];
  var _index = 0;

  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) async {
    cursors.add(cursor);
    final result = results[_index++];
    if (result is BusinessEventPage) return result;
    if (result is Completer<BusinessEventPage>) return result.future;
    _throwResult(result);
  }
}

Never _throwResult(Object result) => switch (result) {
  final Exception error => throw error,
  final Error error => throw error,
  _ => throw StateError('Unexpected fake result'),
};
