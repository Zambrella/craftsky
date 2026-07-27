import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Shared paper-cutout card surface: white paper, chunky ink border,
/// rounded corners, and a hard-offset shadow.
class CraftskyCard extends StatelessWidget {
  const CraftskyCard({
    required this.child,
    this.margin,
    this.padding,
    this.clipBehavior = Clip.antiAlias,
    super.key,
  });

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
        border: Border.all(color: theme.colorScheme.onSurface, width: 1.5),
        boxShadow: shadows.dropSm,
      ),
      clipBehavior: clipBehavior,
      child: Material(
        type: MaterialType.transparency,
        clipBehavior: clipBehavior,
        borderRadius: BorderRadius.circular(radii.r3),
        child: child,
      ),
    );
  }
}
