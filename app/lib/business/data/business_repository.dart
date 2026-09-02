import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

abstract interface class BusinessRepository {
  Future<AccountType> updateAccountType(AccountType value);

  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  });

  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  });

  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  });

  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey);

  Future<RecordMutationResult> createEvent(BusinessEventDraft draft);

  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    BusinessEventDraft draft,
  );

  Future<RecordMutationResult> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
  );

  Future<ReportResult> reportEvent(
    Did owner,
    RecordKey rkey,
    ReportSubmission submission,
  );
}
