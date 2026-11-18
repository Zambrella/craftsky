import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Shared paper-cutout card surface: white paper, chunky ink border,
/// rounded corners, and a hard-offset shadow.
class CraftskyCard extends StatelessWidget {
  const new({
    required this.child,
    this.margin,
    this.padding,
    this.clipBehavior = Clip.antiAlias,
    super.key,
  });

  static const _borderWidth = 1.5;

  final Widget child;
  final EdgeInsetsGeometry? margin;
  final EdgeInsetsGeometry? padding;
  final Clip clipBehavior;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;

    return Container(
      margin: margin,
      padding: padding,
      decoration: BoxDecoration(
        color: swatches.paper3,
        borderRadius: BorderRadius.circular(radii.r3),
        border: Border.all(
          color: theme.colorScheme.onSurface,
          width: _borderWidth,
        ),
        boxShadow: shadows.dropSm,
      ),
      clipBehavior: clipBehavior,
      child: Material(
        type: MaterialType.transparency,
        clipBehavior: clipBehavior,
        // Container insets its child by the border width. Match that inner
        // curve so edge-to-edge images meet the border without a paper gap.
        borderRadius: BorderRadius.circular(
          radii.r3 > _borderWidth ? radii.r3 - _borderWidth : 0,
        ),
        child: child,
      ),
    );
  }
}
