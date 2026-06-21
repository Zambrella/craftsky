// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'search_result_state.dart';

class SearchPostResultsStateMapper
    extends ClassMapperBase<SearchPostResultsState> {
  SearchPostResultsStateMapper._();

  static SearchPostResultsStateMapper? _instance;
  static SearchPostResultsStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SearchPostResultsStateMapper._());
      PostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SearchPostResultsState';

  static List<Post> _$items(SearchPostResultsState v) => v.items;
  static const Field<SearchPostResultsState, List<Post>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(SearchPostResultsState v) => v.cursor;
  static const Field<SearchPostResultsState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );
  static String? _$hashtag(SearchPostResultsState v) => v.hashtag;
  static const Field<SearchPostResultsState, String> _f$hashtag = Field(
    'hashtag',
    _$hashtag,
    opt: true,
  );

  @override
  final MappableFields<SearchPostResultsState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
    #hashtag: _f$hashtag,
  };

  static SearchPostResultsState _instantiate(DecodingData data) {
    return SearchPostResultsState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
      hashtag: data.dec(_f$hashtag),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SearchPostResultsState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SearchPostResultsState>(map);
  }

  static SearchPostResultsState fromJson(String json) {
    return ensureInitialized().decodeJson<SearchPostResultsState>(json);
  }
}

mixin SearchPostResultsStateMappable {
  String toJson() {
    return SearchPostResultsStateMapper.ensureInitialized()
        .encodeJson<SearchPostResultsState>(this as SearchPostResultsState);
  }

  Map<String, dynamic> toMap() {
    return SearchPostResultsStateMapper.ensureInitialized()
        .encodeMap<SearchPostResultsState>(this as SearchPostResultsState);
  }

  SearchPostResultsStateCopyWith<
    SearchPostResultsState,
    SearchPostResultsState,
    SearchPostResultsState
  >
  get copyWith =>
      _SearchPostResultsStateCopyWithImpl<
        SearchPostResultsState,
        SearchPostResultsState
      >(this as SearchPostResultsState, $identity, $identity);
  @override
  String toString() {
    return SearchPostResultsStateMapper.ensureInitialized().stringifyValue(
      this as SearchPostResultsState,
    );
  }

  @override
  bool operator ==(Object other) {
    return SearchPostResultsStateMapper.ensureInitialized().equalsValue(
      this as SearchPostResultsState,
      other,
    );
  }

  @override
  int get hashCode {
    return SearchPostResultsStateMapper.ensureInitialized().hashValue(
      this as SearchPostResultsState,
    );
  }
}

