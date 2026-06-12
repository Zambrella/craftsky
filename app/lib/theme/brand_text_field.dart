import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// A text field dressed in the CraftSky paper-cutout style: heavy label
/// above the field, chunky ink-black outline, and a hard-offset shadow
/// that appears while the field is focused — "lifting" the field off the
/// page when the user is interacting with it. Helper / error text sits
/// below in muted ink.
///
/// The surrounding [InputDecorationTheme] (set in `AppTheme`) gives the
/// inner [TextField] its radius, border width, and fill — this widget
/// adds the shadow and positions the label and helper outside the
/// decorated area.
class BrandTextField extends StatefulWidget {
  const BrandTextField({
    required this.label,
    super.key,
    this.controller,
    this.initialValue,
    this.focusNode,
    this.hintText,
    this.helperText,
    this.errorText,
    this.labelLeading,
    this.labelTrailing,
    this.labelStyle,
    this.helperStyle,
    this.helperAlignment = AlignmentDirectional.centerStart,
    this.textFieldKey,
    this.prefixIcon,
    this.prefixText,
    this.suffixIcon,
    this.suffixText,
    this.maxLines = 1,
    this.minLines,
    this.obscureText = false,
    this.keyboardType,
    this.textInputAction,
    this.inputFormatters,
    this.autofillHints,
    this.onChanged,
    this.onSubmitted,
    this.enabled = true,
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
  final Widget? labelLeading;
  final Widget? labelTrailing;
  final TextStyle? labelStyle;
  final TextStyle? helperStyle;
  final AlignmentGeometry helperAlignment;
  final Key? textFieldKey;
  final Widget? prefixIcon;
  final String? prefixText;
  final Widget? suffixIcon;
  final String? suffixText;
  final int? maxLines;
  final int? minLines;
  final bool obscureText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final List<TextInputFormatter>? inputFormatters;
  final Iterable<String>? autofillHints;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final bool enabled;

  @override
  State<BrandTextField> createState() => _BrandTextFieldState();
}

class _BrandTextFieldState extends State<BrandTextField> {
  FocusNode? _internalFocusNode;
  FocusNode get _focusNode =>
      widget.focusNode ?? (_internalFocusNode ??= FocusNode());
  TextEditingController? _internalController;
  TextEditingController? get _controller =>
      widget.controller ??
      (_internalController ??= _createInternalController());

  bool _focused = false;

  TextEditingController _createInternalController() {
    return TextEditingController(text: widget.initialValue ?? '');
  }

  @override
  void initState() {
    super.initState();
    _focusNode.addListener(_onFocusChange);
  }

  @override
  void didUpdateWidget(covariant BrandTextField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.focusNode != widget.focusNode) {
      (oldWidget.focusNode ?? _internalFocusNode)?.removeListener(
        _onFocusChange,
      );
      _focusNode.addListener(_onFocusChange);
    }
    if (widget.controller == null &&
        oldWidget.initialValue != widget.initialValue) {
      final controller = _controller;
      final nextText = widget.initialValue ?? '';
      if (controller != null && controller.text != nextText) {
        controller.value = TextEditingValue(
          text: nextText,
          selection: TextSelection.collapsed(offset: nextText.length),
        );
      }
    }
  }

  void _onFocusChange() {
    final hasFocus = _focusNode.hasFocus;
    if (hasFocus != _focused) {
      setState(() => _focused = hasFocus);
    }
  }

  @override
  void dispose() {
    _focusNode.removeListener(_onFocusChange);
    _internalFocusNode?.dispose();
    _internalController?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final colors = theme.colorScheme;
    final hasError = widget.errorText != null;

    return CraftskyFieldScaffold(
      label: widget.label,
      focusNode: _focusNode,
      helperText: widget.helperText,
      errorText: widget.errorText,
      labelLeading: widget.labelLeading,
      labelTrailing: widget.labelTrailing,
      labelStyle: widget.labelStyle,
      helperStyle: widget.helperStyle,
      helperAlignment: widget.helperAlignment,
      enabled: widget.enabled,
      semanticHint: widget.hintText,
      semanticValue: _controller?.text,
      textFieldSemantics: true,
      child: TextField(
        key: widget.textFieldKey,
        controller: _controller,
        focusNode: _focusNode,
        enabled: widget.enabled,
        maxLines: widget.maxLines,
        minLines: widget.minLines,
        obscureText: widget.obscureText,
        keyboardType: widget.keyboardType,
        textInputAction: widget.textInputAction,
        inputFormatters: widget.inputFormatters,
        autofillHints: widget.autofillHints,
        onChanged: widget.onChanged,
        onSubmitted: widget.onSubmitted,
        style: theme.textTheme.bodyLarge,
        decoration: InputDecoration(
          // The label/helper/error are rendered outside this decoration
          // by the surrounding column — see above and below.
          hintText: widget.hintText,
          prefixIcon: widget.prefixIcon,
          prefixText: widget.prefixText,
          suffixIcon: widget.suffixIcon,
          suffixText: widget.suffixText,
          errorBorder: hasError
              ? OutlineInputBorder(
                  borderRadius: BorderRadius.circular(radii.r3),
                  borderSide: BorderSide(color: colors.error, width: 2),
                )
              : null,
          focusedErrorBorder: hasError
              ? OutlineInputBorder(
                  borderRadius: BorderRadius.circular(radii.r3),
                  borderSide: BorderSide(color: colors.error, width: 2),
                )
              : null,
        ),
      ),
    );
  }
}
