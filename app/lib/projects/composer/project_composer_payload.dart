import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';

enum ProjectComposerValidationCode { required, invalidGauge, tooManyValues }

class ProjectComposerValidationError {
  const ProjectComposerValidationError({
    required this.fieldName,
    required this.code,
  });

  final String fieldName;
  final ProjectComposerValidationCode code;
}

class ProjectComposerPayloadResult {
  const ProjectComposerPayloadResult.project(this.project) : errors = const [];
  const ProjectComposerPayloadResult.errors(this.errors) : project = null;

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

  final detailErrors = _detailErrorsFrom(craftType, formValues);
  if (detailErrors.isNotEmpty) {
    return ProjectComposerPayloadResult.errors(detailErrors);
  }

  final common = ProjectCommon(
    craftType: craftType,
    status:
        _stringValue(formValues[ProjectComposerFields.status]) ??
        ProjectOptionCatalogs.finishedStatusToken,
    title: _stringValue(formValues[ProjectComposerFields.title]),
    pattern: _patternFrom(formValues),
    materials: _stringList(formValues[ProjectComposerFields.materials]),
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
  final stitches = _stringValue(values[stitchesField]);
  final rows = _stringValue(values[rowsField]);
  final measurement = _stringValue(values[measurementField]);
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
  final stitches = _positiveInt(_stringValue(values[stitchesField]));
  final rows = _positiveInt(_stringValue(values[rowsField]));
  final measurement = _positiveInt(_stringValue(values[measurementField]));
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

int? _positiveInt(String? value) {
  if (value == null || !RegExp(r'^\d+$').hasMatch(value)) return null;
  final parsed = int.parse(value);
  return parsed > 0 ? parsed : null;
}

ProjectPattern? _patternFrom(Map<String, dynamic> values) {
  final pattern = ProjectPattern(
    url: _stringValue(values[ProjectComposerFields.patternUrl]),
    name: _stringValue(values[ProjectComposerFields.patternName]),
    difficulty: _stringValue(values[ProjectComposerFields.patternDifficulty]),
    designer: _stringValue(values[ProjectComposerFields.patternDesigner]),
    publisher: _stringValue(values[ProjectComposerFields.patternPublisher]),
  );

  if (pattern.url == null &&
      pattern.name == null &&
      pattern.difficulty == null &&
      pattern.designer == null &&
      pattern.publisher == null) {
    return null;
  }
  return pattern;
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
