import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flex_color_scheme/flex_color_scheme.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// CraftSky theme — paper-cutout direction. Warm cream paper, ink-black rules,
/// cobalt + electric red accents. See `docs/design/design-system.md` and
/// `docs/design/colors_and_type.css` for the source of truth.
///
/// Dark mode translates the same paper-cutout system onto warm charcoal paper.
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

  static const _darkPage = Color(0xFF171513);
  static const _darkSunken = Color(0xFF12100F);
  static const _darkSurface = Color(0xFF24201D);
  static const _darkSurfaceHigh = Color(0xFF2D2925);
  static const _darkInk2 = Color(0xFFCFC6BB);
  static const _darkInk3 = Color(0xFF968D84);
  static const _darkInk4 = Color(0xFF6F6862);
  static const _darkCobalt = Color(0xFF7890FF);

  static const _darkSwatches = BrandSwatchTheme(
    paper: _darkPage,
    paper2: _darkSunken,
    paper3: _darkSurface,
    butter: Color(0xFF5A4815),
    clay: Color(0xFF6A2F1A),
    moss: Color(0xFF425626),
    onMoss: BrandColors.paper,
    sky: Color(0xFF274964),
    lilac: Color(0xFF493B63),
    wip: Color(0xFF5A4815),
    done: Color(0xFF425626),
    borderHair: Color(0x3DF5EFE4),
  );

  static const _darkColors = FlexSchemeColor(
    primary: _darkCobalt,
    primaryContainer: BrandColors.cobalt,
    secondary: BrandColors.red,
    secondaryContainer: Color(0xFF5A211D),
    tertiary: BrandColors.butter,
    tertiaryContainer: BrandColors.clay,
    appBarColor: _darkPage,
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
      navigationRailTheme: _navigationRailTheme(base),
      tabBarTheme: _tabBarTheme(base),
      segmentedButtonTheme: _segmentedButtonTheme(base.colorScheme),
      timePickerTheme: _timePickerTheme(base),
    );
  }

  static ThemeData _buildDark() {
    final base0 = FlexThemeData.dark(
      colors: _darkColors,
      scaffoldBackground: _darkPage,
      surface: _darkSurface,
      subThemesData: const FlexSubThemesData(
        interactionEffects: true,
        tintedDisabledControls: true,
        cardRadius: 14,
        elevatedButtonRadius: 999,
        filledButtonRadius: 999,
        outlinedButtonRadius: 999,
        textButtonRadius: 999,
        inputDecoratorRadius: 14,
        inputDecoratorIsFilled: true,
        inputDecoratorFillColor: _darkSurface,
        inputDecoratorBorderType: FlexInputBorderType.outline,
        inputDecoratorBorderWidth: 1.5,
        inputDecoratorFocusedBorderWidth: 2,
        inputDecoratorBorderSchemeColor: SchemeColor.onSurface,
        inputDecoratorUnfocusedBorderIsColored: true,
        chipRadius: 999,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
      textTheme: _textTheme(ink: BrandColors.paper, ink2: _darkInk2),
    );
    final base = base0.copyWith(
      colorScheme: base0.colorScheme.copyWith(
        primary: _darkCobalt,
        onPrimary: _darkPage,
        primaryContainer: BrandColors.cobalt,
        onPrimaryContainer: BrandColors.paper3,
        secondary: BrandColors.red,
        onSecondary: _darkPage,
        surface: _darkSurface,
        onSurface: BrandColors.paper,
        onSurfaceVariant: _darkInk2,
        outline: _darkInk3,
        outlineVariant: _darkInk4,
        surfaceContainerLowest: _darkSunken,
        surfaceContainerLow: _darkPage,
        surfaceContainer: const Color(0xFF1D1A17),
        surfaceContainerHigh: _darkSurface,
        surfaceContainerHighest: _darkSurfaceHigh,
      ),
    );
    return base.copyWith(
      extensions: _extensions(base.colorScheme, dark: true),
      appBarTheme: _appBarTheme(base),
      navigationBarTheme: _navigationBarTheme(base),
      navigationRailTheme: _navigationRailTheme(base),
      tabBarTheme: _tabBarTheme(base),
      segmentedButtonTheme: _segmentedButtonTheme(base.colorScheme),
      timePickerTheme: _timePickerTheme(base),
    );
  }

  static TimePickerThemeData _timePickerTheme(ThemeData base) {
    final selectorStyle = base.textTheme.headlineLarge?.copyWith(
      fontSize: 42,
      height: 1,
    );
    return base.timePickerTheme.copyWith(
      hourMinuteTextStyle: selectorStyle,
      timeSelectorSeparatorTextStyle: WidgetStatePropertyAll(selectorStyle),
    );
  }

  static SegmentedButtonThemeData _segmentedButtonTheme(ColorScheme colors) {
    final swatches = colors.brightness == Brightness.dark
        ? _darkSwatches
        : const BrandSwatchTheme();
    return SegmentedButtonThemeData(
      style: ButtonStyle(
        backgroundColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.disabled)) return null;
          return states.contains(WidgetState.selected) ? swatches.moss : null;
        }),
        foregroundColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.disabled)) return null;
          return states.contains(WidgetState.selected)
              ? swatches.onMoss
              : colors.onSurface;
        }),
        overlayColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.disabled)) {
            return Colors.transparent;
          }
          final interactionColor = states.contains(WidgetState.selected)
              ? swatches.onMoss
              : swatches.moss;
          if (states.contains(WidgetState.pressed)) {
            return interactionColor.withValues(alpha: 0.12);
          }
          if (states.contains(WidgetState.focused)) {
            return interactionColor.withValues(alpha: 0.10);
          }
          if (states.contains(WidgetState.hovered)) {
            return interactionColor.withValues(alpha: 0.08);
          }
          return Colors.transparent;
        }),
      ),
    );
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
      backgroundColor: base.scaffoldBackgroundColor,
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
      backgroundColor: base.scaffoldBackgroundColor,
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

  /// NavigationRail: use the primary brand colour for the selected route,
  /// with sufficient contrast inside the selected indicator.
  static NavigationRailThemeData _navigationRailTheme(ThemeData base) {
    final colors = base.colorScheme;
    return NavigationRailThemeData(
      indicatorColor: colors.primary,
      selectedIconTheme: IconThemeData(color: colors.onPrimary),
      selectedLabelTextStyle: base.textTheme.labelLarge?.copyWith(
        color: colors.primary,
      ),
    );
  }

  static List<ThemeExtension<dynamic>> _extensions(
    ColorScheme scheme, {
    bool dark = false,
  }) {
    if (dark) {
      return <ThemeExtension<dynamic>>[
        const SpacingTheme(),
        const RadiusTheme(),
        const DurationTheme(),
        const BrandShadowTheme(
          drop: [BoxShadow(color: Color(0xFF090807), offset: Offset(6, 6))],
          dropSm: [
            BoxShadow(color: Color(0xFF090807), offset: Offset(3, 3)),
          ],
          dropLg: [
            BoxShadow(color: Color(0xFF090807), offset: Offset(10, 10)),
          ],
          paper1: [
            BoxShadow(color: Color(0x52090807), offset: Offset(0, 2)),
            BoxShadow(
              color: Color(0x3D090807),
              offset: Offset(0, 8),
              blurRadius: 20,
            ),
          ],
          paper2: [
            BoxShadow(
              color: Color(0x66090807),
              offset: Offset(0, 20),
              blurRadius: 40,
            ),
          ],
        ),
        _darkSwatches,
        const SemanticColorsTheme(
          error: BrandColors.red,
          warning: BrandColors.butter,
          success: Color(0xFF9EBC68),
          info: _darkCobalt,
          errorSurface: Color(0xFF491B18),
          warningSurface: Color(0xFF493D19),
          successSurface: Color(0xFF253318),
          infoSurface: Color(0xFF18285D),
        ),
      ];
    }
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
