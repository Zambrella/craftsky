import 'dart:typed_data';

import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_mutation_controller.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-008 create and update pass complete drafts and current CID',
    () async {
      final repository = _Repository();
      final container = _container(repository);
      addTearDown(container.dispose);
      final controller = container.read(
        businessEventMutationControllerProvider.notifier,
      );

      expect(await controller.create(_draft()), isTrue);
      expect(
        await controller.update(_event(), _draft(status: 'cancelled')),
        isTrue,
      );

      expect(repository.created.single.name, 'Fibre fair');
      expect(repository.updated.single.expectedCid.toString(), 'bafy-current');
      expect(repository.updated.single.draft.status, 'cancelled');
    },
  );

  test(
    'AT-008 successful lifecycle movement restarts both owner views',
    () async {
      final repository = _Repository()
        ..ownerPages = {
          OwnerEventFilter.upcoming: [
            BusinessEventPage(items: [_event()]),
            const BusinessEventPage(items: []),
          ],
          OwnerEventFilter.history: [
            const BusinessEventPage(items: []),
            BusinessEventPage(items: [_event(status: 'cancelled')]),
          ],
        };
      final container = _container(repository);
      addTearDown(container.dispose);
      final upcomingSubscription = container.listen(
        ownerBusinessEventsProvider(OwnerEventFilter.upcoming),
        (_, _) {},
        fireImmediately: true,
      );
      final historySubscription = container.listen(
        ownerBusinessEventsProvider(OwnerEventFilter.history),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(upcomingSubscription.close);
      addTearDown(historySubscription.close);
      await Future.wait([
        container.read(
          ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
        ),
        container.read(
          ownerBusinessEventsProvider(OwnerEventFilter.history).future,
        ),
      ]);

      expect(
        await container
            .read(businessEventMutationControllerProvider.notifier)
            .changeStatus(_event(), 'cancelled'),
        isTrue,
      );
      final restarted = await Future.wait([
        container.read(
          ownerBusinessEventsProvider(OwnerEventFilter.upcoming).future,
        ),
        container.read(
          ownerBusinessEventsProvider(OwnerEventFilter.history).future,
        ),
      ]);

      expect(restarted[0].items, isEmpty);
      expect(restarted[1].items.single.status.value, 'cancelled');
    },
  );

  test('IT-008 cancellation and postponement are PUT edits', () async {
    final repository = _Repository();
    final container = _container(repository);
    addTearDown(container.dispose);
    final controller = container.read(
      businessEventMutationControllerProvider.notifier,
    );

    await controller.changeStatus(_event(), 'cancelled');
    await controller.changeStatus(_event(), 'postponed');

    expect(repository.updated.map((call) => call.draft.status), [
      'cancelled',
      'postponed',
    ]);
    expect(repository.deletes, isEmpty);
  });

  test(
    'AT-008 IT-008 lifecycle conflict retry rebases status onto the '
    'authoritative event',
    () async {
      final authoritativeImage = BusinessImageView(
        cid: 'bafy-authoritative-image',
        mime: 'image/webp',
        size: 456,
        alt: 'Updated event poster',
        thumb: 'https://cdn.example/authoritative-thumb',
        fullsize: 'https://cdn.example/authoritative-full',
      );
      final authoritative = _event(
        cid: 'bafy-authoritative',
        name: 'Concurrent fibre festival',
        startsAt: DateTime.utc(2026, 10, 8, 14),
        endsAt: DateTime.utc(2026, 10, 8, 19),
        summary: 'Changed by another editor',
        venueName: 'New hall',
        eventUri: 'https://example.com/new-event',
        registrationUri: 'https://example.com/new-registration',
        image: authoritativeImage,
      );
      final repository = _Repository()
        ..error = const ApiBadRequest(
          'pds_record_conflict',
          details: ApiFailureDetails(statusCode: 409),
        )
        ..currentEvent = authoritative;
      final container = _container(repository);
      addTearDown(container.dispose);
      final controller = container.read(
        businessEventMutationControllerProvider.notifier,
      );

      expect(await controller.changeStatus(_event(), 'cancelled'), isFalse);
      repository.error = null;
      expect(await controller.retryConflict(), isTrue);

      final retry = repository.updated.last;
      expect(retry.expectedCid.toString(), 'bafy-authoritative');
      expect(retry.draft.name, authoritative.name);
      expect(retry.draft.startsAt, DateTime(2026, 10, 8, 14));
      expect(retry.draft.endsAt, DateTime(2026, 10, 8, 19));
      expect(retry.draft.summary, authoritative.summary);
      expect(retry.draft.venueName, authoritative.venueName);
      expect(retry.draft.eventUri, authoritative.eventUri);
      expect(retry.draft.registrationUri, authoritative.registrationUri);
      expect(
        retry.draft.image.toJson(),
        ExistingBusinessImageDraft(authoritativeImage).toJson(),
      );
      expect(retry.draft.status, 'cancelled');
    },
  );

  test('IT-008 destructive deletion requires confirmation', () async {
    final repository = _Repository();
    final container = _container(repository);
    addTearDown(container.dispose);
    final controller = container.read(
      businessEventMutationControllerProvider.notifier,
    );

    expect(await controller.delete(_event(), confirmed: false), isFalse);
    expect(repository.deletes, isEmpty);
    expect(await controller.delete(_event(), confirmed: true), isTrue);
    expect(repository.deletes.single.expectedCid.toString(), 'bafy-current');
  });

  test(
    'IT-008 stale CID preserves conflict and reloads current event',
    () async {
      final repository = _Repository()
        ..error = const ApiBadRequest(
          'pds_record_conflict',
          details: ApiFailureDetails(statusCode: 409),
        );
      final container = _container(repository);
      addTearDown(container.dispose);
      final controller = container.read(
        businessEventMutationControllerProvider.notifier,
      );

      expect(await controller.update(_event(), _draft()), isFalse);
      expect(
        container.read(businessEventMutationControllerProvider).status,
        EventMutationStatus.conflict,
      );
      repository.error = null;
      final reloaded = await controller.reloadConflict();

      expect(reloaded?.cid.toString(), 'bafy-server');
      expect(repository.gets, 1);
      expect(
        container.read(businessEventMutationControllerProvider).status,
        EventMutationStatus.ready,
      );
    },
  );

  test(
    'IT-013 accepted event replacement and create retain local previews only '
    'in projections',
    () async {
      final repository = _Repository();
      final container = _container(repository);
      addTearDown(container.dispose);
      final controller = container.read(
        businessEventMutationControllerProvider.notifier,
      );
      final replacementBytes = Uint8List.fromList([1, 2, 3]);
      final createBytes = Uint8List.fromList([4, 5, 6]);

      expect(
        await controller.update(
          _event(),
          _draft(
            image: UploadedBusinessImageDraft(
              cid: 'bafy-replacement',
              mime: 'image/png',
              size: 3,
              alt: 'Replacement poster',
              localPreviewBytes: replacementBytes,
            ),
          ),
        ),
        isTrue,
      );
      expect(
        await controller.create(
          _draft(
            image: UploadedBusinessImageDraft(
              cid: 'bafy-created-image',
              mime: 'image/png',
              size: 3,
              alt: 'Created poster',
              localPreviewBytes: createBytes,
            ),
          ),
        ),
        isTrue,
      );

      final acceptedEvents = container
          .read(businessProjectionOverlayProvider)
          .values
          .map((overlay) => overlay.acceptedView)
          .whereType<BusinessEvent>()
          .toList();
      expect(
        acceptedEvents.map((event) => event.image?.previewBytes),
        containsAll(<Uint8List>[replacementBytes, createBytes]),
      );
      for (final draft in [
        repository.updated.single.draft,
        repository.created.single,
      ]) {
        final image =
            draft.toCreateJson(
                  BusinessTimeZoneService.initialized(),
                )['image']
                as Map<String, dynamic>;
        expect(image.keys, unorderedEquals(['image', 'alt']));
        expect(image, isNot(contains('previewBytes')));
        expect(image, isNot(contains('thumb')));
        expect(image, isNot(contains('fullsize')));
      }
    },
  );

  test('AT-008 conflict retry reloads before using the current CID', () async {
    final repository = _Repository()
      ..error = const ApiBadRequest(
        'pds_record_conflict',
        details: ApiFailureDetails(statusCode: 409),
      )
      ..currentEvent = _event(
        cid: 'bafy-server',
        name: 'Concurrent server name',
      );
    final container = _container(repository);
    addTearDown(container.dispose);
    final controller = container.read(
      businessEventMutationControllerProvider.notifier,
    );

    expect(
      await controller.update(_event(), _draft(name: 'User correction')),
      isFalse,
    );
    repository.error = null;
    expect(await controller.retryConflict(), isTrue);

    expect(repository.gets, 1);
    expect(repository.updated.map((call) => call.expectedCid.toString()), [
      'bafy-current',
      'bafy-server',
    ]);
    expect(repository.updated.last.draft.name, 'User correction');
  });
}

