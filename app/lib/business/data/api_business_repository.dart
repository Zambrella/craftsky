import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

class ApiBusinessRepository implements BusinessRepository {
  const ApiBusinessRepository(this._api, this._timeZones);

  final BusinessApiClient _api;
  final BusinessTimeZoneService _timeZones;

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required String operationKey,
    required Cid? expectedCid,
  }) => _api.putBusinessProfile(
    body,
    operationKey: operationKey,
    expectedCid: expectedCid,
  );

  @override
  Future<void> deleteBusinessProfile({
    required String operationKey,
    required Cid expectedCid,
  }) => _api.deleteBusinessProfile(
    operationKey: operationKey,
    expectedCid: expectedCid,
  );

  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) => _api.listProfileEvents(owner, cursor: cursor, limit: limit);

  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) => _api.listOwnerEvents(filter, cursor: cursor, limit: limit);

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) =>
      _api.getEvent(owner, rkey);

  @override
  Future<RecordMutationResult> createEvent(
    BusinessEventDraft draft, {
    required String operationKey,
  }) => _api.createEvent(
    draft.toCreateJson(_timeZones),
    operationKey: operationKey,
  );

  @override
  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    BusinessEventDraft draft, {
    required String operationKey,
  }) => _api.updateEvent(
    owner,
    rkey,
    expectedCid,
    draft.toUpdateJson(_timeZones),
    operationKey: operationKey,
  );

  @override
  Future<void> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid, {
    required String operationKey,
  }) => _api.deleteEvent(
    owner,
    rkey,
    expectedCid,
    operationKey: operationKey,
  );

  @override
  Future<ReportResult> reportEvent(
    Did owner,
    RecordKey rkey,
    ReportSubmission submission,
  ) => _api.reportEvent(owner, rkey, submission);
}
