import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'timeline_page.mapper.dart';

/// Home timeline page. Unlike profile/search pages, timeline rows are feed
/// items with stable identities and optional reasons.
@MappableClass(ignoreNull: true)
class TimelinePage with TimelinePageMappable {
  const TimelinePage({required this.items, this.cursor});

  final List<TimelineItem> items;
  final String? cursor;
}

@MappableClass(ignoreNull: true)
class TimelineItem with TimelineItemMappable {
  const TimelineItem({
    required this.itemKey,
    required this.post,
    this.reason,
  });

  final String itemKey;
  final Post post;
  final RepostReason? reason;
}

@MappableEnum()
enum RepostReasonType { repost }

@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [AtUriMapper(), CidMapper()],
)
class RepostReason with RepostReasonMappable {
  RepostReason({
    required this.type,
    required this.by,
    required String uri,
    required this.createdAt,
    required this.indexedAt,
    String? cid,
  }) : uri = AtUri.parse(uri),
       cid = cid == null ? null : Cid.parse(cid);

  final RepostReasonType type;
  final PostAuthor by;
  final AtUri uri;
  final Cid? cid;
  final DateTime createdAt;
  final DateTime indexedAt;
}
