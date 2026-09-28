import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'project_browse_filters.mapper.dart';

final class CraftTypeFilterToken {
  const CraftTypeFilterToken(this.value);

  final String value;
}

final class ProjectTypeFilterToken {
  const ProjectTypeFilterToken(this.value);

  final String value;
}

final class PatternDifficultyFilterToken {
  const PatternDifficultyFilterToken(this.value);

  final String value;
}

final class DesignTagFilterToken {
  const DesignTagFilterToken(this.value);

  final String value;
}

enum ProjectBrowseFilterFamily {
  status,
  projectType,
  projectSubtype,
  patternDifficulty,
  color,
  designTag,
  yarnWeight,
  piecingTechnique,
  quiltingMethod,
  selfDrafted,
}

@MappableClass()
class ProjectBrowseQuery with ProjectBrowseQueryMappable {
  const ProjectBrowseQuery({
    this.craftTypes = const [],
    this.filters = const ProjectBrowseFilters(),
    this.sort = SearchSort.chronological,
  });

  factory ProjectBrowseQuery.tokens({
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
  const ProjectBrowseFilters({
    this.status = const [],
    this.projectType = const [],
    this.projectSubtype = const [],
    this.patternDifficulty = const [],
    this.color = const [],
    this.designTag = const [],
    this.yarnWeight = const [],
    this.piecingTechnique = const [],
    this.quiltingMethod = const [],
    this.selfDrafted = false,
  });

  factory ProjectBrowseFilters.tokens({
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
  final List<String> patternDifficulty;
  final List<String> color;
  final List<String> designTag;
  final List<String> yarnWeight;
  final List<String> piecingTechnique;
  final List<String> quiltingMethod;
  final bool selfDrafted;

  Map<String, Object> toQueryParameters() => {
    if (status.isNotEmpty) 'status': status,
    if (projectType.isNotEmpty) 'projectType': projectType,
    if (projectSubtype.isNotEmpty) 'projectSubtype': projectSubtype,
    if (patternDifficulty.isNotEmpty) 'patternDifficulty': patternDifficulty,
    if (color.isNotEmpty) 'color': color,
    if (designTag.isNotEmpty) 'designTag': designTag,
    if (yarnWeight.isNotEmpty) 'yarnWeight': yarnWeight,
    if (piecingTechnique.isNotEmpty) 'piecingTechnique': piecingTechnique,
    if (quiltingMethod.isNotEmpty) 'quiltingMethod': quiltingMethod,
    if (selfDrafted) 'selfDrafted': true,
  };

  List<String> valuesFor(ProjectBrowseFilterFamily family) => switch (family) {
    ProjectBrowseFilterFamily.status => status,
    ProjectBrowseFilterFamily.projectType => projectType,
    ProjectBrowseFilterFamily.projectSubtype => projectSubtype,
    ProjectBrowseFilterFamily.patternDifficulty => patternDifficulty,
    ProjectBrowseFilterFamily.color => color,
    ProjectBrowseFilterFamily.designTag => designTag,
    ProjectBrowseFilterFamily.yarnWeight => yarnWeight,
    ProjectBrowseFilterFamily.piecingTechnique => piecingTechnique,
    ProjectBrowseFilterFamily.quiltingMethod => quiltingMethod,
    ProjectBrowseFilterFamily.selfDrafted =>
      selfDrafted ? const ['true'] : const [],
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
      ProjectBrowseFilterFamily.selfDrafted => copyWith(
        selfDrafted: values.contains('true'),
      ),
    };
  }
}
