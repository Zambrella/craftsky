import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';

enum ProjectComposerValidationCode {
  required,
  invalidGauge,
  tooManyValues,
  tooLong,
}

class ProjectComposerValidationError {
  const new({
    required this.fieldName,
    required this.code,
  });

  final String fieldName;
  final ProjectComposerValidationCode code;
}

class ProjectComposerPayloadResult {
  const new project(this.project) : errors = const [];
  const new errors(this.errors) : project = null;

  final Project? project;
  final List<ProjectComposerValidationError> errors;
}

ProjectComposerPayloadResult buildProjectComposerPayload({
  required Map<String, dynamic> formValues,
}) {
  final craftType = _stringValue(formValues[ProjectComposerFields.craftType]);
  if (craftType == null) {
    return const ProjectComposerPayloadResult.errors([
      ProjectComposerValidationError(
        fieldName: ProjectComposerFields.craftType,
        code: ProjectComposerValidationCode.required,
      ),
    ]);
  }

  final textLimits = <String, RecordTextLimit>{
    ProjectComposerFields.title: RecordTextLimits.projectTitle,
    ProjectComposerFields.duration: RecordTextLimits.projectDuration,
    ProjectComposerFields.patternName: RecordTextLimits.patternText,
    ProjectComposerFields.patternDesigner: RecordTextLimits.patternText,
    ProjectComposerFields.patternPublisher: RecordTextLimits.patternText,
    if (craftType == ProjectOptionCatalogs.sewingCraftToken) ...{
      ProjectComposerFields.sewingSizeMade: RecordTextLimits.projectSize,
      ProjectComposerFields.sewingFitNotes: RecordTextLimits.fitNotes,
    },
    if (craftType == ProjectOptionCatalogs.knittingCraftToken)
      ProjectComposerFields.knittingFinishedSize: RecordTextLimits.projectSize,
    if (craftType == ProjectOptionCatalogs.crochetCraftToken)
      ProjectComposerFields.crochetFinishedSize: RecordTextLimits.projectSize,
    if (craftType == ProjectOptionCatalogs.quiltingCraftToken)
      ProjectComposerFields.quiltingSize: RecordTextLimits.projectSize,
  };
  final textErrors = [
    for (final entry in textLimits.entries)
      if (!entry.value.accepts(_stringValue(formValues[entry.key]) ?? ''))
        ProjectComposerValidationError(
          fieldName: entry.key,
          code: ProjectComposerValidationCode.tooLong,
        ),
    if (_materialsList(formValues[ProjectComposerFields.materials])?.any(
          (material) => !RecordTextLimits.material.accepts(material.text),
        ) ??
        false)
      const ProjectComposerValidationError(
        fieldName: ProjectComposerFields.materials,
        code: ProjectComposerValidationCode.tooLong,
      ),
  ];
  if (textErrors.isNotEmpty) {
    return ProjectComposerPayloadResult.errors(textErrors);
  }

  final detailErrors = _detailErrorsFrom(craftType, formValues);
  if (detailErrors.isNotEmpty) {
    return ProjectComposerPayloadResult.errors(detailErrors);
  }

  final common = ProjectCommon(
    craftType: craftType,
    additionalCrafts: [
      for (final craft
          in (_stringList(formValues[ProjectComposerFields.additionalCrafts]) ??
                  <String>[])
              .toSet())
        if (craft != craftType) ProjectCraft(craftType: craft),
    ],
    status: formValues.containsKey(ProjectComposerFields.status)
        ? _stringValue(formValues[ProjectComposerFields.status])
        : ProjectOptionCatalogs.finishedStatusToken,
    title: _stringValue(formValues[ProjectComposerFields.title]),
    duration: _stringValue(formValues[ProjectComposerFields.duration]),
    pattern: _patternFrom(formValues),
    materials: _materialsList(formValues[ProjectComposerFields.materials]),
    colors: _stringList(formValues[ProjectComposerFields.colours]),
    designTags: _stringList(formValues[ProjectComposerFields.designTags]),
  );

  return ProjectComposerPayloadResult.project(
    Project(common: common, details: _detailsFrom(craftType, formValues)),
  );
}

ProjectDetails? _detailsFrom(String craftType, Map<String, dynamic> values) {
  return switch (craftType) {
    ProjectOptionCatalogs.sewingCraftToken => _sewingDetailsFrom(values),
    ProjectOptionCatalogs.knittingCraftToken => _knittingDetailsFrom(values),
    ProjectOptionCatalogs.crochetCraftToken => _crochetDetailsFrom(values),
    ProjectOptionCatalogs.quiltingCraftToken => _quiltingDetailsFrom(values),
    _ => null,
  };
}

