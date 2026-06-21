import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'project_browse_filters.mapper.dart';

@MappableClass()
class ProjectBrowseQuery with ProjectBrowseQueryMappable {
  const ProjectBrowseQuery({
    this.craftTypes = const [],
    this.filters = const ProjectBrowseFilters(),
    this.sort = SearchSort.chronological,
  });

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
}
