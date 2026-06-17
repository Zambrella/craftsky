// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'project.dart';

class ProjectMapper extends ClassMapperBase<Project> {
  ProjectMapper._();

  static ProjectMapper? _instance;
  static ProjectMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectMapper._());
      MapperContainer.globals.useAll([ProjectDetailsMapper()]);
      ProjectCommonMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'Project';

  static ProjectCommon _$common(Project v) => v.common;
  static const Field<Project, ProjectCommon> _f$common = Field(
    'common',
    _$common,
  );
  static ProjectDetails? _$details(Project v) => v.details;
  static const Field<Project, ProjectDetails> _f$details = Field(
    'details',
    _$details,
    opt: true,
    hook: ProjectDetailsFieldHook(),
  );

  @override
  final MappableFields<Project> fields = const {
    #common: _f$common,
    #details: _f$details,
  };
  @override
  final bool ignoreNull = true;

  static Project _instantiate(DecodingData data) {
    return Project(common: data.dec(_f$common), details: data.dec(_f$details));
  }

  @override
  final Function instantiate = _instantiate;

  static Project fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<Project>(map);
  }

  static Project fromJson(String json) {
    return ensureInitialized().decodeJson<Project>(json);
  }
}

mixin ProjectMappable {
  String toJson() {
    return ProjectMapper.ensureInitialized().encodeJson<Project>(
      this as Project,
    );
  }

  Map<String, dynamic> toMap() {
    return ProjectMapper.ensureInitialized().encodeMap<Project>(
      this as Project,
    );
  }

  ProjectCopyWith<Project, Project, Project> get copyWith =>
      _ProjectCopyWithImpl<Project, Project>(
        this as Project,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProjectMapper.ensureInitialized().stringifyValue(this as Project);
  }

  @override
  bool operator ==(Object other) {
    return ProjectMapper.ensureInitialized().equalsValue(
      this as Project,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectMapper.ensureInitialized().hashValue(this as Project);
  }
}

extension ProjectValueCopy<$R, $Out> on ObjectCopyWith<$R, Project, $Out> {
  ProjectCopyWith<$R, Project, $Out> get $asProject =>
      $base.as((v, t, t2) => _ProjectCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProjectCopyWith<$R, $In extends Project, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ProjectCommonCopyWith<$R, ProjectCommon, ProjectCommon> get common;
  $R call({ProjectCommon? common, ProjectDetails? details});
  ProjectCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ProjectCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, Project, $Out>
    implements ProjectCopyWith<$R, Project, $Out> {
  _ProjectCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<Project> $mapper =
      ProjectMapper.ensureInitialized();
  @override
  ProjectCommonCopyWith<$R, ProjectCommon, ProjectCommon> get common =>
      $value.common.copyWith.$chain((v) => call(common: v));
  @override
  $R call({ProjectCommon? common, Object? details = $none}) => $apply(
    FieldCopyWithData({
      if (common != null) #common: common,
      if (details != $none) #details: details,
    }),
  );
  @override
  Project $make(CopyWithData data) => Project(
    common: data.get(#common, or: $value.common),
    details: data.get(#details, or: $value.details),
  );

  @override
  ProjectCopyWith<$R2, Project, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProjectCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectCommonMapper extends ClassMapperBase<ProjectCommon> {
  ProjectCommonMapper._();

  static ProjectCommonMapper? _instance;
  static ProjectCommonMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectCommonMapper._());
      ProjectPatternMapper.ensureInitialized();
      ProjectMaterialMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectCommon';

  static String _$craftType(ProjectCommon v) => v.craftType;
  static const Field<ProjectCommon, String> _f$craftType = Field(
    'craftType',
    _$craftType,
  );
  static String? _$status(ProjectCommon v) => v.status;
  static const Field<ProjectCommon, String> _f$status = Field(
    'status',
    _$status,
    opt: true,
  );
  static String? _$title(ProjectCommon v) => v.title;
  static const Field<ProjectCommon, String> _f$title = Field(
    'title',
    _$title,
    opt: true,
  );
  static String? _$duration(ProjectCommon v) => v.duration;
  static const Field<ProjectCommon, String> _f$duration = Field(
    'duration',
    _$duration,
    opt: true,
  );
  static ProjectPattern? _$pattern(ProjectCommon v) => v.pattern;
  static const Field<ProjectCommon, ProjectPattern> _f$pattern = Field(
    'pattern',
    _$pattern,
    opt: true,
  );
  static List<ProjectMaterial>? _$materials(ProjectCommon v) => v.materials;
  static const Field<ProjectCommon, List<ProjectMaterial>> _f$materials = Field(
    'materials',
    _$materials,
    opt: true,
  );
  static List<String>? _$colors(ProjectCommon v) => v.colors;
  static const Field<ProjectCommon, List<String>> _f$colors = Field(
    'colors',
    _$colors,
    opt: true,
  );
  static List<String>? _$designTags(ProjectCommon v) => v.designTags;
  static const Field<ProjectCommon, List<String>> _f$designTags = Field(
    'designTags',
    _$designTags,
    opt: true,
  );
  static List<String>? _$tags(ProjectCommon v) => v.tags;
  static const Field<ProjectCommon, List<String>> _f$tags = Field(
    'tags',
    _$tags,
    opt: true,
  );

  @override
  final MappableFields<ProjectCommon> fields = const {
    #craftType: _f$craftType,
    #status: _f$status,
    #title: _f$title,
    #duration: _f$duration,
    #pattern: _f$pattern,
    #materials: _f$materials,
    #colors: _f$colors,
    #designTags: _f$designTags,
    #tags: _f$tags,
  };
  @override
  final bool ignoreNull = true;

  static ProjectCommon _instantiate(DecodingData data) {
    return ProjectCommon(
      craftType: data.dec(_f$craftType),
      status: data.dec(_f$status),
      title: data.dec(_f$title),
      duration: data.dec(_f$duration),
      pattern: data.dec(_f$pattern),
      materials: data.dec(_f$materials),
      colors: data.dec(_f$colors),
      designTags: data.dec(_f$designTags),
      tags: data.dec(_f$tags),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectCommon fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectCommon>(map);
  }

  static ProjectCommon fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectCommon>(json);
  }
}

mixin ProjectCommonMappable {
  String toJson() {
    return ProjectCommonMapper.ensureInitialized().encodeJson<ProjectCommon>(
      this as ProjectCommon,
    );
  }

  Map<String, dynamic> toMap() {
    return ProjectCommonMapper.ensureInitialized().encodeMap<ProjectCommon>(
      this as ProjectCommon,
    );
  }

  ProjectCommonCopyWith<ProjectCommon, ProjectCommon, ProjectCommon>
  get copyWith => _ProjectCommonCopyWithImpl<ProjectCommon, ProjectCommon>(
    this as ProjectCommon,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ProjectCommonMapper.ensureInitialized().stringifyValue(
      this as ProjectCommon,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectCommonMapper.ensureInitialized().equalsValue(
      this as ProjectCommon,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectCommonMapper.ensureInitialized().hashValue(
      this as ProjectCommon,
    );
  }
}

extension ProjectCommonValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectCommon, $Out> {
  ProjectCommonCopyWith<$R, ProjectCommon, $Out> get $asProjectCommon =>
      $base.as((v, t, t2) => _ProjectCommonCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProjectCommonCopyWith<$R, $In extends ProjectCommon, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ProjectPatternCopyWith<$R, ProjectPattern, ProjectPattern>? get pattern;
  ListCopyWith<
    $R,
    ProjectMaterial,
    ProjectMaterialCopyWith<$R, ProjectMaterial, ProjectMaterial>
  >?
  get materials;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>? get colors;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>? get designTags;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>? get tags;
  $R call({
    String? craftType,
    String? status,
    String? title,
    String? duration,
    ProjectPattern? pattern,
    List<ProjectMaterial>? materials,
    List<String>? colors,
    List<String>? designTags,
    List<String>? tags,
  });
  ProjectCommonCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ProjectCommonCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectCommon, $Out>
    implements ProjectCommonCopyWith<$R, ProjectCommon, $Out> {
  _ProjectCommonCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectCommon> $mapper =
      ProjectCommonMapper.ensureInitialized();
  @override
  ProjectPatternCopyWith<$R, ProjectPattern, ProjectPattern>? get pattern =>
      $value.pattern?.copyWith.$chain((v) => call(pattern: v));
  @override
  ListCopyWith<
    $R,
    ProjectMaterial,
    ProjectMaterialCopyWith<$R, ProjectMaterial, ProjectMaterial>
  >?
  get materials => $value.materials != null
      ? ListCopyWith(
          $value.materials!,
          (v, t) => v.copyWith.$chain(t),
          (v) => call(materials: v),
        )
      : null;
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>? get colors =>
      $value.colors != null
      ? ListCopyWith(
          $value.colors!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(colors: v),
        )
      : null;
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>?
  get designTags => $value.designTags != null
      ? ListCopyWith(
          $value.designTags!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(designTags: v),
        )
      : null;
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>? get tags =>
      $value.tags != null
      ? ListCopyWith(
          $value.tags!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(tags: v),
        )
      : null;
  @override
  $R call({
    String? craftType,
    Object? status = $none,
    Object? title = $none,
    Object? duration = $none,
    Object? pattern = $none,
    Object? materials = $none,
    Object? colors = $none,
    Object? designTags = $none,
    Object? tags = $none,
  }) => $apply(
    FieldCopyWithData({
      if (craftType != null) #craftType: craftType,
      if (status != $none) #status: status,
      if (title != $none) #title: title,
      if (duration != $none) #duration: duration,
      if (pattern != $none) #pattern: pattern,
      if (materials != $none) #materials: materials,
      if (colors != $none) #colors: colors,
      if (designTags != $none) #designTags: designTags,
      if (tags != $none) #tags: tags,
    }),
  );
  @override
  ProjectCommon $make(CopyWithData data) => ProjectCommon(
    craftType: data.get(#craftType, or: $value.craftType),
    status: data.get(#status, or: $value.status),
    title: data.get(#title, or: $value.title),
    duration: data.get(#duration, or: $value.duration),
    pattern: data.get(#pattern, or: $value.pattern),
    materials: data.get(#materials, or: $value.materials),
    colors: data.get(#colors, or: $value.colors),
    designTags: data.get(#designTags, or: $value.designTags),
    tags: data.get(#tags, or: $value.tags),
  );

  @override
  ProjectCommonCopyWith<$R2, ProjectCommon, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProjectCommonCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectPatternMapper extends ClassMapperBase<ProjectPattern> {
  ProjectPatternMapper._();

  static ProjectPatternMapper? _instance;
  static ProjectPatternMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectPatternMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectPattern';

  static String? _$url(ProjectPattern v) => v.url;
  static const Field<ProjectPattern, String> _f$url = Field(
    'url',
    _$url,
    opt: true,
  );
  static String? _$name(ProjectPattern v) => v.name;
  static const Field<ProjectPattern, String> _f$name = Field(
    'name',
    _$name,
    opt: true,
  );
  static List<Map<String, dynamic>>? _$nameFacets(ProjectPattern v) =>
      v.nameFacets;
  static const Field<ProjectPattern, List<Map<String, dynamic>>> _f$nameFacets =
      Field('nameFacets', _$nameFacets, opt: true);
  static String? _$difficulty(ProjectPattern v) => v.difficulty;
  static const Field<ProjectPattern, String> _f$difficulty = Field(
    'difficulty',
    _$difficulty,
    opt: true,
  );
  static String? _$designer(ProjectPattern v) => v.designer;
  static const Field<ProjectPattern, String> _f$designer = Field(
    'designer',
    _$designer,
    opt: true,
  );
  static List<Map<String, dynamic>>? _$designerFacets(ProjectPattern v) =>
      v.designerFacets;
  static const Field<ProjectPattern, List<Map<String, dynamic>>>
  _f$designerFacets = Field('designerFacets', _$designerFacets, opt: true);
  static String? _$publisher(ProjectPattern v) => v.publisher;
  static const Field<ProjectPattern, String> _f$publisher = Field(
    'publisher',
    _$publisher,
    opt: true,
  );
  static List<Map<String, dynamic>>? _$publisherFacets(ProjectPattern v) =>
      v.publisherFacets;
  static const Field<ProjectPattern, List<Map<String, dynamic>>>
  _f$publisherFacets = Field('publisherFacets', _$publisherFacets, opt: true);

  @override
  final MappableFields<ProjectPattern> fields = const {
    #url: _f$url,
    #name: _f$name,
    #nameFacets: _f$nameFacets,
    #difficulty: _f$difficulty,
    #designer: _f$designer,
    #designerFacets: _f$designerFacets,
    #publisher: _f$publisher,
    #publisherFacets: _f$publisherFacets,
  };
  @override
  final bool ignoreNull = true;

  static ProjectPattern _instantiate(DecodingData data) {
    return ProjectPattern(
      url: data.dec(_f$url),
      name: data.dec(_f$name),
      nameFacets: data.dec(_f$nameFacets),
      difficulty: data.dec(_f$difficulty),
      designer: data.dec(_f$designer),
      designerFacets: data.dec(_f$designerFacets),
      publisher: data.dec(_f$publisher),
      publisherFacets: data.dec(_f$publisherFacets),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectPattern fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectPattern>(map);
  }

  static ProjectPattern fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectPattern>(json);
  }
}

mixin ProjectPatternMappable {
  String toJson() {
    return ProjectPatternMapper.ensureInitialized().encodeJson<ProjectPattern>(
      this as ProjectPattern,
    );
  }

  Map<String, dynamic> toMap() {
    return ProjectPatternMapper.ensureInitialized().encodeMap<ProjectPattern>(
      this as ProjectPattern,
    );
  }

  ProjectPatternCopyWith<ProjectPattern, ProjectPattern, ProjectPattern>
  get copyWith => _ProjectPatternCopyWithImpl<ProjectPattern, ProjectPattern>(
    this as ProjectPattern,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ProjectPatternMapper.ensureInitialized().stringifyValue(
      this as ProjectPattern,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectPatternMapper.ensureInitialized().equalsValue(
      this as ProjectPattern,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectPatternMapper.ensureInitialized().hashValue(
      this as ProjectPattern,
    );
  }
}

extension ProjectPatternValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectPattern, $Out> {
  ProjectPatternCopyWith<$R, ProjectPattern, $Out> get $asProjectPattern =>
      $base.as((v, t, t2) => _ProjectPatternCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProjectPatternCopyWith<$R, $In extends ProjectPattern, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get nameFacets;
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get designerFacets;
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get publisherFacets;
  $R call({
    String? url,
    String? name,
    List<Map<String, dynamic>>? nameFacets,
    String? difficulty,
    String? designer,
    List<Map<String, dynamic>>? designerFacets,
    String? publisher,
    List<Map<String, dynamic>>? publisherFacets,
  });
  ProjectPatternCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectPatternCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectPattern, $Out>
    implements ProjectPatternCopyWith<$R, ProjectPattern, $Out> {
  _ProjectPatternCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectPattern> $mapper =
      ProjectPatternMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get nameFacets => $value.nameFacets != null
      ? ListCopyWith(
          $value.nameFacets!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(nameFacets: v),
        )
      : null;
  @override
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get designerFacets => $value.designerFacets != null
      ? ListCopyWith(
          $value.designerFacets!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(designerFacets: v),
        )
      : null;
  @override
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get publisherFacets => $value.publisherFacets != null
      ? ListCopyWith(
          $value.publisherFacets!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(publisherFacets: v),
        )
      : null;
  @override
  $R call({
    Object? url = $none,
    Object? name = $none,
    Object? nameFacets = $none,
    Object? difficulty = $none,
    Object? designer = $none,
    Object? designerFacets = $none,
    Object? publisher = $none,
    Object? publisherFacets = $none,
  }) => $apply(
    FieldCopyWithData({
      if (url != $none) #url: url,
      if (name != $none) #name: name,
      if (nameFacets != $none) #nameFacets: nameFacets,
      if (difficulty != $none) #difficulty: difficulty,
      if (designer != $none) #designer: designer,
      if (designerFacets != $none) #designerFacets: designerFacets,
      if (publisher != $none) #publisher: publisher,
      if (publisherFacets != $none) #publisherFacets: publisherFacets,
    }),
  );
  @override
  ProjectPattern $make(CopyWithData data) => ProjectPattern(
    url: data.get(#url, or: $value.url),
    name: data.get(#name, or: $value.name),
    nameFacets: data.get(#nameFacets, or: $value.nameFacets),
    difficulty: data.get(#difficulty, or: $value.difficulty),
    designer: data.get(#designer, or: $value.designer),
    designerFacets: data.get(#designerFacets, or: $value.designerFacets),
    publisher: data.get(#publisher, or: $value.publisher),
    publisherFacets: data.get(#publisherFacets, or: $value.publisherFacets),
  );

  @override
  ProjectPatternCopyWith<$R2, ProjectPattern, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProjectPatternCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectMaterialMapper extends ClassMapperBase<ProjectMaterial> {
  ProjectMaterialMapper._();

  static ProjectMaterialMapper? _instance;
  static ProjectMaterialMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectMaterialMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectMaterial';

  static String _$text(ProjectMaterial v) => v.text;
  static const Field<ProjectMaterial, String> _f$text = Field('text', _$text);
  static List<Map<String, dynamic>>? _$facets(ProjectMaterial v) => v.facets;
  static const Field<ProjectMaterial, List<Map<String, dynamic>>> _f$facets =
      Field('facets', _$facets, opt: true);

  @override
  final MappableFields<ProjectMaterial> fields = const {
    #text: _f$text,
    #facets: _f$facets,
  };
  @override
  final bool ignoreNull = true;

  static ProjectMaterial _instantiate(DecodingData data) {
    return ProjectMaterial(
      text: data.dec(_f$text),
      facets: data.dec(_f$facets),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectMaterial fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectMaterial>(map);
  }

  static ProjectMaterial fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectMaterial>(json);
  }
}

mixin ProjectMaterialMappable {
  String toJson() {
    return ProjectMaterialMapper.ensureInitialized()
        .encodeJson<ProjectMaterial>(this as ProjectMaterial);
  }

  Map<String, dynamic> toMap() {
    return ProjectMaterialMapper.ensureInitialized().encodeMap<ProjectMaterial>(
      this as ProjectMaterial,
    );
  }

  ProjectMaterialCopyWith<ProjectMaterial, ProjectMaterial, ProjectMaterial>
  get copyWith =>
      _ProjectMaterialCopyWithImpl<ProjectMaterial, ProjectMaterial>(
        this as ProjectMaterial,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProjectMaterialMapper.ensureInitialized().stringifyValue(
      this as ProjectMaterial,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectMaterialMapper.ensureInitialized().equalsValue(
      this as ProjectMaterial,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectMaterialMapper.ensureInitialized().hashValue(
      this as ProjectMaterial,
    );
  }
}

extension ProjectMaterialValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectMaterial, $Out> {
  ProjectMaterialCopyWith<$R, ProjectMaterial, $Out> get $asProjectMaterial =>
      $base.as((v, t, t2) => _ProjectMaterialCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProjectMaterialCopyWith<$R, $In extends ProjectMaterial, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get facets;
  $R call({String? text, List<Map<String, dynamic>>? facets});
  ProjectMaterialCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProjectMaterialCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectMaterial, $Out>
    implements ProjectMaterialCopyWith<$R, ProjectMaterial, $Out> {
  _ProjectMaterialCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectMaterial> $mapper =
      ProjectMaterialMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get facets => $value.facets != null
      ? ListCopyWith(
          $value.facets!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(facets: v),
        )
      : null;
  @override
  $R call({String? text, Object? facets = $none}) => $apply(
    FieldCopyWithData({
      if (text != null) #text: text,
      if (facets != $none) #facets: facets,
    }),
  );
  @override
  ProjectMaterial $make(CopyWithData data) => ProjectMaterial(
    text: data.get(#text, or: $value.text),
    facets: data.get(#facets, or: $value.facets),
  );

  @override
  ProjectMaterialCopyWith<$R2, ProjectMaterial, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProjectMaterialCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProjectGaugeMapper extends ClassMapperBase<ProjectGauge> {
  ProjectGaugeMapper._();

  static ProjectGaugeMapper? _instance;
  static ProjectGaugeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProjectGaugeMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProjectGauge';

  static int _$stitches(ProjectGauge v) => v.stitches;
  static const Field<ProjectGauge, int> _f$stitches = Field(
    'stitches',
    _$stitches,
  );
  static int _$measurement(ProjectGauge v) => v.measurement;
  static const Field<ProjectGauge, int> _f$measurement = Field(
    'measurement',
    _$measurement,
  );
  static String _$unit(ProjectGauge v) => v.unit;
  static const Field<ProjectGauge, String> _f$unit = Field('unit', _$unit);
  static int? _$rows(ProjectGauge v) => v.rows;
  static const Field<ProjectGauge, int> _f$rows = Field(
    'rows',
    _$rows,
    opt: true,
  );

  @override
  final MappableFields<ProjectGauge> fields = const {
    #stitches: _f$stitches,
    #measurement: _f$measurement,
    #unit: _f$unit,
    #rows: _f$rows,
  };
  @override
  final bool ignoreNull = true;

  static ProjectGauge _instantiate(DecodingData data) {
    return ProjectGauge(
      stitches: data.dec(_f$stitches),
      measurement: data.dec(_f$measurement),
      unit: data.dec(_f$unit),
      rows: data.dec(_f$rows),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProjectGauge fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProjectGauge>(map);
  }

  static ProjectGauge fromJson(String json) {
    return ensureInitialized().decodeJson<ProjectGauge>(json);
  }
}

mixin ProjectGaugeMappable {
  String toJson() {
    return ProjectGaugeMapper.ensureInitialized().encodeJson<ProjectGauge>(
      this as ProjectGauge,
    );
  }

  Map<String, dynamic> toMap() {
    return ProjectGaugeMapper.ensureInitialized().encodeMap<ProjectGauge>(
      this as ProjectGauge,
    );
  }

  ProjectGaugeCopyWith<ProjectGauge, ProjectGauge, ProjectGauge> get copyWith =>
      _ProjectGaugeCopyWithImpl<ProjectGauge, ProjectGauge>(
        this as ProjectGauge,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProjectGaugeMapper.ensureInitialized().stringifyValue(
      this as ProjectGauge,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProjectGaugeMapper.ensureInitialized().equalsValue(
      this as ProjectGauge,
      other,
    );
  }

  @override
  int get hashCode {
    return ProjectGaugeMapper.ensureInitialized().hashValue(
      this as ProjectGauge,
    );
  }
}

extension ProjectGaugeValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProjectGauge, $Out> {
  ProjectGaugeCopyWith<$R, ProjectGauge, $Out> get $asProjectGauge =>
      $base.as((v, t, t2) => _ProjectGaugeCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProjectGaugeCopyWith<$R, $In extends ProjectGauge, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({int? stitches, int? measurement, String? unit, int? rows});
  ProjectGaugeCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ProjectGaugeCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProjectGauge, $Out>
    implements ProjectGaugeCopyWith<$R, ProjectGauge, $Out> {
  _ProjectGaugeCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProjectGauge> $mapper =
      ProjectGaugeMapper.ensureInitialized();
  @override
  $R call({
    int? stitches,
    int? measurement,
    String? unit,
    Object? rows = $none,
  }) => $apply(
    FieldCopyWithData({
      if (stitches != null) #stitches: stitches,
      if (measurement != null) #measurement: measurement,
      if (unit != null) #unit: unit,
      if (rows != $none) #rows: rows,
    }),
  );
  @override
  ProjectGauge $make(CopyWithData data) => ProjectGauge(
    stitches: data.get(#stitches, or: $value.stitches),
    measurement: data.get(#measurement, or: $value.measurement),
    unit: data.get(#unit, or: $value.unit),
    rows: data.get(#rows, or: $value.rows),
  );

  @override
  ProjectGaugeCopyWith<$R2, ProjectGauge, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProjectGaugeCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class KnittingProjectDetailsMapper
    extends ClassMapperBase<KnittingProjectDetails> {
  KnittingProjectDetailsMapper._();

  static KnittingProjectDetailsMapper? _instance;
  static KnittingProjectDetailsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = KnittingProjectDetailsMapper._());
      ProjectGaugeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'KnittingProjectDetails';

  static String? _$projectType(KnittingProjectDetails v) => v.projectType;
  static const Field<KnittingProjectDetails, String> _f$projectType = Field(
    'projectType',
    _$projectType,
    opt: true,
  );
  static String? _$projectSubtype(KnittingProjectDetails v) => v.projectSubtype;
  static const Field<KnittingProjectDetails, String> _f$projectSubtype = Field(
    'projectSubtype',
    _$projectSubtype,
    opt: true,
  );
  static String? _$yarnWeight(KnittingProjectDetails v) => v.yarnWeight;
  static const Field<KnittingProjectDetails, String> _f$yarnWeight = Field(
    'yarnWeight',
    _$yarnWeight,
    opt: true,
  );
  static String? _$needleSizeMm(KnittingProjectDetails v) => v.needleSizeMm;
  static const Field<KnittingProjectDetails, String> _f$needleSizeMm = Field(
    'needleSizeMm',
    _$needleSizeMm,
    opt: true,
  );
  static ProjectGauge? _$gauge(KnittingProjectDetails v) => v.gauge;
  static const Field<KnittingProjectDetails, ProjectGauge> _f$gauge = Field(
    'gauge',
    _$gauge,
    opt: true,
  );
  static String? _$finishedSize(KnittingProjectDetails v) => v.finishedSize;
  static const Field<KnittingProjectDetails, String> _f$finishedSize = Field(
    'finishedSize',
    _$finishedSize,
    opt: true,
  );

  @override
  final MappableFields<KnittingProjectDetails> fields = const {
    #projectType: _f$projectType,
    #projectSubtype: _f$projectSubtype,
    #yarnWeight: _f$yarnWeight,
    #needleSizeMm: _f$needleSizeMm,
    #gauge: _f$gauge,
    #finishedSize: _f$finishedSize,
  };
  @override
  final bool ignoreNull = true;

  static KnittingProjectDetails _instantiate(DecodingData data) {
    return KnittingProjectDetails(
      projectType: data.dec(_f$projectType),
      projectSubtype: data.dec(_f$projectSubtype),
      yarnWeight: data.dec(_f$yarnWeight),
      needleSizeMm: data.dec(_f$needleSizeMm),
      gauge: data.dec(_f$gauge),
      finishedSize: data.dec(_f$finishedSize),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static KnittingProjectDetails fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<KnittingProjectDetails>(map);
  }

  static KnittingProjectDetails fromJson(String json) {
    return ensureInitialized().decodeJson<KnittingProjectDetails>(json);
  }
}

mixin KnittingProjectDetailsMappable {
  String toJson() {
    return KnittingProjectDetailsMapper.ensureInitialized()
        .encodeJson<KnittingProjectDetails>(this as KnittingProjectDetails);
  }

  Map<String, dynamic> toMap() {
    return KnittingProjectDetailsMapper.ensureInitialized()
        .encodeMap<KnittingProjectDetails>(this as KnittingProjectDetails);
  }

  KnittingProjectDetailsCopyWith<
    KnittingProjectDetails,
    KnittingProjectDetails,
    KnittingProjectDetails
  >
  get copyWith =>
      _KnittingProjectDetailsCopyWithImpl<
        KnittingProjectDetails,
        KnittingProjectDetails
      >(this as KnittingProjectDetails, $identity, $identity);
  @override
  String toString() {
    return KnittingProjectDetailsMapper.ensureInitialized().stringifyValue(
      this as KnittingProjectDetails,
    );
  }

  @override
  bool operator ==(Object other) {
    return KnittingProjectDetailsMapper.ensureInitialized().equalsValue(
      this as KnittingProjectDetails,
      other,
    );
  }

  @override
  int get hashCode {
    return KnittingProjectDetailsMapper.ensureInitialized().hashValue(
      this as KnittingProjectDetails,
    );
  }
}

extension KnittingProjectDetailsValueCopy<$R, $Out>
    on ObjectCopyWith<$R, KnittingProjectDetails, $Out> {
  KnittingProjectDetailsCopyWith<$R, KnittingProjectDetails, $Out>
  get $asKnittingProjectDetails => $base.as(
    (v, t, t2) => _KnittingProjectDetailsCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class KnittingProjectDetailsCopyWith<
  $R,
  $In extends KnittingProjectDetails,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ProjectGaugeCopyWith<$R, ProjectGauge, ProjectGauge>? get gauge;
  $R call({
    String? projectType,
    String? projectSubtype,
    String? yarnWeight,
    String? needleSizeMm,
    ProjectGauge? gauge,
    String? finishedSize,
  });
  KnittingProjectDetailsCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _KnittingProjectDetailsCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, KnittingProjectDetails, $Out>
    implements
        KnittingProjectDetailsCopyWith<$R, KnittingProjectDetails, $Out> {
  _KnittingProjectDetailsCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<KnittingProjectDetails> $mapper =
      KnittingProjectDetailsMapper.ensureInitialized();
  @override
  ProjectGaugeCopyWith<$R, ProjectGauge, ProjectGauge>? get gauge =>
      $value.gauge?.copyWith.$chain((v) => call(gauge: v));
  @override
  $R call({
    Object? projectType = $none,
    Object? projectSubtype = $none,
    Object? yarnWeight = $none,
    Object? needleSizeMm = $none,
    Object? gauge = $none,
    Object? finishedSize = $none,
  }) => $apply(
    FieldCopyWithData({
      if (projectType != $none) #projectType: projectType,
      if (projectSubtype != $none) #projectSubtype: projectSubtype,
      if (yarnWeight != $none) #yarnWeight: yarnWeight,
      if (needleSizeMm != $none) #needleSizeMm: needleSizeMm,
      if (gauge != $none) #gauge: gauge,
      if (finishedSize != $none) #finishedSize: finishedSize,
    }),
  );
  @override
  KnittingProjectDetails $make(CopyWithData data) => KnittingProjectDetails(
    projectType: data.get(#projectType, or: $value.projectType),
    projectSubtype: data.get(#projectSubtype, or: $value.projectSubtype),
    yarnWeight: data.get(#yarnWeight, or: $value.yarnWeight),
    needleSizeMm: data.get(#needleSizeMm, or: $value.needleSizeMm),
    gauge: data.get(#gauge, or: $value.gauge),
    finishedSize: data.get(#finishedSize, or: $value.finishedSize),
  );

  @override
  KnittingProjectDetailsCopyWith<$R2, KnittingProjectDetails, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _KnittingProjectDetailsCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class CrochetProjectDetailsMapper
    extends ClassMapperBase<CrochetProjectDetails> {
  CrochetProjectDetailsMapper._();

  static CrochetProjectDetailsMapper? _instance;
  static CrochetProjectDetailsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CrochetProjectDetailsMapper._());
      ProjectGaugeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'CrochetProjectDetails';

  static String? _$projectType(CrochetProjectDetails v) => v.projectType;
  static const Field<CrochetProjectDetails, String> _f$projectType = Field(
    'projectType',
    _$projectType,
    opt: true,
  );
  static String? _$projectSubtype(CrochetProjectDetails v) => v.projectSubtype;
  static const Field<CrochetProjectDetails, String> _f$projectSubtype = Field(
    'projectSubtype',
    _$projectSubtype,
    opt: true,
  );
  static String? _$yarnWeight(CrochetProjectDetails v) => v.yarnWeight;
  static const Field<CrochetProjectDetails, String> _f$yarnWeight = Field(
    'yarnWeight',
    _$yarnWeight,
    opt: true,
  );
  static String? _$hookSizeMm(CrochetProjectDetails v) => v.hookSizeMm;
  static const Field<CrochetProjectDetails, String> _f$hookSizeMm = Field(
    'hookSizeMm',
    _$hookSizeMm,
    opt: true,
  );
  static ProjectGauge? _$gauge(CrochetProjectDetails v) => v.gauge;
  static const Field<CrochetProjectDetails, ProjectGauge> _f$gauge = Field(
    'gauge',
    _$gauge,
    opt: true,
  );
  static String? _$finishedSize(CrochetProjectDetails v) => v.finishedSize;
  static const Field<CrochetProjectDetails, String> _f$finishedSize = Field(
    'finishedSize',
    _$finishedSize,
    opt: true,
  );

  @override
  final MappableFields<CrochetProjectDetails> fields = const {
    #projectType: _f$projectType,
    #projectSubtype: _f$projectSubtype,
    #yarnWeight: _f$yarnWeight,
    #hookSizeMm: _f$hookSizeMm,
    #gauge: _f$gauge,
    #finishedSize: _f$finishedSize,
  };
  @override
  final bool ignoreNull = true;

  static CrochetProjectDetails _instantiate(DecodingData data) {
    return CrochetProjectDetails(
      projectType: data.dec(_f$projectType),
      projectSubtype: data.dec(_f$projectSubtype),
      yarnWeight: data.dec(_f$yarnWeight),
      hookSizeMm: data.dec(_f$hookSizeMm),
      gauge: data.dec(_f$gauge),
      finishedSize: data.dec(_f$finishedSize),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static CrochetProjectDetails fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CrochetProjectDetails>(map);
  }

  static CrochetProjectDetails fromJson(String json) {
    return ensureInitialized().decodeJson<CrochetProjectDetails>(json);
  }
}

mixin CrochetProjectDetailsMappable {
  String toJson() {
    return CrochetProjectDetailsMapper.ensureInitialized()
        .encodeJson<CrochetProjectDetails>(this as CrochetProjectDetails);
  }

  Map<String, dynamic> toMap() {
    return CrochetProjectDetailsMapper.ensureInitialized()
        .encodeMap<CrochetProjectDetails>(this as CrochetProjectDetails);
  }

  CrochetProjectDetailsCopyWith<
    CrochetProjectDetails,
    CrochetProjectDetails,
    CrochetProjectDetails
  >
  get copyWith =>
      _CrochetProjectDetailsCopyWithImpl<
        CrochetProjectDetails,
        CrochetProjectDetails
      >(this as CrochetProjectDetails, $identity, $identity);
  @override
  String toString() {
    return CrochetProjectDetailsMapper.ensureInitialized().stringifyValue(
      this as CrochetProjectDetails,
    );
  }

  @override
  bool operator ==(Object other) {
    return CrochetProjectDetailsMapper.ensureInitialized().equalsValue(
      this as CrochetProjectDetails,
      other,
    );
  }

  @override
  int get hashCode {
    return CrochetProjectDetailsMapper.ensureInitialized().hashValue(
      this as CrochetProjectDetails,
    );
  }
}

extension CrochetProjectDetailsValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CrochetProjectDetails, $Out> {
  CrochetProjectDetailsCopyWith<$R, CrochetProjectDetails, $Out>
  get $asCrochetProjectDetails => $base.as(
    (v, t, t2) => _CrochetProjectDetailsCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class CrochetProjectDetailsCopyWith<
  $R,
  $In extends CrochetProjectDetails,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ProjectGaugeCopyWith<$R, ProjectGauge, ProjectGauge>? get gauge;
  $R call({
    String? projectType,
    String? projectSubtype,
    String? yarnWeight,
    String? hookSizeMm,
    ProjectGauge? gauge,
    String? finishedSize,
  });
  CrochetProjectDetailsCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _CrochetProjectDetailsCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CrochetProjectDetails, $Out>
    implements CrochetProjectDetailsCopyWith<$R, CrochetProjectDetails, $Out> {
  _CrochetProjectDetailsCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CrochetProjectDetails> $mapper =
      CrochetProjectDetailsMapper.ensureInitialized();
  @override
  ProjectGaugeCopyWith<$R, ProjectGauge, ProjectGauge>? get gauge =>
      $value.gauge?.copyWith.$chain((v) => call(gauge: v));
  @override
  $R call({
    Object? projectType = $none,
    Object? projectSubtype = $none,
    Object? yarnWeight = $none,
    Object? hookSizeMm = $none,
    Object? gauge = $none,
    Object? finishedSize = $none,
  }) => $apply(
    FieldCopyWithData({
      if (projectType != $none) #projectType: projectType,
      if (projectSubtype != $none) #projectSubtype: projectSubtype,
      if (yarnWeight != $none) #yarnWeight: yarnWeight,
      if (hookSizeMm != $none) #hookSizeMm: hookSizeMm,
      if (gauge != $none) #gauge: gauge,
      if (finishedSize != $none) #finishedSize: finishedSize,
    }),
  );
  @override
  CrochetProjectDetails $make(CopyWithData data) => CrochetProjectDetails(
    projectType: data.get(#projectType, or: $value.projectType),
    projectSubtype: data.get(#projectSubtype, or: $value.projectSubtype),
    yarnWeight: data.get(#yarnWeight, or: $value.yarnWeight),
    hookSizeMm: data.get(#hookSizeMm, or: $value.hookSizeMm),
    gauge: data.get(#gauge, or: $value.gauge),
    finishedSize: data.get(#finishedSize, or: $value.finishedSize),
  );

  @override
  CrochetProjectDetailsCopyWith<$R2, CrochetProjectDetails, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _CrochetProjectDetailsCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SewingProjectDetailsMapper extends ClassMapperBase<SewingProjectDetails> {
  SewingProjectDetailsMapper._();

  static SewingProjectDetailsMapper? _instance;
  static SewingProjectDetailsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SewingProjectDetailsMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SewingProjectDetails';

  static String? _$projectType(SewingProjectDetails v) => v.projectType;
  static const Field<SewingProjectDetails, String> _f$projectType = Field(
    'projectType',
    _$projectType,
    opt: true,
  );
  static String? _$projectSubtype(SewingProjectDetails v) => v.projectSubtype;
  static const Field<SewingProjectDetails, String> _f$projectSubtype = Field(
    'projectSubtype',
    _$projectSubtype,
    opt: true,
  );
  static String? _$sizeMade(SewingProjectDetails v) => v.sizeMade;
  static const Field<SewingProjectDetails, String> _f$sizeMade = Field(
    'sizeMade',
    _$sizeMade,
    opt: true,
  );
  static String? _$fitNotes(SewingProjectDetails v) => v.fitNotes;
  static const Field<SewingProjectDetails, String> _f$fitNotes = Field(
    'fitNotes',
    _$fitNotes,
    opt: true,
  );

  @override
  final MappableFields<SewingProjectDetails> fields = const {
    #projectType: _f$projectType,
    #projectSubtype: _f$projectSubtype,
    #sizeMade: _f$sizeMade,
    #fitNotes: _f$fitNotes,
  };
  @override
  final bool ignoreNull = true;

  static SewingProjectDetails _instantiate(DecodingData data) {
    return SewingProjectDetails(
      projectType: data.dec(_f$projectType),
      projectSubtype: data.dec(_f$projectSubtype),
      sizeMade: data.dec(_f$sizeMade),
      fitNotes: data.dec(_f$fitNotes),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SewingProjectDetails fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SewingProjectDetails>(map);
  }

  static SewingProjectDetails fromJson(String json) {
    return ensureInitialized().decodeJson<SewingProjectDetails>(json);
  }
}

mixin SewingProjectDetailsMappable {
  String toJson() {
    return SewingProjectDetailsMapper.ensureInitialized()
        .encodeJson<SewingProjectDetails>(this as SewingProjectDetails);
  }

  Map<String, dynamic> toMap() {
    return SewingProjectDetailsMapper.ensureInitialized()
        .encodeMap<SewingProjectDetails>(this as SewingProjectDetails);
  }

  SewingProjectDetailsCopyWith<
    SewingProjectDetails,
    SewingProjectDetails,
    SewingProjectDetails
  >
  get copyWith =>
      _SewingProjectDetailsCopyWithImpl<
        SewingProjectDetails,
        SewingProjectDetails
      >(this as SewingProjectDetails, $identity, $identity);
  @override
  String toString() {
    return SewingProjectDetailsMapper.ensureInitialized().stringifyValue(
      this as SewingProjectDetails,
    );
  }

  @override
  bool operator ==(Object other) {
    return SewingProjectDetailsMapper.ensureInitialized().equalsValue(
      this as SewingProjectDetails,
      other,
    );
  }

  @override
  int get hashCode {
    return SewingProjectDetailsMapper.ensureInitialized().hashValue(
      this as SewingProjectDetails,
    );
  }
}

extension SewingProjectDetailsValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SewingProjectDetails, $Out> {
  SewingProjectDetailsCopyWith<$R, SewingProjectDetails, $Out>
  get $asSewingProjectDetails => $base.as(
    (v, t, t2) => _SewingProjectDetailsCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SewingProjectDetailsCopyWith<
  $R,
  $In extends SewingProjectDetails,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? projectType,
    String? projectSubtype,
    String? sizeMade,
    String? fitNotes,
  });
  SewingProjectDetailsCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SewingProjectDetailsCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SewingProjectDetails, $Out>
    implements SewingProjectDetailsCopyWith<$R, SewingProjectDetails, $Out> {
  _SewingProjectDetailsCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SewingProjectDetails> $mapper =
      SewingProjectDetailsMapper.ensureInitialized();
  @override
  $R call({
    Object? projectType = $none,
    Object? projectSubtype = $none,
    Object? sizeMade = $none,
    Object? fitNotes = $none,
  }) => $apply(
    FieldCopyWithData({
      if (projectType != $none) #projectType: projectType,
      if (projectSubtype != $none) #projectSubtype: projectSubtype,
      if (sizeMade != $none) #sizeMade: sizeMade,
      if (fitNotes != $none) #fitNotes: fitNotes,
    }),
  );
  @override
  SewingProjectDetails $make(CopyWithData data) => SewingProjectDetails(
    projectType: data.get(#projectType, or: $value.projectType),
    projectSubtype: data.get(#projectSubtype, or: $value.projectSubtype),
    sizeMade: data.get(#sizeMade, or: $value.sizeMade),
    fitNotes: data.get(#fitNotes, or: $value.fitNotes),
  );

  @override
  SewingProjectDetailsCopyWith<$R2, SewingProjectDetails, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SewingProjectDetailsCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class QuiltingProjectDetailsMapper
    extends ClassMapperBase<QuiltingProjectDetails> {
  QuiltingProjectDetailsMapper._();

  static QuiltingProjectDetailsMapper? _instance;
  static QuiltingProjectDetailsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = QuiltingProjectDetailsMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'QuiltingProjectDetails';

  static String? _$projectType(QuiltingProjectDetails v) => v.projectType;
  static const Field<QuiltingProjectDetails, String> _f$projectType = Field(
    'projectType',
    _$projectType,
    opt: true,
  );
  static String? _$projectSubtype(QuiltingProjectDetails v) => v.projectSubtype;
  static const Field<QuiltingProjectDetails, String> _f$projectSubtype = Field(
    'projectSubtype',
    _$projectSubtype,
    opt: true,
  );
  static String? _$size(QuiltingProjectDetails v) => v.size;
  static const Field<QuiltingProjectDetails, String> _f$size = Field(
    'size',
    _$size,
    opt: true,
  );
  static String? _$piecingTechnique(QuiltingProjectDetails v) =>
      v.piecingTechnique;
  static const Field<QuiltingProjectDetails, String> _f$piecingTechnique =
      Field('piecingTechnique', _$piecingTechnique, opt: true);
  static String? _$quiltingMethod(QuiltingProjectDetails v) => v.quiltingMethod;
  static const Field<QuiltingProjectDetails, String> _f$quiltingMethod = Field(
    'quiltingMethod',
    _$quiltingMethod,
    opt: true,
  );

  @override
  final MappableFields<QuiltingProjectDetails> fields = const {
    #projectType: _f$projectType,
    #projectSubtype: _f$projectSubtype,
    #size: _f$size,
    #piecingTechnique: _f$piecingTechnique,
    #quiltingMethod: _f$quiltingMethod,
  };
  @override
  final bool ignoreNull = true;

  static QuiltingProjectDetails _instantiate(DecodingData data) {
    return QuiltingProjectDetails(
      projectType: data.dec(_f$projectType),
      projectSubtype: data.dec(_f$projectSubtype),
      size: data.dec(_f$size),
      piecingTechnique: data.dec(_f$piecingTechnique),
      quiltingMethod: data.dec(_f$quiltingMethod),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static QuiltingProjectDetails fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<QuiltingProjectDetails>(map);
  }

  static QuiltingProjectDetails fromJson(String json) {
    return ensureInitialized().decodeJson<QuiltingProjectDetails>(json);
  }
}

mixin QuiltingProjectDetailsMappable {
  String toJson() {
    return QuiltingProjectDetailsMapper.ensureInitialized()
        .encodeJson<QuiltingProjectDetails>(this as QuiltingProjectDetails);
  }

  Map<String, dynamic> toMap() {
    return QuiltingProjectDetailsMapper.ensureInitialized()
        .encodeMap<QuiltingProjectDetails>(this as QuiltingProjectDetails);
  }

  QuiltingProjectDetailsCopyWith<
    QuiltingProjectDetails,
    QuiltingProjectDetails,
    QuiltingProjectDetails
  >
  get copyWith =>
      _QuiltingProjectDetailsCopyWithImpl<
        QuiltingProjectDetails,
        QuiltingProjectDetails
      >(this as QuiltingProjectDetails, $identity, $identity);
  @override
  String toString() {
    return QuiltingProjectDetailsMapper.ensureInitialized().stringifyValue(
      this as QuiltingProjectDetails,
    );
  }

  @override
  bool operator ==(Object other) {
    return QuiltingProjectDetailsMapper.ensureInitialized().equalsValue(
      this as QuiltingProjectDetails,
      other,
    );
  }

  @override
  int get hashCode {
    return QuiltingProjectDetailsMapper.ensureInitialized().hashValue(
      this as QuiltingProjectDetails,
    );
  }
}

extension QuiltingProjectDetailsValueCopy<$R, $Out>
    on ObjectCopyWith<$R, QuiltingProjectDetails, $Out> {
  QuiltingProjectDetailsCopyWith<$R, QuiltingProjectDetails, $Out>
  get $asQuiltingProjectDetails => $base.as(
    (v, t, t2) => _QuiltingProjectDetailsCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class QuiltingProjectDetailsCopyWith<
  $R,
  $In extends QuiltingProjectDetails,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? projectType,
    String? projectSubtype,
    String? size,
    String? piecingTechnique,
    String? quiltingMethod,
  });
  QuiltingProjectDetailsCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _QuiltingProjectDetailsCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, QuiltingProjectDetails, $Out>
    implements
        QuiltingProjectDetailsCopyWith<$R, QuiltingProjectDetails, $Out> {
  _QuiltingProjectDetailsCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<QuiltingProjectDetails> $mapper =
      QuiltingProjectDetailsMapper.ensureInitialized();
  @override
  $R call({
    Object? projectType = $none,
    Object? projectSubtype = $none,
    Object? size = $none,
    Object? piecingTechnique = $none,
    Object? quiltingMethod = $none,
  }) => $apply(
    FieldCopyWithData({
      if (projectType != $none) #projectType: projectType,
      if (projectSubtype != $none) #projectSubtype: projectSubtype,
      if (size != $none) #size: size,
      if (piecingTechnique != $none) #piecingTechnique: piecingTechnique,
      if (quiltingMethod != $none) #quiltingMethod: quiltingMethod,
    }),
  );
  @override
  QuiltingProjectDetails $make(CopyWithData data) => QuiltingProjectDetails(
    projectType: data.get(#projectType, or: $value.projectType),
    projectSubtype: data.get(#projectSubtype, or: $value.projectSubtype),
    size: data.get(#size, or: $value.size),
    piecingTechnique: data.get(#piecingTechnique, or: $value.piecingTechnique),
    quiltingMethod: data.get(#quiltingMethod, or: $value.quiltingMethod),
  );

  @override
  QuiltingProjectDetailsCopyWith<$R2, QuiltingProjectDetails, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _QuiltingProjectDetailsCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class UnknownProjectDetailsMapper
    extends ClassMapperBase<UnknownProjectDetails> {
  UnknownProjectDetailsMapper._();

  static UnknownProjectDetailsMapper? _instance;
  static UnknownProjectDetailsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UnknownProjectDetailsMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'UnknownProjectDetails';

  static Map<String, dynamic> _$raw(UnknownProjectDetails v) => v.raw;
  static const Field<UnknownProjectDetails, Map<String, dynamic>> _f$raw =
      Field('raw', _$raw);
  static String? _$type(UnknownProjectDetails v) => v.type;
  static const Field<UnknownProjectDetails, String> _f$type = Field(
    'type',
    _$type,
    opt: true,
  );

  @override
  final MappableFields<UnknownProjectDetails> fields = const {
    #raw: _f$raw,
    #type: _f$type,
  };
  @override
  final bool ignoreNull = true;

  static UnknownProjectDetails _instantiate(DecodingData data) {
    return UnknownProjectDetails(
      raw: data.dec(_f$raw),
      type: data.dec(_f$type),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UnknownProjectDetails fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UnknownProjectDetails>(map);
  }

  static UnknownProjectDetails fromJson(String json) {
    return ensureInitialized().decodeJson<UnknownProjectDetails>(json);
  }
}

mixin UnknownProjectDetailsMappable {
  String toJson() {
    return UnknownProjectDetailsMapper.ensureInitialized()
        .encodeJson<UnknownProjectDetails>(this as UnknownProjectDetails);
  }

  Map<String, dynamic> toMap() {
    return UnknownProjectDetailsMapper.ensureInitialized()
        .encodeMap<UnknownProjectDetails>(this as UnknownProjectDetails);
  }

  UnknownProjectDetailsCopyWith<
    UnknownProjectDetails,
    UnknownProjectDetails,
    UnknownProjectDetails
  >
  get copyWith =>
      _UnknownProjectDetailsCopyWithImpl<
        UnknownProjectDetails,
        UnknownProjectDetails
      >(this as UnknownProjectDetails, $identity, $identity);
  @override
  String toString() {
    return UnknownProjectDetailsMapper.ensureInitialized().stringifyValue(
      this as UnknownProjectDetails,
    );
  }

  @override
  bool operator ==(Object other) {
    return UnknownProjectDetailsMapper.ensureInitialized().equalsValue(
      this as UnknownProjectDetails,
      other,
    );
  }

  @override
  int get hashCode {
    return UnknownProjectDetailsMapper.ensureInitialized().hashValue(
      this as UnknownProjectDetails,
    );
  }
}

extension UnknownProjectDetailsValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UnknownProjectDetails, $Out> {
  UnknownProjectDetailsCopyWith<$R, UnknownProjectDetails, $Out>
  get $asUnknownProjectDetails => $base.as(
    (v, t, t2) => _UnknownProjectDetailsCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class UnknownProjectDetailsCopyWith<
  $R,
  $In extends UnknownProjectDetails,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  MapCopyWith<$R, String, dynamic, ObjectCopyWith<$R, dynamic, dynamic>?>
  get raw;
  $R call({Map<String, dynamic>? raw, String? type});
  UnknownProjectDetailsCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UnknownProjectDetailsCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UnknownProjectDetails, $Out>
    implements UnknownProjectDetailsCopyWith<$R, UnknownProjectDetails, $Out> {
  _UnknownProjectDetailsCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UnknownProjectDetails> $mapper =
      UnknownProjectDetailsMapper.ensureInitialized();
  @override
  MapCopyWith<$R, String, dynamic, ObjectCopyWith<$R, dynamic, dynamic>?>
  get raw => MapCopyWith(
    $value.raw,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(raw: v),
  );
  @override
  $R call({Map<String, dynamic>? raw, Object? type = $none}) => $apply(
    FieldCopyWithData({
      if (raw != null) #raw: raw,
      if (type != $none) #type: type,
    }),
  );
  @override
  UnknownProjectDetails $make(CopyWithData data) => UnknownProjectDetails(
    raw: data.get(#raw, or: $value.raw),
    type: data.get(#type, or: $value.type),
  );

  @override
  UnknownProjectDetailsCopyWith<$R2, UnknownProjectDetails, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _UnknownProjectDetailsCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

