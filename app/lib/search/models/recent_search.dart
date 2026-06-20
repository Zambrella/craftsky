import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/search_sort.dart';

enum RecentSearchType {
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
    RecentSearchType.hashtag => HashtagRecentSearchPayload(
      tag: map['tag'] as String,
      sort: _sortFromMap(map),
    ),
    RecentSearchType.profile => ProfileRecentSearchPayload(
      q: map['q'] as String,
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

class HashtagRecentSearchPayload extends RecentSearchPayload {
  const HashtagRecentSearchPayload({
    required this.tag,
    this.sort = SearchSort.chronological,
  });

  final String tag;
  final SearchSort sort;

  @override
  Map<String, dynamic> toMap() => {'tag': tag, 'sort': sort.wireValue};
}

class ProfileRecentSearchPayload extends RecentSearchPayload {
  const ProfileRecentSearchPayload({required this.q});

  final String q;

  @override
  Map<String, dynamic> toMap() => {'q': q};
}

class PostRecentSearchPayload extends RecentSearchPayload {
  const PostRecentSearchPayload({
    required this.q,
    this.sort = SearchSort.chronological,
  });

  final String q;
  final SearchSort sort;

  @override
  Map<String, dynamic> toMap() => {'q': q, 'sort': sort.wireValue};
}

class ProjectRecentSearchPayload extends RecentSearchPayload {
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

class SaveRecentSearchRequest {
  const SaveRecentSearchRequest({
    required this.type,
    required this.displayLabel,
    required this.payload,
  });

  final RecentSearchType type;
  final String displayLabel;
  final RecentSearchPayload payload;

  Map<String, dynamic> toMap() => {
    'type': type.wireValue,
    'displayLabel': displayLabel,
    'payload': payload.toMap(),
  };
}

class RecentSearchItem {
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

class RecentSearchPage {
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
