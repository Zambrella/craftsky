import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-013 a pre-mutation detail read cannot discard an accepted overlay',
    () async {
      final read = Completer<BusinessEvent>();
      final repository = _DetailRepository(read.future);
      final container = _detailContainer(repository);
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

      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final controller = container.read(pdsRecordOperationControllerProvider);
      _installSharedEventOverlay(
        controller,
        lease,
        _event(cid: 'bafy-accepted', name: 'Accepted'),
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
      expect(
        controller.activeOverlays.single.optimisticValue,
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
      final container = _detailContainer(repository);
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

      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final controller = container.read(pdsRecordOperationControllerProvider);
      _installSharedEventOverlay(
        controller,
        lease,
        _event(cid: 'bafy-accepted', name: 'Accepted'),
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
      expect(controller.activeOverlays, isNotEmpty);
    },
  );

  test(
    'IT-013 a pre-mutation detail error cannot replace accepted state',
    () async {
      final read = Completer<BusinessEvent>();
      final repository = _DetailRepository(read.future);
      final container = _detailContainer(repository);
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

      final lease = AccountSessionLease(
        account: target.account,
        sessionGeneration: 0,
      );
      final controller = container.read(pdsRecordOperationControllerProvider);
      _installSharedEventOverlay(
        controller,
        lease,
        _event(cid: 'bafy-accepted', name: 'Accepted'),
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
      expect(controller.activeOverlays, isNotEmpty);
    },
  );
}

ProviderContainer _detailContainer(_DetailRepository repository) =>
    ProviderContainer(
      overrides: [
        businessRepositoryProvider.overrideWithValue(repository),
        pdsRecordOperationControllerProvider.overrideWithValue(
          PdsRecordOperationController(schedule: (_, _) {}),
        ),
      ],
    );

void _installSharedEventOverlay(
  PdsRecordOperationController controller,
  AccountSessionLease lease,
  BusinessEvent accepted,
) {
  final scope = businessEventMutationScope(lease, accepted.uri.toString());
  final token = controller.begin(
    scope: scope,
    operationKey: 'test-${accepted.cid}',
    endpoint: '/test',
    immutableBody: '{}',
  );
  final reconciliation = PdsAddressedUpdateReconciliation(
    uri: accepted.uri.toString(),
    acceptedCid: accepted.cid.toString(),
    controlledContent: businessEventProjection(accepted).content,
  );
  controller.markAccepted(
    token,
    optimisticValue: accepted,
    agrees: (value) =>
        value is PdsRecordProjection && reconciliation.agrees(value),
  );
}

BusinessEvent _event({required String cid, required String name}) =>
    BusinessEvent(
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
      createdAt: DateTime.utc(2026, 8, 30),
      past: false,
      publicSuppressionReasons: const [],
      upcomingExclusionReasons: const [],
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
