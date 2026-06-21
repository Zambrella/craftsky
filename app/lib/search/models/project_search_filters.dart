import 'package:dart_mappable/dart_mappable.dart';

part 'project_search_filters.mapper.dart';

/// Supported repeated project search filter families.
@MappableClass()
class ProjectSearchFilters with ProjectSearchFiltersMappable {
  const ProjectSearchFilters({
    this.craftType = const [],
    this.projectType = const [],
    this.patternDifficulty = const [],
    this.color = const [],
    this.material = const [],
    this.designTag = const [],
    this.projectTag = const [],
  });

  factory ProjectSearchFilters.fromMap(Map<String, dynamic> map) =>
      ProjectSearchFilters(
        craftType: _strings(map['craftType']),
        projectType: _strings(map['projectType']),
        patternDifficulty: _strings(map['patternDifficulty']),
        color: _strings(map['color']),
        material: _strings(map['material']),
        designTag: _strings(map['designTag']),
        projectTag: _strings(map['projectTag']),
      );

  final List<String> craftType;
  final List<String> projectType;
  final List<String> patternDifficulty;
  final List<String> color;
  final List<String> material;
  final List<String> designTag;
  final List<String> projectTag;

  Map<String, List<String>> toQueryParameters() => _nonEmptyMap();

  Map<String, List<String>> toPayloadMap() => _nonEmptyMap();

  Map<String, List<String>> _nonEmptyMap() => {
    if (craftType.isNotEmpty) 'craftType': craftType,
    if (projectType.isNotEmpty) 'projectType': projectType,
    if (patternDifficulty.isNotEmpty) 'patternDifficulty': patternDifficulty,
    if (color.isNotEmpty) 'color': color,
    if (material.isNotEmpty) 'material': material,
    if (designTag.isNotEmpty) 'designTag': designTag,
    if (projectTag.isNotEmpty) 'projectTag': projectTag,
  };
}

List<String> _strings(Object? value) =>
    value is List ? [for (final item in value) item as String] : const [];
