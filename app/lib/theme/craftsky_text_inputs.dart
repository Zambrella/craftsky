import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';

class CraftskyTextInput extends StatelessWidget {
  const CraftskyTextInput({
    required this.label,
    super.key,
    this.controller,
    this.initialValue,
    this.focusNode,
    this.hintText,
    this.helperText,
    this.errorText,
    this.textFieldKey,
    this.enabled = true,
    this.keyboardType,
    this.textInputAction,
    this.inputFormatters,
    this.autofillHints,
    this.minLines,
    this.maxLines = 1,
    this.onChanged,
    this.onSubmitted,
  }) : assert(
         controller == null || initialValue == null,
         'Provide either controller or initialValue, not both.',
       );

  final String label;
  final TextEditingController? controller;
  final String? initialValue;
  final FocusNode? focusNode;
  final String? hintText;
  final String? helperText;
  final String? errorText;
  final Key? textFieldKey;
  final bool enabled;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final List<TextInputFormatter>? inputFormatters;
  final Iterable<String>? autofillHints;
  final int? minLines;
  final int? maxLines;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;

  @override
  Widget build(BuildContext context) {
    return BrandTextField(
      label: label,
      controller: controller,
      initialValue: initialValue,
      focusNode: focusNode,
      hintText: hintText,
      helperText: helperText,
      errorText: errorText,
      textFieldKey: textFieldKey,
      enabled: enabled,
      keyboardType: keyboardType,
      textInputAction: textInputAction,
      inputFormatters: inputFormatters,
      autofillHints: autofillHints,
      minLines: minLines,
      maxLines: maxLines,
      onChanged: onChanged,
      onSubmitted: onSubmitted,
    );
  }
}

class CraftskyMultilineTextInput extends CraftskyTextInput {
  const CraftskyMultilineTextInput({
    required super.label,
    super.key,
    super.controller,
    super.initialValue,
    super.focusNode,
    super.hintText,
    super.helperText,
    super.errorText,
    super.textFieldKey,
    super.enabled,
    super.inputFormatters,
    super.autofillHints,
    super.minLines = 3,
    super.maxLines = 6,
    super.textInputAction = TextInputAction.newline,
    super.onChanged,
    super.onSubmitted,
  }) : super(keyboardType: TextInputType.multiline);
}

enum CraftskyNumberInputMode { integer, decimal }

class CraftskyNumberInput extends StatefulWidget {
  const CraftskyNumberInput({
    required this.label,
    super.key,
    this.controller,
    this.initialValue,
    this.focusNode,
    this.hintText,
    this.helperText,
    this.errorText,
    this.prefixText,
    this.suffixText,
    this.textFieldKey,
    this.enabled = true,
    this.mode = CraftskyNumberInputMode.decimal,
    this.onChanged,
    this.onSubmitted,
  }) : assert(
         controller == null || initialValue == null,
         'Provide either controller or initialValue, not both.',
       );

  final String label;
  final TextEditingController? controller;
  final num? initialValue;
  final FocusNode? focusNode;
  final String? hintText;
  final String? helperText;
  final String? errorText;
  final String? prefixText;
  final String? suffixText;
  final Key? textFieldKey;
  final bool enabled;
  final CraftskyNumberInputMode mode;
  final ValueChanged<num?>? onChanged;
  final ValueChanged<num?>? onSubmitted;

  @override
  State<CraftskyNumberInput> createState() => _CraftskyNumberInputState();
}

class _CraftskyNumberInputState extends State<CraftskyNumberInput> {
  TextEditingController? _internalController;

  TextEditingController get _controller =>
      widget.controller ??
      (_internalController ??= TextEditingController(text: _formatInitial()));

  String _formatInitial() => switch (widget.initialValue) {
    final int value => value.toString(),
    final num value => value.toString(),
    null => '',
  };

