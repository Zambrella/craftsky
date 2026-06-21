// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'search_queries.dart';

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
      SearchSortMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostSearchQuery';

  static String _$q(PostSearchQuery v) => v.q;
  static const Field<PostSearchQuery, String> _f$q = Field('q', _$q);
  static SearchSort _$sort(PostSearchQuery v) => v.sort;
  static const Field<PostSearchQuery, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );

  @override
  final MappableFields<PostSearchQuery> fields = const {
    #q: _f$q,
    #sort: _f$sort,
  };

  static PostSearchQuery _instantiate(DecodingData data) {
    return PostSearchQuery(q: data.dec(_f$q), sort: data.dec(_f$sort));
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
  $R call({String? q, SearchSort? sort});
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
  $R call({String? q, SearchSort? sort}) => $apply(
    FieldCopyWithData({if (q != null) #q: q, if (sort != null) #sort: sort}),
  );
  @override
  PostSearchQuery $make(CopyWithData data) => PostSearchQuery(
    q: data.get(#q, or: $value.q),
    sort: data.get(#sort, or: $value.sort),
  );

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
      SearchSortMapper.ensureInitialized();
      ProjectSearchFiltersMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectSearchQuery';

  static String? _$q(ProjectSearchQuery v) => v.q;
  static const Field<ProjectSearchQuery, String> _f$q = Field(
    'q',
    _$q,
    opt: true,
  );
  static SearchSort _$sort(ProjectSearchQuery v) => v.sort;
  static const Field<ProjectSearchQuery, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );
  static ProjectSearchFilters _$filters(ProjectSearchQuery v) => v.filters;
  static const Field<ProjectSearchQuery, ProjectSearchFilters> _f$filters =
      Field('filters', _$filters, opt: true, def: const ProjectSearchFilters());

  @override
  final MappableFields<ProjectSearchQuery> fields = const {
    #q: _f$q,
    #sort: _f$sort,
    #filters: _f$filters,
  };

  static ProjectSearchQuery _instantiate(DecodingData data) {
    return ProjectSearchQuery(
      q: data.dec(_f$q),
      sort: data.dec(_f$sort),
      filters: data.dec(_f$filters),
    );
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
  ProjectSearchFiltersCopyWith<$R, ProjectSearchFilters, ProjectSearchFilters>
  get filters;
  $R call({String? q, SearchSort? sort, ProjectSearchFilters? filters});
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
  ProjectSearchFiltersCopyWith<$R, ProjectSearchFilters, ProjectSearchFilters>
  get filters => $value.filters.copyWith.$chain((v) => call(filters: v));
  @override
  $R call({
    Object? q = $none,
    SearchSort? sort,
    ProjectSearchFilters? filters,
  }) => $apply(
    FieldCopyWithData({
      if (q != $none) #q: q,
      if (sort != null) #sort: sort,
      if (filters != null) #filters: filters,
    }),
  );
  @override
  ProjectSearchQuery $make(CopyWithData data) => ProjectSearchQuery(
    q: data.get(#q, or: $value.q),
    sort: data.get(#sort, or: $value.sort),
    filters: data.get(#filters, or: $value.filters),
  );

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