List<ProjectComposerValidationError> _detailErrorsFrom(
  String craftType,
  Map<String, dynamic> values,
) {
  return switch (craftType) {
    ProjectOptionCatalogs.knittingCraftToken => _gaugeErrors(
      stitchesField: ProjectComposerFields.knittingGaugeStitches,
      rowsField: ProjectComposerFields.knittingGaugeRows,
      measurementField: ProjectComposerFields.knittingGaugeMeasurement,
      unitField: ProjectComposerFields.knittingGaugeUnit,
      values: values,
    ),
    ProjectOptionCatalogs.crochetCraftToken => _gaugeErrors(
      stitchesField: ProjectComposerFields.crochetGaugeStitches,
      rowsField: ProjectComposerFields.crochetGaugeRows,
      measurementField: ProjectComposerFields.crochetGaugeMeasurement,
      unitField: ProjectComposerFields.crochetGaugeUnit,
      values: values,
    ),
    _ => const <ProjectComposerValidationError>[],
  };
}

SewingProjectDetails? _sewingDetailsFrom(Map<String, dynamic> values) {
  final details = SewingProjectDetails(
    projectType: _stringValue(values[ProjectComposerFields.sewingProjectType]),
    projectSubtype: _stringValue(
      values[ProjectComposerFields.sewingProjectSubtype],
    ),
    sizeMade: _stringValue(values[ProjectComposerFields.sewingSizeMade]),
    fitNotes: _stringValue(values[ProjectComposerFields.sewingFitNotes]),
  );

  if (details.projectType == null &&
      details.projectSubtype == null &&
      details.sizeMade == null &&
      details.fitNotes == null) {
    return null;
  }
  return details;
}

KnittingProjectDetails? _knittingDetailsFrom(Map<String, dynamic> values) {
  final gauge = _gaugeFrom(
    stitchesField: ProjectComposerFields.knittingGaugeStitches,
    rowsField: ProjectComposerFields.knittingGaugeRows,
    measurementField: ProjectComposerFields.knittingGaugeMeasurement,
    unitField: ProjectComposerFields.knittingGaugeUnit,
    values: values,
  );
  final details = KnittingProjectDetails(
    projectType: _stringValue(
      values[ProjectComposerFields.knittingProjectType],
    ),
    projectSubtype: _stringValue(
      values[ProjectComposerFields.knittingProjectSubtype],
    ),
    yarnWeight: _stringValue(values[ProjectComposerFields.knittingYarnWeight]),
    needleSizeMm: _stringValue(
      values[ProjectComposerFields.knittingNeedleSize],
    ),
    gauge: gauge,
    finishedSize: _stringValue(
      values[ProjectComposerFields.knittingFinishedSize],
    ),
  );

  if (details.projectType == null &&
      details.projectSubtype == null &&
      details.yarnWeight == null &&
      details.needleSizeMm == null &&
      details.gauge == null &&
      details.finishedSize == null) {
    return null;
  }
  return details;
}

CrochetProjectDetails? _crochetDetailsFrom(Map<String, dynamic> values) {
  final gauge = _gaugeFrom(
    stitchesField: ProjectComposerFields.crochetGaugeStitches,
    rowsField: ProjectComposerFields.crochetGaugeRows,
    measurementField: ProjectComposerFields.crochetGaugeMeasurement,
    unitField: ProjectComposerFields.crochetGaugeUnit,
    values: values,
  );
  final details = CrochetProjectDetails(
    projectType: _stringValue(values[ProjectComposerFields.crochetProjectType]),
    projectSubtype: _stringValue(
      values[ProjectComposerFields.crochetProjectSubtype],
    ),
    yarnWeight: _stringValue(values[ProjectComposerFields.crochetYarnWeight]),
    hookSizeMm: _stringValue(values[ProjectComposerFields.crochetHookSize]),
    gauge: gauge,
    finishedSize: _stringValue(
      values[ProjectComposerFields.crochetFinishedSize],
    ),
  );

  if (details.projectType == null &&
      details.projectSubtype == null &&
      details.yarnWeight == null &&
      details.hookSizeMm == null &&
      details.gauge == null &&
      details.finishedSize == null) {
    return null;
  }
  return details;
}