extension SearchPostResultsStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SearchPostResultsState, $Out> {
  SearchPostResultsStateCopyWith<$R, SearchPostResultsState, $Out>
  get $asSearchPostResultsState => $base.as(
    (v, t, t2) => _SearchPostResultsStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SearchPostResultsStateCopyWith<
  $R,
  $In extends SearchPostResultsState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items;
  $R call({List<Post>? items, String? cursor, String? hashtag});
  SearchPostResultsStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SearchPostResultsStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SearchPostResultsState, $Out>
    implements
        SearchPostResultsStateCopyWith<$R, SearchPostResultsState, $Out> {
  _SearchPostResultsStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SearchPostResultsState> $mapper =
      SearchPostResultsStateMapper.ensureInitialized();
  @override
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items =>
      ListCopyWith(
        $value.items,
        (v, t) => v.copyWith.$chain(t),
        (v) => call(items: v),
      );
  @override
  $R call({
    List<Post>? items,
    Object? cursor = $none,
    Object? hashtag = $none,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
      if (hashtag != $none) #hashtag: hashtag,
    }),
  );
  @override
  SearchPostResultsState $make(CopyWithData data) => SearchPostResultsState(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
    hashtag: data.get(#hashtag, or: $value.hashtag),
  );

  @override
  SearchPostResultsStateCopyWith<$R2, SearchPostResultsState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SearchPostResultsStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProfileSearchResultsStateMapper
    extends ClassMapperBase<ProfileSearchResultsState> {
  ProfileSearchResultsStateMapper._();

  static ProfileSearchResultsStateMapper? _instance;
  static ProfileSearchResultsStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ProfileSearchResultsStateMapper._(),
      );
      ProfileSearchResultMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileSearchResultsState';

  static List<ProfileSearchResult> _$items(ProfileSearchResultsState v) =>
      v.items;
  static const Field<ProfileSearchResultsState, List<ProfileSearchResult>>
  _f$items = Field('items', _$items);
  static String? _$cursor(ProfileSearchResultsState v) => v.cursor;
  static const Field<ProfileSearchResultsState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<ProfileSearchResultsState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static ProfileSearchResultsState _instantiate(DecodingData data) {
    return ProfileSearchResultsState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileSearchResultsState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileSearchResultsState>(map);
  }

  static ProfileSearchResultsState fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileSearchResultsState>(json);
  }
}

mixin ProfileSearchResultsStateMappable {
  String toJson() {
    return ProfileSearchResultsStateMapper.ensureInitialized()
        .encodeJson<ProfileSearchResultsState>(
          this as ProfileSearchResultsState,
        );
  }

  Map<String, dynamic> toMap() {
    return ProfileSearchResultsStateMapper.ensureInitialized()
        .encodeMap<ProfileSearchResultsState>(
          this as ProfileSearchResultsState,
        );
  }

  ProfileSearchResultsStateCopyWith<
    ProfileSearchResultsState,
    ProfileSearchResultsState,
    ProfileSearchResultsState
  >
  get copyWith =>
      _ProfileSearchResultsStateCopyWithImpl<
        ProfileSearchResultsState,
        ProfileSearchResultsState
      >(this as ProfileSearchResultsState, $identity, $identity);
  @override
  String toString() {
    return ProfileSearchResultsStateMapper.ensureInitialized().stringifyValue(
      this as ProfileSearchResultsState,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileSearchResultsStateMapper.ensureInitialized().equalsValue(
      this as ProfileSearchResultsState,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileSearchResultsStateMapper.ensureInitialized().hashValue(
      this as ProfileSearchResultsState,
    );
  }
}

extension ProfileSearchResultsStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileSearchResultsState, $Out> {
  ProfileSearchResultsStateCopyWith<$R, ProfileSearchResultsState, $Out>
  get $asProfileSearchResultsState => $base.as(
    (v, t, t2) => _ProfileSearchResultsStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileSearchResultsStateCopyWith<
  $R,
  $In extends ProfileSearchResultsState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ProfileSearchResult,
    ProfileSearchResultCopyWith<$R, ProfileSearchResult, ProfileSearchResult>
  >
  get items;
  $R call({List<ProfileSearchResult>? items, String? cursor});
  ProfileSearchResultsStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileSearchResultsStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileSearchResultsState, $Out>
    implements
        ProfileSearchResultsStateCopyWith<$R, ProfileSearchResultsState, $Out> {
  _ProfileSearchResultsStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileSearchResultsState> $mapper =
      ProfileSearchResultsStateMapper.ensureInitialized();
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
  $R call({List<ProfileSearchResult>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  ProfileSearchResultsState $make(CopyWithData data) =>
      ProfileSearchResultsState(
        items: data.get(#items, or: $value.items),
        cursor: data.get(#cursor, or: $value.cursor),
      );

  @override
  ProfileSearchResultsStateCopyWith<$R2, ProfileSearchResultsState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileSearchResultsStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class HashtagSearchResultsStateMapper
    extends ClassMapperBase<HashtagSearchResultsState> {
  HashtagSearchResultsStateMapper._();

  static HashtagSearchResultsStateMapper? _instance;
  static HashtagSearchResultsStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = HashtagSearchResultsStateMapper._(),
      );
      HashtagSearchResultMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagSearchResultsState';

  static List<HashtagSearchResult> _$items(HashtagSearchResultsState v) =>
      v.items;
  static const Field<HashtagSearchResultsState, List<HashtagSearchResult>>
  _f$items = Field('items', _$items);
  static String? _$cursor(HashtagSearchResultsState v) => v.cursor;
  static const Field<HashtagSearchResultsState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<HashtagSearchResultsState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static HashtagSearchResultsState _instantiate(DecodingData data) {
    return HashtagSearchResultsState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagSearchResultsState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagSearchResultsState>(map);
  }

  static HashtagSearchResultsState fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagSearchResultsState>(json);
  }
}

mixin HashtagSearchResultsStateMappable {
  String toJson() {
    return HashtagSearchResultsStateMapper.ensureInitialized()
        .encodeJson<HashtagSearchResultsState>(
          this as HashtagSearchResultsState,
        );
  }

  Map<String, dynamic> toMap() {
    return HashtagSearchResultsStateMapper.ensureInitialized()
        .encodeMap<HashtagSearchResultsState>(
          this as HashtagSearchResultsState,
        );
  }

  HashtagSearchResultsStateCopyWith<
    HashtagSearchResultsState,
    HashtagSearchResultsState,
    HashtagSearchResultsState
  >
  get copyWith =>
      _HashtagSearchResultsStateCopyWithImpl<
        HashtagSearchResultsState,
        HashtagSearchResultsState
      >(this as HashtagSearchResultsState, $identity, $identity);
  @override
  String toString() {
    return HashtagSearchResultsStateMapper.ensureInitialized().stringifyValue(
      this as HashtagSearchResultsState,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagSearchResultsStateMapper.ensureInitialized().equalsValue(
      this as HashtagSearchResultsState,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagSearchResultsStateMapper.ensureInitialized().hashValue(
      this as HashtagSearchResultsState,
    );
  }
}

extension HashtagSearchResultsStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagSearchResultsState, $Out> {
  HashtagSearchResultsStateCopyWith<$R, HashtagSearchResultsState, $Out>
  get $asHashtagSearchResultsState => $base.as(
    (v, t, t2) => _HashtagSearchResultsStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagSearchResultsStateCopyWith<
  $R,
  $In extends HashtagSearchResultsState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    HashtagSearchResult,
    HashtagSearchResultCopyWith<$R, HashtagSearchResult, HashtagSearchResult>
  >
  get items;
  $R call({List<HashtagSearchResult>? items, String? cursor});
  HashtagSearchResultsStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagSearchResultsStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagSearchResultsState, $Out>
    implements
        HashtagSearchResultsStateCopyWith<$R, HashtagSearchResultsState, $Out> {
  _HashtagSearchResultsStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagSearchResultsState> $mapper =
      HashtagSearchResultsStateMapper.ensureInitialized();
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
  $R call({List<HashtagSearchResult>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  HashtagSearchResultsState $make(CopyWithData data) =>
      HashtagSearchResultsState(
        items: data.get(#items, or: $value.items),
        cursor: data.get(#cursor, or: $value.cursor),
      );

  @override
  HashtagSearchResultsStateCopyWith<$R2, HashtagSearchResultsState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _HashtagSearchResultsStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