ProviderContainer _container(_Repository repository) => ProviderContainer(
  overrides: [
    businessRepositoryProvider.overrideWithValue(repository),
    businessTimeZoneServiceProvider.overrideWithValue(
      BusinessTimeZoneService.initialized(),
    ),
  ],
);

BusinessEventDraft _draft({
  String name = 'Fibre fair',
  String status = 'scheduled',
  BusinessImageDraft image = const MissingBusinessImageDraft(),
}) => BusinessEventDraft(
  name: name,
  startsAt: DateTime(2026, 9, 5, 10),
  endsAt: DateTime(2026, 9, 5, 12),
  roles: const ['vendor'],
  mode: 'in-person',
  status: status,
  timeZone: 'UTC',
  isAllDay: false,
  image: image,
);

BusinessEvent _event({
  String cid = 'bafy-current',
  String name = 'Fibre fair',
  String status = 'scheduled',
  DateTime? startsAt,
  DateTime? endsAt,
  String? summary,
  String? venueName,
  String? eventUri,
  String? registrationUri,
  BusinessImageView? image,
}) => BusinessEvent(
  did: 'did:plc:owner',
  rkey: '3m4event',
  uri: 'at://did:plc:owner/social.craftsky.business.event/3m4event',
  cid: cid,
  name: name,
  startsAt: startsAt ?? DateTime.utc(2026, 9, 5, 10),
  endsAt: endsAt ?? DateTime.utc(2026, 9, 5, 12),
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  mode: const BusinessOpenValue(value: 'in-person', known: true),
  status: BusinessOpenValue(value: status, known: true),
  timeZone: 'UTC',
  isAllDay: false,
  summary: summary,
  venueName: venueName,
  eventUri: eventUri,
  registrationUri: registrationUri,
  image: image,
  createdAt: DateTime.utc(2026, 8, 30),
  past: false,
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);

