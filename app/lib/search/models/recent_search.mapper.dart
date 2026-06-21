// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'recent_search.dart';

class RecentSearchTypeMapper extends EnumMapper<RecentSearchType> {
  RecentSearchTypeMapper._();

  static RecentSearchTypeMapper? _instance;
  static RecentSearchTypeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RecentSearchTypeMapper._());
    }
    return _instance!;
  }

  static RecentSearchType fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  RecentSearchType decode(dynamic value) {
    switch (value) {
      case r'hashtag':
        return RecentSearchType.hashtag;
      case r'profile':
        return RecentSearchType.profile;
      case r'post':
        return RecentSearchType.post;
      case r'project':
        return RecentSearchType.project;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(RecentSearchType self) {
    switch (self) {
      case RecentSearchType.hashtag:
        return r'hashtag';
      case RecentSearchType.profile:
        return r'profile';
      case RecentSearchType.post:
        return r'post';
      case RecentSearchType.project:
        return r'project';
    }
  }
}

extension RecentSearchTypeMapperExtension on RecentSearchType {
  String toValue() {
    RecentSearchTypeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<RecentSearchType>(this) as String;
  }
}

class HashtagRecentSearchPayloadMapper
    extends ClassMapperBase<HashtagRecentSearchPayload> {
  HashtagRecentSearchPayloadMapper._();

  static HashtagRecentSearchPayloadMapper? _instance;
  static HashtagRecentSearchPayloadMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = HashtagRecentSearchPayloadMapper._(),
      );
      SearchSortMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagRecentSearchPayload';

  static String _$tag(HashtagRecentSearchPayload v) => v.tag;
  static const Field<HashtagRecentSearchPayload, String> _f$tag = Field(
    'tag',
    _$tag,
  );
  static SearchSort _$sort(HashtagRecentSearchPayload v) => v.sort;
  static const Field<HashtagRecentSearchPayload, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );

  @override
  final MappableFields<HashtagRecentSearchPayload> fields = const {
    #tag: _f$tag,
    #sort: _f$sort,
  };

  static HashtagRecentSearchPayload _instantiate(DecodingData data) {
    return HashtagRecentSearchPayload(
      tag: data.dec(_f$tag),
      sort: data.dec(_f$sort),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagRecentSearchPayload fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagRecentSearchPayload>(map);
  }

  static HashtagRecentSearchPayload fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagRecentSearchPayload>(json);
  }
}

