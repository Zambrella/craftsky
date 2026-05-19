import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'interaction_write_response.mapper.dart';

@MappableClass(
  includeCustomMappers: [AtUriMapper(), CidMapper(), RecordKeyMapper()],
)
class InteractionWriteResponse with InteractionWriteResponseMappable {
  InteractionWriteResponse({
    required String uri,
    required String cid,
    required String rkey,
    required this.subject,
    required this.createdAt,
  }) : uri = AtUri.parse(uri),
       cid = Cid.parse(cid),
       rkey = RecordKey.parse(rkey);

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
  final PostRef subject;
  final DateTime createdAt;
}
