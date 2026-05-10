import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'interaction_write_response.mapper.dart';

@MappableClass()
class InteractionWriteResponse with InteractionWriteResponseMappable {
  const InteractionWriteResponse({
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.subject,
    required this.createdAt,
  });

  final String uri;
  final String cid;
  final String rkey;
  final PostRef subject;
  final DateTime createdAt;
}
