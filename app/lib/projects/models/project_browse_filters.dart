import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'project_browse_filters.mapper.dart';

final class CraftTypeFilterToken {
  const new(this.value);

  final String value;
}

final class ProjectTypeFilterToken {
  const new(this.value);

  final String value;
}

final class PatternDifficultyFilterToken {
  const new(this.value);

  final String value;
}

final class DesignTagFilterToken {
  const new(this.value);

  final String value;
}

enum ProjectBrowseFilterFamily {
  status,
  projectType,
  projectSubtype,
  projectItem,
  patternDifficulty,
  color,
  designTag,
  yarnWeight,
  piecingTechnique,
  quiltingMethod,
  knittingMethod,
  feltingMethod,
  selfDrafted,
  technique,
  techniqueSubtype,
  embroideryMethod,
  embroideryStyle,
  spinningTool,
  fibrePreparation,
  draftingTechnique,
  plyingTechnique,
  finishedYarnMassUnit,
  finishedYarnLengthUnit,
  materialForm,
  dyeType,
  applicationMethod,
  colorEffect,
  loopHeightUnit,
  cordConstruction,
  knot,
  loomType,
  weaveStructure,
  materialRole,
  settUnit,
  feltingForm,
  needleGauge,
  needleSizeMm,
  hookSizeMm,
  gaugeUnit,
  crochetMethod;

  bool get isDetail => index >= technique.index;
}

@MappableClass()
class ProjectBrowseQuery with ProjectBrowseQueryMappable {
  const new({
    this.craftTypes = const [],
    this.filters = const ProjectBrowseFilters(),
    this.sort = SearchSort.chronological,
  });

  factory tokens({
    List<CraftTypeFilterToken> craftTypes = const [],
    ProjectBrowseFilters filters = const ProjectBrowseFilters(),
    SearchSort sort = SearchSort.chronological,
  }) => ProjectBrowseQuery(
    craftTypes: [for (final token in craftTypes) token.value],
    filters: filters,
    sort: sort,
  );

  final List<String> craftTypes;
  final ProjectBrowseFilters filters;
  final SearchSort sort;
}

@MappableClass()
class ProjectBrowseFilters with ProjectBrowseFiltersMappable {
  const new({
    this.status = const [],
    this.projectType = const [],
    this.projectSubtype = const [],
    this.projectItem = const [],
    this.patternDifficulty = const [],
    this.color = const [],
    this.designTag = const [],
    this.yarnWeight = const [],
    this.piecingTechnique = const [],
    this.quiltingMethod = const [],
    this.knittingMethod = const [],
    this.feltingMethod = const [],
    this.detailFilters = const {},
    this.selfDrafted = false,
  });

  factory tokens({
    List<ProjectTypeFilterToken> projectType = const [],
    List<PatternDifficultyFilterToken> patternDifficulty = const [],
    List<String> color = const [],
    List<DesignTagFilterToken> designTag = const [],
  }) => ProjectBrowseFilters(
    projectType: [for (final token in projectType) token.value],
    patternDifficulty: [for (final token in patternDifficulty) token.value],
    color: color,
    designTag: [for (final token in designTag) token.value],
  );

  final List<String> status;
  final List<String> projectType;
  final List<String> projectSubtype;
  final List<String> projectItem;
  final List<String> patternDifficulty;
  final List<String> color;
  final List<String> designTag;
  final List<String> yarnWeight;
  final List<String> piecingTechnique;
  final List<String> quiltingMethod;
  final List<String> knittingMethod;
  final List<String> feltingMethod;
  final Map<String, List<String>> detailFilters;
  final bool selfDrafted;

  Map<String, Object> toQueryParameters() => {
    if (status.isNotEmpty) 'status': status,
    if (projectType.isNotEmpty) 'projectType': projectType,
    if (projectSubtype.isNotEmpty) 'projectSubtype': projectSubtype,
    if (projectItem.isNotEmpty) 'projectItem': projectItem,
    if (patternDifficulty.isNotEmpty) 'patternDifficulty': patternDifficulty,
    if (color.isNotEmpty) 'color': color,
    if (designTag.isNotEmpty) 'designTag': designTag,
    if (yarnWeight.isNotEmpty) 'yarnWeight': yarnWeight,
    if (piecingTechnique.isNotEmpty) 'piecingTechnique': piecingTechnique,
    if (quiltingMethod.isNotEmpty) 'quiltingMethod': quiltingMethod,
    if (knittingMethod.isNotEmpty) 'knittingMethod': knittingMethod,
    if (feltingMethod.isNotEmpty) 'feltingMethod': feltingMethod,
    for (final family in ProjectBrowseFilterFamily.values)
      if (family.isDetail && valuesFor(family).isNotEmpty)
        family.name: valuesFor(family),
    if (selfDrafted) 'selfDrafted': true,
  };

