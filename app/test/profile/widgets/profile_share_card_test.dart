import 'dart:ui' as ui;

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card_dialog.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card_image.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:qr_flutter/qr_flutter.dart';

Profile _profile({String? name = 'Rowan Ellis', List<String>? crafts}) =>
    Profile(
      did: 'did:plc:abcdefghijklmnopqrstuvwx',
      handle: 'rowan.makes.craftsky.social',
      displayName: name,
      crafts: crafts ?? ['sewing', 'knitting', 'quilting'],
    );

Widget _app(Widget child, {ThemeData? theme, double scale = 1}) =>
    ProviderScope(
      child: MaterialApp(
        theme: theme ?? AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: MediaQuery(
          data: MediaQueryData(
            disableAnimations: true,
            textScaler: TextScaler.linear(scale),
            boldText: scale > 1,
          ),
          child: Scaffold(body: Center(child: child)),
        ),
      ),
    );

Future<Uint8List> _capture(WidgetTester tester, GlobalKey key) async {
  await tester.runAsync(() async {
    final context = tester.element(find.byType(ProfileShareCard));
    await precacheImage(const AssetImage(shareCardLogo), context);
    await precacheImage(const AssetImage(shareCardTransparentLogo), context);
  });
  await tester.pumpAndSettle();
  final pending = captureProfileShareCard(key);
  await tester.pump();
  final bytes = await tester.runAsync(() => pending);
  return bytes!;
}

void main() {
  testWidgets('bundled thread shader compiles for the Flutter engine', (
    tester,
  ) async {
    final program = await tester.runAsync(
      () => ui.FragmentProgram.fromAsset('shaders/share_card_threads.frag'),
    );
    expect(program, isNotNull);
    program!.fragmentShader().dispose();
  });

  testWidgets(
    'centres identity, tightens spacing and puts the QR mark on white',
    (
      tester,
    ) async {
      await tester.pumpWidget(
        _app(
          SizedBox(width: 360, child: ProfileShareCard(profile: _profile())),
        ),
      );
      await tester.pumpAndSettle();
      final name = find.text('Rowan Ellis');
      final handle = find.text('@rowan.makes.craftsky.social');
      expect(name, findsOneWidget);
      expect(handle, findsOneWidget);
      expect(find.byType(CraftIcon), findsNWidgets(3));
      final centre = tester.getCenter(find.byType(ProfileShareCard)).dx;
      expect(tester.getCenter(name).dx, closeTo(centre, 0.1));
      expect(tester.getCenter(handle).dx, closeTo(centre, 0.1));
      expect(tester.getCenter(find.byType(ClipOval)).dx, closeTo(centre, 0.1));
      expect(
        tester.getTopLeft(handle).dy - tester.getBottomLeft(name).dy,
        lessThan(4),
      );
      final qr = tester.widget<QrImageView>(find.byType(QrImageView));
      expect(qr.padding, const EdgeInsets.all(9));
      expect(qr.errorCorrectionLevel, QrErrorCorrectLevel.H);
      final mark = tester.widget<ColoredBox>(
        find.byKey(const Key('share-card-qr-logo')),
      );
      expect(mark.color, Colors.white);
      expect(
        find.byWidgetPredicate(
          (w) =>
              w is Image &&
              w.image == const AssetImage(shareCardTransparentLogo),
        ),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'only Share remains and the overlay still honours accessibility',
    (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 568);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        _app(
          ProfileShareCardDialog(
            profile: _profile(
              name: 'Rowan Alexandra Ellis and the textile studio',
              crafts: [
                'sewing',
                'knitting',
                'quilting',
                'crochet',
                'embroidery',
              ],
            ),
          ),
          scale: 2,
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('share-card-download')), findsNothing);
      expect(find.text('Download'), findsNothing);
      expect(
        MediaQuery.textScalerOf(tester.element(find.text('Your share card')))
            .scale(14),
        28,
      );
      expect(
        MediaQuery.textScalerOf(tester.element(find.text('Embroidery')))
            .scale(14),
        14,
      );
      await tester.ensureVisible(find.byKey(const Key('share-card-share')));
      await tester.tap(find.byKey(const Key('share-card-share')));
      await tester.pumpAndSettle();
      expect(find.text('Share coming soon.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'same 4:5 artwork and 1200x1500 PNG across themes, scales and widths',
    (
      tester,
    ) async {
      Uint8List? baseline;
      for (final width in [272.0, 560.0]) {
        final key = GlobalKey();
        final large = width == 560;
        await tester.pumpWidget(
          _app(
            SingleChildScrollView(
              child: SizedBox(
                width: width,
                child: ProfileShareCard(profile: _profile(), artworkKey: key),
              ),
            ),
            scale: large ? 3 : 1,
            theme: large
                ? AppTheme.darkThemeData.copyWith(
                    textTheme: AppTheme.darkThemeData.textTheme.apply(
                      fontFamily: 'monospace',
                      fontSizeFactor: 2,
                      bodyColor: Colors.red,
                    ),
                  )
                : AppTheme.lightThemeData,
          ),
        );
        await tester.pumpAndSettle();
        final size = tester.getSize(find.byType(ProfileShareCard));
        expect(size.width / size.height, closeTo(4 / 5, 0.0001));
        expect(tester.getSize(find.byKey(key)), shareCardCanvasSize);
        final png = await _capture(tester, key);
        final header = ByteData.sublistView(png);
        expect(header.getUint32(16), 1200);
        expect(header.getUint32(20), 1500);
        if (baseline == null) {
          baseline = png;
        } else {
          expect(png, baseline, reason: 'The artwork must be pixel-identical.');
        }
        expect(tester.takeException(), isNull);
      }
    },
  );

  testWidgets('long identity and unknown crafts stay inside the fixed canvas', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        SizedBox(
          width: 272,
          child: ProfileShareCard(
            profile: _profile(
              name: List.filled(8, 'Rowan Alexandra Ellis').join(' '),
              crafts: [
                'sewing',
                'knitting',
                'quilting',
                'crochet',
                'embroidery',
                'social.craftsky.actor.defs#weaving',
                'a-future-craft',
              ],
            ),
          ),
        ),
        scale: 3,
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('a-future-craft'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('missing name and crafts retain a useful card', (tester) async {
    await tester.pumpWidget(
      _app(
        SizedBox(
          width: 320,
          child: ProfileShareCard(
            profile: _profile(name: '  ', crafts: []),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('rowan.makes.craftsky.social'), findsOneWidget);
    expect(find.text('R'), findsOneWidget);
    expect(find.byType(CraftIcon), findsNothing);
    expect(find.byType(QrImageView), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
