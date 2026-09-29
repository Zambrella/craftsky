import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum EventMutationStatus { ready, saving, ambiguous, conflict, error }

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
    (operationKey) => ref
        .read(businessRepositoryProvider)
        .createEvent(draft, operationKey: operationKey),
  );

  Future<bool> update(BusinessEvent event, BusinessEventDraft draft) =>
      _mutateDraft(
        draft,
        (operationKey) => ref
            .read(businessRepositoryProvider)
            .updateEvent(
              event.did,
              event.rkey,
              event.cid,
              draft,
              operationKey: operationKey,
            ),
        event: event,
      );

  Future<bool> changeStatus(BusinessEvent event, String status) {
    final draft = BusinessEventDraft.fromEvent(
      event,
      timeZones: ref.read(businessTimeZoneServiceProvider),
    ).copyWith(status: status);
    return _mutateDraft(
      draft,
      (operationKey) => ref
          .read(businessRepositoryProvider)
          .updateEvent(
            event.did,
            event.rkey,
            event.cid,
            draft,
            operationKey: operationKey,
          ),
      event: event,
      retryOnConflict: (current) => changeStatus(current, status),
    );
  }

  Future<bool> delete(BusinessEvent event, {required bool confirmed}) async {
    if (!confirmed || state.status == EventMutationStatus.saving) return false;
    final ownership = captureActiveAccountOperation(ref);
    final lease = ownership?.session ?? _testLease(event.did);
    final endpoint = '/v1/events/${event.did}/${event.rkey}';
    final immutableBody = 'DELETE\nIf-Match:${event.cid}\n';
    final commandController = ref.read(pdsRecordOperationControllerProvider);
    final commandToken = commandController.beginOrRetry(
      scope: businessEventMutationScope(lease, event.uri.toString()),
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = commandToken.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;
    state = const EventMutationState(status: EventMutationStatus.saving);
    while (true) {
      try {
        await ref
            .read(businessRepositoryProvider)
            .deleteEvent(
              event.did,
              event.rkey,
              event.cid,
              operationKey: operationKey,
            );
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markAmbiguous(
          commandToken,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) {
          state = const EventMutationState(
            status: EventMutationStatus.ambiguous,
          );
          return false;
        }
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !commandController.canRetry(
              commandToken,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          return false;
        }
        continue;
      } on Object catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markFailed(commandToken);
        _setFailure(
          error,
          event,
          retry: (current) => delete(current, confirmed: true),
        );
        return false;
      }
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      final reconciliation = PdsAddressedDeleteReconciliation(
        uri: event.uri.toString(),
      );
      if (!commandController.markAccepted(
        commandToken,
        optimisticValue: null,
        agrees: (value) => reconciliation.agrees(
          value is PdsRecordProjection ? value : null,
        ),
        refresh: _invalidateReads,
      )) {
        return false;
      }
      _conflictedEvent = null;
      _conflictRetry = null;
      state = const EventMutationState();
      _invalidateReads();
      return true;
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
    Future<RecordMutationResult> Function(String operationKey) operation, {
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
    final frozenDraft = draft.copyWith(roles: List.unmodifiable(draft.roles));
    final timeZones = ref.read(businessTimeZoneServiceProvider);
    final body = event == null
        ? frozenDraft.toCreateJson(timeZones)
        : frozenDraft.toUpdateJson(timeZones);
    final immutableBody = jsonEncode(body);
    final endpoint = event == null
        ? '/v1/events'
        : '/v1/events/${event.did}/${event.rkey}';
    final ownership = captureActiveAccountOperation(ref);
    final lease =
        ownership?.session ?? (event == null ? null : _testLease(event.did));
    final commandLease = lease ?? _testLease(Did.parse('did:plc:local-test'));
    final commandController = ref.read(pdsRecordOperationControllerProvider);
    final commandToken = commandController.beginOrRetry(
      scope: event == null
          ? businessEventCreateMutationScope(commandLease)
          : businessEventMutationScope(commandLease, event.uri.toString()),
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = commandToken.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;
    state = const EventMutationState(status: EventMutationStatus.saving);
    while (true) {
      late final RecordMutationResult result;
      try {
        result = await operation(operationKey);
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markAmbiguous(
          commandToken,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) {
          state = const EventMutationState(
            status: EventMutationStatus.ambiguous,
          );
          return false;
        }
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !commandController.canRetry(
              commandToken,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          return false;
        }
        continue;
      } on Object catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markFailed(commandToken);
        _setFailure(
          error,
          event,
          retry:
              retryOnConflict ??
              (event == null
                  ? null
                  : (current) => update(current, frozenDraft)),
        );
        return false;
      }
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      final accepted = _acceptedEvent(result, frozenDraft, event: event);
      if (accepted != null) {
        final acceptedLease = lease ?? _testLease(accepted.did);
        final acceptedScope = businessEventMutationScope(
          acceptedLease,
          accepted.uri.toString(),
        );
        final controlledContent = businessEventControlledContent(
          event == null
              ? frozenDraft.toCreateJson(timeZones)
              : frozenDraft.toUpdateJson(timeZones),
        );
        final reconciliation = event == null
            ? PdsAppendReconciliation(
                selectedUri: accepted.uri.toString(),
                acceptedCid: result.cid.toString(),
                controlledContent: controlledContent,
              )
            : PdsAddressedUpdateReconciliation(
                uri: accepted.uri.toString(),
                acceptedCid: result.cid.toString(),
                controlledContent: controlledContent,
              );
        if (!commandController.markAccepted(
          commandToken,
          overlayScope: acceptedScope,
          optimisticValue: accepted,
          agrees: (value) =>
              value is PdsRecordProjection && reconciliation.agrees(value),
          refresh: _invalidateReads,
        )) {
          return false;
        }
      } else if (!commandController.markAcceptedWithoutOverlay(commandToken)) {
        return false;
      }
      _conflictedEvent = null;
      _conflictRetry = null;
      state = const EventMutationState();
      _invalidateReads();
      return true;
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
