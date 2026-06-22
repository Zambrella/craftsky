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
  projectType,
  patternDifficulty,
  color,
  material,
  designTag,
  projectTag,
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
    this.projectType = const [],
    this.patternDifficulty = const [],
    this.color = const [],
    this.material = const [],
    this.designTag = const [],
    this.projectTag = const [],
  });

  factory ProjectBrowseFilters.tokens({
    List<ProjectTypeFilterToken> projectType = const [],
    List<PatternDifficultyFilterToken> patternDifficulty = const [],
    List<String> color = const [],
    List<String> material = const [],
    List<DesignTagFilterToken> designTag = const [],
    List<String> projectTag = const [],
  }) => ProjectBrowseFilters(
    projectType: [for (final token in projectType) token.value],
    patternDifficulty: [for (final token in patternDifficulty) token.value],
    color: color,
    material: material,
    designTag: [for (final token in designTag) token.value],
    projectTag: projectTag,
  );

  final List<String> projectType;
  final List<String> patternDifficulty;
  final List<String> color;
  final List<String> material;
  final List<String> designTag;
  final List<String> projectTag;

  Map<String, List<String>> toQueryParameters() => {
    if (projectType.isNotEmpty) 'projectType': projectType,
    if (patternDifficulty.isNotEmpty) 'patternDifficulty': patternDifficulty,
    if (color.isNotEmpty) 'color': color,
    if (material.isNotEmpty) 'material': material,
    if (designTag.isNotEmpty) 'designTag': designTag,
    if (projectTag.isNotEmpty) 'projectTag': projectTag,
  };

  List<String> valuesFor(ProjectBrowseFilterFamily family) => switch (family) {
    ProjectBrowseFilterFamily.projectType => projectType,
    ProjectBrowseFilterFamily.patternDifficulty => patternDifficulty,
    ProjectBrowseFilterFamily.color => color,
    ProjectBrowseFilterFamily.material => material,
    ProjectBrowseFilterFamily.designTag => designTag,
    ProjectBrowseFilterFamily.projectTag => projectTag,
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
      ProjectBrowseFilterFamily.projectType => copyWith(projectType: values),
      ProjectBrowseFilterFamily.patternDifficulty => copyWith(
        patternDifficulty: values,
      ),
      ProjectBrowseFilterFamily.color => copyWith(color: values),
      ProjectBrowseFilterFamily.material => copyWith(material: values),
      ProjectBrowseFilterFamily.designTag => copyWith(designTag: values),
      ProjectBrowseFilterFamily.projectTag => copyWith(projectTag: values),
    };
  }
}