QuiltingProjectDetails? _quiltingDetailsFrom(Map<String, dynamic> values) {
  final details = QuiltingProjectDetails(
    projectType: _stringValue(
      values[ProjectComposerFields.quiltingProjectType],
    ),
    projectSubtype: _stringValue(
      values[ProjectComposerFields.quiltingProjectSubtype],
    ),
    size: _stringValue(values[ProjectComposerFields.quiltingSize]),
    piecingTechnique: _stringValue(
      values[ProjectComposerFields.quiltingPiecingTechnique],
    ),
    quiltingMethod: _stringValue(
      values[ProjectComposerFields.quiltingMethod],
    ),
  );

  if (details.projectType == null &&
      details.projectSubtype == null &&
      details.size == null &&
      details.piecingTechnique == null &&
      details.quiltingMethod == null) {
    return null;
  }
  return details;
}

List<ProjectComposerValidationError> _gaugeErrors({
  required String stitchesField,
  required String rowsField,
  required String measurementField,
  required String unitField,
  required Map<String, dynamic> values,
}) {
  final stitches = values[stitchesField];
  final rows = values[rowsField];
  final measurement = values[measurementField];
  final unit = _stringValue(values[unitField]);
  final hasAny = [
    stitches,
    rows,
    measurement,
    unit,
  ].any((value) => value != null);
  if (!hasAny) return const <ProjectComposerValidationError>[];

  final valid =
      _positiveInt(stitches) != null &&
      (rows == null || _positiveInt(rows) != null) &&
      _positiveInt(measurement) != null &&
      unit != null;
  if (valid) return const <ProjectComposerValidationError>[];
  return [
    ProjectComposerValidationError(
      fieldName: stitchesField,
      code: ProjectComposerValidationCode.invalidGauge,
    ),
  ];
}

ProjectGauge? _gaugeFrom({
  required String stitchesField,
  required String rowsField,
  required String measurementField,
  required String unitField,
  required Map<String, dynamic> values,
}) {
  final stitches = _positiveInt(values[stitchesField]);
  final rows = _positiveInt(values[rowsField]);
  final measurement = _positiveInt(values[measurementField]);
  final unit = _stringValue(values[unitField]);
  if (stitches == null && rows == null && measurement == null && unit == null) {
    return null;
  }
  return ProjectGauge(
    stitches: stitches!,
    rows: rows,
    measurement: measurement!,
    unit: unit!,
  );
}

int? _positiveInt(Object? value) {
  if (value is int) return value > 0 ? value : null;
  if (value is num && value % 1 == 0) {
    final parsed = value.toInt();
    return parsed > 0 ? parsed : null;
  }
  final text = _stringValue(value);
  if (text == null || !RegExp(r'^\d+$').hasMatch(text)) return null;
  final parsed = int.parse(text);
  return parsed > 0 ? parsed : null;
}

ProjectPattern? _patternFrom(Map<String, dynamic> values) {
  final pattern = ProjectPattern(
    url: _stringValue(values[ProjectComposerFields.patternUrl]),
    name: _patternNameValue(values[ProjectComposerFields.patternName]),
    difficulty: _stringValue(values[ProjectComposerFields.patternDifficulty]),
    selfDrafted: values[ProjectComposerFields.patternSelfDrafted] == true
        ? true
        : null,
    designer: _stringValue(values[ProjectComposerFields.patternDesigner]),
    publisher: _stringValue(values[ProjectComposerFields.patternPublisher]),
  );

  if (pattern.url == null &&
      pattern.name == null &&
      pattern.difficulty == null &&
      pattern.selfDrafted == null &&
      pattern.designer == null &&
      pattern.publisher == null) {
    return null;
  }
  return pattern;
}

String? _patternNameValue(Object? value) {
  final text = _stringValue(value);
  return text == '#' ? null : text;
}

String? _stringValue(Object? value) {
  if (value is! String) return null;
  final trimmed = value.trim();
  return trimmed.isEmpty ? null : trimmed;
}

List<String>? _stringList(Object? value) {
  if (value is! Iterable) return null;
  final values = value
      .whereType<String>()
      .map((item) => item.trim())
      .where((item) => item.isNotEmpty)
      .toList(growable: false);
  return values.isEmpty ? null : values;
}

List<ProjectMaterial>? _materialsList(Object? value) {
  if (value is! Iterable) return null;
  final values = value
      .whereType<ProjectMaterial>()
      .map((material) {
        final text = material.text.trim();
        if (text.isEmpty) return null;
        return ProjectMaterial(text: text, facets: material.facets);
      })
      .nonNulls
      .toList(growable: false);
  return values.isEmpty ? null : values;
}
