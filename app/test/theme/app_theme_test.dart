import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final (:name, :loadTheme) in [
    (name: 'light', loadTheme: () => AppTheme.lightThemeData),
    (name: 'dark', loadTheme: () => AppTheme.darkThemeData),
  ]) {
    testWidgets(
      '$name theme gives segmented buttons the moss color contract',
      (tester) async {
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
        expect(
          style.foregroundColor?.resolve({}),
          theme.colorScheme.onSurface,
        );
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
      },
    );

    testWidgets(
      '$name theme keeps time picker selector typography compact',
      (tester) async {
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
      },
    );

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
}
