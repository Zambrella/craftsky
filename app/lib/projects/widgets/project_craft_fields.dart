import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_item_select_options.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:material_ui/material_ui.dart';

typedef _DetailValues = Map<String, Object?>;

class ProjectCraftFields extends StatelessWidget {
  const new({required this.craft, super.key});

  final String craft;

  @override
  Widget build(BuildContext context) {
    final types = ProjectOptionCatalogs.projectTypesForCraft(craft);
    final methods = ProjectDetailFields.authoringForCraft(craft);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: 12,
      children: [
        if (types.isNotEmpty) _ProjectItemFields(craft: craft),
        if (methods.isNotEmpty)
          FormBuilderField<_DetailValues>(
            name: ProjectDetailFields.formName(craft),
            builder: (field) {
              final values = field.value ?? const <String, Object?>{};
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: 12,
                children: [
                  for (final detail in methods)
                    _MethodControl(
                      key: ValueKey(detail.key),
                      detail: detail,
                      craft: craft,
                      value: ProjectDetailFields.authoringValue(
                        craft,
                        detail.key,
                        values,
                      ),
                      onChanged: (value) {
                        final updated = _DetailValues.from(values);
                        if (value == null) {
                          updated.remove(detail.key);
                        } else {
                          updated[detail.key] = value;
                        }
                        field.didChange(updated);
                      },
                    ),
                ],
              );
            },
          ),
      ],
    );
  }
}

class _ProjectItemFields extends StatelessWidget {
  const new({required this.craft});

  final String craft;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final typeFieldName = ProjectComposerFields.projectTypeFor(craft);
    final subtypeFieldName = ProjectComposerFields.projectSubtypeFor(craft);
    return FormBuilderField<String>(
      name: typeFieldName,
      builder: (typeField) => FormBuilderField<String>(
        name: subtypeFieldName,
        builder: (subtypeField) {
          final type = typeField.value;
          final subtype = subtypeField.value;
          final options = projectItemSelectOptions(
            craft,
            retainedType: type,
            retainedSubtype: subtype,
          );
          final selected = selectedProjectItem(options, type, subtype);
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 12,
            children: [
              CraftskySingleSelectInput<String>(
                keyPrefix: typeFieldName,
                label: craft == ProjectOptionCatalogs.dyeingCraftToken
                    ? l10n.projectComposerDyedItemLabel
                    : l10n.projectsFilterProjectType,
                value: selected,
                options: options,
                searchThreshold: 0,
                onChanged: (next) {
                  if (next == selected) return;
                  final option = options
                      .where((option) => option.value == next)
                      .firstOrNull;
                  final parent = option?.parentValue;
                  subtypeField.didChange(parent == null ? null : next);
                  typeField.didChange(parent ?? next);
                },
              ),
              if (type == ProjectOptionCatalogs.otherProjectTypeToken)
                _OtherProjectName(
                  name: subtype,
                  onChanged: subtypeField.didChange,
                ),
            ],
          );
        },
      ),
    );
  }
}

class _OtherProjectName extends StatefulWidget {
  const new({required this.name, required this.onChanged});

  final String? name;
  final ValueChanged<String?> onChanged;

  @override
  State<_OtherProjectName> createState() => _OtherProjectNameState();
}

class _OtherProjectNameState extends State<_OtherProjectName> {
  late final _controller = TextEditingController(text: widget.name);

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return CraftskyTextInput(
      label: AppLocalizations.of(context).projectComposerCustomSubtypeLabel,
      textFieldKey: const Key('project-custom-subtype'),
      controller: _controller,
      maxBytes: RecordTextLimits.projectSubtype.bytes,
      onChanged: (value) {
        final trimmed = value.trim();
        widget.onChanged(trimmed.isEmpty ? null : trimmed);
      },
    );
  }
}

class _MethodControl extends StatelessWidget {
  const new({
    required this.detail,
    required this.craft,
    required this.value,
    required this.onChanged,
    super.key,
  });

  final ProjectDetailField detail;
  final String craft;
  final Object? value;
  final ValueChanged<Object?> onChanged;

  @override
  Widget build(BuildContext context) {
    final options = detail.choicesFor(craft);
    if (detail.input == ProjectDetailInput.choices) {
      final selected =
          (value as List?)?.whereType<String>().toList() ?? const [];
      return CraftskySearchableMultiSelectInput<String>(
        label: detail.label,
        values: selected,
        options: alphabetizedSelectOptions([
          ..._selectOptions(options),
          for (final value in selected)
            if (!options.any((option) => option.value == value))
              CraftskySelectOption(
                value: value,
                label: _retainedLabel(value),
              ),
        ]),
        maxSelected: detail.maximum,
        maxSelectedErrorText: AppLocalizations.of(context)
            .projectComposerMultiSelectMaxSelectedError(detail.maximum),
        onChanged: onChanged,
      );
    }
    return CraftskySingleSelectInput<String>(
      label: detail.label,
      value: value as String?,
      options: _selectOptions(options, retained: value as String?),
      onChanged: onChanged,
    );
  }
}

List<CraftskySelectOption<String>> _selectOptions(
  List<ProjectOption> options, {
  String? retained,
}) => alphabetizedSelectOptions([
  for (final option in options)
    CraftskySelectOption(
      value: option.value,
      label: option.label,
      group: option.group,
      aliases: option.aliases,
    ),
  if (retained != null && !options.any((option) => option.value == retained))
    CraftskySelectOption(
      value: retained,
      label: _retainedLabel(retained),
    ),
], lastValue: ProjectOptionCatalogs.otherProjectTypeToken);

String _retainedLabel(String value) {
  final label = ProjectOptionCatalogs.projectValueLabel(value);
  return label == value ? value.split('#').last : label;
}
