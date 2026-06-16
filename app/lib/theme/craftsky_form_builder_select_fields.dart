import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';

export 'package:craftsky_app/theme/craftsky_select_inputs.dart'
    show
        CraftskyFormSearchableMultiSelectField,
        CraftskyFormSingleSelectField,
        CraftskyFormTokenField,
        CraftskySearchableMultiSelectInput,
        CraftskySelectOption,
        CraftskySingleSelectInput,
        CraftskyTokenInput;

class CraftskyFormBuilderDropdownField<T>
    extends CraftskyFormSingleSelectField<T> {
  const CraftskyFormBuilderDropdownField({
    required super.name,
    required super.label,
    required super.options,
    super.key,
    super.initialValue,
    super.helperText,
    super.enabled,
    super.searchThreshold,
    super.searchHintText,
    super.validator,
    super.onChanged,
    super.required,
    super.requiredLabel,
  });
}

class CraftskyFormBuilderMultiSelectField<T> extends StatelessWidget {
  const CraftskyFormBuilderMultiSelectField({
    required this.name,
    required this.label,
    super.key,
    this.options = const [],
    this.initialValue = const [],
    this.helperText,
    this.enabled = true,
    this.validator,
    this.onChanged,
    this.allowCustomValues = false,
    this.maxSelected,
    this.searchHintText,
    this.customValueHintText,
    this.addCustomValueLabel,
    this.disabledText,
    this.maxSelectedErrorText,
  }) : assert(
         !allowCustomValues || T == String,
         'allowCustomValues only supports String values.',
       );

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final List<T> initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<List<T>>? validator;
  final ValueChanged<List<T>>? onChanged;

  /// Uses a free-text token input instead of known options. Only supported for
  /// `CraftskyFormBuilderMultiSelectField<String>`.
  final bool allowCustomValues;
  final int? maxSelected;
  final String? searchHintText;
  final String? customValueHintText;
  final String? addCustomValueLabel;
  final String? disabledText;
  final String? maxSelectedErrorText;

  @override
  Widget build(BuildContext context) {
    if (allowCustomValues && options.isEmpty) {
      return CraftskyFormTokenField(
        name: name,
        label: label,
        initialValue: initialValue.whereType<String>().toList(growable: false),
        helperText: helperText,
        enabled: enabled,
        validator: (values) => validator?.call(values?.cast<T>()),
        onChanged: (values) => onChanged?.call(values.cast<T>()),
        maxSelected: maxSelected,
        inputHintText: customValueHintText,
        addButtonLabel: addCustomValueLabel,
        disabledText: disabledText,
        maxSelectedErrorText: maxSelectedErrorText,
      );
    }

    return CraftskyFormSearchableMultiSelectField<T>(
      name: name,
      label: label,
      options: options,
      initialValue: initialValue,
      helperText: helperText,
      enabled: enabled,
      validator: validator,
      onChanged: onChanged,
      maxSelected: maxSelected,
      searchHintText: searchHintText,
      disabledText: disabledText,
      maxSelectedErrorText: maxSelectedErrorText,
    );
  }
}

class CraftskyFormBuilderRadioField<T> extends StatelessWidget {
  const CraftskyFormBuilderRadioField({
    required this.name,
    required this.label,
    required this.options,
    super.key,
    this.initialValue,
    this.helperText,
    this.enabled = true,
    this.validator,
    this.onChanged,
  });

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final T? initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<T>? validator;
  final ValueChanged<T?>? onChanged;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<T>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return CraftskyFieldScaffold(
          label: label,
          helperText: field.errorText == null ? helperText : null,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          semanticValue: field.value?.toString(),
          child: InputDecorator(
            decoration: InputDecoration(
              enabled: field.widget.enabled,
              contentPadding: EdgeInsets.zero,
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                for (final option in options)
                  RadioListTile<T>(
                    key: Key('$name-radio-${option.value}'),
                    value: option.value,
                    // Flutter's RadioGroup replacement is still migrating;
                    // keep RadioListTile wired directly for current support.
                    // ignore: deprecated_member_use
                    groupValue: field.value,
                    title: Text(option.label),
                    subtitle: option.description == null
                        ? null
                        : Text(option.description!),
                    // Flutter's RadioGroup replacement is still migrating;
                    // keep RadioListTile wired directly for current support.
                    // ignore: deprecated_member_use
                    onChanged: field.widget.enabled
                        ? (value) {
                            field.didChange(value);
                            onChanged?.call(value);
                          }
                        : null,
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
