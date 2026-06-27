// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'project_browse_filters.dart';

class ProjectBrowseQueryMapper extends ClassMapperBase<ProjectBrowseQuery> {
  ProjectBrowseQueryMapper._();

  static ProjectBrowseQueryMapper? _instance;
  static ProjectBrowseQueryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectBrowseQueryMapper._());
      ProjectBrowseFiltersMapper.ensureInitialized();
      SearchSortMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectBrowseQuery';

  static List<String> _$craftTypes(ProjectBrowseQuery v) => v.craftTypes;
  static const Field<ProjectBrowseQuery, List<String>> _f$craftTypes = Field(
    'craftTypes',
    _$craftTypes,
    opt: true,
    def: const [],
  );
  static ProjectBrowseFilters _$filters(ProjectBrowseQuery v) => v.filters;
  static const Field<ProjectBrowseQuery, ProjectBrowseFilters> _f$filters =
      Field('filters', _$filters, opt: true, def: const ProjectBrowseFilters());
  static SearchSort _$sort(ProjectBrowseQuery v) => v.sort;
  static const Field<ProjectBrowseQuery, SearchSort> _f$sort = Field(
    'sort',
    _$sort,
    opt: true,
    def: SearchSort.chronological,
  );

  @override
  final MappableFields<ProjectBrowseQuery> fields = const {
    #craftTypes: _f$craftTypes,
    #filters: _f$filters,
    #sort: _f$sort,
  };

  static ProjectBrowseQuery _instantiate(DecodingData data) {
    return ProjectBrowseQuery(
      craftTypes: data.dec(_f$craftTypes),
      filters: data.dec(_f$filters),
      sort: data.dec(_f$sort),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectBrowseQuery fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectBrowseQuery>(map);
  }

  static ProjectBrowseQuery fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectBrowseQuery>(json);
  }
}

