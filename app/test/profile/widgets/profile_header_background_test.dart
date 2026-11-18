import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test('every craft background key maps to a bundled craft icon', () {
    expect(
      profileCraftBackgrounds.keys,
      profileBackgroundCatalogue.skip(1).take(5),
    );
    for (final craft in profileCraftBackgrounds.values) {
      expect(CraftIcon.assetPathFor(craft), isNotNull);
    }
  });

  testWidgets(
    'retired textures render the selected base without asset requests',
    (
      tester,
    ) async {
      for (final background in [
        'bayerdark',
        'cubedark',
        'dotcrossdark',
        'scallopdark',
        'skewdark',
        'x2',
      ]) {
        await tester.pumpWidget(
          MaterialApp(
            theme: AppTheme.lightThemeData,
            home: SizedBox(
              width: 180,
              height: 80,
              child: ProfileHeaderBackground(
                customisation: ProfileCustomisation(
                  colour: 'seafoam',
                  background: background,
                ),
              ),
            ),
          ),
        );
        final box = tester.widget<ColoredBox>(
          find.byKey(const Key('profile-header-background')),
        );
        expect(box.color, const Color(0xFF246B60));
        expect(find.byType(Image), findsNothing);
        expect(
          find.byKey(const Key('profile-header-background-texture')),
          findsNothing,
        );
        expect(tester.takeException(), isNull);
      }
    },
  );

  testWidgets('none paints only the selected base colour', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        home: const SizedBox(
          width: 180,
          height: 80,
          child: ProfileHeaderBackground(
            customisation: ProfileCustomisation(colour: 'rose'),
          ),
        ),
      ),
    );

    final box = tester.widget<ColoredBox>(
      find.byKey(const Key('profile-header-background')),
    );
    expect(box.color, const Color(0xFFD61535));
    expect(find.byType(Image), findsNothing);
  });

  testWidgets('craft background repeats tinted icons within the header', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        home: const SizedBox(
          width: 180,
          height: 80,
          child: ProfileHeaderBackground(
            customisation: ProfileCustomisation(
              colour: 'seafoam',
              background: 'craft-crochet',
            ),
          ),
        ),
      ),
    );

    final icons = tester.widgetList<CraftIcon>(find.byType(CraftIcon)).toList();
    expect(icons.length, greaterThan(1));
    expect(icons.every((icon) => icon.craft == 'crochet'), isTrue);
    expect(icons.every((icon) => icon.color?.a == 0.18), isTrue);
    expect(
      find.byKey(const Key('profile-header-background-texture')),
      findsOneWidget,
    );
  });
}
