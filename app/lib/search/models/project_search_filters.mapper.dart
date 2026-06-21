// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'project_search_filters.dart';

class ProjectSearchFiltersMapper extends ClassMapperBase<ProjectSearchFilters> {
  ProjectSearchFiltersMapper._();

  static ProjectSearchFiltersMapper? _instance;
  static ProjectSearchFiltersMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectSearchFiltersMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectSearchFilters';

  static List<String> _$craftType(ProjectSearchFilters v) => v.craftType;
  static const Field<ProjectSearchFilters, List<String>> _f$craftType = Field(
    'craftType',
    _$craftType,
    opt: true,
    def: const [],
  );
  static List<String> _$projectType(ProjectSearchFilters v) => v.projectType;
  static const Field<ProjectSearchFilters, List<String>> _f$projectType = Field(
    'projectType',
    _$projectType,
    opt: true,
    def: const [],
  );
  static List<String> _$patternDifficulty(ProjectSearchFilters v) =>
      v.patternDifficulty;
  static const Field<ProjectSearchFilters, List<String>> _f$patternDifficulty =
      Field('patternDifficulty', _$patternDifficulty, opt: true, def: const []);
  static List<String> _$color(ProjectSearchFilters v) => v.color;
  static const Field<ProjectSearchFilters, List<String>> _f$color = Field(
    'color',
    _$color,
    opt: true,
    def: const [],
  );
  static List<String> _$material(ProjectSearchFilters v) => v.material;
  static const Field<ProjectSearchFilters, List<String>> _f$material = Field(
    'material',
    _$material,
    opt: true,
    def: const [],
  );
  static List<String> _$designTag(ProjectSearchFilters v) => v.designTag;
  static const Field<ProjectSearchFilters, List<String>> _f$designTag = Field(
    'designTag',
    _$designTag,
    opt: true,
    def: const [],
  );
  static List<String> _$projectTag(ProjectSearchFilters v) => v.projectTag;
  static const Field<ProjectSearchFilters, List<String>> _f$projectTag = Field(
    'projectTag',
    _$projectTag,
    opt: true,
    def: const [],
  );

  @override
  final MappableFields<ProjectSearchFilters> fields = const {
    #craftType: _f$craftType,
    #projectType: _f$projectType,
    #patternDifficulty: _f$patternDifficulty,
    #color: _f$color,
    #material: _f$material,
    #designTag: _f$designTag,
    #projectTag: _f$projectTag,
  };

  static ProjectSearchFilters _instantiate(DecodingData data) {
    return ProjectSearchFilters(
      craftType: data.dec(_f$craftType),
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

  static ProjectSearchFilters fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectSearchFilters>(map);
  }

  static ProjectSearchFilters fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectSearchFilters>(json);
  }
}

mixin ProjectSearchFiltersMappable {
  String toJson() {
    return ProjectSearchFiltersMapper.ensureInitialized()
        .encodeJson<ProjectSearchFilters>(this as ProjectSearchFilters);
  }

  Map<String, dynamic> toMap() {
    return ProjectSearchFiltersMapper.ensureInitialized()
        .encodeMap<ProjectSearchFilters>(this as ProjectSearchFilters);
  }

  ProjectSearchFiltersCopyWith<
    ProjectSearchFilters,
    ProjectSearchFilters,
    ProjectSearchFilters
  >
  get copyWith =>
      _ProjectSearchFiltersCopyWithImpl<
        ProjectSearchFilters,
        ProjectSearchFilters
      >(this as ProjectSearchFilters, $identity, $identity);
  @override
  String toString() {
    return ProjectSearchFiltersMapper.ensureInitialized().stringifyValue(
      this as ProjectSearchFilters,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectSearchFiltersMapper.ensureInitialized().equalsValue(
      this as ProjectSearchFilters,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectSearchFiltersMapper.ensureInitialized().hashValue(
      this as ProjectSearchFilters,
    );
  }
}

extension ProjectSearchFiltersValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectSearchFilters, $Out> {
  ProjectSearchFiltersCopyWith<$R, ProjectSearchFilters, $Out>
  get $asProjectSearchFilters => $base.as(
    (v, t, t2) => _ProjectSearchFiltersCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProjectSearchFiltersCopyWith<
  $R,
  $In extends ProjectSearchFilters,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get craftType;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get projectType;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get patternDifficulty;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get color;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get material;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get designTag;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get projectTag;
  $R call({
    List<String>? craftType,
    List<String>? projectType,
    List<String>? patternDifficulty,
    List<String>? color,
    List<String>? material,
    List<String>? designTag,
    List<String>? projectTag,
  });
  ProjectSearchFiltersCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectSearchFiltersCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectSearchFilters, $Out>
    implements ProjectSearchFiltersCopyWith<$R, ProjectSearchFilters, $Out> {
  _ProjectSearchFiltersCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectSearchFilters> $mapper =
      ProjectSearchFiltersMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get craftType =>
      ListCopyWith(
        $value.craftType,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(craftType: v),
      );
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
    List<String>? craftType,
    List<String>? projectType,
    List<String>? patternDifficulty,
    List<String>? color,
    List<String>? material,
    List<String>? designTag,
    List<String>? projectTag,
  }) => $apply(
    FieldCopyWithData({
      if (craftType != null) #craftType: craftType,
      if (projectType != null) #projectType: projectType,
      if (patternDifficulty != null) #patternDifficulty: patternDifficulty,
      if (color != null) #color: color,
      if (material != null) #material: material,
      if (designTag != null) #designTag: designTag,
      if (projectTag != null) #projectTag: projectTag,
    }),
  );
  @override
  ProjectSearchFilters $make(CopyWithData data) => ProjectSearchFilters(
    craftType: data.get(#craftType, or: $value.craftType),
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
  ProjectSearchFiltersCopyWith<$R2, ProjectSearchFilters, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProjectSearchFiltersCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