mixin ProjectBrowseQueryMappable {
  String toJson() {
    return ProjectBrowseQueryMapper.ensureInitialized()
        .encodeJson<ProjectBrowseQuery>(this as ProjectBrowseQuery);
  }

  Map<String, dynamic> toMap() {
    return ProjectBrowseQueryMapper.ensureInitialized()
        .encodeMap<ProjectBrowseQuery>(this as ProjectBrowseQuery);
  }

  ProjectBrowseQueryCopyWith<
    ProjectBrowseQuery,
    ProjectBrowseQuery,
    ProjectBrowseQuery
  >
  get copyWith =>
      _ProjectBrowseQueryCopyWithImpl<ProjectBrowseQuery, ProjectBrowseQuery>(
        this as ProjectBrowseQuery,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProjectBrowseQueryMapper.ensureInitialized().stringifyValue(
      this as ProjectBrowseQuery,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectBrowseQueryMapper.ensureInitialized().equalsValue(
      this as ProjectBrowseQuery,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectBrowseQueryMapper.ensureInitialized().hashValue(
      this as ProjectBrowseQuery,
    );
  }
}

extension ProjectBrowseQueryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectBrowseQuery, $Out> {
  ProjectBrowseQueryCopyWith<$R, ProjectBrowseQuery, $Out>
  get $asProjectBrowseQuery => $base.as(
    (v, t, t2) => _ProjectBrowseQueryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProjectBrowseQueryCopyWith<
  $R,
  $In extends ProjectBrowseQuery,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get craftTypes;
  ProjectBrowseFiltersCopyWith<$R, ProjectBrowseFilters, ProjectBrowseFilters>
  get filters;
  $R call({
    List<String>? craftTypes,
    ProjectBrowseFilters? filters,
    SearchSort? sort,
  });
  ProjectBrowseQueryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectBrowseQueryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectBrowseQuery, $Out>
    implements ProjectBrowseQueryCopyWith<$R, ProjectBrowseQuery, $Out> {
  _ProjectBrowseQueryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectBrowseQuery> $mapper =
      ProjectBrowseQueryMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get craftTypes =>
      ListCopyWith(
        $value.craftTypes,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(craftTypes: v),
      );
  @override
  ProjectBrowseFiltersCopyWith<$R, ProjectBrowseFilters, ProjectBrowseFilters>
  get filters => $value.filters.copyWith.$chain((v) => call(filters: v));
  @override
  $R call({
    List<String>? craftTypes,
    ProjectBrowseFilters? filters,
    SearchSort? sort,
  }) => $apply(
    FieldCopyWithData({
      if (craftTypes != null) #craftTypes: craftTypes,
      if (filters != null) #filters: filters,
      if (sort != null) #sort: sort,
    }),
  );
  @override
  ProjectBrowseQuery $make(CopyWithData data) => ProjectBrowseQuery(
    craftTypes: data.get(#craftTypes, or: $value.craftTypes),
    filters: data.get(#filters, or: $value.filters),
    sort: data.get(#sort, or: $value.sort),
  );

  @override
  ProjectBrowseQueryCopyWith<$R2, ProjectBrowseQuery, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProjectBrowseQueryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectBrowseFiltersMapper extends ClassMapperBase<ProjectBrowseFilters> {
  ProjectBrowseFiltersMapper._();

  static ProjectBrowseFiltersMapper? _instance;
  static ProjectBrowseFiltersMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectBrowseFiltersMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectBrowseFilters';

  static List<String> _$projectType(ProjectBrowseFilters v) => v.projectType;
  static const Field<ProjectBrowseFilters, List<String>> _f$projectType = Field(
    'projectType',
    _$projectType,
    opt: true,
    def: const [],
  );
  static List<String> _$patternDifficulty(ProjectBrowseFilters v) =>
      v.patternDifficulty;
  static const Field<ProjectBrowseFilters, List<String>> _f$patternDifficulty =
      Field('patternDifficulty', _$patternDifficulty, opt: true, def: const []);
  static List<String> _$color(ProjectBrowseFilters v) => v.color;
  static const Field<ProjectBrowseFilters, List<String>> _f$color = Field(
    'color',
    _$color,
    opt: true,
    def: const [],
  );
  static List<String> _$material(ProjectBrowseFilters v) => v.material;
  static const Field<ProjectBrowseFilters, List<String>> _f$material = Field(
    'material',
    _$material,
    opt: true,
    def: const [],
  );
  static List<String> _$designTag(ProjectBrowseFilters v) => v.designTag;
  static const Field<ProjectBrowseFilters, List<String>> _f$designTag = Field(
    'designTag',
    _$designTag,
    opt: true,
    def: const [],
  );
  static List<String> _$projectTag(ProjectBrowseFilters v) => v.projectTag;
  static const Field<ProjectBrowseFilters, List<String>> _f$projectTag = Field(
    'projectTag',
    _$projectTag,
    opt: true,
    def: const [],
  );

  @override
  final MappableFields<ProjectBrowseFilters> fields = const {
    #projectType: _f$projectType,
    #patternDifficulty: _f$patternDifficulty,
    #color: _f$color,
    #material: _f$material,
    #designTag: _f$designTag,
    #projectTag: _f$projectTag,
  };

  static ProjectBrowseFilters _instantiate(DecodingData data) {
    return ProjectBrowseFilters(
      projectType: data.dec(_f$projectType),
      patternDifficulty: data.dec(_f$patternDifficulty),
      color: data.dec(_f$color),
      material: data.dec(_f$material),
      designTag: data.dec(_f$designTag),
      projectTag: data.dec(_f$projectTag),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectBrowseFilters fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectBrowseFilters>(map);
  }

  static ProjectBrowseFilters fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectBrowseFilters>(json);
  }
}

mixin ProjectBrowseFiltersMappable {
  String toJson() {
    return ProjectBrowseFiltersMapper.ensureInitialized()
        .encodeJson<ProjectBrowseFilters>(this as ProjectBrowseFilters);
  }

  Map<String, dynamic> toMap() {
    return ProjectBrowseFiltersMapper.ensureInitialized()
        .encodeMap<ProjectBrowseFilters>(this as ProjectBrowseFilters);
  }

  ProjectBrowseFiltersCopyWith<
    ProjectBrowseFilters,
    ProjectBrowseFilters,
    ProjectBrowseFilters
  >
  get copyWith =>
      _ProjectBrowseFiltersCopyWithImpl<
        ProjectBrowseFilters,
        ProjectBrowseFilters
      >(this as ProjectBrowseFilters, $identity, $identity);
  @override
  String toString() {
    return ProjectBrowseFiltersMapper.ensureInitialized().stringifyValue(
      this as ProjectBrowseFilters,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectBrowseFiltersMapper.ensureInitialized().equalsValue(
      this as ProjectBrowseFilters,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectBrowseFiltersMapper.ensureInitialized().hashValue(
      this as ProjectBrowseFilters,
    );
  }
}

extension ProjectBrowseFiltersValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectBrowseFilters, $Out> {
  ProjectBrowseFiltersCopyWith<$R, ProjectBrowseFilters, $Out>
  get $asProjectBrowseFilters => $base.as(
    (v, t, t2) => _ProjectBrowseFiltersCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProjectBrowseFiltersCopyWith<
  $R,
  $In extends ProjectBrowseFilters,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get projectType;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get patternDifficulty;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get color;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get material;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get designTag;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get projectTag;
  $R call({
    List<String>? projectType,
    List<String>? patternDifficulty,
    List<String>? color,
    List<String>? material,
    List<String>? designTag,
    List<String>? projectTag,
  });
  ProjectBrowseFiltersCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectBrowseFiltersCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectBrowseFilters, $Out>
    implements ProjectBrowseFiltersCopyWith<$R, ProjectBrowseFilters, $Out> {
  _ProjectBrowseFiltersCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectBrowseFilters> $mapper =
      ProjectBrowseFiltersMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get projectType => ListCopyWith(
    $value.projectType,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(projectType: v),
  );
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get patternDifficulty => ListCopyWith(
    $value.patternDifficulty,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(patternDifficulty: v),
  );
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get color =>
      ListCopyWith(
        $value.color,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(color: v),
      );
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get material =>
      ListCopyWith(
        $value.material,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(material: v),
      );
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get designTag =>
      ListCopyWith(
        $value.designTag,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(designTag: v),
      );
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get projectTag =>
      ListCopyWith(
        $value.projectTag,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(projectTag: v),
      );
  @override
  $R call({
    List<String>? projectType,
    List<String>? patternDifficulty,
    List<String>? color,
    List<String>? material,
    List<String>? designTag,
    List<String>? projectTag,
  }) => $apply(
    FieldCopyWithData({
      if (projectType != null) #projectType: projectType,
      if (patternDifficulty != null) #patternDifficulty: patternDifficulty,
      if (color != null) #color: color,
      if (material != null) #material: material,
      if (designTag != null) #designTag: designTag,
      if (projectTag != null) #projectTag: projectTag,
    }),
  );
  @override
  ProjectBrowseFilters $make(CopyWithData data) => ProjectBrowseFilters(
    projectType: data.get(#projectType, or: $value.projectType),
    patternDifficulty: data.get(
      #patternDifficulty,
      or: $value.patternDifficulty,
    ),
    color: data.get(#color, or: $value.color),
    material: data.get(#material, or: $value.material),
    designTag: data.get(#designTag, or: $value.designTag),
    projectTag: data.get(#projectTag, or: $value.projectTag),
  );

  @override
  ProjectBrowseFiltersCopyWith<$R2, ProjectBrowseFilters, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProjectBrowseFiltersCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

