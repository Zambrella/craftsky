import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final (:name, :loadTheme) in [
    (name: 'light', loadTheme: () => AppTheme.lightThemeData),
    (name: 'dark', loadTheme: () => AppTheme.darkThemeData),
  ]) {
    testWidgets('$name theme gives segmented buttons the moss color contract', (
      tester,
    ) async {
      final theme = loadTheme();
      final swatches = theme.extension<BrandSwatchTheme>()!;
      final style = theme.segmentedButtonTheme.style!;

      expect(
        style.backgroundColor?.resolve({WidgetState.selected}),
        swatches.moss,
      );
      expect(
        style.foregroundColor?.resolve({WidgetState.selected}),
        swatches.onMoss,
      );
      expect(style.foregroundColor?.resolve({}), theme.colorScheme.onSurface);
      expect(
        style.overlayColor?.resolve({WidgetState.pressed}),
        swatches.moss.withValues(alpha: 0.12),
      );
      expect(
        style.overlayColor?.resolve({
          WidgetState.selected,
          WidgetState.pressed,
        }),
        swatches.onMoss.withValues(alpha: 0.12),
      );
    });

    testWidgets('$name theme keeps time picker selector typography compact', (
      tester,
    ) async {
      final theme = loadTheme();
      final selectorStyle = theme.timePickerTheme.hourMinuteTextStyle!;
      final separatorStyle = theme
          .timePickerTheme
          .timeSelectorSeparatorTextStyle!
          .resolve({})!;

      expect(
        selectorStyle,
        theme.textTheme.headlineLarge?.copyWith(fontSize: 42, height: 1),
      );
      expect(separatorStyle, selectorStyle);
      expect(selectorStyle.fontSize, 42);
    });

    testWidgets('$name theme gives FABs the primary cutout treatment', (
      tester,
    ) async {
      final theme = loadTheme();
      final fab = theme.floatingActionButtonTheme;
      final shape = fab.shape! as StadiumBorder;

      expect(fab.backgroundColor, theme.colorScheme.primary);
      expect(fab.foregroundColor, theme.colorScheme.onPrimary);
      expect(fab.elevation, 3);
      expect(fab.hoverElevation, 4);
      expect(fab.highlightElevation, 0);
      expect(shape.side.color, theme.colorScheme.onSurface);
      expect(shape.side.width, 1.5);
      expect(fab.extendedTextStyle?.fontWeight, FontWeight.w800);
    });
  }

  test('dark theme uses the Midnight Paper palette', () {
    final theme = AppTheme.darkThemeData;
    final colors = theme.colorScheme;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    expect(theme.brightness, Brightness.dark);
    expect(theme.scaffoldBackgroundColor, const Color(0xFF171513));
    expect(colors.primary, const Color(0xFF7890FF));
    expect(colors.primaryContainer, const Color(0xFF1535D6));
    expect(colors.onSurface, const Color(0xFFF5EFE4));
    expect(colors.onSurfaceVariant, const Color(0xFFCFC6BB));
    expect(colors.outline, const Color(0xFF968D84));
    expect(swatches.paper, const Color(0xFF171513));
    expect(swatches.paper2, const Color(0xFF12100F));
    expect(swatches.paper3, const Color(0xFF24201D));
    expect(swatches.butter, const Color(0xFF5A4815));
    expect(swatches.clay, const Color(0xFF6A2F1A));
    expect(swatches.moss, const Color(0xFF425626));
    expect(swatches.sky, const Color(0xFF274964));
    expect(swatches.lilac, const Color(0xFF493B63));
    expect(swatches.borderHair, const Color(0x3DF5EFE4));
  });

  test('dark theme keeps CraftSky chrome', () {
    final theme = AppTheme.darkThemeData;

    expect(theme.appBarTheme.backgroundColor, const Color(0xFF171513));
    expect(theme.navigationBarTheme.backgroundColor, const Color(0xFF171513));
    expect(theme.cardTheme.shape, isA<RoundedRectangleBorder>());
  });
}
