import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'business_event.mapper.dart';

enum OwnerEventFilter { upcoming, history }

@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [
    DidMapper(),
    RecordKeyMapper(),
    AtUriMapper(),
    CidMapper(),
  ],
)
class BusinessEvent with BusinessEventMappable {
  BusinessEvent({
    required String did,
    required String rkey,
    required String uri,
    required String cid,
    required this.name,
    required this.startsAt,
    required this.endsAt,
    required this.roles,
    required this.status,
    required this.isAllDay,
    required this.createdAt,
    required this.past,
    required this.publicSuppressionReasons,
    required this.upcomingExclusionReasons,
    this.mode,
    this.timeZone,
    this.summary,
    this.venueName,
    this.eventUri,
    this.registrationUri,
    this.image,
  }) : did = Did.parse(did),
       rkey = RecordKey.parse(rkey),
       uri = AtUri.parse(uri),
       cid = Cid.parse(cid);

  final Did did;
  final RecordKey rkey;
  final AtUri uri;
  final Cid cid;
  final String name;
  final DateTime startsAt;
  final DateTime endsAt;
  final List<BusinessOpenValue> roles;
  final BusinessOpenValue? mode;
  final BusinessOpenValue status;
  final String? timeZone;
  final bool isAllDay;
  final String? summary;
  final String? venueName;
  final String? eventUri;
  final String? registrationUri;
  final BusinessImageView? image;
  final DateTime createdAt;
  final bool past;
  final List<String> publicSuppressionReasons;
  final List<String> upcomingExclusionReasons;
}

@MappableClass(ignoreNull: true)
class BusinessEventPage with BusinessEventPageMappable {
  const BusinessEventPage({required this.items, this.cursor});

  final List<BusinessEvent> items;
  final String? cursor;
}

@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [
    DidMapper(),
    RecordKeyMapper(),
    AtUriMapper(),
    CidMapper(),
  ],
)
class RecordMutationResult with RecordMutationResultMappable {
  RecordMutationResult({
    required String cid,
    String? did,
    String? rkey,
    String? uri,
  }) : cid = Cid.parse(cid),
       did = did == null ? null : Did.parse(did),
       rkey = rkey == null ? null : RecordKey.parse(rkey),
       uri = uri == null ? null : AtUri.parse(uri);

  final Cid cid;
  final Did? did;
  final RecordKey? rkey;
  final AtUri? uri;
}
