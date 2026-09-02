import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

class ApiBusinessRepository implements BusinessRepository {
  const ApiBusinessRepository(this._api, this._timeZones);

  final BusinessApiClient _api;
  final BusinessTimeZoneService _timeZones;

  @override
  Future<AccountType> updateAccountType(AccountType value) =>
      _api.updateAccountType(value);

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) => _api.putBusinessProfile(body, expectedCid: expectedCid);

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
  Future<RecordMutationResult> createEvent(BusinessEventDraft draft) =>
      _api.createEvent(draft.toCreateJson(_timeZones));

  @override
  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    BusinessEventDraft draft,
  ) => _api.updateEvent(
    owner,
    rkey,
    expectedCid,
    draft.toUpdateJson(_timeZones),
  );

  @override
  Future<RecordMutationResult> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
  ) => _api.deleteEvent(owner, rkey, expectedCid);

  @override
  Future<ReportResult> reportEvent(
    Did owner,
    RecordKey rkey,
    ReportSubmission submission,
  ) => _api.reportEvent(owner, rkey, submission);
}
