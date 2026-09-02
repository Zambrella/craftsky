import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-015 exact pre-write CID retains an accepted update', () {
    final overlay = BusinessProjectionOverlay<String>.upsert(
      lease: _lease(),
      requestGeneration: 1,
      preWriteCid: Cid.parse('bafy-before'),
      acceptedCid: Cid.parse('bafy-accepted'),
      acceptedView: 'accepted',
    );

    final result = overlay.reconcile(
      authoritativeCid: Cid.parse('bafy-before'),
      authoritativeView: 'stale',
    );

    expect(result.view, 'accepted');
    expect(result.overlay, same(overlay));
  });

  test('UT-015 create settles only on accepted CID and adopts divergence', () {
    final overlay = BusinessProjectionOverlay<String>.upsert(
      lease: _lease(),
      requestGeneration: 1,
      preWriteCid: null,
      acceptedCid: Cid.parse('bafy-accepted'),
      acceptedView: 'accepted',
    );

    final absent = overlay.reconcile(
      authoritativeCid: null,
      authoritativeView: null,
    );
    final accepted = overlay.reconcile(
      authoritativeCid: Cid.parse('bafy-accepted'),
      authoritativeView: 'projected',
    );
    final divergent = overlay.reconcile(
      authoritativeCid: Cid.parse('bafy-opaque-third'),
      authoritativeView: 'concurrent',
    );

    expect(absent.view, 'accepted');
    expect(absent.overlay, same(overlay));
    expect(accepted.view, 'projected');
    expect(accepted.overlay, isNull);
    expect(divergent.view, 'concurrent');
    expect(divergent.overlay, isNull);
  });

  test('UT-015 delete tombstone hides deleted CID then adopts recreation', () {
    final overlay = BusinessProjectionOverlay<String>.delete(
      lease: _lease(),
      requestGeneration: 1,
      deletedCid: Cid.parse('bafy-deleted'),
      acceptedCid: Cid.parse('bafy-delete-response'),
    );

    final lag = overlay.reconcile(
      authoritativeCid: Cid.parse('bafy-deleted'),
      authoritativeView: 'deleted record',
    );
    final absent = overlay.reconcile(
      authoritativeCid: null,
      authoritativeView: null,
    );
    final recreated = overlay.reconcile(
      authoritativeCid: Cid.parse('bafy-recreated'),
      authoritativeView: 'recreated',
    );

    expect(lag.view, isNull);
    expect(lag.overlay, same(overlay));
    expect(absent.overlay, isNull);
    expect(recreated.view, 'recreated');
    expect(recreated.overlay, isNull);
  });

  test('UT-015 refresh failure retains retryable overlay without timeout', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final key = _eventKey();
    final lease = _lease();
    final generation = controller.beginMutation(key, lease);
    expect(
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: Cid.parse('bafy-accepted'),
        acceptedView: 'accepted',
      ),
      isTrue,
    );
    final fence = controller.captureRead(lease);

    expect(
      controller.markReadFailure(
        key: key,
        fence: fence,
        error: StateError('offline'),
      ),
      isTrue,
    );

    final overlay = container.read(businessProjectionOverlayProvider)[key]!;
    expect(overlay.acceptedView, 'accepted');
    expect(overlay.retryMetadata?.failureCount, 1);
  });

  test('UT-015 explicit reload warns and clears only after confirmation', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final key = _eventKey();
    final lease = _lease();
    final generation = controller.beginMutation(key, lease);
    controller.acceptUpsert(
      key: key,
      lease: lease,
      requestGeneration: generation,
      preWriteCid: Cid.parse('bafy-before'),
      acceptedCid: Cid.parse('bafy-accepted'),
      acceptedView: 'accepted',
    );

    expect(
      controller.discardForExplicitReload(
        key: key,
        lease: lease,
        confirmed: false,
      ),
      BusinessProjectionReloadDecision.warningRequired,
    );
    expect(container.read(businessProjectionOverlayProvider), contains(key));
    expect(
      controller.discardForExplicitReload(
        key: key,
        lease: lease,
        confirmed: true,
      ),
      BusinessProjectionReloadDecision.discarded,
    );
    expect(
      container.read(businessProjectionOverlayProvider),
      isNot(contains(key)),
    );
  });

  test('UT-015 rejects wrong lease and superseded request generation', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final key = _eventKey();
    final lease = _lease();
    final superseded = controller.beginMutation(key, lease);
    final current = controller.beginMutation(key, lease);

    expect(
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: superseded,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: Cid.parse('bafy-old-request'),
        acceptedView: 'old request',
      ),
      isFalse,
    );
    expect(
      controller.acceptUpsert(
        key: key,
        lease: _lease(generation: 2),
        requestGeneration: current,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: Cid.parse('bafy-wrong-session'),
        acceptedView: 'wrong session',
      ),
      isFalse,
    );
    expect(container.read(businessProjectionOverlayProvider), isEmpty);
  });

  test('IT-013 chained upserts retain the newest view across lagging CIDs', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final key = _eventKey();
    final lease = _lease();

    final first = controller.beginMutation(key, lease);
    expect(
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: first,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: Cid.parse('bafy-first'),
        acceptedView: 'first',
      ),
      isTrue,
    );
    final second = controller.beginMutation(key, lease);
    expect(
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: second,
        preWriteCid: Cid.parse('bafy-first'),
        acceptedCid: Cid.parse('bafy-second'),
        acceptedView: 'second',
      ),
      isTrue,
    );

    for (final laggingCid in ['bafy-before', 'bafy-first']) {
      final result = controller.reconcile<String>(
        key: key,
        fence: controller.captureRead(lease),
        authoritativeCid: Cid.parse(laggingCid),
        authoritativeView: 'lagging',
      );
      expect(result.view, 'second');
      expect(result.overlay, isNotNull);
    }

    final settled = controller.reconcile<String>(
      key: key,
      fence: controller.captureRead(lease),
      authoritativeCid: Cid.parse('bafy-second'),
      authoritativeView: 'settled',
    );
    expect(settled.view, 'settled');
    expect(settled.overlay, isNull);
  });

  test('UT-015 ignores a read from the wrong session lease', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final key = _eventKey();
    final lease = _lease();
    final generation = controller.beginMutation(key, lease);
    controller.acceptUpsert(
      key: key,
      lease: lease,
      requestGeneration: generation,
      preWriteCid: Cid.parse('bafy-before'),
      acceptedCid: Cid.parse('bafy-accepted'),
      acceptedView: 'accepted',
    );
    final wrongLease = _lease(generation: 2);

    expect(
      controller.reconcile<String>(
        key: key,
        fence: controller.captureRead(wrongLease),
        authoritativeCid: Cid.parse('bafy-before'),
        authoritativeView: 'wrong session',
      ),
      isA<BusinessProjectionReconciliation<String>>()
          .having((result) => result.isStale, 'is stale', isTrue)
          .having((result) => result.view, 'view', isNull),
    );
    expect(container.read(businessProjectionOverlayProvider), contains(key));
  });

  test('IT-013 lagging event lists retain update then settle or diverge', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final lease = _lease();
    final key = _eventKey();
    final accepted = _event(cid: 'bafy-accepted', name: 'Accepted');
    final generation = controller.beginMutation(key, lease);
    controller.acceptUpsert(
      key: key,
      lease: lease,
      requestGeneration: generation,
      preWriteCid: Cid.parse('bafy-before'),
      acceptedCid: accepted.cid,
      acceptedView: accepted,
    );

    final lag = reconcileBusinessEventList(
      controller: controller,
      lease: lease,
      fence: controller.captureRead(lease),
      owner: accepted.did,
      authoritative: [_event(cid: 'bafy-before', name: 'Stale')],
    );
    expect(lag.events.single.name, 'Accepted');
    expect(lag.events, hasLength(1));

    final projected = reconcileBusinessEventList(
      controller: controller,
      lease: lease,
      fence: controller.captureRead(lease),
      owner: accepted.did,
      authoritative: [_event(cid: 'bafy-accepted', name: 'Projected')],
    );
    expect(projected.events.single.name, 'Projected');
    expect(container.read(businessProjectionOverlayProvider), isEmpty);

    final nextGeneration = controller.beginMutation(key, lease);
    controller.acceptUpsert(
      key: key,
      lease: lease,
      requestGeneration: nextGeneration,
      preWriteCid: Cid.parse('bafy-accepted'),
      acceptedCid: Cid.parse('bafy-next-accepted'),
      acceptedView: _event(cid: 'bafy-next-accepted', name: 'Next accepted'),
    );
    final divergent = reconcileBusinessEventList(
      controller: controller,
      lease: lease,
      fence: controller.captureRead(lease),
      owner: accepted.did,
      authoritative: [_event(cid: 'bafy-third', name: 'Concurrent')],
    );
    expect(divergent.events.single.name, 'Concurrent');
    expect(container.read(businessProjectionOverlayProvider), isEmpty);
  });

  test(
    'IT-013 event list and detail keep accepted previews only through exact '
    'CID or absence lag',
    () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      final controller = container.read(
        businessProjectionOverlayProvider.notifier,
      );
      final lease = _lease();
      final key = _eventKey();
      final previewBytes = Uint8List.fromList([1, 2, 3]);
      final accepted = _event(
        cid: 'bafy-accepted',
        name: 'Accepted',
        image: BusinessImageView.localPreview(
          cid: 'bafy-image-accepted',
          mime: 'image/png',
          size: 3,
          alt: 'Accepted poster',
          previewBytes: previewBytes,
        ),
      );
      var generation = controller.beginMutation(key, lease);
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: accepted.cid,
        acceptedView: accepted,
      );

      final laggingList = reconcileBusinessEventList(
        controller: controller,
        lease: lease,
        fence: controller.captureRead(lease),
        owner: accepted.did,
        authoritative: [_event(cid: 'bafy-before', name: 'Stale')],
      );
      expect(
        laggingList.events.single.image?.previewBytes,
        same(previewBytes),
      );

      final projected = _event(
        cid: 'bafy-accepted',
        name: 'Projected',
        image: _networkImage('accepted'),
      );
      expect(
        controller
            .reconcile<BusinessEvent>(
              key: key,
              fence: controller.captureRead(lease),
              authoritativeCid: projected.cid,
              authoritativeView: projected,
            )
            .view
            ?.image
            ?.previewBytes,
        isNull,
      );

      generation = controller.beginMutation(key, lease);
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: null,
        acceptedCid: accepted.cid,
        acceptedView: accepted,
      );
      expect(
        controller
            .reconcile<BusinessEvent>(
              key: key,
              fence: controller.captureRead(lease),
              authoritativeCid: null,
              authoritativeView: null,
            )
            .view
            ?.image
            ?.previewBytes,
        same(previewBytes),
      );
      final divergent = _event(
        cid: 'bafy-third',
        name: 'Concurrent',
        image: _networkImage('third'),
      );
      expect(
        controller
            .reconcile<BusinessEvent>(
              key: key,
              fence: controller.captureRead(lease),
              authoritativeCid: divergent.cid,
              authoritativeView: divergent,
            )
            .view
            ?.image
            ?.previewBytes,
        isNull,
      );
      expect(container.read(businessProjectionOverlayProvider), isEmpty);
    },
  );

  test('IT-013 create absence and delete tombstone reconcile in lists', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final lease = _lease();
    final key = _eventKey();
    final created = _event(cid: 'bafy-created', name: 'Created');
    var generation = controller.beginMutation(key, lease);
    controller.acceptUpsert(
      key: key,
      lease: lease,
      requestGeneration: generation,
      preWriteCid: null,
      acceptedCid: created.cid,
      acceptedView: created,
    );

    expect(
      reconcileBusinessEventList(
        controller: controller,
        lease: lease,
        fence: controller.captureRead(lease),
        owner: created.did,
        authoritative: const [],
      ).events.single.name,
      'Created',
    );
    reconcileBusinessEventList(
      controller: controller,
      lease: lease,
      fence: controller.captureRead(lease),
      owner: created.did,
      authoritative: [created],
    );
    expect(container.read(businessProjectionOverlayProvider), isEmpty);

    generation = controller.beginMutation(key, lease);
    controller.acceptDelete(
      key: key,
      lease: lease,
      requestGeneration: generation,
      deletedCid: created.cid,
      acceptedCid: Cid.parse('bafy-delete-response'),
    );
    expect(
      reconcileBusinessEventList(
        controller: controller,
        lease: lease,
        fence: controller.captureRead(lease),
        owner: created.did,
        authoritative: [created],
      ).events,
      isEmpty,
    );
    final recreated = _event(cid: 'bafy-recreated', name: 'Recreated');
    expect(
      reconcileBusinessEventList(
        controller: controller,
        lease: lease,
        fence: controller.captureRead(lease),
        owner: created.did,
        authoritative: [recreated],
      ).events.single.name,
      'Recreated',
    );
  });

  test('IT-013 a newer account generation supersedes an older read fence', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final lease = _lease();
    final oldRead = controller.captureRead(lease);

    controller.beginMutation(_eventKey(), lease);

    expect(controller.isReadCurrent(oldRead), isFalse);
    expect(
      controller.reconcile<String>(
        key: _eventKey(),
        fence: oldRead,
        authoritativeCid: Cid.parse('bafy-stale'),
        authoritativeView: 'stale',
      ),
      isA<BusinessProjectionReconciliation<String>>()
          .having((result) => result.isStale, 'is stale', isTrue)
          .having((result) => result.view, 'view', isNull),
    );
  });

  test('UT-015 list reconciliation reports any stale returned record', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(
      businessProjectionOverlayProvider.notifier,
    );
    final lease = _lease();
    final fence = controller.captureRead(lease);
    controller.beginMutation(_eventKey(), lease);

    final result = reconcileBusinessEventList(
      controller: controller,
      lease: lease,
      fence: fence,
      owner: Did.parse('did:plc:owner'),
      authoritative: [_event(cid: 'bafy-stale', name: 'Stale')],
    );

    expect(result.isStale, isTrue);
    expect(result.events, isEmpty);
  });

  test(
    'IT-013 a pre-mutation detail read cannot discard an accepted overlay',
    () async {
      final read = Completer<BusinessEvent>();
      final repository = _DetailRepository(read.future);
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      final target = BusinessEventDetailTarget(
        account: AccountKey('did:plc:account'),
        owner: Did.parse('did:plc:owner'),
        rkey: RecordKey.parse('3m4event'),
      );
      final pending = container.read(
        businessEventDetailProvider(target).future,
      );
      await repository.started.future;

      final controller = container.read(
        businessProjectionOverlayProvider.notifier,
      );
      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        target.account,
        target.owner,
        target.rkey,
      );
      final generation = controller.beginMutation(key, lease);
      expect(
        controller.acceptUpsert(
          key: key,
          lease: lease,
          requestGeneration: generation,
          preWriteCid: Cid.parse('bafy-before'),
          acceptedCid: Cid.parse('bafy-accepted'),
          acceptedView: _event(cid: 'bafy-accepted', name: 'Accepted'),
        ),
        isTrue,
      );

      read.complete(_event(cid: 'bafy-third', name: 'Obsolete third CID'));

      expect(
        await pending,
        isA<BusinessEventDetailAvailable>().having(
          (state) => state.event.name,
          'event name',
          'Accepted',
        ),
      );
      expect(container.read(businessProjectionOverlayProvider), contains(key));
      expect(
        container.read(businessProjectionOverlayProvider)[key]?.acceptedView,
        isA<BusinessEvent>().having(
          (event) => event.name,
          'name',
          'Accepted',
        ),
      );
    },
  );

  test(
    'IT-013 a pre-mutation detail not-found cannot discard accepted state',
    () async {
      final read = Completer<BusinessEvent>();
      final repository = _DetailRepository(read.future);
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      final target = BusinessEventDetailTarget(
        account: AccountKey('did:plc:account'),
        owner: Did.parse('did:plc:owner'),
        rkey: RecordKey.parse('3m4event'),
      );
      final pending = container.read(
        businessEventDetailProvider(target).future,
      );
      await repository.started.future;

      final controller = container.read(
        businessProjectionOverlayProvider.notifier,
      );
      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        target.account,
        target.owner,
        target.rkey,
      );
      final generation = controller.beginMutation(key, lease);
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: Cid.parse('bafy-accepted'),
        acceptedView: _event(cid: 'bafy-accepted', name: 'Accepted'),
      );

      read.completeError(const ApiBadRequest('event_not_found'));

      expect(
        await pending,
        isA<BusinessEventDetailAvailable>().having(
          (state) => state.event.name,
          'event name',
          'Accepted',
        ),
      );
      expect(container.read(businessProjectionOverlayProvider), contains(key));
    },
  );

  test(
    'IT-013 a pre-mutation detail error cannot replace accepted state',
    () async {
      final read = Completer<BusinessEvent>();
      final repository = _DetailRepository(read.future);
      final container = ProviderContainer(
        overrides: [businessRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      final target = BusinessEventDetailTarget(
        account: AccountKey('did:plc:account'),
        owner: Did.parse('did:plc:owner'),
        rkey: RecordKey.parse('3m4event'),
      );
      final pending = container.read(
        businessEventDetailProvider(target).future,
      );
      await repository.started.future;

      final controller = container.read(
        businessProjectionOverlayProvider.notifier,
      );
      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final key = BusinessProjectionKey.event(
        target.account,
        target.owner,
        target.rkey,
      );
      final generation = controller.beginMutation(key, lease);
      controller.acceptUpsert(
        key: key,
        lease: lease,
        requestGeneration: generation,
        preWriteCid: Cid.parse('bafy-before'),
        acceptedCid: Cid.parse('bafy-accepted'),
        acceptedView: _event(cid: 'bafy-accepted', name: 'Accepted'),
      );

      read.completeError(StateError('obsolete failure'));

      expect(
        await pending,
        isA<BusinessEventDetailAvailable>().having(
          (state) => state.event.name,
          'event name',
          'Accepted',
        ),
      );
      expect(container.read(businessProjectionOverlayProvider), contains(key));
    },
  );
}

AccountSessionLease _lease({int generation = 1}) => AccountSessionLease(
  account: AccountKey('did:plc:account'),
  sessionGeneration: generation,
);

BusinessProjectionKey _eventKey() => BusinessProjectionKey.event(
  AccountKey('did:plc:account'),
  Did.parse('did:plc:owner'),
  RecordKey.parse('3m4event'),
);

BusinessEvent _event({
  required String cid,
  required String name,
  BusinessImageView? image,
}) => BusinessEvent(
  did: 'did:plc:owner',
  rkey: '3m4event',
  uri: 'at://did:plc:owner/social.craftsky.business.event/3m4event',
  cid: cid,
  name: name,
  startsAt: DateTime.utc(2026, 9, 5, 10),
  endsAt: DateTime.utc(2026, 9, 5, 12),
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  mode: const BusinessOpenValue(value: 'in-person', known: true),
  status: const BusinessOpenValue(value: 'scheduled', known: true),
  timeZone: 'UTC',
  isAllDay: false,
  image: image,
  createdAt: DateTime.utc(2026, 8, 30),
  past: false,
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);

BusinessImageView _networkImage(String suffix) => BusinessImageView(
  cid: 'bafy-image-$suffix',
  mime: 'image/jpeg',
  size: 10,
  alt: '$suffix poster',
  thumb: 'https://cdn.example/$suffix-thumb',
  fullsize: 'https://cdn.example/$suffix-full',
);

final class _DetailRepository extends Fake implements BusinessRepository {
  _DetailRepository(this.result);

  final Future<BusinessEvent> result;
  final started = Completer<void>();

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) {
    if (!started.isCompleted) started.complete();
    return result;
  }
}
