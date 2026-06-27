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
    final base0 = FlexThemeData.light(
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
        // Chunky paper-cutout input: thick ink border, 14px corners, white
        // fill on paper background. Labels render outside the field per the
        // design — see `BrandTextField`.
        inputDecoratorRadius: 14,
        inputDecoratorIsFilled: true,
        inputDecoratorFillColor: BrandColors.paper3,
        inputDecoratorBorderType: FlexInputBorderType.outline,
        inputDecoratorBorderWidth: 1.5,
        inputDecoratorFocusedBorderWidth: 2,
        inputDecoratorBorderSchemeColor: SchemeColor.onSurface,
        inputDecoratorUnfocusedBorderIsColored: true,
        // Chips are pills.
        chipRadius: 999,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
      textTheme: _textTheme(ink: BrandColors.ink, ink2: BrandColors.ink2),
    );
    // Pin the `on-surface` family to the brand's four ink levels so callers
    // can read brand text strengths directly from `colorScheme` in standard
    // Material vocabulary, no `BrandColors.X` import needed:
    //   ink  → onSurface          (full-strength text, primary surface)
    //   ink2 → onSurfaceVariant   (M3's canonical secondary text)
    //   ink3 → outline            (tertiary text + decorative borders)
    //   ink4 → outlineVariant     (faintest tier; dividers, disabled lines)
    // M3 only has two text strengths officially — using outline/outlineVariant
    // for ink3/ink4 is a deliberate departure that lets the brand's four-level
    // hierarchy live inside the standard ColorScheme surface.
    final base = base0.copyWith(
      colorScheme: base0.colorScheme.copyWith(
        onSurface: BrandColors.ink,
        onSurfaceVariant: BrandColors.ink2,
        outline: BrandColors.ink3,
        outlineVariant: BrandColors.ink4,
      ),
    );
    return base.copyWith(
      extensions: _extensions(base.colorScheme),
      appBarTheme: _appBarTheme(base),
      navigationBarTheme: _navigationBarTheme(base),
      tabBarTheme: _tabBarTheme(base),
    );
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

  /// AppBar: paper background matching the scaffold, no elevation/tint, and a
  /// chunky ink rule along the bottom edge so it reads as a hand-cut
  /// paper-cutout boundary rather than a raised Material surface.
  static AppBarTheme _appBarTheme(ThemeData base) {
    return AppBarTheme(
      backgroundColor: BrandColors.paper,
      surfaceTintColor: Colors.transparent,
      foregroundColor: base.colorScheme.onSurface,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: base.textTheme.titleLarge,
      shape: Border(
        bottom: BorderSide(color: base.colorScheme.onSurface, width: 1.5),
      ),
    );
  }

  static TabBarThemeData _tabBarTheme(ThemeData base) {
    final muted = base.colorScheme.outline;
    final onSurface = base.colorScheme.onSurface;
    return TabBarThemeData(
      labelStyle: base.textTheme.labelMedium,
      unselectedLabelStyle: base.textTheme.labelMedium?.copyWith(color: muted),
      labelColor: onSurface,
      unselectedLabelColor: muted,
      indicatorColor: onSurface,
      dividerColor: Colors.transparent,
    );
  }

  /// NavigationBar: paper background matching the scaffold, chunky ink rule
  /// along the top edge (mirroring the AppBar), primary-coloured indicator +
  /// label for the selected destination, and faded ink for unselected tabs.
  /// The Material 3 tap-highlight is suppressed — the paper-cutout look
  /// prefers a clean surface without ripple/tint overlays.
  static NavigationBarThemeData _navigationBarTheme(ThemeData base) {
    final colors = base.colorScheme;
    // `outline` carries ink3 after the ColorScheme override in _buildLight.
    final unselected = colors.outline;
    return NavigationBarThemeData(
      backgroundColor: BrandColors.paper,
      surfaceTintColor: Colors.transparent,
      // No pill behind the selected icon — the primary-coloured icon + label
      // carry the selected state on their own.
      indicatorColor: Colors.transparent,
      overlayColor: const WidgetStatePropertyAll(Colors.transparent),
      elevation: 0,
      height: 64,
      labelTextStyle: WidgetStateProperty.resolveWith((states) {
        const base0 = TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w600,
        );
        if (states.contains(WidgetState.selected)) {
          return base0.copyWith(color: colors.primary);
        }
        return base0.copyWith(color: unselected);
      }),
      iconTheme: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) {
          return IconThemeData(color: colors.primary);
        }
        return IconThemeData(color: unselected);
      }),
      // Top hairline rule: the NavigationBar ships as a Material with its own
      // shape, so we wrap with a decoration — but NavigationBarThemeData
      // doesn't expose shape. The AppShell adds a Border on the wrapper
      // Container instead (see app_shell.dart).
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
        errorSurface: BrandColors.redSoft,
        warningSurface: BrandColors.butter,
        successSurface: BrandColors.moss,
        infoSurface: BrandColors.cobaltSoft,
      ),
    ];
  }
}