final class _UpdateCall {
  const _UpdateCall(this.expectedCid, this.draft);
  final Cid expectedCid;
  final BusinessEventDraft draft;
}

final class _DeleteCall {
  const _DeleteCall(this.expectedCid);
  final Cid expectedCid;
}

final class _Repository extends Fake implements BusinessRepository {
  final created = <BusinessEventDraft>[];
  final updated = <_UpdateCall>[];
  final deletes = <_DeleteCall>[];
  Exception? error;
  int gets = 0;
  BusinessEvent? currentEvent;
  Map<OwnerEventFilter, List<BusinessEventPage>> ownerPages = const {};
  final ownerIndices = <OwnerEventFilter, int>{};

  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) async {
    final index = ownerIndices.update(
      filter,
      (value) => value + 1,
      ifAbsent: () => 0,
    );
    return ownerPages[filter]![index];
  }

  @override
  Future<RecordMutationResult> createEvent(BusinessEventDraft draft) async {
    created.add(draft);
    if (error case final value?) throw value;
    return RecordMutationResult(
      did: 'did:plc:owner',
      rkey: 'created',
      uri: 'at://did:plc:owner/social.craftsky.business.event/created',
      cid: 'bafy-created',
    );
  }

  @override
  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    BusinessEventDraft draft,
  ) async {
    updated.add(_UpdateCall(expectedCid, draft));
    if (error case final value?) throw value;
    return RecordMutationResult(cid: 'bafy-updated');
  }

  @override
  Future<RecordMutationResult> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
  ) async {
    deletes.add(_DeleteCall(expectedCid));
    if (error case final value?) throw value;
    return RecordMutationResult(cid: expectedCid.toString());
  }

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) async {
    gets++;
    return currentEvent ?? _event(cid: 'bafy-server');
  }
}
