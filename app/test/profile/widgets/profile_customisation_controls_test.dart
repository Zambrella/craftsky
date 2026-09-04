import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Color _paintedSurface(WidgetTester tester, Color expected) {
  final colours = tester
      .widgetList<DecoratedBox>(find.byType(DecoratedBox))
      .map((box) => box.decoration)
      .whereType<ShapeDecoration>()
      .map((decoration) => decoration.color)
      .whereType<Color>();
  return colours.firstWhere((colour) => colour == expected);
}

Widget _subject({
  required FocusNode focusNode,
  String colour = 'orchid',
  ThemeMode themeMode = ThemeMode.light,
}) {
  return MaterialApp(
    theme: AppTheme.lightThemeData,
    darkTheme: AppTheme.darkThemeData,
    themeMode: themeMode,
    home: Scaffold(
      body: ProfileCustomisationTheme(
        customisation: ProfileCustomisation(colour: colour),
        child: Row(
          children: [
            ChunkyButton(
              focusNode: focusNode,
              onPressed: () {},
              child: const Text('Primary'),
            ),
            ChunkyIconButton(
              onPressed: () {},
              icon: CraftskyIcons.settings,
              tooltip: 'Secondary',
            ),
          ],
        ),
      ),
    ),
  );
}

void main() {
  testWidgets(
    'IR-001 profile bundle governs real Chunky control interaction states',
    (tester) async {
      final focusNode = FocusNode();
      addTearDown(focusNode.dispose);
      final bundle = profileColourBundles['orchid']!;
      final base = profileColour(bundle.base);
      final foreground = profileColour(bundle.foreground);
      final hover = profileColour(bundle.hover);
      final pressed = profileColour(bundle.pressed);
      final soft = profileColour(bundle.softContainer);

      await tester.pumpWidget(_subject(focusNode: focusNode));

      expect(_paintedSurface(tester, base), base);
      expect(_paintedSurface(tester, soft), soft);
      final primary = tester.widget<ChunkyButton>(
        find.widgetWithText(ChunkyButton, 'Primary'),
      );
      final secondary = tester.widget<ChunkyButton>(
        find.descendant(
          of: find.byType(ChunkyIconButton),
          matching: find.byType(ChunkyButton),
        ),
      );
      expect(
        primary
            .defaultStyleOf(tester.element(find.text('Primary')))
            .foregroundColor
            ?.resolve({}),
        foreground,
      );
      expect(
        primary
            .defaultStyleOf(tester.element(find.text('Primary')))
            .overlayColor
            ?.resolve({WidgetState.hovered}),
        Colors.transparent,
      );
      expect(
        secondary
            .defaultStyleOf(
              tester.element(find.byIcon(CraftskyIcons.settings)),
            )
            .foregroundColor
            ?.resolve({}),
        profileColour('#111318'),
      );

      final mouse = await tester.createGesture(kind: PointerDeviceKind.mouse);
      addTearDown(mouse.removePointer);
      await mouse.addPointer(location: Offset.zero);
      await mouse.moveTo(tester.getCenter(find.text('Primary')));
      await tester.pumpAndSettle();
      expect(_paintedSurface(tester, hover), hover);

      await mouse.moveTo(Offset.zero);
      focusNode.requestFocus();
      await tester.pumpAndSettle();
      expect(_paintedSurface(tester, hover), hover);

      final press = await tester.startGesture(
        tester.getCenter(find.text('Primary')),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 120));
      expect(_paintedSurface(tester, pressed), pressed);
      await press.up();
      await tester.pumpAndSettle();
    },
  );

  testWidgets('dark Ink keeps identity black and uses parchment interactions', (
    tester,
  ) async {
    final focusNode = FocusNode();
    addTearDown(focusNode.dispose);
    final bundle = profileColourBundles['ink']!;

    await tester.pumpWidget(
      _subject(
        focusNode: focusNode,
        colour: 'ink',
        themeMode: ThemeMode.dark,
      ),
    );

    final context = tester.element(find.text('Primary'));
    final theme = Theme.of(context);
    final primary = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Primary'),
    );

    expect(theme.colorScheme.primary, profileColour(bundle.darkAccent));
    expect(theme.colorScheme.onPrimary, profileColour(bundle.darkForeground));
    expect(
      primary.defaultStyleOf(context).foregroundColor?.resolve({}),
      profileColour(bundle.darkForeground),
    );
    expect(profileColour(bundle.base), BrandColors.ink);
  });
}
