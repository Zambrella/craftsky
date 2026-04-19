import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flex_color_scheme/flex_color_scheme.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// CraftSky theme — paper-cutout direction. Warm cream paper, ink-black rules,
/// cobalt + electric red accents. See `docs/design/design-system.md` and
/// `docs/design/colors_and_type.css` for the source of truth.
///
/// Dark mode is a stub; the brand is paper-warm by nature and a proper dark
/// palette hasn't been designed yet.
class AppTheme {
  AppTheme._();

  static final ThemeData lightThemeData = _buildLight();
  static final ThemeData darkThemeData = _buildDark();

  static const _lightColors = FlexSchemeColor(
    primary: BrandColors.cobalt,
    primaryContainer: BrandColors.cobaltSoft,
    secondary: BrandColors.red,
    secondaryContainer: BrandColors.redSoft,
    tertiary: BrandColors.butter,
    tertiaryContainer: BrandColors.clay,
    appBarColor: BrandColors.paper,
    error: BrandColors.red,
  );

  static ThemeData _buildLight() {
    final base = FlexThemeData.light(
      colors: _lightColors,
      scaffoldBackground: BrandColors.paper,
      surface: BrandColors.paper3,
      subThemesData: const FlexSubThemesData(
        interactionEffects: true,
        tintedDisabledControls: true,
        // Cards get the chunky 14px corner from the design system.
        cardRadius: 14,
        // Primary pill buttons.
        elevatedButtonRadius: 999,
        filledButtonRadius: 999,
        outlinedButtonRadius: 999,
        textButtonRadius: 999,
        // Form fields sit almost square — 2px corners per the system.
        inputDecoratorRadius: 2,
        inputDecoratorIsFilled: true,
        inputDecoratorBorderType: FlexInputBorderType.outline,
        // Chips are pills.
        chipRadius: 999,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
      textTheme: _textTheme(ink: BrandColors.ink, ink2: BrandColors.ink2),
    );
    return base.copyWith(extensions: _extensions(base.colorScheme));
  }

  // Dark mode is intentionally minimal — brand is paper-warm and a proper
  // dark palette has not been designed yet. Keeping a Material fallback so
  // the system still has something to return.
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

  /// Outfit for UI, DM Serif Display for editorial display, JetBrains Mono
  /// for code-ish content. Weight, size, and letter-spacing follow the rhythm
  /// notes in `docs/design/design-system.md` ("Typography").
  static TextTheme _textTheme({required Color ink, required Color ink2}) {
    final display = GoogleFonts.dmSerifDisplayTextTheme();
    final ui = GoogleFonts.outfitTextTheme();

    return TextTheme(
      // Display — chunky serif with tight line-height, editorial scale.
      displayLarge: display.displayLarge?.copyWith(
        fontSize: 96,
        height: 0.95,
        letterSpacing: -0.025 * 96,
        color: ink,
      ),
      displayMedium: display.displayMedium?.copyWith(
        fontSize: 64,
        height: 1.02,
        letterSpacing: -0.02 * 64,
        color: ink,
      ),
      displaySmall: display.displaySmall?.copyWith(
        fontSize: 42,
        height: 1.05,
        letterSpacing: -0.02 * 42,
        color: ink,
      ),

      // Headlines — Outfit, heavy, tight tracking.
      headlineLarge: ui.headlineLarge?.copyWith(
        fontSize: 42,
        fontWeight: FontWeight.w800,
        height: 1.1,
        letterSpacing: -0.03 * 42,
        color: ink,
      ),
      headlineMedium: ui.headlineMedium?.copyWith(
        fontSize: 30,
        fontWeight: FontWeight.w700,
        height: 1.15,
        letterSpacing: -0.02 * 30,
        color: ink,
      ),
      headlineSmall: ui.headlineSmall?.copyWith(
        fontSize: 22,
        fontWeight: FontWeight.w700,
        height: 1.2,
        letterSpacing: -0.015 * 22,
        color: ink,
      ),

      // Titles — for card titles, list section heads. Outfit, heavy.
      titleLarge: ui.titleLarge?.copyWith(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        height: 1.25,
        color: ink,
      ),
      titleMedium: ui.titleMedium?.copyWith(
        fontSize: 16,
        fontWeight: FontWeight.w600,
        height: 1.3,
        color: ink,
      ),
      titleSmall: ui.titleSmall?.copyWith(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        height: 1.3,
        color: ink,
      ),

      // Body — Outfit 400, roomy line-height.
      bodyLarge: ui.bodyLarge?.copyWith(
        fontSize: 16,
        fontWeight: FontWeight.w400,
        height: 1.5,
        color: ink,
      ),
      bodyMedium: ui.bodyMedium?.copyWith(
        fontSize: 14,
        fontWeight: FontWeight.w400,
        height: 1.5,
        color: ink,
      ),
      bodySmall: ui.bodySmall?.copyWith(
        fontSize: 13,
        fontWeight: FontWeight.w400,
        height: 1.45,
        color: ink2,
      ),

      // Labels — buttons (heavy) and eyebrow labels (uppercase, tracked).
      labelLarge: ui.labelLarge?.copyWith(
        fontSize: 15,
        fontWeight: FontWeight.w700,
        height: 1.2,
        color: ink,
      ),
      labelMedium: ui.labelMedium?.copyWith(
        fontSize: 13,
        fontWeight: FontWeight.w600,
        height: 1.3,
        color: ink,
      ),
      labelSmall: ui.labelSmall?.copyWith(
        fontSize: 12,
        fontWeight: FontWeight.w700,
        height: 1.2,
        letterSpacing: 0.14 * 12,
        color: ink2,
      ),
    );
  }

  static List<ThemeExtension<dynamic>> _extensions(ColorScheme scheme) {
    return <ThemeExtension<dynamic>>[
      const SpacingTheme(),
      const RadiusTheme(),
      const DurationTheme(),
      const BrandShadowTheme(),
      const BrandSwatchTheme(),
      const SemanticColorsTheme(
        error: BrandColors.red,
        warning: BrandColors.butter,
        success: BrandColors.moss,
        info: BrandColors.cobalt,
      ),
    ];
  }
}
