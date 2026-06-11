import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';

class CraftskyFormBuilderTextField extends StatelessWidget {
  const CraftskyFormBuilderTextField({
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
        return _CraftskyFormBuilderTextAdapter(
          field: field,
          label: label,
          controller: controller,
          focusNode: focusNode,
          hintText: hintText,
          helperText: helperText,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
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

class CraftskyFormBuilderMultilineTextField
    extends CraftskyFormBuilderTextField {
  const CraftskyFormBuilderMultilineTextField({
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
    super.textInputAction = TextInputAction.newline,
    super.minLines = 3,
    super.maxLines = 6,
    super.onChanged,
    super.onSubmitted,
    super.textFieldKey,
  }) : super(keyboardType: TextInputType.multiline);
}

class _CraftskyFormBuilderTextAdapter extends StatefulWidget {
  const _CraftskyFormBuilderTextAdapter({
    required this.field,
    required this.label,
    required this.controller,
    required this.focusNode,
    required this.hintText,
    required this.helperText,
    required this.keyboardType,
    required this.textInputAction,
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
  final int? minLines;
  final int? maxLines;
  final ValueChanged<String?>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final Key? textFieldKey;

  @override
  State<_CraftskyFormBuilderTextAdapter> createState() =>
      _CraftskyFormBuilderTextAdapterState();
}

class _CraftskyFormBuilderTextAdapterState
    extends State<_CraftskyFormBuilderTextAdapter> {
  TextEditingController? _internalController;

  TextEditingController get _controller =>
      widget.controller ??
      (_internalController ??= TextEditingController(text: widget.field.value));

  @override
  void didUpdateWidget(covariant _CraftskyFormBuilderTextAdapter oldWidget) {
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
    return BrandTextField(
      label: widget.label,
      controller: _controller,
      focusNode: widget.focusNode,
      hintText: widget.hintText,
      helperText: widget.helperText,
      errorText: widget.field.errorText,
      keyboardType: widget.keyboardType,
      textInputAction: widget.textInputAction,
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
