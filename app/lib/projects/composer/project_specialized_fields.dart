import 'package:craftsky_app/projects/options/project_detail_fields.dart';

typedef ProjectDetailValues = Map<String, Object?>;

class ProjectSpecializedFields {
  const new(this.values, {this.valid = true});

  factory fromForm(
    String craft,
    Map<String, Object?> form, {
    Map<String, Object?>? originalForm,
  }) {
    final fieldName = ProjectDetailFields.formName(craft);
    final raw = form[fieldName];
    if (raw is! ProjectDetailValues) return const ProjectSpecializedFields({});
    final original = originalForm?[fieldName];
    return _normalize(
      raw,
      ProjectDetailFields.forCraft(craft),
      original: original is ProjectDetailValues ? original : null,
    );
  }

  final ProjectDetailValues values;
  final bool valid;
}

ProjectSpecializedFields _normalize(
  ProjectDetailValues raw,
  List<ProjectDetailField> fields, {
  ProjectDetailValues? original,
}) {
  final values = ProjectDetailValues.from(raw);
  var valid = true;
  for (final field in fields) {
    final value = raw[field.key];
    var normalized = value;
    switch (field.input) {
      case ProjectDetailInput.text ||
          ProjectDetailInput.decimal ||
          ProjectDetailInput.choice ||
          ProjectDetailInput.count:
        // Previously published strings remain authoritative if untouched.
        if (value != null && original?[field.key] == value) continue;
        final text = value?.toString().trim();
        normalized = text == '' ? null : text;
        if (normalized != null) {
          if (!field.limit.accepts(text!)) valid = false;
          if (field.input == ProjectDetailInput.count) {
            final number = int.tryParse(text);
            if (number == null || number < 1) valid = false;
            normalized = number;
          } else if (field.input == ProjectDetailInput.decimal) {
            final decimal = text.replaceAll(',', '.');
            if (!RegExp(r'^\d+(\.\d+)?$').hasMatch(decimal)) valid = false;
            normalized = decimal;
          }
        }
      case ProjectDetailInput.choices:
        final entries = (value as List?)?.whereType<String>().toList();
        if ((entries?.length ?? 0) > field.maximum) valid = false;
        normalized =
            entries == null || (entries.isEmpty && !field.explicitEmpty)
            ? null
            : entries;
      case ProjectDetailInput.entries || ProjectDetailInput.selection:
        final entries = <ProjectDetailValues>[];
        final originals = {
          for (final entry
              in (original?[field.key] as List? ?? const [])
                  .whereType<ProjectDetailValues>())
            if (entry['existingIndex'] case final int index) index: entry,
        };
        for (final entry
            in (value as List? ?? const []).whereType<ProjectDetailValues>()) {
          final child = _normalize(
            entry,
            field.children,
            original: originals[entry['existingIndex']],
          );
          if (!child.valid) valid = false;
          if (child.values.isNotEmpty) entries.add(child.values);
        }
        if (entries.length > field.maximum) valid = false;
        normalized = entries.isEmpty ? null : entries;
      case ProjectDetailInput.object:
        final child = _normalize(
          value as ProjectDetailValues? ?? const {},
          field.children,
          original: original?[field.key] as ProjectDetailValues?,
        );
        if (!child.valid) valid = false;
        normalized = child.values.isEmpty ? null : child.values;
    }
    if (normalized == null) {
      values.remove(field.key);
    } else {
      values[field.key] = normalized;
    }
  }
  if (values.keys.any((key) => key != 'existingIndex') &&
      fields.any((field) => field.required && !values.containsKey(field.key))) {
    valid = false;
  }
  return ProjectSpecializedFields(values, valid: valid);
}