mixin HashtagRecentSearchPayloadMappable {
  String toJson() {
    return HashtagRecentSearchPayloadMapper.ensureInitialized()
        .encodeJson<HashtagRecentSearchPayload>(
          this as HashtagRecentSearchPayload,
        );
  }

  Map<String, dynamic> toMap() {
    return HashtagRecentSearchPayloadMapper.ensureInitialized()
        .encodeMap<HashtagRecentSearchPayload>(
          this as HashtagRecentSearchPayload,
        );
  }

  HashtagRecentSearchPayloadCopyWith<
    HashtagRecentSearchPayload,
    HashtagRecentSearchPayload,
    HashtagRecentSearchPayload
  >
  get copyWith =>
      _HashtagRecentSearchPayloadCopyWithImpl<
        HashtagRecentSearchPayload,
        HashtagRecentSearchPayload
      >(this as HashtagRecentSearchPayload, $identity, $identity);
  @override
  String toString() {
    return HashtagRecentSearchPayloadMapper.ensureInitialized().stringifyValue(
      this as HashtagRecentSearchPayload,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagRecentSearchPayloadMapper.ensureInitialized().equalsValue(
      this as HashtagRecentSearchPayload,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagRecentSearchPayloadMapper.ensureInitialized().hashValue(
      this as HashtagRecentSearchPayload,
    );
  }
}

extension HashtagRecentSearchPayloadValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagRecentSearchPayload, $Out> {
  HashtagRecentSearchPayloadCopyWith<$R, HashtagRecentSearchPayload, $Out>
  get $asHashtagRecentSearchPayload => $base.as(
    (v, t, t2) => _HashtagRecentSearchPayloadCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagRecentSearchPayloadCopyWith<
  $R,
  $In extends HashtagRecentSearchPayload,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? tag, SearchSort? sort});
  HashtagRecentSearchPayloadCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagRecentSearchPayloadCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagRecentSearchPayload, $Out>
    implements
        HashtagRecentSearchPayloadCopyWith<
          $R,
          HashtagRecentSearchPayload,
          $Out
        > {
  _HashtagRecentSearchPayloadCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagRecentSearchPayload> $mapper =
      HashtagRecentSearchPayloadMapper.ensureInitialized();
  @override
  $R call({String? tag, SearchSort? sort}) => $apply(
    FieldCopyWithData({
      if (tag != null) #tag: tag,
      if (sort != null) #sort: sort,
    }),
  );
  @override
  HashtagRecentSearchPayload $make(CopyWithData data) =>
      HashtagRecentSearchPayload(
        tag: data.get(#tag, or: $value.tag),
        sort: data.get(#sort, or: $value.sort),
      );

  @override
  HashtagRecentSearchPayloadCopyWith<$R2, HashtagRecentSearchPayload, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _HashtagRecentSearchPayloadCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProfileRecentSearchPayloadMapper
    extends ClassMapperBase<ProfileRecentSearchPayload> {
  ProfileRecentSearchPayloadMapper._();

  static ProfileRecentSearchPayloadMapper? _instance;
  static ProfileRecentSearchPayloadMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ProfileRecentSearchPayloadMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileRecentSearchPayload';

  static String _$q(ProfileRecentSearchPayload v) => v.q;
  static const Field<ProfileRecentSearchPayload, String> _f$q = Field('q', _$q);

  @override
  final MappableFields<ProfileRecentSearchPayload> fields = const {#q: _f$q};

  static ProfileRecentSearchPayload _instantiate(DecodingData data) {
    return ProfileRecentSearchPayload(q: data.dec(_f$q));
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileRecentSearchPayload fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileRecentSearchPayload>(map);
  }

  static ProfileRecentSearchPayload fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileRecentSearchPayload>(json);
  }
}

mixin ProfileRecentSearchPayloadMappable {
  String toJson() {
    return ProfileRecentSearchPayloadMapper.ensureInitialized()
        .encodeJson<ProfileRecentSearchPayload>(
          this as ProfileRecentSearchPayload,
        );
  }

  Map<String, dynamic> toMap() {
    return ProfileRecentSearchPayloadMapper.ensureInitialized()
        .encodeMap<ProfileRecentSearchPayload>(
          this as ProfileRecentSearchPayload,
        );
  }

  ProfileRecentSearchPayloadCopyWith<
    ProfileRecentSearchPayload,
    ProfileRecentSearchPayload,
    ProfileRecentSearchPayload
  >
  get copyWith =>
      _ProfileRecentSearchPayloadCopyWithImpl<
        ProfileRecentSearchPayload,
        ProfileRecentSearchPayload
      >(this as ProfileRecentSearchPayload, $identity, $identity);
  @override
  String toString() {
    return ProfileRecentSearchPayloadMapper.ensureInitialized().stringifyValue(
      this as ProfileRecentSearchPayload,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileRecentSearchPayloadMapper.ensureInitialized().equalsValue(
      this as ProfileRecentSearchPayload,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileRecentSearchPayloadMapper.ensureInitialized().hashValue(
      this as ProfileRecentSearchPayload,
    );
  }
}

extension ProfileRecentSearchPayloadValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileRecentSearchPayload, $Out> {
  ProfileRecentSearchPayloadCopyWith<$R, ProfileRecentSearchPayload, $Out>
  get $asProfileRecentSearchPayload => $base.as(
    (v, t, t2) => _ProfileRecentSearchPayloadCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileRecentSearchPayloadCopyWith<
  $R,
  $In extends ProfileRecentSearchPayload,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? q});
  ProfileRecentSearchPayloadCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileRecentSearchPayloadCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileRecentSearchPayload, $Out>
    implements
        ProfileRecentSearchPayloadCopyWith<
          $R,
          ProfileRecentSearchPayload,
          $Out
        > {
  _ProfileRecentSearchPayloadCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileRecentSearchPayload> $mapper =
      ProfileRecentSearchPayloadMapper.ensureInitialized();
  @override
  $R call({String? q}) => $apply(FieldCopyWithData({if (q != null) #q: q}));
  @override
  ProfileRecentSearchPayload $make(CopyWithData data) =>
      ProfileRecentSearchPayload(q: data.get(#q, or: $value.q));

  @override
  ProfileRecentSearchPayloadCopyWith<$R2, ProfileRecentSearchPayload, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileRecentSearchPayloadCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostRecentSearchPayloadMapper
    extends ClassMapperBase<PostRecentSearchPayload> {
  PostRecentSearchPayloadMapper._();

  static PostRecentSearchPayloadMapper? _instance;
  static PostRecentSearchPayloadMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = PostRecentSearchPayloadMapper._(),
      );
      SearchSortMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostRecentSearchPayload';

  static String _$q(PostRecentSearchPayload v) => v.q;
  static const Field<PostRecentSearchPayload, String> _f$q = Field('q', _$q);
  static SearchSort _$sort(PostRecentSearchPayload v) => v.sort;
  static const Field<PostRecentSearchPayload, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );

  @override
  final MappableFields<PostRecentSearchPayload> fields = const {
    #q: _f$q,
    #sort: _f$sort,
  };

  static PostRecentSearchPayload _instantiate(DecodingData data) {
    return PostRecentSearchPayload(q: data.dec(_f$q), sort: data.dec(_f$sort));
  }

  @override
  final Function instantiate = _instantiate;

  static PostRecentSearchPayload fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostRecentSearchPayload>(map);
  }

  static PostRecentSearchPayload fromJson(String json) {
    return ensureInitialized().decodeJson<PostRecentSearchPayload>(json);
  }
}

mixin PostRecentSearchPayloadMappable {
  String toJson() {
    return PostRecentSearchPayloadMapper.ensureInitialized()
        .encodeJson<PostRecentSearchPayload>(this as PostRecentSearchPayload);
  }

  Map<String, dynamic> toMap() {
    return PostRecentSearchPayloadMapper.ensureInitialized()
        .encodeMap<PostRecentSearchPayload>(this as PostRecentSearchPayload);
  }

  PostRecentSearchPayloadCopyWith<
    PostRecentSearchPayload,
    PostRecentSearchPayload,
    PostRecentSearchPayload
  >
  get copyWith =>
      _PostRecentSearchPayloadCopyWithImpl<
        PostRecentSearchPayload,
        PostRecentSearchPayload
      >(this as PostRecentSearchPayload, $identity, $identity);
  @override
  String toString() {
    return PostRecentSearchPayloadMapper.ensureInitialized().stringifyValue(
      this as PostRecentSearchPayload,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostRecentSearchPayloadMapper.ensureInitialized().equalsValue(
      this as PostRecentSearchPayload,
      other,
    );
  }

  @override
  int get hashCode {
    return PostRecentSearchPayloadMapper.ensureInitialized().hashValue(
      this as PostRecentSearchPayload,
    );
  }
}

extension PostRecentSearchPayloadValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostRecentSearchPayload, $Out> {
  PostRecentSearchPayloadCopyWith<$R, PostRecentSearchPayload, $Out>
  get $asPostRecentSearchPayload => $base.as(
    (v, t, t2) => _PostRecentSearchPayloadCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class PostRecentSearchPayloadCopyWith<
  $R,
  $In extends PostRecentSearchPayload,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? q, SearchSort? sort});
  PostRecentSearchPayloadCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostRecentSearchPayloadCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostRecentSearchPayload, $Out>
    implements
        PostRecentSearchPayloadCopyWith<$R, PostRecentSearchPayload, $Out> {
  _PostRecentSearchPayloadCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostRecentSearchPayload> $mapper =
      PostRecentSearchPayloadMapper.ensureInitialized();
  @override
  $R call({String? q, SearchSort? sort}) => $apply(
    FieldCopyWithData({if (q != null) #q: q, if (sort != null) #sort: sort}),
  );
  @override
  PostRecentSearchPayload $make(CopyWithData data) => PostRecentSearchPayload(
    q: data.get(#q, or: $value.q),
    sort: data.get(#sort, or: $value.sort),
  );

  @override
  PostRecentSearchPayloadCopyWith<$R2, PostRecentSearchPayload, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _PostRecentSearchPayloadCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectRecentSearchPayloadMapper
    extends ClassMapperBase<ProjectRecentSearchPayload> {
  ProjectRecentSearchPayloadMapper._();

  static ProjectRecentSearchPayloadMapper? _instance;
  static ProjectRecentSearchPayloadMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ProjectRecentSearchPayloadMapper._(),
      );
      SearchSortMapper.ensureInitialized();
      ProjectSearchFiltersMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectRecentSearchPayload';

  static String? _$q(ProjectRecentSearchPayload v) => v.q;
  static const Field<ProjectRecentSearchPayload, String> _f$q = Field(
    'q',
    _$q,
    opt: true,
  );
  static SearchSort _$sort(ProjectRecentSearchPayload v) => v.sort;
  static const Field<ProjectRecentSearchPayload, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );
  static ProjectSearchFilters _$filters(ProjectRecentSearchPayload v) =>
      v.filters;
  static const Field<ProjectRecentSearchPayload, ProjectSearchFilters>
  _f$filters = Field(
    'filters',
    _$filters,
    opt: true,
    def: const ProjectSearchFilters(),
  );

  @override
  final MappableFields<ProjectRecentSearchPayload> fields = const {
    #q: _f$q,
    #sort: _f$sort,
    #filters: _f$filters,
  };

  static ProjectRecentSearchPayload _instantiate(DecodingData data) {
    return ProjectRecentSearchPayload(
      q: data.dec(_f$q),
      sort: data.dec(_f$sort),
      filters: data.dec(_f$filters),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectRecentSearchPayload fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectRecentSearchPayload>(map);
  }

  static ProjectRecentSearchPayload fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectRecentSearchPayload>(json);
  }
}

mixin ProjectRecentSearchPayloadMappable {
  String toJson() {
    return ProjectRecentSearchPayloadMapper.ensureInitialized()
        .encodeJson<ProjectRecentSearchPayload>(
          this as ProjectRecentSearchPayload,
        );
  }

  Map<String, dynamic> toMap() {
    return ProjectRecentSearchPayloadMapper.ensureInitialized()
        .encodeMap<ProjectRecentSearchPayload>(
          this as ProjectRecentSearchPayload,
        );
  }

  ProjectRecentSearchPayloadCopyWith<
    ProjectRecentSearchPayload,
    ProjectRecentSearchPayload,
    ProjectRecentSearchPayload
  >
  get copyWith =>
      _ProjectRecentSearchPayloadCopyWithImpl<
        ProjectRecentSearchPayload,
        ProjectRecentSearchPayload
      >(this as ProjectRecentSearchPayload, $identity, $identity);
  @override
  String toString() {
    return ProjectRecentSearchPayloadMapper.ensureInitialized().stringifyValue(
      this as ProjectRecentSearchPayload,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectRecentSearchPayloadMapper.ensureInitialized().equalsValue(
      this as ProjectRecentSearchPayload,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectRecentSearchPayloadMapper.ensureInitialized().hashValue(
      this as ProjectRecentSearchPayload,
    );
  }
}

extension ProjectRecentSearchPayloadValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectRecentSearchPayload, $Out> {
  ProjectRecentSearchPayloadCopyWith<$R, ProjectRecentSearchPayload, $Out>
  get $asProjectRecentSearchPayload => $base.as(
    (v, t, t2) => _ProjectRecentSearchPayloadCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProjectRecentSearchPayloadCopyWith<
  $R,
  $In extends ProjectRecentSearchPayload,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ProjectSearchFiltersCopyWith<$R, ProjectSearchFilters, ProjectSearchFilters>
  get filters;
  $R call({String? q, SearchSort? sort, ProjectSearchFilters? filters});
  ProjectRecentSearchPayloadCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectRecentSearchPayloadCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectRecentSearchPayload, $Out>
    implements
        ProjectRecentSearchPayloadCopyWith<
          $R,
          ProjectRecentSearchPayload,
          $Out
        > {
  _ProjectRecentSearchPayloadCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectRecentSearchPayload> $mapper =
      ProjectRecentSearchPayloadMapper.ensureInitialized();
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
  ProjectRecentSearchPayload $make(CopyWithData data) =>
      ProjectRecentSearchPayload(
        q: data.get(#q, or: $value.q),
        sort: data.get(#sort, or: $value.sort),
        filters: data.get(#filters, or: $value.filters),
      );

  @override
  ProjectRecentSearchPayloadCopyWith<$R2, ProjectRecentSearchPayload, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProjectRecentSearchPayloadCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SaveRecentSearchRequestMapper
    extends ClassMapperBase<SaveRecentSearchRequest> {
  SaveRecentSearchRequestMapper._();

  static SaveRecentSearchRequestMapper? _instance;
  static SaveRecentSearchRequestMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = SaveRecentSearchRequestMapper._(),
      );
      RecentSearchTypeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SaveRecentSearchRequest';

  static RecentSearchType _$type(SaveRecentSearchRequest v) => v.type;
  static const Field<SaveRecentSearchRequest, RecentSearchType> _f$type = Field(
    'type',
    _$type,
  );
  static String _$displayLabel(SaveRecentSearchRequest v) => v.displayLabel;
  static const Field<SaveRecentSearchRequest, String> _f$displayLabel = Field(
    'displayLabel',
    _$displayLabel,
  );
  static RecentSearchPayload _$payload(SaveRecentSearchRequest v) => v.payload;
  static const Field<SaveRecentSearchRequest, RecentSearchPayload> _f$payload =
      Field('payload', _$payload);

  @override
  final MappableFields<SaveRecentSearchRequest> fields = const {
    #type: _f$type,
    #displayLabel: _f$displayLabel,
    #payload: _f$payload,
  };

  static SaveRecentSearchRequest _instantiate(DecodingData data) {
    return SaveRecentSearchRequest(
      type: data.dec(_f$type),
      displayLabel: data.dec(_f$displayLabel),
      payload: data.dec(_f$payload),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SaveRecentSearchRequest fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SaveRecentSearchRequest>(map);
  }

  static SaveRecentSearchRequest fromJson(String json) {
    return ensureInitialized().decodeJson<SaveRecentSearchRequest>(json);
  }
}

mixin SaveRecentSearchRequestMappable {
  String toJson() {
    return SaveRecentSearchRequestMapper.ensureInitialized()
        .encodeJson<SaveRecentSearchRequest>(this as SaveRecentSearchRequest);
  }

  Map<String, dynamic> toMap() {
    return SaveRecentSearchRequestMapper.ensureInitialized()
        .encodeMap<SaveRecentSearchRequest>(this as SaveRecentSearchRequest);
  }

  SaveRecentSearchRequestCopyWith<
    SaveRecentSearchRequest,
    SaveRecentSearchRequest,
    SaveRecentSearchRequest
  >
  get copyWith =>
      _SaveRecentSearchRequestCopyWithImpl<
        SaveRecentSearchRequest,
        SaveRecentSearchRequest
      >(this as SaveRecentSearchRequest, $identity, $identity);
  @override
  String toString() {
    return SaveRecentSearchRequestMapper.ensureInitialized().stringifyValue(
      this as SaveRecentSearchRequest,
    );
  }

  @override
  bool operator ==(Object other) {
    return SaveRecentSearchRequestMapper.ensureInitialized().equalsValue(
      this as SaveRecentSearchRequest,
      other,
    );
  }

  @override
  int get hashCode {
    return SaveRecentSearchRequestMapper.ensureInitialized().hashValue(
      this as SaveRecentSearchRequest,
    );
  }
}

extension SaveRecentSearchRequestValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SaveRecentSearchRequest, $Out> {
  SaveRecentSearchRequestCopyWith<$R, SaveRecentSearchRequest, $Out>
  get $asSaveRecentSearchRequest => $base.as(
    (v, t, t2) => _SaveRecentSearchRequestCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SaveRecentSearchRequestCopyWith<
  $R,
  $In extends SaveRecentSearchRequest,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    RecentSearchType? type,
    String? displayLabel,
    RecentSearchPayload? payload,
  });
  SaveRecentSearchRequestCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SaveRecentSearchRequestCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SaveRecentSearchRequest, $Out>
    implements
        SaveRecentSearchRequestCopyWith<$R, SaveRecentSearchRequest, $Out> {
  _SaveRecentSearchRequestCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SaveRecentSearchRequest> $mapper =
      SaveRecentSearchRequestMapper.ensureInitialized();
  @override
  $R call({
    RecentSearchType? type,
    String? displayLabel,
    RecentSearchPayload? payload,
  }) => $apply(
    FieldCopyWithData({
      if (type != null) #type: type,
      if (displayLabel != null) #displayLabel: displayLabel,
      if (payload != null) #payload: payload,
    }),
  );
  @override
  SaveRecentSearchRequest $make(CopyWithData data) => SaveRecentSearchRequest(
    type: data.get(#type, or: $value.type),
    displayLabel: data.get(#displayLabel, or: $value.displayLabel),
    payload: data.get(#payload, or: $value.payload),
  );

  @override
  SaveRecentSearchRequestCopyWith<$R2, SaveRecentSearchRequest, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SaveRecentSearchRequestCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class RecentSearchItemMapper extends ClassMapperBase<RecentSearchItem> {
  RecentSearchItemMapper._();

  static RecentSearchItemMapper? _instance;
  static RecentSearchItemMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RecentSearchItemMapper._());
      RecentSearchTypeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'RecentSearchItem';

  static String _$id(RecentSearchItem v) => v.id;
  static const Field<RecentSearchItem, String> _f$id = Field('id', _$id);
  static RecentSearchType _$type(RecentSearchItem v) => v.type;
  static const Field<RecentSearchItem, RecentSearchType> _f$type = Field(
    'type',
    _$type,
  );
  static String _$displayLabel(RecentSearchItem v) => v.displayLabel;
  static const Field<RecentSearchItem, String> _f$displayLabel = Field(
    'displayLabel',
    _$displayLabel,
  );
  static RecentSearchPayload _$payload(RecentSearchItem v) => v.payload;
  static const Field<RecentSearchItem, RecentSearchPayload> _f$payload = Field(
    'payload',
    _$payload,
  );
  static DateTime _$updatedAt(RecentSearchItem v) => v.updatedAt;
  static const Field<RecentSearchItem, DateTime> _f$updatedAt = Field(
    'updatedAt',
    _$updatedAt,
  );

  @override
  final MappableFields<RecentSearchItem> fields = const {
    #id: _f$id,
    #type: _f$type,
    #displayLabel: _f$displayLabel,
    #payload: _f$payload,
    #updatedAt: _f$updatedAt,
  };

  static RecentSearchItem _instantiate(DecodingData data) {
    return RecentSearchItem(
      id: data.dec(_f$id),
      type: data.dec(_f$type),
      displayLabel: data.dec(_f$displayLabel),
      payload: data.dec(_f$payload),
      updatedAt: data.dec(_f$updatedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static RecentSearchItem fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<RecentSearchItem>(map);
  }

  static RecentSearchItem fromJson(String json) {
    return ensureInitialized().decodeJson<RecentSearchItem>(json);
  }
}

mixin RecentSearchItemMappable {
  String toJson() {
    return RecentSearchItemMapper.ensureInitialized()
        .encodeJson<RecentSearchItem>(this as RecentSearchItem);
  }

  Map<String, dynamic> toMap() {
    return RecentSearchItemMapper.ensureInitialized()
        .encodeMap<RecentSearchItem>(this as RecentSearchItem);
  }

  RecentSearchItemCopyWith<RecentSearchItem, RecentSearchItem, RecentSearchItem>
  get copyWith =>
      _RecentSearchItemCopyWithImpl<RecentSearchItem, RecentSearchItem>(
        this as RecentSearchItem,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return RecentSearchItemMapper.ensureInitialized().stringifyValue(
      this as RecentSearchItem,
    );
  }

  @override
  bool operator ==(Object other) {
    return RecentSearchItemMapper.ensureInitialized().equalsValue(
      this as RecentSearchItem,
      other,
    );
  }

  @override
  int get hashCode {
    return RecentSearchItemMapper.ensureInitialized().hashValue(
      this as RecentSearchItem,
    );
  }
}

extension RecentSearchItemValueCopy<$R, $Out>
    on ObjectCopyWith<$R, RecentSearchItem, $Out> {
  RecentSearchItemCopyWith<$R, RecentSearchItem, $Out>
  get $asRecentSearchItem =>
      $base.as((v, t, t2) => _RecentSearchItemCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class RecentSearchItemCopyWith<$R, $In extends RecentSearchItem, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? id,
    RecentSearchType? type,
    String? displayLabel,
    RecentSearchPayload? payload,
    DateTime? updatedAt,
  });
  RecentSearchItemCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _RecentSearchItemCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, RecentSearchItem, $Out>
    implements RecentSearchItemCopyWith<$R, RecentSearchItem, $Out> {
  _RecentSearchItemCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<RecentSearchItem> $mapper =
      RecentSearchItemMapper.ensureInitialized();
  @override
  $R call({
    String? id,
    RecentSearchType? type,
    String? displayLabel,
    RecentSearchPayload? payload,
    DateTime? updatedAt,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (type != null) #type: type,
      if (displayLabel != null) #displayLabel: displayLabel,
      if (payload != null) #payload: payload,
      if (updatedAt != null) #updatedAt: updatedAt,
    }),
  );
  @override
  RecentSearchItem $make(CopyWithData data) => RecentSearchItem(
    id: data.get(#id, or: $value.id),
    type: data.get(#type, or: $value.type),
    displayLabel: data.get(#displayLabel, or: $value.displayLabel),
    payload: data.get(#payload, or: $value.payload),
    updatedAt: data.get(#updatedAt, or: $value.updatedAt),
  );

  @override
  RecentSearchItemCopyWith<$R2, RecentSearchItem, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _RecentSearchItemCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class RecentSearchPageMapper extends ClassMapperBase<RecentSearchPage> {
  RecentSearchPageMapper._();

  static RecentSearchPageMapper? _instance;
  static RecentSearchPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RecentSearchPageMapper._());
      RecentSearchItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'RecentSearchPage';

  static List<RecentSearchItem> _$items(RecentSearchPage v) => v.items;
  static const Field<RecentSearchPage, List<RecentSearchItem>> _f$items = Field(
    'items',
    _$items,
  );

  @override
  final MappableFields<RecentSearchPage> fields = const {#items: _f$items};

  static RecentSearchPage _instantiate(DecodingData data) {
    return RecentSearchPage(items: data.dec(_f$items));
  }

  @override
  final Function instantiate = _instantiate;

  static RecentSearchPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<RecentSearchPage>(map);
  }

  static RecentSearchPage fromJson(String json) {
    return ensureInitialized().decodeJson<RecentSearchPage>(json);
  }
}

mixin RecentSearchPageMappable {
  String toJson() {
    return RecentSearchPageMapper.ensureInitialized()
        .encodeJson<RecentSearchPage>(this as RecentSearchPage);
  }

  Map<String, dynamic> toMap() {
    return RecentSearchPageMapper.ensureInitialized()
        .encodeMap<RecentSearchPage>(this as RecentSearchPage);
  }

  RecentSearchPageCopyWith<RecentSearchPage, RecentSearchPage, RecentSearchPage>
  get copyWith =>
      _RecentSearchPageCopyWithImpl<RecentSearchPage, RecentSearchPage>(
        this as RecentSearchPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return RecentSearchPageMapper.ensureInitialized().stringifyValue(
      this as RecentSearchPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return RecentSearchPageMapper.ensureInitialized().equalsValue(
      this as RecentSearchPage,
      other,
    );
  }

  @override
  int get hashCode {
    return RecentSearchPageMapper.ensureInitialized().hashValue(
      this as RecentSearchPage,
    );
  }
}

extension RecentSearchPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, RecentSearchPage, $Out> {
  RecentSearchPageCopyWith<$R, RecentSearchPage, $Out>
  get $asRecentSearchPage =>
      $base.as((v, t, t2) => _RecentSearchPageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class RecentSearchPageCopyWith<$R, $In extends RecentSearchPage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    RecentSearchItem,
    RecentSearchItemCopyWith<$R, RecentSearchItem, RecentSearchItem>
  >
  get items;
  $R call({List<RecentSearchItem>? items});
  RecentSearchPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _RecentSearchPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, RecentSearchPage, $Out>
    implements RecentSearchPageCopyWith<$R, RecentSearchPage, $Out> {
  _RecentSearchPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<RecentSearchPage> $mapper =
      RecentSearchPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    RecentSearchItem,
    RecentSearchItemCopyWith<$R, RecentSearchItem, RecentSearchItem>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<RecentSearchItem>? items}) =>
      $apply(FieldCopyWithData({if (items != null) #items: items}));
  @override
  RecentSearchPage $make(CopyWithData data) =>
      RecentSearchPage(items: data.get(#items, or: $value.items));

  @override
  RecentSearchPageCopyWith<$R2, RecentSearchPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _RecentSearchPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

