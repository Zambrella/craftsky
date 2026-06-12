part of '../craftsky_select_inputs.dart';

class CraftskyFormSingleSelectField<T> extends StatelessWidget {
  const CraftskyFormSingleSelectField({
    required this.name,
    required this.label,
    required this.options,
    super.key,
    this.initialValue,
    this.helperText,
    this.enabled = true,
    this.searchThreshold = 5,
    this.searchHintText = 'Search',
    this.keyPrefix,
    this.validator,
    this.onChanged,
  });

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final T? initialValue;
  final String? helperText;
  final bool enabled;
  final int searchThreshold;
  final String searchHintText;
  final String? keyPrefix;
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
        return CraftskySingleSelectInput<T>(
          label: label,
          options: options,
          value: field.value,
          helperText: field.errorText == null ? helperText : null,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          searchThreshold: searchThreshold,
          searchHintText: searchHintText,
          keyPrefix: keyPrefix ?? name,
          onChanged: (value) {
            field.didChange(value);
            onChanged?.call(value);
          },
        );
      },
    );
  }
}

class CraftskyFormSearchableMultiSelectField<T> extends StatelessWidget {
  const CraftskyFormSearchableMultiSelectField({
    required this.name,
    required this.label,
    required this.options,
    super.key,
    this.initialValue = const [],
    this.helperText,
    this.enabled = true,
    this.validator,
    this.onChanged,
    this.maxSelected,
    this.searchHintText,
    this.disabledText,
    this.maxSelectedErrorText,
    this.keyPrefix,
  });

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final List<T> initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<List<T>>? validator;
  final ValueChanged<List<T>>? onChanged;
  final int? maxSelected;
  final String? searchHintText;
  final String? disabledText;
  final String? maxSelectedErrorText;
  final String? keyPrefix;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<T>>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return CraftskySearchableMultiSelectInput<T>(
          label: label,
          options: options,
          values: List<T>.from(field.value ?? const []),
          helperText: helperText,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          maxSelected: maxSelected,
          searchHintText: searchHintText,
          disabledText: disabledText,
          maxSelectedErrorText: maxSelectedErrorText,
          keyPrefix: keyPrefix ?? name,
          onChanged: (values) {
            field.didChange(values);
            onChanged?.call(values);
          },
        );
      },
    );
  }
}

class CraftskyFormTokenField extends StatelessWidget {
  const CraftskyFormTokenField({
    required this.name,
    required this.label,
    super.key,
    this.initialValue = const [],
    this.helperText,
    this.enabled = true,
    this.validator,
    this.onChanged,
    this.maxSelected,
    this.inputHintText,
    this.addButtonLabel,
    this.disabledText,
    this.maxSelectedErrorText,
    this.keyPrefix,
  });

  final String name;
  final String label;
  final List<String> initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<List<String>>? validator;
  final ValueChanged<List<String>>? onChanged;
  final int? maxSelected;
  final String? inputHintText;
  final String? addButtonLabel;
  final String? disabledText;
  final String? maxSelectedErrorText;
  final String? keyPrefix;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<String>>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return CraftskyTokenInput(
          label: label,
          values: List<String>.from(field.value ?? const []),
          helperText: helperText,
          errorText: field.errorText,
          enabled: field.widget.enabled,
          maxSelected: maxSelected,
          inputHintText: inputHintText,
          addButtonLabel: addButtonLabel,
          disabledText: disabledText,
          maxSelectedErrorText: maxSelectedErrorText,
          keyPrefix: keyPrefix ?? name,
          onChanged: (values) {
            field.didChange(values);
            onChanged?.call(values);
          },
        );
      },
    );
  }
}