  @override
  void didUpdateWidget(covariant CraftskyNumberInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.controller == null &&
        oldWidget.initialValue != widget.initialValue) {
      final value = _formatInitial();
      if (_controller.text != value) {
        _controller.value = TextEditingValue(
          text: value,
          selection: TextSelection.collapsed(offset: value.length),
        );
      }
    }
  }

  @override
  void dispose() {
    _internalController?.dispose();
    super.dispose();
  }

  List<TextInputFormatter> get _inputFormatters {
    return switch (widget.mode) {
      CraftskyNumberInputMode.integer => [_NumberTextInputFormatter.integer()],
      CraftskyNumberInputMode.decimal => [_NumberTextInputFormatter.decimal()],
    };
  }

  num? _parse(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed == '-' || trimmed == '.') return null;
    return switch (widget.mode) {
      CraftskyNumberInputMode.integer => int.tryParse(trimmed),
      CraftskyNumberInputMode.decimal => num.tryParse(trimmed),
    };
  }

  @override
  Widget build(BuildContext context) {
    return BrandTextField(
      label: widget.label,
      controller: _controller,
      focusNode: widget.focusNode,
      hintText: widget.hintText,
      helperText: widget.helperText,
      errorText: widget.errorText,
      prefixText: widget.prefixText,
      suffixText: widget.suffixText,
      textFieldKey: widget.textFieldKey,
      enabled: widget.enabled,
      keyboardType: TextInputType.numberWithOptions(
        decimal: widget.mode == CraftskyNumberInputMode.decimal,
        signed: true,
      ),
      textInputAction: TextInputAction.next,
      inputFormatters: _inputFormatters,
      onChanged: (value) => widget.onChanged?.call(_parse(value)),
      onSubmitted: (value) => widget.onSubmitted?.call(_parse(value)),
    );
  }
}

class _NumberTextInputFormatter extends TextInputFormatter {
  _NumberTextInputFormatter._(this._pattern);

  factory _NumberTextInputFormatter.integer() {
    return _NumberTextInputFormatter._(RegExp(r'^-?\d*$'));
  }

  factory _NumberTextInputFormatter.decimal() {
    return _NumberTextInputFormatter._(RegExp(r'^-?\d*\.?\d*$'));
  }

  final RegExp _pattern;

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    return _pattern.hasMatch(newValue.text) ? newValue : oldValue;
  }
}

class CraftskyFormTextField extends StatelessWidget {
  const CraftskyFormTextField({
    required this.name,
    required this.label,
    super.key,
    this.controller,
    this.focusNode,
    this.initialValue,
    this.hintText,
    this.helperText,
    this.enabled = true,
    this.validator,
    this.keyboardType,
    this.textInputAction,
    this.inputFormatters,
    this.autofillHints,
    this.minLines,
    this.maxLines = 1,
    this.onChanged,
    this.onSubmitted,
    this.textFieldKey,
  }) : assert(
         controller == null || initialValue == null,
         'Provide either controller or initialValue, not both.',
       );

  final String name;
  final String label;
  final TextEditingController? controller;
  final FocusNode? focusNode;
  final String? initialValue;
  final String? hintText;
  final String? helperText;
  final bool enabled;
  final FormFieldValidator<String>? validator;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final List<TextInputFormatter>? inputFormatters;
  final Iterable<String>? autofillHints;
  final int? minLines;
  final int? maxLines;
  final ValueChanged<String?>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final Key? textFieldKey;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<String>(
      name: name,
      initialValue: initialValue ?? controller?.text ?? '',
      enabled: enabled,
      focusNode: focusNode,
      validator: validator,
      builder: (field) {
        return _CraftskyFormTextAdapter(
          field: field,
          label: label,
          controller: controller,
          focusNode: focusNode,
          hintText: hintText,
          helperText: helperText,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
          inputFormatters: inputFormatters,
          autofillHints: autofillHints,
          minLines: minLines,
          maxLines: maxLines,
          onChanged: onChanged,
          onSubmitted: onSubmitted,
          textFieldKey: textFieldKey,
        );
      },
    );
  }
}

class CraftskyFormMultilineTextField extends CraftskyFormTextField {
  const CraftskyFormMultilineTextField({
    required super.name,
    required super.label,
    super.key,
    super.controller,
    super.focusNode,
    super.initialValue,
    super.hintText,
    super.helperText,
    super.enabled,
    super.validator,
    super.inputFormatters,
    super.autofillHints,
    super.textInputAction = TextInputAction.newline,
    super.minLines = 3,
    super.maxLines = 6,
    super.onChanged,
    super.onSubmitted,
    super.textFieldKey,
  }) : super(keyboardType: TextInputType.multiline);
}