  int get selectedValueCount => toQueryParameters().values
      .whereType<List<String>>()
      .fold(0, (count, values) => count + values.length);

  // Keep an untouched legacy filter's original type and subtype intersection
  // on the wire. The picker displays the narrower of those selections.
  List<String> get selectedProjectItems {
    if (projectItem.isNotEmpty) return projectItem;
    return projectSubtype.isEmpty ? projectType : projectSubtype;
  }

  List<String> valuesFor(ProjectBrowseFilterFamily family) => switch (family) {
    ProjectBrowseFilterFamily.status => status,
    ProjectBrowseFilterFamily.projectType => projectType,
    ProjectBrowseFilterFamily.projectSubtype => projectSubtype,
    ProjectBrowseFilterFamily.projectItem => selectedProjectItems,
    ProjectBrowseFilterFamily.patternDifficulty => patternDifficulty,
    ProjectBrowseFilterFamily.color => color,
    ProjectBrowseFilterFamily.designTag => designTag,
    ProjectBrowseFilterFamily.yarnWeight => yarnWeight,
    ProjectBrowseFilterFamily.piecingTechnique => piecingTechnique,
    ProjectBrowseFilterFamily.quiltingMethod => quiltingMethod,
    ProjectBrowseFilterFamily.knittingMethod => knittingMethod,
    ProjectBrowseFilterFamily.feltingMethod => feltingMethod,
    ProjectBrowseFilterFamily.selfDrafted =>
      selfDrafted ? const ['true'] : const [],
    _ => detailFilters[family.name] ?? const [],
  };

  ProjectBrowseFilters toggleValue(
    ProjectBrowseFilterFamily family,
    String value,
  ) {
    final values = valuesFor(family);
    return values.contains(value)
        ? withoutValue(family, value)
        : withValue(family, value);
  }

  ProjectBrowseFilters withValue(
    ProjectBrowseFilterFamily family,
    String value,
  ) {
    final values = valuesFor(family);
    if (values.contains(value)) return this;
    return withValues(family, [...values, value]);
  }

  ProjectBrowseFilters withoutValue(
    ProjectBrowseFilterFamily family,
    String value,
  ) {
    return withValues(
      family,
      valuesFor(family).where((item) => item != value).toList(),
    );
  }

  ProjectBrowseFilters withValues(
    ProjectBrowseFilterFamily family,
    List<String> values,
  ) {
    return switch (family) {
      ProjectBrowseFilterFamily.status => copyWith(status: values),
      ProjectBrowseFilterFamily.projectType => copyWith(projectType: values),
      ProjectBrowseFilterFamily.projectSubtype => copyWith(
        projectSubtype: values,
      ),
      ProjectBrowseFilterFamily.projectItem => copyWith(
        projectItem: values,
        projectType: const [],
        projectSubtype: const [],
      ),
      ProjectBrowseFilterFamily.patternDifficulty => copyWith(
        patternDifficulty: values,
      ),
      ProjectBrowseFilterFamily.color => copyWith(color: values),
      ProjectBrowseFilterFamily.designTag => copyWith(designTag: values),
      ProjectBrowseFilterFamily.yarnWeight => copyWith(yarnWeight: values),
      ProjectBrowseFilterFamily.piecingTechnique => copyWith(
        piecingTechnique: values,
      ),
      ProjectBrowseFilterFamily.quiltingMethod => copyWith(
        quiltingMethod: values,
      ),
      ProjectBrowseFilterFamily.knittingMethod => copyWith(
        knittingMethod: values,
      ),
      ProjectBrowseFilterFamily.feltingMethod => copyWith(
        feltingMethod: values,
      ),
      ProjectBrowseFilterFamily.selfDrafted => copyWith(
        selfDrafted: values.contains('true'),
      ),
      _ => copyWith(
        detailFilters: {
          for (final entry in detailFilters.entries)
            if (entry.key != family.name) entry.key: entry.value,
          if (values.isNotEmpty) family.name: values,
        },
      ),
    };
  }
}
