import 'package:flutter/material.dart';

/// Raw CraftSky palette from the design system.
///
/// Mirrors `docs/design/colors_and_type.css`. Prefer reading semantic slots
/// from `Theme.of(context).colorScheme` or the brand-specific theme
/// extensions — use these only when defining the theme itself or when a
/// slot is genuinely absent from the `ColorScheme`/extension surface.
abstract final class BrandColors {
  // Paper — warm, not beige.
  static const paper = Color(0xFFF5EFE4);
  static const paper2 = Color(0xFFEFE7D6);
  static const paper3 = Color(0xFFFFFFFF);

  // Ink — near-black with warmth. Never pure #000.
  static const ink = Color(0xFF161210);
  static const ink2 = Color(0xFF3E3733);
  static const ink3 = Color(0xFF7A716B);
  static const ink4 = Color(0xFFA69E97);

  // Cobalt — the confident blue. Primary.
  static const cobalt = Color(0xFF1535D6);
  static const cobaltDeep = Color(0xFF0C1F8C);
  static const cobaltSoft = Color(0xFFE4E8FC);

  // Electric red — the accent. Sparingly.
  static const red = Color(0xFFF03A2E);
  static const redDeep = Color(0xFFB82016);
  static const redSoft = Color(0xFFFDDED9);

  // Supporting paper swatches — cutout backgrounds, chips, large surfaces.
  static const butter = Color(0xFFF7D46A);
  static const clay = Color(0xFFE27B4A);
  static const moss = Color(0xFF6E8B3D);
  static const sky = Color(0xFF9BC2E6);
  static const lilac = Color(0xFFC9B8E8);

  // Hairline: 15% ink for internal divisions inside cards.
  static const borderHair = Color(0x26161210);
}
