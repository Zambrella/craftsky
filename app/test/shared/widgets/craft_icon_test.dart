import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test('resolves profile IDs and project lexicon tokens', () {
    for (final MapEntry(:key, :value) in {
      'knitting': 'knitting',
      'crochet': 'crochet',
      ProjectOptionCatalogs.sewingCraftToken: 'sewing',
      ProjectOptionCatalogs.embroideryCraftToken: 'embroidery',
      ProjectOptionCatalogs.quiltingCraftToken: 'quilting',
      ProjectOptionCatalogs.spinningCraftToken: 'spinning',
      ProjectOptionCatalogs.dyeingCraftToken: 'dyeing',
      ProjectOptionCatalogs.punchNeedleCraftToken: 'punch-needle',
      'punch-needle': 'punch-needle',
      ProjectOptionCatalogs.macrameCraftToken: 'macrame',
      ProjectOptionCatalogs.weavingCraftToken: 'weaving',
      ProjectOptionCatalogs.feltingCraftToken: 'felting',
    }.entries) {
      expect(
        CraftIcon.assetPathFor(key),
        'assets/design/icons/$value.svg',
      );
    }
    expect(CraftIcon.assetPathFor('pottery'), isNull);
  });

  testWidgets('renders an asset-backed icon beside its label', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: CraftIconLabel(
            craft: ProjectOptionCatalogs.punchNeedleCraftToken,
            label: 'Punch Needle',
          ),
        ),
      ),
    );

    expect(
      find.byKey(
        const ValueKey('craft-icon-assets/design/icons/punch-needle.svg'),
      ),
      findsOneWidget,
    );
    expect(find.text('Punch Needle'), findsOneWidget);
  });

  testWidgets('keeps unsupported crafts text-only', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: CraftIconLabel(craft: 'pottery', label: 'Pottery'),
        ),
      ),
    );

    expect(find.byType(SvgPicture), findsNothing);
    expect(find.text('Pottery'), findsOneWidget);
  });

  testWidgets('flexible labels wrap without overflowing beside an icon', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 100,
            child: CraftIconLabel(
              craft: 'embroidery',
              label: 'Embroidery',
              flexibleLabel: true,
            ),
          ),
        ),
      ),
    );

    expect(tester.takeException(), isNull);
  });
}
