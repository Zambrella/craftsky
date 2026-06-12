import 'package:craftsky_app/theme/craftsky_text_inputs.dart';

class CraftskyFormBuilderTextField extends CraftskyFormTextField {
  const CraftskyFormBuilderTextField({
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
    super.keyboardType,
    super.textInputAction,
    super.inputFormatters,
    super.autofillHints,
    super.minLines,
    super.maxLines,
    super.onChanged,
    super.onSubmitted,
    super.textFieldKey,
  });
}

class CraftskyFormBuilderMultilineTextField
    extends CraftskyFormMultilineTextField {
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
    super.inputFormatters,
    super.autofillHints,
    super.textInputAction,
    super.minLines,
    super.maxLines,
    super.onChanged,
    super.onSubmitted,
    super.textFieldKey,
  });
}
