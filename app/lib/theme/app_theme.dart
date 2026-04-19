import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:dynamic_color/dynamic_color.dart';
import 'package:flex_color_scheme/flex_color_scheme.dart';
import 'package:flutter/material.dart';

class AppTheme {
  AppTheme._();

  static final ThemeData lightThemeData = _buildLight();
  static final ThemeData darkThemeData = _buildDark();

  static ThemeData _buildLight() {
    final base = FlexThemeData.light(
      scheme: FlexScheme.material,
      subThemesData: const FlexSubThemesData(
        interactionEffects: true,
        tintedDisabledControls: true,
        defaultRadius: 8,
        inputDecoratorIsFilled: true,
        inputDecoratorBorderType: FlexInputBorderType.outline,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
    );
    return base.copyWith(extensions: _extensions(base.colorScheme));
  }

  static ThemeData _buildDark() {
    final base = FlexThemeData.dark(
      scheme: FlexScheme.material,
      subThemesData: const FlexSubThemesData(
        interactionEffects: true,
        tintedDisabledControls: true,
        defaultRadius: 8,
        inputDecoratorIsFilled: true,
        inputDecoratorBorderType: FlexInputBorderType.outline,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
    );
    return base.copyWith(extensions: _extensions(base.colorScheme));
  }

  static List<ThemeExtension<dynamic>> _extensions(ColorScheme scheme) {
    return <ThemeExtension<dynamic>>[
      const SpacingTheme(),
      const RadiusTheme(),
      const DurationTheme(),
      // Error harmonizes toward the scheme's error slot (which already carries
      // the error hue); the other three have no dedicated scheme slot, so they
      // harmonize toward primary.
      SemanticColorsTheme(
        error: Colors.red.harmonizeWith(scheme.error),
        warning: Colors.orange.harmonizeWith(scheme.primary),
        success: Colors.green.harmonizeWith(scheme.primary),
        info: Colors.blue.harmonizeWith(scheme.primary),
      ),
    ];
  }
}
