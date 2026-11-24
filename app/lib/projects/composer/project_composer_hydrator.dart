import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
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
  final colours = common['colors'] as List<Object?>?;
  final designTags = common['designTags'] as List<Object?>?;
  values
    ..[ProjectComposerFields.craftType] = common['craftType']
    ..[ProjectComposerFields.additionalCrafts] = <String>[
      for (final craft in (common['additionalCrafts'] as List? ?? const []))
        if (craft case {'craftType': final String craftType}) craftType,
    ]
    ..[ProjectComposerFields.status] = common['status']
    ..[ProjectComposerFields.title] = common['title']
    ..[ProjectComposerFields.duration] = common['duration']
    ..[ProjectComposerFields.colours] = colours == null
        ? null
        : List<String>.from(colours)
    ..[ProjectComposerFields.designTags] = designTags == null
        ? null
        : List<String>.from(designTags);

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
  final patterns = common.containsKey('patterns')
      ? common['patterns'] as List? ?? const []
      : [?common['pattern']];
  if (common.containsKey('patterns') || common['pattern'] is Map) {
    values[ProjectComposerFields.patterns] = [
      for (final pattern in patterns)
        if (pattern is Map)
          ProjectPatternMapper.fromMap(Map<String, dynamic>.from(pattern)),
    ];
  }
  if (patterns.firstOrNull case final Map<Object?, Object?> pattern) {
    values
      ..[ProjectComposerFields.patternUrl] = pattern['url']
      ..[ProjectComposerFields.patternName] = pattern['name']
      ..[ProjectComposerFields.patternDifficulty] = pattern['difficulty']
      ..[ProjectComposerFields.patternSelfDrafted] =
          pattern['selfDrafted'] == true
      ..[ProjectComposerFields.patternDesigner] = pattern['designer']
      ..[ProjectComposerFields.patternPublisher] = pattern['publisher'];
  }

  final details = project['details'];
  if (details is! Map<Object?, Object?>) return values;
  final craftType = common['craftType'];
  if (craftType is String &&
      details[r'$type'] == ProjectDetailFields.typeForCraft(craftType)) {
    if (details.containsKey('projectType')) {
      values[ProjectComposerFields.projectTypeFor(craftType)] =
          details['projectType'];
    }
    if (details.containsKey('projectSubtype')) {
      values[ProjectComposerFields.projectSubtypeFor(craftType)] =
          details['projectSubtype'];
    }
    final fields = ProjectDetailFields.forCraft(craftType);
    if (fields.isNotEmpty) {
      values[ProjectDetailFields.formName(craftType)] = {
        for (final field in fields)
          if (details.containsKey(field.key)) field.key: details[field.key],
      };
    }
  }
  if (craftType == ProjectOptionCatalogs.knittingCraftToken) {
    values[ProjectComposerFields.knittingNeedleSizes] = _toolSizes(
      details,
      'needleSizesMm',
      'needleSizeMm',
    );
  }
  if (craftType == ProjectOptionCatalogs.crochetCraftToken) {
    values[ProjectComposerFields.crochetHookSizes] = _toolSizes(
      details,
      'hookSizesMm',
      'hookSizeMm',
    );
  }
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

List<String>? _toolSizes(
  Map<Object?, Object?> details,
  String listKey,
  String scalarKey,
) {
  if (details.containsKey(listKey)) {
    return [
      for (final item in details[listKey] as List? ?? const [])
        if (item is String) item,
    ];
  }
  if (details[scalarKey] case final String size) return [size];
  return null;
}
