// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'search_suggestions.dart';

class SearchSuggestionsMapper extends ClassMapperBase<SearchSuggestions> {
  SearchSuggestionsMapper._();

  static SearchSuggestionsMapper? _instance;
  static SearchSuggestionsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SearchSuggestionsMapper._());
      SearchSuggestionProfileSectionMapper.ensureInitialized();
      SearchSuggestionHashtagSectionMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SearchSuggestions';

  static SearchSuggestionProfileSection _$profiles(SearchSuggestions v) =>
      v.profiles;
  static const Field<SearchSuggestions, SearchSuggestionProfileSection>
  _f$profiles = Field('profiles', _$profiles);
  static SearchSuggestionHashtagSection _$hashtags(SearchSuggestions v) =>
      v.hashtags;
  static const Field<SearchSuggestions, SearchSuggestionHashtagSection>
  _f$hashtags = Field('hashtags', _$hashtags);

  @override
  final MappableFields<SearchSuggestions> fields = const {
    #profiles: _f$profiles,
    #hashtags: _f$hashtags,
  };

  static SearchSuggestions _instantiate(DecodingData data) {
    return SearchSuggestions(
      profiles: data.dec(_f$profiles),
      hashtags: data.dec(_f$hashtags),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SearchSuggestions fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SearchSuggestions>(map);
  }

  static SearchSuggestions fromJson(String json) {
    return ensureInitialized().decodeJson<SearchSuggestions>(json);
  }
}

mixin SearchSuggestionsMappable {
  String toJson() {
    return SearchSuggestionsMapper.ensureInitialized()
        .encodeJson<SearchSuggestions>(this as SearchSuggestions);
  }

  Map<String, dynamic> toMap() {
    return SearchSuggestionsMapper.ensureInitialized()
        .encodeMap<SearchSuggestions>(this as SearchSuggestions);
  }

  SearchSuggestionsCopyWith<
    SearchSuggestions,
    SearchSuggestions,
    SearchSuggestions
  >
  get copyWith =>
      _SearchSuggestionsCopyWithImpl<SearchSuggestions, SearchSuggestions>(
        this as SearchSuggestions,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return SearchSuggestionsMapper.ensureInitialized().stringifyValue(
      this as SearchSuggestions,
    );
  }

  @override
  bool operator ==(Object other) {
    return SearchSuggestionsMapper.ensureInitialized().equalsValue(
      this as SearchSuggestions,
      other,
    );
  }

  @override
  int get hashCode {
    return SearchSuggestionsMapper.ensureInitialized().hashValue(
      this as SearchSuggestions,
    );
  }
}

