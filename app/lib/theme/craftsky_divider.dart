import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Zero-padding hairline divider using the Craftsky border hair colour.
class CraftskyDivider extends StatelessWidget {
  const CraftskyDivider({
    this.axis = Axis.horizontal,
    this.thickness = 1,
    this.indent = 0,
    this.endIndent = 0,
    this.color,
    super.key,
  });

  final Axis axis;
  final double thickness;
  final double indent;
  final double endIndent;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final swatches = Theme.of(context).extension<BrandSwatchTheme>()!;
    final dividerColor = color ?? swatches.borderHair;

    return switch (axis) {
      Axis.horizontal => Padding(
        padding: EdgeInsetsDirectional.only(start: indent, end: endIndent),
        child: ColoredBox(
          color: dividerColor,
          child: SizedBox(height: thickness, width: double.infinity),
        ),
      ),
      Axis.vertical => Padding(
        padding: EdgeInsetsDirectional.only(top: indent, bottom: endIndent),
        child: ColoredBox(
          color: dividerColor,
          child: SizedBox(width: thickness, height: double.infinity),
        ),
      ),
    };
  }
}
