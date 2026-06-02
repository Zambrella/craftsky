import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

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
    this.suffixIcon,
    this.maxLines = 1,
    this.minLines,
    this.obscureText = false,
    this.keyboardType,
    this.textInputAction,
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
  final Widget? suffixIcon;
  final int? maxLines;
  final int? minLines;
  final bool obscureText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
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
    final sp = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final durations = theme.extension<DurationTheme>()!;
    final colors = theme.colorScheme;

    final hasError = widget.errorText != null;
    final belowText = widget.errorText ?? widget.helperText;

    final shadowOffset = shadows.dropSm.first.offset;
    final shadowColor = shadows.dropSm.first.color;
    // Unfocused: field sits at the shadow's resting position (shadow hidden
    // behind). Focused: field lifts back to origin, revealing the shadow.
    final lift = _focused ? Offset.zero : shadowOffset;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            if (widget.labelLeading != null) ...[
              widget.labelLeading!,
              SizedBox(width: sp.sp1),
            ],
            Expanded(
              child: Text(
                widget.label,
                style:
                    widget.labelStyle ??
                    theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w800,
                      color: hasError ? colors.error : colors.onSurface,
                    ),
              ),
            ),
            if (widget.labelTrailing != null) widget.labelTrailing!,
          ],
        ),
        SizedBox(height: sp.sp2),
        _FocusLift(
          lift: lift,
          shadowOffset: shadowOffset,
          shadowColor: shadowColor,
          borderRadius: BorderRadius.circular(radii.r3),
          duration: durations.fast,
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
            onChanged: widget.onChanged,
            onSubmitted: widget.onSubmitted,
            style: theme.textTheme.bodyLarge,
            decoration: InputDecoration(
              // The label/helper/error are rendered outside this decoration
              // by the surrounding column — see above and below.
              hintText: widget.hintText,
              prefixIcon: widget.prefixIcon,
              suffixIcon: widget.suffixIcon,
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
        ),
        if (belowText != null) ...[
          SizedBox(height: sp.sp2),
          Align(
            alignment: hasError
                ? AlignmentDirectional.centerStart
                : widget.helperAlignment,
            child: Text(
              belowText,
              style:
                  widget.helperStyle ??
                  theme.textTheme.bodyMedium?.copyWith(
                    color: hasError ? colors.error : colors.onSurfaceVariant,
                  ),
            ),
          ),
        ],
      ],
    );
  }
}

/// Stacks a static hard-offset shadow behind [child] and animates the child
/// between resting on top of the shadow (unfocused) and lifted back to
/// origin (focused). Reserves the outer rect so siblings don't shift as the
/// field lifts.
class _FocusLift extends StatelessWidget {
  const _FocusLift({
    required this.lift,
    required this.shadowOffset,
    required this.shadowColor,
    required this.borderRadius,
    required this.duration,
    required this.child,
  });

  /// Offset applied to the child. Origin = fully lifted (focused).
  /// [shadowOffset] = resting on the shadow (unfocused).
  final Offset lift;
  final Offset shadowOffset;
  final Color shadowColor;
  final BorderRadius borderRadius;
  final Duration duration;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final shadowPadding = EdgeInsets.only(
      left: shadowOffset.dx < 0 ? -shadowOffset.dx : 0,
      top: shadowOffset.dy < 0 ? -shadowOffset.dy : 0,
      right: shadowOffset.dx > 0 ? shadowOffset.dx : 0,
      bottom: shadowOffset.dy > 0 ? shadowOffset.dy : 0,
    );

    // The unpositioned child sizes the Stack; Positioned.fill stretches the
    // shadow underneath to match. Siblings below don't jump as the field
    // lifts because the Stack's size is driven by the child, not the shadow.
    return Padding(
      padding: shadowPadding,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned.fill(
            child: Transform.translate(
              offset: shadowOffset,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: shadowColor,
                  borderRadius: borderRadius,
                ),
              ),
            ),
          ),
          TweenAnimationBuilder<Offset>(
            tween: Tween<Offset>(begin: lift, end: lift),
            duration: duration,
            curve: Curves.easeOut,
            builder: (context, value, innerChild) {
              return Transform.translate(offset: value, child: innerChild);
            },
            child: child,
          ),
        ],
      ),
    );
  }
}
