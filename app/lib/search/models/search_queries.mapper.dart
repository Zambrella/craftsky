// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'search_queries.dart';

class SearchSuggestionQueryMapper
    extends ClassMapperBase<SearchSuggestionQuery> {
  SearchSuggestionQueryMapper._();

  static SearchSuggestionQueryMapper? _instance;
  static SearchSuggestionQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SearchSuggestionQueryMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SearchSuggestionQuery';

  static String _$q(SearchSuggestionQuery v) => v.q;
  static const Field<SearchSuggestionQuery, String> _f$q = Field('q', _$q);
  static List<SearchSuggestionType> _$types(SearchSuggestionQuery v) => v.types;
  static const Field<SearchSuggestionQuery, List<SearchSuggestionType>>
  _f$types = Field('types', _$types, opt: true, def: const []);
  static int? _$profileLimit(SearchSuggestionQuery v) => v.profileLimit;
  static const Field<SearchSuggestionQuery, int> _f$profileLimit = Field(
    'profileLimit',
    _$profileLimit,
    opt: true,
  );
  static int? _$hashtagLimit(SearchSuggestionQuery v) => v.hashtagLimit;
  static const Field<SearchSuggestionQuery, int> _f$hashtagLimit = Field(
    'hashtagLimit',
    _$hashtagLimit,
    opt: true,
  );

  @override
  final MappableFields<SearchSuggestionQuery> fields = const {
    #q: _f$q,
    #types: _f$types,
    #profileLimit: _f$profileLimit,
    #hashtagLimit: _f$hashtagLimit,
  };

  static SearchSuggestionQuery _instantiate(DecodingData data) {
    return SearchSuggestionQuery(
      q: data.dec(_f$q),
      types: data.dec(_f$types),
      profileLimit: data.dec(_f$profileLimit),
      hashtagLimit: data.dec(_f$hashtagLimit),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SearchSuggestionQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SearchSuggestionQuery>(map);
  }

  static SearchSuggestionQuery fromJson(String json) {
    return ensureInitialized().decodeJson<SearchSuggestionQuery>(json);
  }
}

mixin SearchSuggestionQueryMappable {
  String toJson() {
    return SearchSuggestionQueryMapper.ensureInitialized()
        .encodeJson<SearchSuggestionQuery>(this as SearchSuggestionQuery);
  }

  Map<String, dynamic> toMap() {
    return SearchSuggestionQueryMapper.ensureInitialized()
        .encodeMap<SearchSuggestionQuery>(this as SearchSuggestionQuery);
  }

  SearchSuggestionQueryCopyWith<
    SearchSuggestionQuery,
    SearchSuggestionQuery,
    SearchSuggestionQuery
  >
  get copyWith =>
      _SearchSuggestionQueryCopyWithImpl<
        SearchSuggestionQuery,
        SearchSuggestionQuery
      >(this as SearchSuggestionQuery, $identity, $identity);
  @override
  String toString() {
    return SearchSuggestionQueryMapper.ensureInitialized().stringifyValue(
      this as SearchSuggestionQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return SearchSuggestionQueryMapper.ensureInitialized().equalsValue(
      this as SearchSuggestionQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return SearchSuggestionQueryMapper.ensureInitialized().hashValue(
      this as SearchSuggestionQuery,
    );
  }
}

extension SearchSuggestionQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SearchSuggestionQuery, $Out> {
  SearchSuggestionQueryCopyWith<$R, SearchSuggestionQuery, $Out>
  get $asSearchSuggestionQuery => $base.as(
    (v, t, t2) => _SearchSuggestionQueryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SearchSuggestionQueryCopyWith<
  $R,
  $In extends SearchSuggestionQuery,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    SearchSuggestionType,
    ObjectCopyWith<$R, SearchSuggestionType, SearchSuggestionType>
  >
  get types;
  $R call({
    String? q,
    List<SearchSuggestionType>? types,
    int? profileLimit,
    int? hashtagLimit,
  });
  SearchSuggestionQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SearchSuggestionQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SearchSuggestionQuery, $Out>
    implements SearchSuggestionQueryCopyWith<$R, SearchSuggestionQuery, $Out> {
  _SearchSuggestionQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SearchSuggestionQuery> $mapper =
      SearchSuggestionQueryMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    SearchSuggestionType,
    ObjectCopyWith<$R, SearchSuggestionType, SearchSuggestionType>
  >
  get types => ListCopyWith(
    $value.types,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(types: v),
  );
  @override
  $R call({
    String? q,
    List<SearchSuggestionType>? types,
    Object? profileLimit = $none,
    Object? hashtagLimit = $none,
  }) => $apply(
    FieldCopyWithData({
      if (q != null) #q: q,
      if (types != null) #types: types,
      if (profileLimit != $none) #profileLimit: profileLimit,
      if (hashtagLimit != $none) #hashtagLimit: hashtagLimit,
    }),
  );
  @override
  SearchSuggestionQuery $make(CopyWithData data) => SearchSuggestionQuery(
    q: data.get(#q, or: $value.q),
    types: data.get(#types, or: $value.types),
    profileLimit: data.get(#profileLimit, or: $value.profileLimit),
    hashtagLimit: data.get(#hashtagLimit, or: $value.hashtagLimit),
  );

  @override
  SearchSuggestionQueryCopyWith<$R2, SearchSuggestionQuery, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SearchSuggestionQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class HashtagSearchQueryMapper extends ClassMapperBase<HashtagSearchQuery> {
  HashtagSearchQueryMapper._();

  static HashtagSearchQueryMapper? _instance;
  static HashtagSearchQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = HashtagSearchQueryMapper._());
      SearchSortMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagSearchQuery';

  static String _$tag(HashtagSearchQuery v) => v.tag;
  static const Field<HashtagSearchQuery, String> _f$tag = Field('tag', _$tag);
  static SearchSort _$sort(HashtagSearchQuery v) => v.sort;
  static const Field<HashtagSearchQuery, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );

  @override
  final MappableFields<HashtagSearchQuery> fields = const {
    #tag: _f$tag,
    #sort: _f$sort,
  };

  static HashtagSearchQuery _instantiate(DecodingData data) {
    return HashtagSearchQuery(tag: data.dec(_f$tag), sort: data.dec(_f$sort));
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagSearchQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagSearchQuery>(map);
  }

  static HashtagSearchQuery fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagSearchQuery>(json);
  }
}

mixin HashtagSearchQueryMappable {
  String toJson() {
    return HashtagSearchQueryMapper.ensureInitialized()
        .encodeJson<HashtagSearchQuery>(this as HashtagSearchQuery);
  }

  Map<String, dynamic> toMap() {
    return HashtagSearchQueryMapper.ensureInitialized()
        .encodeMap<HashtagSearchQuery>(this as HashtagSearchQuery);
  }

  HashtagSearchQueryCopyWith<
    HashtagSearchQuery,
    HashtagSearchQuery,
    HashtagSearchQuery
  >
  get copyWith =>
      _HashtagSearchQueryCopyWithImpl<HashtagSearchQuery, HashtagSearchQuery>(
        this as HashtagSearchQuery,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return HashtagSearchQueryMapper.ensureInitialized().stringifyValue(
      this as HashtagSearchQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagSearchQueryMapper.ensureInitialized().equalsValue(
      this as HashtagSearchQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagSearchQueryMapper.ensureInitialized().hashValue(
      this as HashtagSearchQuery,
    );
  }
}

extension HashtagSearchQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagSearchQuery, $Out> {
  HashtagSearchQueryCopyWith<$R, HashtagSearchQuery, $Out>
  get $asHashtagSearchQuery => $base.as(
    (v, t, t2) => _HashtagSearchQueryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagSearchQueryCopyWith<
  $R,
  $In extends HashtagSearchQuery,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? tag, SearchSort? sort});
  HashtagSearchQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagSearchQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagSearchQuery, $Out>
    implements HashtagSearchQueryCopyWith<$R, HashtagSearchQuery, $Out> {
  _HashtagSearchQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagSearchQuery> $mapper =
      HashtagSearchQueryMapper.ensureInitialized();
  @override
  $R call({String? tag, SearchSort? sort}) => $apply(
    FieldCopyWithData({
      if (tag != null) #tag: tag,
      if (sort != null) #sort: sort,
    }),
  );
  @override
  HashtagSearchQuery $make(CopyWithData data) => HashtagSearchQuery(
    tag: data.get(#tag, or: $value.tag),
    sort: data.get(#sort, or: $value.sort),
  );

  @override
  HashtagSearchQueryCopyWith<$R2, HashtagSearchQuery, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _HashtagSearchQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class HashtagResultSearchQueryMapper
    extends ClassMapperBase<HashtagResultSearchQuery> {
  HashtagResultSearchQueryMapper._();

  static HashtagResultSearchQueryMapper? _instance;
  static HashtagResultSearchQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = HashtagResultSearchQueryMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagResultSearchQuery';

  static String _$q(HashtagResultSearchQuery v) => v.q;
  static const Field<HashtagResultSearchQuery, String> _f$q = Field('q', _$q);

  @override
  final MappableFields<HashtagResultSearchQuery> fields = const {#q: _f$q};

  static HashtagResultSearchQuery _instantiate(DecodingData data) {
    return HashtagResultSearchQuery(q: data.dec(_f$q));
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagResultSearchQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagResultSearchQuery>(map);
  }

  static HashtagResultSearchQuery fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagResultSearchQuery>(json);
  }
}

mixin HashtagResultSearchQueryMappable {
  String toJson() {
    return HashtagResultSearchQueryMapper.ensureInitialized()
        .encodeJson<HashtagResultSearchQuery>(this as HashtagResultSearchQuery);
  }

  Map<String, dynamic> toMap() {
    return HashtagResultSearchQueryMapper.ensureInitialized()
        .encodeMap<HashtagResultSearchQuery>(this as HashtagResultSearchQuery);
  }

  HashtagResultSearchQueryCopyWith<
    HashtagResultSearchQuery,
    HashtagResultSearchQuery,
    HashtagResultSearchQuery
  >
  get copyWith =>
      _HashtagResultSearchQueryCopyWithImpl<
        HashtagResultSearchQuery,
        HashtagResultSearchQuery
      >(this as HashtagResultSearchQuery, $identity, $identity);
  @override
  String toString() {
    return HashtagResultSearchQueryMapper.ensureInitialized().stringifyValue(
      this as HashtagResultSearchQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagResultSearchQueryMapper.ensureInitialized().equalsValue(
      this as HashtagResultSearchQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagResultSearchQueryMapper.ensureInitialized().hashValue(
      this as HashtagResultSearchQuery,
    );
  }
}

extension HashtagResultSearchQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagResultSearchQuery, $Out> {
  HashtagResultSearchQueryCopyWith<$R, HashtagResultSearchQuery, $Out>
  get $asHashtagResultSearchQuery => $base.as(
    (v, t, t2) => _HashtagResultSearchQueryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagResultSearchQueryCopyWith<
  $R,
  $In extends HashtagResultSearchQuery,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? q});
  HashtagResultSearchQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagResultSearchQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagResultSearchQuery, $Out>
    implements
        HashtagResultSearchQueryCopyWith<$R, HashtagResultSearchQuery, $Out> {
  _HashtagResultSearchQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagResultSearchQuery> $mapper =
      HashtagResultSearchQueryMapper.ensureInitialized();
  @override
  $R call({String? q}) => $apply(FieldCopyWithData({if (q != null) #q: q}));
  @override
  HashtagResultSearchQuery $make(CopyWithData data) =>
      HashtagResultSearchQuery(q: data.get(#q, or: $value.q));

  @override
  HashtagResultSearchQueryCopyWith<$R2, HashtagResultSearchQuery, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _HashtagResultSearchQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProfileSearchQueryMapper extends ClassMapperBase<ProfileSearchQuery> {
  ProfileSearchQueryMapper._();

  static ProfileSearchQueryMapper? _instance;
  static ProfileSearchQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileSearchQueryMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileSearchQuery';

  static String _$q(ProfileSearchQuery v) => v.q;
  static const Field<ProfileSearchQuery, String> _f$q = Field('q', _$q);

  @override
  final MappableFields<ProfileSearchQuery> fields = const {#q: _f$q};

  static ProfileSearchQuery _instantiate(DecodingData data) {
    return ProfileSearchQuery(q: data.dec(_f$q));
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileSearchQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileSearchQuery>(map);
  }

  static ProfileSearchQuery fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileSearchQuery>(json);
  }
}

mixin ProfileSearchQueryMappable {
  String toJson() {
    return ProfileSearchQueryMapper.ensureInitialized()
        .encodeJson<ProfileSearchQuery>(this as ProfileSearchQuery);
  }

  Map<String, dynamic> toMap() {
    return ProfileSearchQueryMapper.ensureInitialized()
        .encodeMap<ProfileSearchQuery>(this as ProfileSearchQuery);
  }

  ProfileSearchQueryCopyWith<
    ProfileSearchQuery,
    ProfileSearchQuery,
    ProfileSearchQuery
  >
  get copyWith =>
      _ProfileSearchQueryCopyWithImpl<ProfileSearchQuery, ProfileSearchQuery>(
        this as ProfileSearchQuery,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfileSearchQueryMapper.ensureInitialized().stringifyValue(
      this as ProfileSearchQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileSearchQueryMapper.ensureInitialized().equalsValue(
      this as ProfileSearchQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileSearchQueryMapper.ensureInitialized().hashValue(
      this as ProfileSearchQuery,
    );
  }
}

extension ProfileSearchQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileSearchQuery, $Out> {
  ProfileSearchQueryCopyWith<$R, ProfileSearchQuery, $Out>
  get $asProfileSearchQuery => $base.as(
    (v, t, t2) => _ProfileSearchQueryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileSearchQueryCopyWith<
  $R,
  $In extends ProfileSearchQuery,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? q});
  ProfileSearchQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileSearchQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileSearchQuery, $Out>
    implements ProfileSearchQueryCopyWith<$R, ProfileSearchQuery, $Out> {
  _ProfileSearchQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileSearchQuery> $mapper =
      ProfileSearchQueryMapper.ensureInitialized();
  @override
  $R call({String? q}) => $apply(FieldCopyWithData({if (q != null) #q: q}));
  @override
  ProfileSearchQuery $make(CopyWithData data) =>
      ProfileSearchQuery(q: data.get(#q, or: $value.q));

  @override
  ProfileSearchQueryCopyWith<$R2, ProfileSearchQuery, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProfileSearchQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostSearchQueryMapper extends ClassMapperBase<PostSearchQuery> {
  PostSearchQueryMapper._();

  static PostSearchQueryMapper? _instance;
  static PostSearchQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostSearchQueryMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'PostSearchQuery';

  static String _$q(PostSearchQuery v) => v.q;
  static const Field<PostSearchQuery, String> _f$q = Field('q', _$q);

  @override
  final MappableFields<PostSearchQuery> fields = const {#q: _f$q};

  static PostSearchQuery _instantiate(DecodingData data) {
    return PostSearchQuery(q: data.dec(_f$q));
  }

  @override
  final Function instantiate = _instantiate;

  static PostSearchQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostSearchQuery>(map);
  }

  static PostSearchQuery fromJson(String json) {
    return ensureInitialized().decodeJson<PostSearchQuery>(json);
  }
}

mixin PostSearchQueryMappable {
  String toJson() {
    return PostSearchQueryMapper.ensureInitialized()
        .encodeJson<PostSearchQuery>(this as PostSearchQuery);
  }

  Map<String, dynamic> toMap() {
    return PostSearchQueryMapper.ensureInitialized().encodeMap<PostSearchQuery>(
      this as PostSearchQuery,
    );
  }

  PostSearchQueryCopyWith<PostSearchQuery, PostSearchQuery, PostSearchQuery>
  get copyWith =>
      _PostSearchQueryCopyWithImpl<PostSearchQuery, PostSearchQuery>(
        this as PostSearchQuery,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostSearchQueryMapper.ensureInitialized().stringifyValue(
      this as PostSearchQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostSearchQueryMapper.ensureInitialized().equalsValue(
      this as PostSearchQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return PostSearchQueryMapper.ensureInitialized().hashValue(
      this as PostSearchQuery,
    );
  }
}

extension PostSearchQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostSearchQuery, $Out> {
  PostSearchQueryCopyWith<$R, PostSearchQuery, $Out> get $asPostSearchQuery =>
      $base.as((v, t, t2) => _PostSearchQueryCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostSearchQueryCopyWith<$R, $In extends PostSearchQuery, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? q});
  PostSearchQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostSearchQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostSearchQuery, $Out>
    implements PostSearchQueryCopyWith<$R, PostSearchQuery, $Out> {
  _PostSearchQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostSearchQuery> $mapper =
      PostSearchQueryMapper.ensureInitialized();
  @override
  $R call({String? q}) => $apply(FieldCopyWithData({if (q != null) #q: q}));
  @override
  PostSearchQuery $make(CopyWithData data) =>
      PostSearchQuery(q: data.get(#q, or: $value.q));

  @override
  PostSearchQueryCopyWith<$R2, PostSearchQuery, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostSearchQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectSearchQueryMapper extends ClassMapperBase<ProjectSearchQuery> {
  ProjectSearchQueryMapper._();

  static ProjectSearchQueryMapper? _instance;
  static ProjectSearchQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectSearchQueryMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectSearchQuery';

  static String _$q(ProjectSearchQuery v) => v.q;
  static const Field<ProjectSearchQuery, String> _f$q = Field('q', _$q);

  @override
  final MappableFields<ProjectSearchQuery> fields = const {#q: _f$q};

  static ProjectSearchQuery _instantiate(DecodingData data) {
    return ProjectSearchQuery(q: data.dec(_f$q));
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectSearchQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectSearchQuery>(map);
  }

  static ProjectSearchQuery fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectSearchQuery>(json);
  }
}

mixin ProjectSearchQueryMappable {
  String toJson() {
    return ProjectSearchQueryMapper.ensureInitialized()
        .encodeJson<ProjectSearchQuery>(this as ProjectSearchQuery);
  }

  Map<String, dynamic> toMap() {
    return ProjectSearchQueryMapper.ensureInitialized()
        .encodeMap<ProjectSearchQuery>(this as ProjectSearchQuery);
  }

  ProjectSearchQueryCopyWith<
    ProjectSearchQuery,
    ProjectSearchQuery,
    ProjectSearchQuery
  >
  get copyWith =>
      _ProjectSearchQueryCopyWithImpl<ProjectSearchQuery, ProjectSearchQuery>(
        this as ProjectSearchQuery,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProjectSearchQueryMapper.ensureInitialized().stringifyValue(
      this as ProjectSearchQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectSearchQueryMapper.ensureInitialized().equalsValue(
      this as ProjectSearchQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectSearchQueryMapper.ensureInitialized().hashValue(
      this as ProjectSearchQuery,
    );
  }
}

extension ProjectSearchQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectSearchQuery, $Out> {
  ProjectSearchQueryCopyWith<$R, ProjectSearchQuery, $Out>
  get $asProjectSearchQuery => $base.as(
    (v, t, t2) => _ProjectSearchQueryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProjectSearchQueryCopyWith<
  $R,
  $In extends ProjectSearchQuery,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? q});
  ProjectSearchQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectSearchQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectSearchQuery, $Out>
    implements ProjectSearchQueryCopyWith<$R, ProjectSearchQuery, $Out> {
  _ProjectSearchQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectSearchQuery> $mapper =
      ProjectSearchQueryMapper.ensureInitialized();
  @override
  $R call({String? q}) => $apply(FieldCopyWithData({if (q != null) #q: q}));
  @override
  ProjectSearchQuery $make(CopyWithData data) =>
      ProjectSearchQuery(q: data.get(#q, or: $value.q));

  @override
  ProjectSearchQueryCopyWith<$R2, ProjectSearchQuery, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProjectSearchQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TopHashtagsQueryMapper extends ClassMapperBase<TopHashtagsQuery> {
  TopHashtagsQueryMapper._();

  static TopHashtagsQueryMapper? _instance;
  static TopHashtagsQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TopHashtagsQueryMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'TopHashtagsQuery';

  static List<String> _$craftTypes(TopHashtagsQuery v) => v.craftTypes;
  static const Field<TopHashtagsQuery, List<String>> _f$craftTypes = Field(
    'craftTypes',
    _$craftTypes,
    opt: true,
    def: const [],
  );
  static int? _$limit(TopHashtagsQuery v) => v.limit;
  static const Field<TopHashtagsQuery, int> _f$limit = Field(
    'limit',
    _$limit,
    opt: true,
  );

  @override
  final MappableFields<TopHashtagsQuery> fields = const {
    #craftTypes: _f$craftTypes,
    #limit: _f$limit,
  };

  static TopHashtagsQuery _instantiate(DecodingData data) {
    return TopHashtagsQuery(
      craftTypes: data.dec(_f$craftTypes),
      limit: data.dec(_f$limit),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static TopHashtagsQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TopHashtagsQuery>(map);
  }

  static TopHashtagsQuery fromJson(String json) {
    return ensureInitialized().decodeJson<TopHashtagsQuery>(json);
  }
}

mixin TopHashtagsQueryMappable {
  String toJson() {
    return TopHashtagsQueryMapper.ensureInitialized()
        .encodeJson<TopHashtagsQuery>(this as TopHashtagsQuery);
  }

  Map<String, dynamic> toMap() {
    return TopHashtagsQueryMapper.ensureInitialized()
        .encodeMap<TopHashtagsQuery>(this as TopHashtagsQuery);
  }

  TopHashtagsQueryCopyWith<TopHashtagsQuery, TopHashtagsQuery, TopHashtagsQuery>
  get copyWith =>
      _TopHashtagsQueryCopyWithImpl<TopHashtagsQuery, TopHashtagsQuery>(
        this as TopHashtagsQuery,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return TopHashtagsQueryMapper.ensureInitialized().stringifyValue(
      this as TopHashtagsQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return TopHashtagsQueryMapper.ensureInitialized().equalsValue(
      this as TopHashtagsQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return TopHashtagsQueryMapper.ensureInitialized().hashValue(
      this as TopHashtagsQuery,
    );
  }
}

extension TopHashtagsQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TopHashtagsQuery, $Out> {
  TopHashtagsQueryCopyWith<$R, TopHashtagsQuery, $Out>
  get $asTopHashtagsQuery =>
      $base.as((v, t, t2) => _TopHashtagsQueryCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TopHashtagsQueryCopyWith<$R, $In extends TopHashtagsQuery, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get craftTypes;
  $R call({List<String>? craftTypes, int? limit});
  TopHashtagsQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _TopHashtagsQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TopHashtagsQuery, $Out>
    implements TopHashtagsQueryCopyWith<$R, TopHashtagsQuery, $Out> {
  _TopHashtagsQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TopHashtagsQuery> $mapper =
      TopHashtagsQueryMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get craftTypes =>
      ListCopyWith(
        $value.craftTypes,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(craftTypes: v),
      );
  @override
  $R call({List<String>? craftTypes, Object? limit = $none}) => $apply(
    FieldCopyWithData({
      if (craftTypes != null) #craftTypes: craftTypes,
      if (limit != $none) #limit: limit,
    }),
  );
  @override
  TopHashtagsQuery $make(CopyWithData data) => TopHashtagsQuery(
    craftTypes: data.get(#craftTypes, or: $value.craftTypes),
    limit: data.get(#limit, or: $value.limit),
  );

  @override
  TopHashtagsQueryCopyWith<$R2, TopHashtagsQuery, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TopHashtagsQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

