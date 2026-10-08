import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final variant in ['ordinary', 'locked', 'disabled', 'destructive']) {
    testWidgets('$variant settings icon follows theme changes', (tester) async {
      final mode = ValueNotifier(ThemeMode.light);
      addTearDown(mode.dispose);
      await tester.pumpWidget(
        ValueListenableBuilder<ThemeMode>(
          valueListenable: mode,
          builder: (_, themeMode, _) => MaterialApp(
            theme: AppTheme.lightThemeData,
            darkTheme: AppTheme.darkThemeData,
            themeMode: themeMode,
            home: Scaffold(
              body: SettingsRowTile(
                descriptor: SettingsRowDescriptor(
                  id: SettingsRowId.account,
                  kind: variant == 'destructive'
                      ? SettingsRowKind.destructiveAction
                      : SettingsRowKind.disclosure,
                ),
                label: 'Account',
                leading: CraftskyIcons.accountSettings,
                locked: variant == 'locked',
                onTap: variant == 'disabled' ? null : () {},
              ),
            ),
          ),
        ),
      );

      for (final themeMode in [
        ThemeMode.light,
        ThemeMode.dark,
        ThemeMode.light,
      ]) {
        mode.value = themeMode;
        await tester.pumpAndSettle();
        final theme = Theme.of(tester.element(find.byType(SettingsRowTile)));
        final leadingColor = _renderedIconColor(
          tester,
          CraftskyIcons.accountSettings,
        );
        if (variant == 'destructive') {
          expect(leadingColor, theme.colorScheme.error);
        } else if (variant == 'locked' && !subscriptionsEnabled) {
          expect(
            leadingColor,
            theme.colorScheme.onSurface.withValues(alpha: 0.72),
          );
        } else {
          expect(
            leadingColor,
            _renderedIconColor(tester, CraftskyIconsBold.next),
            reason: 'Leading icons should inherit the themed ListTile colour',
          );
        }
      }
    });
  }
}

Color? _renderedIconColor(WidgetTester tester, IconData icon) {
  final glyph = tester.widget<RichText>(
    find.descendant(of: find.byIcon(icon), matching: find.byType(RichText)),
  );
  return glyph.text.style?.color;
}
