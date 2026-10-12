import 'package:craftsky_app/theme/craftsky_text_inputs.dart';

class CraftskyFormBuilderTextField extends CraftskyFormTextField {
  const new({
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
    super.textCapitalization,
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
  const new({
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
    super.textCapitalization,
    super.minLines,
    super.maxLines,
    super.onChanged,
    super.onSubmitted,
    super.textFieldKey,
  });
}
