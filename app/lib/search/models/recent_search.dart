import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'recent_search.mapper.dart';

@MappableEnum()
enum RecentSearchType {
  query,
  hashtag,
  profile,
  post,
  project;

  String get wireValue => name;

  static RecentSearchType fromWire(String value) =>
      RecentSearchType.values.singleWhere((type) => type.wireValue == value);
}

sealed class RecentSearchPayload {
  const RecentSearchPayload();

  Map<String, dynamic> toMap();

  static RecentSearchPayload fromMap(
    RecentSearchType type,
    Map<String, dynamic> map,
  ) => switch (type) {
    RecentSearchType.query => QueryRecentSearchPayload(q: map['q'] as String),
    RecentSearchType.hashtag => HashtagRecentSearchPayload(
      tag: map['tag'] as String,
    ),
    RecentSearchType.profile => ProfileRecentSearchPayload(
      did: map['did'] as String,
      handle: map['handle'] as String,
      displayName: map['displayName'] as String?,
      avatar: map['avatar'] as String?,
    ),
    RecentSearchType.post => PostRecentSearchPayload(
      q: map['q'] as String,
      sort: _sortFromMap(map),
    ),
    RecentSearchType.project => ProjectRecentSearchPayload(
      q: map['q'] as String?,
      sort: _sortFromMap(map),
      filters: ProjectSearchFilters.fromMap(
        Map<String, dynamic>.from((map['filters'] as Map?) ?? const {}),
      ),
    ),
  };
}

@MappableClass()
class QueryRecentSearchPayload extends RecentSearchPayload
    with QueryRecentSearchPayloadMappable {
  const QueryRecentSearchPayload({required this.q});

  final String q;

  @override
  Map<String, dynamic> toMap() => {'q': q};
}

@MappableClass()
class HashtagRecentSearchPayload extends RecentSearchPayload
    with HashtagRecentSearchPayloadMappable {
  const HashtagRecentSearchPayload({required this.tag});

  final String tag;

  @override
  Map<String, dynamic> toMap() => {'tag': tag};
}

@MappableClass()
class ProfileRecentSearchPayload extends RecentSearchPayload
    with ProfileRecentSearchPayloadMappable {
  const ProfileRecentSearchPayload({
    required this.did,
    required this.handle,
    this.displayName,
    this.avatar,
  });

  final String did;
  final String handle;
  final String? displayName;
  final String? avatar;

  @override
  Map<String, dynamic> toMap() => {
    'did': did,
    'handle': handle,
    'displayName': ?displayName,
    'avatar': ?avatar,
  };
}

@MappableClass()
class PostRecentSearchPayload extends RecentSearchPayload
    with PostRecentSearchPayloadMappable {
  const PostRecentSearchPayload({
    required this.q,
    this.sort = SearchSort.chronological,
  });

  final String q;
  final SearchSort sort;

  @override
  Map<String, dynamic> toMap() => {'q': q, 'sort': sort.wireValue};
}

@MappableClass()
class ProjectRecentSearchPayload extends RecentSearchPayload
    with ProjectRecentSearchPayloadMappable {
  const ProjectRecentSearchPayload({
    this.q,
    this.sort = SearchSort.chronological,
    this.filters = const ProjectSearchFilters(),
  });

  final String? q;
  final SearchSort sort;
  final ProjectSearchFilters filters;

  @override
  Map<String, dynamic> toMap() => {
    'q': ?q,
    'sort': sort.wireValue,
    if (filters.toPayloadMap().isNotEmpty) 'filters': filters.toPayloadMap(),
  };
}

@MappableClass()
class SaveRecentSearchRequest with SaveRecentSearchRequestMappable {
  const SaveRecentSearchRequest({
    required this.type,
    required this.displayLabel,
    required this.payload,
  });

  final RecentSearchType type;
  final String displayLabel;
  final RecentSearchPayload payload;

  @override
  Map<String, dynamic> toMap() => {
    'type': type.wireValue,
    'displayLabel': displayLabel,
    'payload': payload.toMap(),
  };
}

@MappableClass()
class RecentSearchItem with RecentSearchItemMappable {
  const RecentSearchItem({
    required this.id,
    required this.type,
    required this.displayLabel,
    required this.payload,
    required this.updatedAt,
  });

  factory RecentSearchItem.fromMap(Map<String, dynamic> map) {
    final type = RecentSearchType.fromWire(map['type'] as String);
    return RecentSearchItem(
      id: map['id'] as String,
      type: type,
      displayLabel: map['displayLabel'] as String,
      payload: RecentSearchPayload.fromMap(
        type,
        Map<String, dynamic>.from(map['payload'] as Map),
      ),
      updatedAt: DateTime.parse(map['updatedAt'] as String),
    );
  }

  final String id;
  final RecentSearchType type;
  final String displayLabel;
  final RecentSearchPayload payload;
  final DateTime updatedAt;
}

@MappableClass()
class RecentSearchPage with RecentSearchPageMappable {
  const RecentSearchPage({required this.items});

  factory RecentSearchPage.fromMap(Map<String, dynamic> map) =>
      RecentSearchPage(
        items: [
          for (final item in map['items'] as List)
            RecentSearchItem.fromMap(Map<String, dynamic>.from(item as Map)),
        ],
      );

  final List<RecentSearchItem> items;
}

SearchSort _sortFromMap(Map<String, dynamic> map) {
  final wire = map['sort'] as String? ?? SearchSort.chronological.wireValue;
  return SearchSort.values.singleWhere((sort) => sort.wireValue == wire);
}