class _CraftskyFormTextAdapter extends StatefulWidget {
  const _CraftskyFormTextAdapter({
    required this.field,
    required this.label,
    required this.controller,
    required this.focusNode,
    required this.hintText,
    required this.helperText,
    required this.keyboardType,
    required this.textInputAction,
    required this.inputFormatters,
    required this.autofillHints,
    required this.minLines,
    required this.maxLines,
    required this.onChanged,
    required this.onSubmitted,
    required this.textFieldKey,
  });

  final FormFieldState<String> field;
  final String label;
  final TextEditingController? controller;
  final FocusNode? focusNode;
  final String? hintText;
  final String? helperText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final List<TextInputFormatter>? inputFormatters;
  final Iterable<String>? autofillHints;
  final int? minLines;
  final int? maxLines;
  final ValueChanged<String?>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final Key? textFieldKey;

  @override
  State<_CraftskyFormTextAdapter> createState() =>
      _CraftskyFormTextAdapterState();
}

class _CraftskyFormTextAdapterState extends State<_CraftskyFormTextAdapter> {
  TextEditingController? _internalController;

  TextEditingController get _controller =>
      widget.controller ??
      (_internalController ??= TextEditingController(text: widget.field.value));

  @override
  void didUpdateWidget(covariant _CraftskyFormTextAdapter oldWidget) {
    super.didUpdateWidget(oldWidget);
    final value = widget.field.value ?? '';
    if (_controller.text != value) {
      _controller.value = TextEditingValue(
        text: value,
        selection: TextSelection.collapsed(offset: value.length),
      );
    }
  }

  @override
  void dispose() {
    _internalController?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return CraftskyTextInput(
      label: widget.label,
      controller: _controller,
      focusNode: widget.focusNode,
      hintText: widget.hintText,
      helperText: widget.helperText,
      errorText: widget.field.errorText,
      keyboardType: widget.keyboardType,
      textInputAction: widget.textInputAction,
      inputFormatters: widget.inputFormatters,
      autofillHints: widget.autofillHints,
      minLines: widget.minLines,
      maxLines: widget.maxLines,
      enabled: widget.field.widget.enabled,
      textFieldKey: widget.textFieldKey,
      onChanged: (value) {
        widget.field.didChange(value);
        widget.onChanged?.call(value);
      },
      onSubmitted: widget.onSubmitted,
    );
  }
}

class CraftskyFormNumberField extends StatelessWidget {
  const CraftskyFormNumberField({
    required this.name,
    required this.label,
    super.key,
    this.controller,
    this.focusNode,
    this.initialValue,
    this.hintText,
    this.helperText,
    this.prefixText,
    this.suffixText,
    this.enabled = true,
    this.mode = CraftskyNumberInputMode.decimal,
    this.validator,
    this.onChanged,
    this.textFieldKey,
  }) : assert(
         controller == null || initialValue == null,
         'Provide either controller or initialValue, not both.',
       );

  final String name;
  final String label;
  final TextEditingController? controller;
  final FocusNode? focusNode;
  final num? initialValue;
  final String? hintText;
  final String? helperText;
  final String? prefixText;
  final String? suffixText;
  final bool enabled;
  final CraftskyNumberInputMode mode;
  final FormFieldValidator<num>? validator;
  final ValueChanged<num?>? onChanged;
  final Key? textFieldKey;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<num>(
      name: name,
      initialValue: initialValue ?? _parse(controller?.text),
      enabled: enabled,
      focusNode: focusNode,
      validator: validator,
      builder: (field) {
        return CraftskyNumberInput(
          label: label,
          controller: controller,
          focusNode: focusNode,
          initialValue: field.value,
          hintText: hintText,
          helperText: helperText,
          errorText: field.errorText,
          prefixText: prefixText,
          suffixText: suffixText,
          textFieldKey: textFieldKey,
          mode: mode,
          enabled: field.widget.enabled,
          onChanged: (value) {
            field.didChange(value);
            onChanged?.call(value);
          },
        );
      },
    );
  }

  num? _parse(String? value) {
    if (value == null || value.trim().isEmpty) return null;
    return switch (mode) {
      CraftskyNumberInputMode.integer => int.tryParse(value.trim()),
      CraftskyNumberInputMode.decimal => num.tryParse(value.trim()),
    };
  }
}
