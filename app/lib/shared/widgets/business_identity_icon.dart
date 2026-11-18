import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

/// Matches the storefront's inset glyph to the full-canvas craft artwork.
class BusinessIdentityIcon extends StatelessWidget {
  const new({
    required this.size,
    this.color,
    this.customisation = ProfileCustomisation.defaults,
    super.key,
  });

  final double size;
  final Color? color;
  final ProfileCustomisation customisation;

  @override
  Widget build(BuildContext context) {
    final bundle =
        profileColourBundles[customisation.colour] ??
        profileColourBundles[ProfileCustomisation.defaults.colour]!;
    final accent = profileColour(
      Theme.of(context).brightness == Brightness.dark
          ? bundle.darkAccent
          : bundle.base,
    );
    return SizedBox.square(
      dimension: size,
      child: Transform.scale(
        scale: 1.25,
        child: Icon(
          CraftskyIcons.storefront,
          size: size,
          color: color ?? accent,
        ),
      ),
    );
  }
}
