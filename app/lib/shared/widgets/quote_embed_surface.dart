import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Shared inset surface for quoted posts and embedded CraftSky content.
class QuoteEmbedSurface extends StatelessWidget {
  const new({required this.child, super.key});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    return Material(
      color: swatches.paper2,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.r2),
        side: BorderSide(color: swatches.borderHair),
      ),
      clipBehavior: Clip.antiAlias,
      child: child,
    );
  }
}
