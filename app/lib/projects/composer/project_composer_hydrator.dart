import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';

Map<String, dynamic> hydrateScheduledProjectComposer(
  Map<String, dynamic>? project,
) {
  final values = <String, dynamic>{
    ProjectComposerFields.status: ProjectOptionCatalogs.finishedStatusToken,
  };
  if (project == null) return values;

  final common = project['common'];
  if (common is! Map<Object?, Object?>) return values;
  values
    ..[ProjectComposerFields.craftType] = common['craftType']
    ..[ProjectComposerFields.status] =
        common['status'] ?? ProjectOptionCatalogs.finishedStatusToken
    ..[ProjectComposerFields.title] = common['title']
    ..[ProjectComposerFields.colours] = common['colors']
    ..[ProjectComposerFields.designTags] = common['designTags'];

  if (common['materials'] case final List<dynamic> materials) {
    values[ProjectComposerFields.materials] = [
      for (final material in materials)
        if (material is Map<Object?, Object?> && material['text'] is String)
          ProjectMaterial(
            text: material['text']! as String,
            facets: _facetMaps(material['facets']),
          ),
    ];
  }
  if (common['pattern'] case final Map<Object?, Object?> pattern) {
    values
      ..[ProjectComposerFields.patternUrl] = pattern['url']
      ..[ProjectComposerFields.patternName] = pattern['name']
      ..[ProjectComposerFields.patternDifficulty] = pattern['difficulty']
      ..[ProjectComposerFields.patternDesigner] = pattern['designer']
      ..[ProjectComposerFields.patternPublisher] = pattern['publisher'];
  }

  final details = project['details'];
  if (details is! Map<Object?, Object?>) return values;
  final craftType = common['craftType'];
  final prefix = switch (craftType) {
    ProjectOptionCatalogs.sewingCraftToken => 'sewing',
    ProjectOptionCatalogs.knittingCraftToken => 'knitting',
    ProjectOptionCatalogs.crochetCraftToken => 'crochet',
    ProjectOptionCatalogs.quiltingCraftToken => 'quilting',
    _ => null,
  };
  if (prefix == null) return values;

  final detailFields = <String, String>{
    'projectType': '${prefix}ProjectType',
    'projectSubtype': '${prefix}ProjectSubtype',
    'sizeMade': ProjectComposerFields.sewingSizeMade,
    'fitNotes': ProjectComposerFields.sewingFitNotes,
    'yarnWeight': '${prefix}YarnWeight',
    'needleSizeMm': ProjectComposerFields.knittingNeedleSize,
    'hookSizeMm': ProjectComposerFields.crochetHookSize,
    'finishedSize': '${prefix}FinishedSize',
    'size': ProjectComposerFields.quiltingSize,
    'piecingTechnique': ProjectComposerFields.quiltingPiecingTechnique,
    'quiltingMethod': ProjectComposerFields.quiltingMethod,
  };
  for (final entry in detailFields.entries) {
    if (details.containsKey(entry.key)) {
      values[entry.value] = details[entry.key];
    }
  }
  if (details['gauge'] case final Map<Object?, Object?> gauge) {
    values
      ..['${prefix}GaugeStitches'] = gauge['stitches']
      ..['${prefix}GaugeRows'] = gauge['rows']
      ..['${prefix}GaugeMeasurement'] = gauge['measurement']
      ..['${prefix}GaugeUnit'] = gauge['unit'];
  }
  return values;
}

List<Map<String, dynamic>>? _facetMaps(Object? value) {
  if (value is! List) return null;
  final facets = [
    for (final item in value)
      if (item is Map) Map<String, dynamic>.from(item),
  ];
  return facets.isEmpty ? null : facets;
}
