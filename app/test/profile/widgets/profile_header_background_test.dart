import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('every non-none background key maps to one bundled local PNG', () {
    expect(profileBackgroundAssets.keys, profileBackgroundCatalogue.skip(1));
    for (final asset in profileBackgroundAssets.values) {
      expect(asset, startsWith('assets/profile_backgrounds/'));
      expect(asset, endsWith('.png'));
      expect(asset, isNot(contains('://')));
    }
  });

  testWidgets('texture is clipped, tiled, and tinted by the selected bundle', (
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
              colour: 'teal',
              background: 'x2',
            ),
          ),
        ),
      ),
    );

    expect(find.byType(ClipRect), findsOneWidget);
    final image = tester.widget<Image>(
      find.byKey(const Key('profile-header-background-texture')),
    );
    expect(image.image, isA<AssetImage>());
    expect((image.image as AssetImage).assetName, endsWith('/x2.png'));
    expect(image.repeat, ImageRepeat.repeat);
    expect(image.color?.a, moreOrLessEquals(0.18));
    expect(image.color?.r, moreOrLessEquals(0x11 / 255));
    expect(image.color?.g, moreOrLessEquals(0x13 / 255));
    expect(image.color?.b, moreOrLessEquals(0x18 / 255));
    expect(image.colorBlendMode, BlendMode.srcIn);
  });

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
}
