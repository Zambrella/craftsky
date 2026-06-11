import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';

class CraftskySelectOption<T> {
  const CraftskySelectOption({
    required this.value,
    required this.label,
    this.description,
  });

  final T value;
  final String label;
  final String? description;
}

class CraftskyFormBuilderDropdownField<T> extends StatelessWidget {
  const CraftskyFormBuilderDropdownField({
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
        return InputDecorator(
          decoration: InputDecoration(
            labelText: label,
            helperText: field.errorText == null ? helperText : null,
            errorText: field.errorText,
          ),
          isEmpty: field.value == null,
          child: DropdownButtonHideUnderline(
            child: DropdownButton<T>(
              value: field.value,
              isExpanded: true,
              items: [
                for (final option in options)
                  DropdownMenuItem<T>(
                    value: option.value,
                    child: Text(option.label),
                  ),
              ],
              onChanged: field.widget.enabled
                  ? (value) {
                      field.didChange(value);
                      onChanged?.call(value);
                    }
                  : null,
            ),
          ),
        );
      },
    );
  }
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
  });

  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final List<T> initialValue;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<List<T>>? validator;
  final ValueChanged<List<T>>? onChanged;
  final bool allowCustomValues;
  final int? maxSelected;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<T>>(
      name: name,
      initialValue: initialValue,
      enabled: enabled,
      validator: validator,
      builder: (field) {
        return _CraftskyMultiSelectBody<T>(
          field: field,
          name: name,
          label: label,
          options: options,
          helperText: helperText,
          allowCustomValues: allowCustomValues,
          maxSelected: maxSelected,
          onChanged: onChanged,
        );
      },
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
        return InputDecorator(
          decoration: InputDecoration(
            labelText: label,
            helperText: field.errorText == null ? helperText : null,
            errorText: field.errorText,
            enabled: field.widget.enabled,
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
        );
      },
    );
  }
}

class _CraftskyMultiSelectBody<T> extends StatefulWidget {
  const _CraftskyMultiSelectBody({
    required this.field,
    required this.name,
    required this.label,
    required this.options,
    required this.helperText,
    required this.allowCustomValues,
    required this.maxSelected,
    required this.onChanged,
  });

  final FormFieldState<List<T>> field;
  final String name;
  final String label;
  final List<CraftskySelectOption<T>> options;
  final String? helperText;
  final bool allowCustomValues;
  final int? maxSelected;
  final ValueChanged<List<T>>? onChanged;

  @override
  State<_CraftskyMultiSelectBody<T>> createState() =>
      _CraftskyMultiSelectBodyState<T>();
}

class _CraftskyMultiSelectBodyState<T>
    extends State<_CraftskyMultiSelectBody<T>> {
  late final TextEditingController _customController;
  String? _limitText;

  @override
  void initState() {
    super.initState();
    _customController = TextEditingController();
  }

  @override
  void dispose() {
    _customController.dispose();
    super.dispose();
  }

  List<T> get _selected => List<T>.from(widget.field.value ?? <T>[]);

  bool get _enabled => widget.field.widget.enabled;

  void _setSelected(List<T> selected) {
    widget.field.didChange(List<T>.unmodifiable(selected));
    widget.onChanged?.call(List<T>.unmodifiable(selected));
  }

  bool _atLimit(List<T> selected) {
    final max = widget.maxSelected;
    return max != null && selected.length >= max;
  }

  void _showLimit() {
    final max = widget.maxSelected;
    if (max == null) return;
    setState(() => _limitText = 'You can choose up to $max.');
  }

  void _toggle(T value) {
    if (!_enabled) return;
    final selected = _selected;
    if (selected.contains(value)) {
      selected.remove(value);
      setState(() => _limitText = null);
      _setSelected(selected);
      return;
    }
    if (_atLimit(selected)) {
      _showLimit();
      return;
    }
    selected.add(value);
    setState(() => _limitText = null);
    _setSelected(selected);
  }

  void _addCustom() {
    if (!_enabled) return;
    final text = _customController.text.trim();
    if (text.isEmpty) return;
    final selected = _selected;
    if (_atLimit(selected)) {
      _showLimit();
      return;
    }
    final value = text as T;
    if (!selected.contains(value)) {
      selected.add(value);
      _setSelected(selected);
    }
    _customController.clear();
    setState(() => _limitText = null);
  }

  void _remove(T value) {
    if (!_enabled) return;
    final selected = _selected..remove(value);
    setState(() => _limitText = null);
    _setSelected(selected);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final selected = _selected;
    final errorText = widget.field.errorText ?? _limitText;
    final optionLabelByValue = {
      for (final option in widget.options) option.value: option.label,
    };

    return InputDecorator(
      decoration: InputDecoration(
        labelText: widget.label,
        helperText: errorText == null ? widget.helperText : null,
        errorText: errorText,
        enabled: _enabled,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (selected.isNotEmpty)
            Wrap(
              spacing: 8,
              runSpacing: 4,
              children: [
                for (final value in selected)
                  Chip(
                    label: Text(optionLabelByValue[value] ?? value.toString()),
                    onDeleted: _enabled ? () => _remove(value) : null,
                    deleteIcon: Icon(
                      Icons.close,
                      key: Key('${widget.name}-remove-$value'),
                    ),
                  ),
              ],
            ),
          if (widget.options.isNotEmpty)
            Wrap(
              spacing: 8,
              children: [
                for (final option in widget.options)
                  FilterChip(
                    key: Key('${widget.name}-option-${option.value}'),
                    label: Text(option.label),
                    selected: selected.contains(option.value),
                    onSelected: _enabled ? (_) => _toggle(option.value) : null,
                  ),
              ],
            ),
          if (widget.allowCustomValues) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    key: Key('${widget.name}-custom-input'),
                    controller: _customController,
                    enabled: _enabled,
                    decoration: const InputDecoration(hintText: 'Add item'),
                  ),
                ),
                const SizedBox(width: 8),
                TextButton(
                  key: Key('${widget.name}-add-custom'),
                  onPressed: _enabled ? _addCustom : null,
                  child: const Text('Add'),
                ),
              ],
            ),
          ],
          if (!_enabled)
            Text(
              'Disabled',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
        ],
      ),
    );
  }
}
