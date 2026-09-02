import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum EventMutationStatus { ready, saving, conflict, error }

@immutable
class EventMutationState {
  const EventMutationState({
    this.status = EventMutationStatus.ready,
    this.serverEvent,
    this.validationErrors = const {},
  });

  final EventMutationStatus status;
  final BusinessEvent? serverEvent;
  final Set<EventDraftError> validationErrors;
}

final businessEventMutationControllerProvider =
    NotifierProvider<BusinessEventMutationController, EventMutationState>(
      BusinessEventMutationController.new,
    );

class BusinessEventMutationController extends Notifier<EventMutationState> {
  BusinessEvent? _conflictedEvent;
  Future<bool> Function(BusinessEvent current)? _conflictRetry;

  @override
  EventMutationState build() => const EventMutationState();

  Future<bool> create(BusinessEventDraft draft) => _mutateDraft(
    draft,
    () => ref.read(businessRepositoryProvider).createEvent(draft),
  );

  Future<bool> update(BusinessEvent event, BusinessEventDraft draft) =>
      _mutateDraft(
        draft,
        () => ref
            .read(businessRepositoryProvider)
            .updateEvent(event.did, event.rkey, event.cid, draft),
        event: event,
      );

  Future<bool> changeStatus(BusinessEvent event, String status) {
    final draft = BusinessEventDraft.fromEvent(
      event,
      timeZones: ref.read(businessTimeZoneServiceProvider),
    ).copyWith(status: status);
    return _mutateDraft(
      draft,
      () => ref
          .read(businessRepositoryProvider)
          .updateEvent(event.did, event.rkey, event.cid, draft),
      event: event,
      retryOnConflict: (current) => changeStatus(current, status),
    );
  }

  Future<bool> delete(BusinessEvent event, {required bool confirmed}) async {
    if (!confirmed || state.status == EventMutationStatus.saving) return false;
    final ownership = captureActiveAccountOperation(ref);
    final lease = ownership?.session ?? _testLease(event.did);
    final key = BusinessProjectionKey.event(
      lease.account,
      event.did,
      event.rkey,
    );
    final generation = ref
        .read(businessProjectionOverlayProvider.notifier)
        .beginMutation(key, lease);
    state = const EventMutationState(status: EventMutationStatus.saving);
    try {
      final result = await ref
          .read(businessRepositoryProvider)
          .deleteEvent(event.did, event.rkey, event.cid);
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      if (!ref
          .read(businessProjectionOverlayProvider.notifier)
          .acceptDelete(
            key: key,
            lease: lease,
            requestGeneration: generation,
            deletedCid: event.cid,
            acceptedCid: result.cid,
          )) {
        return false;
      }
      _conflictedEvent = null;
      _conflictRetry = null;
      state = const EventMutationState();
      _invalidateReads();
      return true;
    } on Object catch (error) {
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      _setFailure(
        error,
        event,
        retry: (current) => delete(current, confirmed: true),
      );
      return false;
    }
  }

  Future<BusinessEvent?> reloadConflict() async {
    final event = _conflictedEvent;
    if (event == null || state.status != EventMutationStatus.conflict) {
      return null;
    }
    state = const EventMutationState(status: EventMutationStatus.saving);
    try {
      final current = await ref
          .read(businessRepositoryProvider)
          .getEvent(event.did, event.rkey);
      _conflictedEvent = null;
      _conflictRetry = null;
      state = EventMutationState(serverEvent: current);
      _invalidateReads();
      return current;
    } on Object {
      state = const EventMutationState(status: EventMutationStatus.error);
      return null;
    }
  }

  Future<bool> retryConflict() async {
    final retry = _conflictRetry;
    if (retry == null || state.status != EventMutationStatus.conflict) {
      return false;
    }
    final current = await reloadConflict();
    return current != null && await retry(current);
  }

