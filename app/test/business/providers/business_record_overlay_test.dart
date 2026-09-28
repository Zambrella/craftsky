import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('shared append overlay inserts an accepted create for its lease', () {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final container = ProviderContainer(
      overrides: [
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    addTearDown(container.dispose);
    final accepted = _event(cid: 'bafy-created', name: 'Created');
    final scope = businessEventMutationScope(
      _lease,
      accepted.uri.toString(),
    );
    final token = controller.begin(
      scope: businessEventCreateMutationScope(_lease),
      operationKey: 'create-key',
      endpoint: '/v1/events',
      immutableBody: '{}',
    );
    final reconciliation = PdsAppendReconciliation(
      selectedUri: accepted.uri.toString(),
      acceptedCid: accepted.cid.toString(),
      controlledContent: businessEventProjection(accepted).content,
    );
    controller.markAccepted(
      token,
      overlayScope: scope,
      optimisticValue: accepted,
      agrees: (value) =>
          value is PdsRecordProjection && reconciliation.agrees(value),
    );

    final result = _withRef(
      container,
      (ref) => applyBusinessEventListOverlays(
        ref,
        lease: _lease,
        fence: captureBusinessEventListRead(ref, _lease),
        owner: accepted.did,
        authoritative: const [],
      ),
    );
    final otherLeaseResult = _withRef(
      container,
      (ref) => applyBusinessEventListOverlays(
        ref,
        lease: _otherLease,
        fence: captureBusinessEventListRead(ref, _otherLease),
        owner: accepted.did,
        authoritative: const [],
      ),
    );

    expect(result.events.single.name, 'Created');
    expect(otherLeaseResult.events, isEmpty);
  });

  test('shared addressed overlays mask stale update and delete reads', () {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final container = ProviderContainer(
      overrides: [
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    addTearDown(container.dispose);
    final stale = _event(cid: 'bafy-before', name: 'Before');
    final accepted = _event(cid: 'bafy-after', name: 'After');
    final scope = businessEventMutationScope(_lease, stale.uri.toString());
    var token = controller.begin(
      scope: scope,
      operationKey: 'update-key',
      endpoint: '/v1/events/event',
      immutableBody: '{}',
    );
    final update = PdsAddressedUpdateReconciliation(
      uri: accepted.uri.toString(),
      acceptedCid: accepted.cid.toString(),
      controlledContent: businessEventProjection(accepted).content,
    );
    controller.markAccepted(
      token,
      optimisticValue: accepted,
      agrees: (value) => value is PdsRecordProjection && update.agrees(value),
    );

    expect(
      _withRef(
        container,
        (ref) => applyBusinessEventOverlay(
          ref,
          _lease,
          stale.uri.toString(),
          stale,
        ),
      )?.name,
      'After',
    );
    expect(
      _withRef(
        container,
        (ref) => applyBusinessEventOverlay(
          ref,
          _otherLease,
          stale.uri.toString(),
          stale,
        ),
      ),
      same(stale),
    );

    token = controller.begin(
      scope: scope,
      operationKey: 'delete-key',
      endpoint: '/v1/events/event',
      immutableBody: 'DELETE',
    );
    final deletion = PdsAddressedDeleteReconciliation(
      uri: stale.uri.toString(),
    );
    controller.markAccepted(
      token,
      optimisticValue: null,
      agrees: (value) => deletion.agrees(
        value is PdsRecordProjection ? value : null,
      ),
    );

    expect(
      _withRef(
        container,
        (ref) => applyBusinessEventOverlay(
          ref,
          _lease,
          stale.uri.toString(),
          stale,
        ),
      ),
      isNull,
    );
    _withRef(
      container,
      (ref) => applyBusinessEventOverlay(
        ref,
        _lease,
        stale.uri.toString(),
        null,
      ),
    );
    expect(controller.activeOverlays, isEmpty);
  });

  test('fixed profile controlled fields settle through the shared overlay', () {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final container = ProviderContainer(
      overrides: [
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    addTearDown(container.dispose);
    final accepted = BusinessProfile(
      cid: 'bafy-accepted',
      businessTypes: const [
        BusinessOpenValue(value: 'teacher', known: true),
      ],
      tagline: 'Accepted',
    );
    final scope = businessProfileMutationScope(_lease, _owner);
    final token = controller.begin(
      scope: scope,
      operationKey: 'profile-key',
      endpoint: '/v1/profiles/me/business',
      immutableBody: '{}',
    );
    final controlled = businessProfileProjection(_owner, accepted).content;
    final reconciliation = PdsFixedKeyReconciliation(
      uri: 'at://$_owner/social.craftsky.business.profile/self',
      controlledContent: controlled,
    );
    controller.markAccepted(
      token,
      optimisticValue: accepted,
      agrees: (value) =>
          value is PdsRecordProjection && reconciliation.agrees(value),
    );

    expect(
      _withRef(
        container,
        (ref) => applyBusinessProfileOverlay(ref, _lease, _owner, null),
      ),
      same(accepted),
    );
    final projected = BusinessProfile(
      cid: 'bafy-projected',
      businessTypes: accepted.businessTypes,
      tagline: accepted.tagline,
    );
    expect(
      _withRef(
        container,
        (ref) => applyBusinessProfileOverlay(
          ref,
          _lease,
          _owner,
          projected,
        ),
      ),
      same(projected),
    );
    expect(controller.activeOverlays, isEmpty);
  });

  test('business read fences ignore unrelated shared mutation scopes', () {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final container = ProviderContainer(
      overrides: [
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    addTearDown(container.dispose);
    final fence = _withRef(
      container,
      (ref) => captureBusinessEventListRead(ref, _lease),
    );

    controller.begin(
      scope: PdsMutationScope(lease: _lease, identity: 'like:post'),
      operationKey: 'like-key',
      endpoint: '/v1/posts/post/likes',
      immutableBody: '{}',
    );

    expect(
      _withRef(
        container,
        (ref) => isBusinessRecordReadCurrent(ref, fence),
      ),
      isTrue,
    );
  });
}

T _withRef<T>(ProviderContainer container, T Function(Ref ref) operation) =>
    container.read(Provider<T>(operation));

final _owner = Did.parse('did:plc:owner');

final _lease = AccountSessionLease(
  account: AccountKey('did:plc:viewer'),
  sessionGeneration: 1,
);

final _otherLease = AccountSessionLease(
  account: AccountKey('did:plc:viewer'),
  sessionGeneration: 2,
);

BusinessEvent _event({required String cid, required String name}) =>
    BusinessEvent(
      did: _owner.toString(),
      rkey: 'event',
      uri: 'at://$_owner/social.craftsky.business.event/event',
      cid: cid,
      name: name,
      startsAt: DateTime.utc(2030, 9, 5, 9),
      endsAt: DateTime.utc(2030, 9, 5, 17),
      roles: const [BusinessOpenValue(value: 'vendor', known: true)],
      mode: const BusinessOpenValue(value: 'in-person', known: true),
      status: const BusinessOpenValue(value: 'scheduled', known: true),
      timeZone: 'UTC',
      isAllDay: false,
      createdAt: DateTime.utc(2026, 8, 30),
      past: false,
      publicSuppressionReasons: const [],
      upcomingExclusionReasons: const [],
    );
