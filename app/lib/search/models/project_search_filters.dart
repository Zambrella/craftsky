import 'package:flutter/foundation.dart';

/// Supported repeated project search filter families.
@immutable
class ProjectSearchFilters {
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

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ProjectSearchFilters &&
          _listEquals(craftType, other.craftType) &&
          _listEquals(projectType, other.projectType) &&
          _listEquals(patternDifficulty, other.patternDifficulty) &&
          _listEquals(color, other.color) &&
          _listEquals(material, other.material) &&
          _listEquals(designTag, other.designTag) &&
          _listEquals(projectTag, other.projectTag);

  @override
  int get hashCode => Object.hash(
    Object.hashAll(craftType),
    Object.hashAll(projectType),
    Object.hashAll(patternDifficulty),
    Object.hashAll(color),
    Object.hashAll(material),
    Object.hashAll(designTag),
    Object.hashAll(projectTag),
  );
}

bool _listEquals(List<String> a, List<String> b) {
  if (identical(a, b)) return true;
  if (a.length != b.length) return false;
  for (var i = 0; i < a.length; i++) {
    if (a[i] != b[i]) return false;
  }
  return true;
}

List<String> _strings(Object? value) =>
    value is List ? [for (final item in value) item as String] : const [];