  Future<bool> _mutateDraft(
    BusinessEventDraft draft,
    Future<RecordMutationResult> Function() operation, {
    BusinessEvent? event,
    Future<bool> Function(BusinessEvent current)? retryOnConflict,
  }) async {
    if (state.status == EventMutationStatus.saving ||
        state.status == EventMutationStatus.conflict) {
      return false;
    }
    final errors = draft.validate(ref.read(businessTimeZoneServiceProvider));
    if (errors.isNotEmpty) {
      state = EventMutationState(
        status: EventMutationStatus.error,
        validationErrors: errors,
      );
      return false;
    }
    final ownership = captureActiveAccountOperation(ref);
    final lease =
        ownership?.session ?? (event == null ? null : _testLease(event.did));
    final key = event == null || lease == null
        ? null
        : BusinessProjectionKey.event(lease.account, event.did, event.rkey);
    final generation = key == null
        ? null
        : ref
              .read(businessProjectionOverlayProvider.notifier)
              .beginMutation(key, lease!);
    state = const EventMutationState(status: EventMutationStatus.saving);
    try {
      final result = await operation();
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      final accepted = _acceptedEvent(result, draft, event: event);
      if (accepted != null) {
        final acceptedLease = lease ?? _testLease(accepted.did);
        final acceptedKey =
            key ??
            BusinessProjectionKey.event(
              acceptedLease.account,
              accepted.did,
              accepted.rkey,
            );
        final acceptedGeneration =
            generation ??
            ref
                .read(businessProjectionOverlayProvider.notifier)
                .beginMutation(acceptedKey, acceptedLease);
        if (!ref
            .read(businessProjectionOverlayProvider.notifier)
            .acceptUpsert(
              key: acceptedKey,
              lease: acceptedLease,
              requestGeneration: acceptedGeneration,
              preWriteCid: event?.cid,
              acceptedCid: result.cid,
              acceptedView: accepted,
            )) {
          return false;
        }
      }
      _conflictedEvent = null;
      _conflictRetry = null;
      state = const EventMutationState();
      _invalidateReads();
      return true;
    } on Object catch (error) {
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      _setFailure(
        error,
        event,
        retry:
            retryOnConflict ??
            (event == null ? null : (current) => update(current, draft)),
      );
      return false;
    }
  }

  BusinessEvent? _acceptedEvent(
    RecordMutationResult result,
    BusinessEventDraft draft, {
    required BusinessEvent? event,
  }) {
    final did = event?.did ?? result.did;
    final rkey = event?.rkey ?? result.rkey;
    final uri = event?.uri ?? result.uri;
    if (did == null || rkey == null || uri == null) return null;
    final timeZones = ref.read(businessTimeZoneServiceProvider);
    final range = draft.utcRange(timeZones);
    final startsAt = DateTime.parse(range.startsAt);
    final endsAt = DateTime.parse(range.endsAt);
    final now = DateTime.now().toUtc();
    final acceptedImage = switch (draft.image) {
      ExistingBusinessImageDraft(:final cid)
          when event?.image?.cid.toString() == cid =>
        event?.image,
      UploadedBusinessImageDraft(
        :final cid,
        :final mime,
        :final size,
        :final alt,
        :final aspectRatio,
        previewBytes: final previewBytes?,
      ) =>
        BusinessImageView.localPreview(
          cid: cid,
          mime: mime,
          size: size,
          alt: alt,
          aspectRatio: aspectRatio,
          previewBytes: previewBytes,
        ),
      _ => null,
    };
    return BusinessEvent(
      did: did.toString(),
      rkey: rkey.toString(),
      uri: uri.toString(),
      cid: result.cid.toString(),
      name: draft.name,
      startsAt: startsAt,
      endsAt: endsAt,
      roles: [
        for (final role in draft.roles)
          BusinessOpenValue(value: role, known: true),
      ],
      mode: BusinessOpenValue(value: draft.mode, known: true),
      status: BusinessOpenValue(value: draft.status, known: true),
      timeZone: draft.timeZone,
      isAllDay: draft.isAllDay,
      summary: draft.summary,
      venueName: draft.venueName,
      eventUri: draft.eventUri,
      registrationUri: draft.registrationUri,
      image: acceptedImage,
      createdAt: event?.createdAt ?? now,
      past: !endsAt.isAfter(now),
      publicSuppressionReasons: event?.publicSuppressionReasons ?? const [],
      upcomingExclusionReasons: event?.upcomingExclusionReasons ?? const [],
    );
  }

  void _setFailure(
    Object error,
    BusinessEvent? event, {
    Future<bool> Function(BusinessEvent current)? retry,
  }) {
    final conflict =
        error is ApiBadRequest &&
        (error.code == 'pds_record_conflict' ||
            error.details.statusCode == 409);
    _conflictedEvent = conflict ? event : null;
    _conflictRetry = conflict ? retry : null;
    state = EventMutationState(
      status: conflict
          ? EventMutationStatus.conflict
          : EventMutationStatus.error,
    );
  }

  void _invalidateReads() {
    ref
      ..invalidate(profileBusinessEventsProvider)
      ..invalidate(ownerBusinessEventsProvider)
      ..invalidate(businessEventDetailProvider);
  }
}

AccountSessionLease _testLease(Did owner) => AccountSessionLease(
  account: AccountKey(owner.toString()),
  sessionGeneration: 0,
);