extension SearchSuggestionsValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SearchSuggestions, $Out> {
  SearchSuggestionsCopyWith<$R, SearchSuggestions, $Out>
  get $asSearchSuggestions => $base.as(
    (v, t, t2) => _SearchSuggestionsCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SearchSuggestionsCopyWith<
  $R,
  $In extends SearchSuggestions,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  SearchSuggestionProfileSectionCopyWith<
    $R,
    SearchSuggestionProfileSection,
    SearchSuggestionProfileSection
  >
  get profiles;
  SearchSuggestionHashtagSectionCopyWith<
    $R,
    SearchSuggestionHashtagSection,
    SearchSuggestionHashtagSection
  >
  get hashtags;
  $R call({
    SearchSuggestionProfileSection? profiles,
    SearchSuggestionHashtagSection? hashtags,
  });
  SearchSuggestionsCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SearchSuggestionsCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SearchSuggestions, $Out>
    implements SearchSuggestionsCopyWith<$R, SearchSuggestions, $Out> {
  _SearchSuggestionsCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SearchSuggestions> $mapper =
      SearchSuggestionsMapper.ensureInitialized();
  @override
  SearchSuggestionProfileSectionCopyWith<
    $R,
    SearchSuggestionProfileSection,
    SearchSuggestionProfileSection
  >
  get profiles => $value.profiles.copyWith.$chain((v) => call(profiles: v));
  @override
  SearchSuggestionHashtagSectionCopyWith<
    $R,
    SearchSuggestionHashtagSection,
    SearchSuggestionHashtagSection
  >
  get hashtags => $value.hashtags.copyWith.$chain((v) => call(hashtags: v));
  @override
  $R call({
    SearchSuggestionProfileSection? profiles,
    SearchSuggestionHashtagSection? hashtags,
  }) => $apply(
    FieldCopyWithData({
      if (profiles != null) #profiles: profiles,
      if (hashtags != null) #hashtags: hashtags,
    }),
  );
  @override
  SearchSuggestions $make(CopyWithData data) => SearchSuggestions(
    profiles: data.get(#profiles, or: $value.profiles),
    hashtags: data.get(#hashtags, or: $value.hashtags),
  );

  @override
  SearchSuggestionsCopyWith<$R2, SearchSuggestions, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SearchSuggestionsCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SearchSuggestionProfileSectionMapper
    extends ClassMapperBase<SearchSuggestionProfileSection> {
  SearchSuggestionProfileSectionMapper._();

  static SearchSuggestionProfileSectionMapper? _instance;
  static SearchSuggestionProfileSectionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = SearchSuggestionProfileSectionMapper._(),
      );
      ProfileSearchResultMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SearchSuggestionProfileSection';

  static List<ProfileSearchResult> _$items(SearchSuggestionProfileSection v) =>
      v.items;
  static const Field<SearchSuggestionProfileSection, List<ProfileSearchResult>>
  _f$items = Field('items', _$items);
  static bool _$hasMore(SearchSuggestionProfileSection v) => v.hasMore;
  static const Field<SearchSuggestionProfileSection, bool> _f$hasMore = Field(
    'hasMore',
    _$hasMore,
  );

  @override
  final MappableFields<SearchSuggestionProfileSection> fields = const {
    #items: _f$items,
    #hasMore: _f$hasMore,
  };

  static SearchSuggestionProfileSection _instantiate(DecodingData data) {
    return SearchSuggestionProfileSection(
      items: data.dec(_f$items),
      hasMore: data.dec(_f$hasMore),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SearchSuggestionProfileSection fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SearchSuggestionProfileSection>(map);
  }

  static SearchSuggestionProfileSection fromJson(String json) {
    return ensureInitialized().decodeJson<SearchSuggestionProfileSection>(json);
  }
}

mixin SearchSuggestionProfileSectionMappable {
  String toJson() {
    return SearchSuggestionProfileSectionMapper.ensureInitialized()
        .encodeJson<SearchSuggestionProfileSection>(
          this as SearchSuggestionProfileSection,
        );
  }

  Map<String, dynamic> toMap() {
    return SearchSuggestionProfileSectionMapper.ensureInitialized()
        .encodeMap<SearchSuggestionProfileSection>(
          this as SearchSuggestionProfileSection,
        );
  }

  SearchSuggestionProfileSectionCopyWith<
    SearchSuggestionProfileSection,
    SearchSuggestionProfileSection,
    SearchSuggestionProfileSection
  >
  get copyWith =>
      _SearchSuggestionProfileSectionCopyWithImpl<
        SearchSuggestionProfileSection,
        SearchSuggestionProfileSection
      >(this as SearchSuggestionProfileSection, $identity, $identity);
  @override
  String toString() {
    return SearchSuggestionProfileSectionMapper.ensureInitialized()
        .stringifyValue(this as SearchSuggestionProfileSection);
  }

  @override
  bool operator ==(Object other) {
    return SearchSuggestionProfileSectionMapper.ensureInitialized().equalsValue(
      this as SearchSuggestionProfileSection,
      other,
    );
  }

  @override
  int get hashCode {
    return SearchSuggestionProfileSectionMapper.ensureInitialized().hashValue(
      this as SearchSuggestionProfileSection,
    );
  }
}

extension SearchSuggestionProfileSectionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SearchSuggestionProfileSection, $Out> {
  SearchSuggestionProfileSectionCopyWith<
    $R,
    SearchSuggestionProfileSection,
    $Out
  >
  get $asSearchSuggestionProfileSection => $base.as(
    (v, t, t2) =>
        _SearchSuggestionProfileSectionCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SearchSuggestionProfileSectionCopyWith<
  $R,
  $In extends SearchSuggestionProfileSection,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ProfileSearchResult,
    ProfileSearchResultCopyWith<$R, ProfileSearchResult, ProfileSearchResult>
  >
  get items;
  $R call({List<ProfileSearchResult>? items, bool? hasMore});
  SearchSuggestionProfileSectionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SearchSuggestionProfileSectionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SearchSuggestionProfileSection, $Out>
    implements
        SearchSuggestionProfileSectionCopyWith<
          $R,
          SearchSuggestionProfileSection,
          $Out
        > {
  _SearchSuggestionProfileSectionCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<SearchSuggestionProfileSection> $mapper =
      SearchSuggestionProfileSectionMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ProfileSearchResult,
    ProfileSearchResultCopyWith<$R, ProfileSearchResult, ProfileSearchResult>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<ProfileSearchResult>? items, bool? hasMore}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (hasMore != null) #hasMore: hasMore,
    }),
  );
  @override
  SearchSuggestionProfileSection $make(CopyWithData data) =>
      SearchSuggestionProfileSection(
        items: data.get(#items, or: $value.items),
        hasMore: data.get(#hasMore, or: $value.hasMore),
      );

  @override
  SearchSuggestionProfileSectionCopyWith<
    $R2,
    SearchSuggestionProfileSection,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SearchSuggestionProfileSectionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SearchSuggestionHashtagSectionMapper
    extends ClassMapperBase<SearchSuggestionHashtagSection> {
  SearchSuggestionHashtagSectionMapper._();

  static SearchSuggestionHashtagSectionMapper? _instance;
  static SearchSuggestionHashtagSectionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = SearchSuggestionHashtagSectionMapper._(),
      );
      HashtagSearchResultMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SearchSuggestionHashtagSection';

  static List<HashtagSearchResult> _$items(SearchSuggestionHashtagSection v) =>
      v.items;
  static const Field<SearchSuggestionHashtagSection, List<HashtagSearchResult>>
  _f$items = Field('items', _$items);
  static bool _$hasMore(SearchSuggestionHashtagSection v) => v.hasMore;
  static const Field<SearchSuggestionHashtagSection, bool> _f$hasMore = Field(
    'hasMore',
    _$hasMore,
  );

  @override
  final MappableFields<SearchSuggestionHashtagSection> fields = const {
    #items: _f$items,
    #hasMore: _f$hasMore,
  };

  static SearchSuggestionHashtagSection _instantiate(DecodingData data) {
    return SearchSuggestionHashtagSection(
      items: data.dec(_f$items),
      hasMore: data.dec(_f$hasMore),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SearchSuggestionHashtagSection fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SearchSuggestionHashtagSection>(map);
  }

  static SearchSuggestionHashtagSection fromJson(String json) {
    return ensureInitialized().decodeJson<SearchSuggestionHashtagSection>(json);
  }
}

mixin SearchSuggestionHashtagSectionMappable {
  String toJson() {
    return SearchSuggestionHashtagSectionMapper.ensureInitialized()
        .encodeJson<SearchSuggestionHashtagSection>(
          this as SearchSuggestionHashtagSection,
        );
  }

  Map<String, dynamic> toMap() {
    return SearchSuggestionHashtagSectionMapper.ensureInitialized()
        .encodeMap<SearchSuggestionHashtagSection>(
          this as SearchSuggestionHashtagSection,
        );
  }

  SearchSuggestionHashtagSectionCopyWith<
    SearchSuggestionHashtagSection,
    SearchSuggestionHashtagSection,
    SearchSuggestionHashtagSection
  >
  get copyWith =>
      _SearchSuggestionHashtagSectionCopyWithImpl<
        SearchSuggestionHashtagSection,
        SearchSuggestionHashtagSection
      >(this as SearchSuggestionHashtagSection, $identity, $identity);
  @override
  String toString() {
    return SearchSuggestionHashtagSectionMapper.ensureInitialized()
        .stringifyValue(this as SearchSuggestionHashtagSection);
  }

  @override
  bool operator ==(Object other) {
    return SearchSuggestionHashtagSectionMapper.ensureInitialized().equalsValue(
      this as SearchSuggestionHashtagSection,
      other,
    );
  }

  @override
  int get hashCode {
    return SearchSuggestionHashtagSectionMapper.ensureInitialized().hashValue(
      this as SearchSuggestionHashtagSection,
    );
  }
}

extension SearchSuggestionHashtagSectionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SearchSuggestionHashtagSection, $Out> {
  SearchSuggestionHashtagSectionCopyWith<
    $R,
    SearchSuggestionHashtagSection,
    $Out
  >
  get $asSearchSuggestionHashtagSection => $base.as(
    (v, t, t2) =>
        _SearchSuggestionHashtagSectionCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SearchSuggestionHashtagSectionCopyWith<
  $R,
  $In extends SearchSuggestionHashtagSection,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    HashtagSearchResult,
    HashtagSearchResultCopyWith<$R, HashtagSearchResult, HashtagSearchResult>
  >
  get items;
  $R call({List<HashtagSearchResult>? items, bool? hasMore});
  SearchSuggestionHashtagSectionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SearchSuggestionHashtagSectionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SearchSuggestionHashtagSection, $Out>
    implements
        SearchSuggestionHashtagSectionCopyWith<
          $R,
          SearchSuggestionHashtagSection,
          $Out
        > {
  _SearchSuggestionHashtagSectionCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<SearchSuggestionHashtagSection> $mapper =
      SearchSuggestionHashtagSectionMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    HashtagSearchResult,
    HashtagSearchResultCopyWith<$R, HashtagSearchResult, HashtagSearchResult>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<HashtagSearchResult>? items, bool? hasMore}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (hasMore != null) #hasMore: hasMore,
    }),
  );
  @override
  SearchSuggestionHashtagSection $make(CopyWithData data) =>
      SearchSuggestionHashtagSection(
        items: data.get(#items, or: $value.items),
        hasMore: data.get(#hasMore, or: $value.hasMore),
      );

  @override
  SearchSuggestionHashtagSectionCopyWith<
    $R2,
    SearchSuggestionHashtagSection,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SearchSuggestionHashtagSectionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

