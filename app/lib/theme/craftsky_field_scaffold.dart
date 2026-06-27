import 'dart:async';

import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class CraftskyFieldScaffold extends StatefulWidget {
  const CraftskyFieldScaffold({
    required this.label,
    required this.child,
    super.key,
    this.focusNode,
    this.helperText,
    this.errorText,
    this.counterText,
    this.labelLeading,
    this.labelTrailing,
    this.betweenLabelAndChild,
    this.labelStyle,
    this.showLabel = true,
    this.helperStyle,
    this.helperAlignment = AlignmentDirectional.centerStart,
    this.enabled = true,
    this.required = false,
    this.requiredLabel = 'required',
    this.semanticHint,
    this.semanticValue,
    this.textFieldSemantics = false,
  });

  final String label;
  final Widget child;
  final FocusNode? focusNode;
  final String? helperText;
  final String? errorText;
  final String? counterText;
  final Widget? labelLeading;
  final Widget? labelTrailing;
  final Widget? betweenLabelAndChild;
  final TextStyle? labelStyle;
  final bool showLabel;
  final TextStyle? helperStyle;
  final AlignmentGeometry helperAlignment;
  final bool enabled;
  final bool required;
  final String requiredLabel;
  final String? semanticHint;
  final String? semanticValue;
  final bool textFieldSemantics;

  @override
  State<CraftskyFieldScaffold> createState() => _CraftskyFieldScaffoldState();
}

class _CraftskyFieldScaffoldState extends State<CraftskyFieldScaffold> {
  FocusNode? _internalFocusNode;
  FocusNode get _focusNode =>
      widget.focusNode ?? (_internalFocusNode ??= FocusNode());

  bool _focused = false;

  @override
  void initState() {
    super.initState();
    _focused = _focusNode.hasFocus;
    _focusNode.addListener(_onFocusChange);
  }

  @override
  void didUpdateWidget(covariant CraftskyFieldScaffold oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.focusNode != widget.focusNode) {
      (oldWidget.focusNode ?? _internalFocusNode)?.removeListener(
        _onFocusChange,
      );
      _focusNode.addListener(_onFocusChange);
      _focused = _focusNode.hasFocus;
    }
  }

  void _onFocusChange() {
    final hasFocus = _focusNode.hasFocus;
    if (hasFocus != _focused) {
      setState(() => _focused = hasFocus);
    }
    if (hasFocus) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted || !_focusNode.hasFocus) return;
        unawaited(
          Scrollable.ensureVisible(
            context,
            alignment: 0.08,
            duration: const Duration(milliseconds: 180),
            curve: Curves.easeOut,
          ),
        );
      });
    }
  }

  @override
  void dispose() {
    _focusNode.removeListener(_onFocusChange);
    _internalFocusNode?.dispose();
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
    final labelStyle =
        widget.labelStyle ??
        theme.textTheme.titleMedium?.copyWith(
          fontWeight: FontWeight.w800,
          color: hasError ? colors.error : colors.onSurface,
        );
    final requiredLabelStyle = theme.textTheme.labelSmall?.copyWith(
      color: colors.onSurfaceVariant,
      fontWeight: FontWeight.w600,
    );

    final shadowOffset = shadows.dropSm.first.offset;
    final shadowColor = shadows.dropSm.first.color;
    final lift = _focused ? Offset.zero : shadowOffset;
    final semanticLabel = widget.required
        ? '${widget.label}, ${widget.requiredLabel}'
        : widget.label;
    final semanticHint = [
      widget.semanticHint,
      if (hasError) widget.errorText,
      if (!widget.enabled) 'Disabled',
    ].whereType<String>().where((text) => text.isNotEmpty).join('. ');

    return Semantics(
      container: true,
      textField: widget.textFieldSemantics,
      enabled: widget.enabled,
      label: semanticLabel,
      hint: semanticHint.isEmpty ? null : semanticHint,
      value: widget.semanticValue,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (widget.showLabel) ...[
            Row(
              children: [
                if (widget.labelLeading != null) ...[
                  widget.labelLeading!,
                  SizedBox(width: sp.sp1),
                ],
                Expanded(
                  child: ExcludeSemantics(
                    child: Text.rich(
                      TextSpan(
                        children: [
                          TextSpan(text: widget.label),
                          if (widget.required)
                            TextSpan(
                              text: '  ${widget.requiredLabel}',
                              style: requiredLabelStyle,
                            ),
                        ],
                      ),
                      style: labelStyle,
                      textAlign: TextAlign.start,
                    ),
                  ),
                ),
                if (widget.labelTrailing != null)
                  ExcludeSemantics(child: widget.labelTrailing),
              ],
            ),
            SizedBox(height: sp.sp2),
          ],
          if (widget.betweenLabelAndChild != null) ...[
            widget.betweenLabelAndChild!,
            SizedBox(height: sp.sp2),
          ],
          CraftskyFocusLift(
            lift: lift,
            shadowOffset: shadowOffset,
            shadowColor: shadowColor,
            borderRadius: BorderRadius.circular(radii.r3),
            duration: durations.fast,
            child: widget.child,
          ),
          if (belowText != null || widget.counterText != null) ...[
            SizedBox(height: sp.sp2),
            ExcludeSemantics(
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Align(
                      alignment: hasError
                          ? AlignmentDirectional.centerStart
                          : widget.helperAlignment,
                      child: belowText == null
                          ? const SizedBox.shrink()
                          : Text(
                              belowText,
                              style:
                                  widget.helperStyle ??
                                  theme.textTheme.bodyMedium?.copyWith(
                                    color: hasError
                                        ? colors.error
                                        : colors.onSurfaceVariant,
                                  ),
                            ),
                    ),
                  ),
                  if (widget.counterText != null) ...[
                    SizedBox(width: sp.sp2),
                    Text(
                      widget.counterText!,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: hasError
                            ? colors.error
                            : colors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class CraftskyFocusLift extends StatelessWidget {
  const CraftskyFocusLift({
    required this.lift,
    required this.shadowOffset,
    required this.shadowColor,
    required this.borderRadius,
    required this.duration,
    required this.child,
    super.key,
  });

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
