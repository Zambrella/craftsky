import 'dart:async';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/sharing/craft_foil_surface.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card_dialog.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card_image.dart';
import 'package:craftsky_app/profile/sharing/share_card_motion.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as raster;
import 'package:material_ui/material_ui.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../../test_support/serialized_sentry_transport.dart';

class _Paths extends PathProviderPlatform {
  new(this.path);
  final String path;
  @override
  Future<String> getTemporaryPath() async => path;
}

Future<void> _advanceUntil(
  WidgetTester tester,
  bool Function() complete,
) async {
  for (var attempt = 0; attempt < 200 && !complete(); attempt++) {
    await tester.pump();
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 10)),
    );
  }
  expect(complete(), isTrue);
}

Profile _profile({String? name = 'Rowan Ellis', List<String>? crafts}) =>
    Profile(
      did: 'did:plc:abcdefghijklmnopqrstuvwx',
      handle: 'rowan.makes.craftsky.social',
      displayName: name,
      crafts: crafts ?? ['sewing', 'knitting', 'quilting'],
    );

Widget _app(
  Widget child, {
  ThemeData? theme,
  double scale = 1,
  ErrorReporter? reporter,
}) => ProviderScope(
  overrides: [
    if (reporter != null) errorReporterProvider.overrideWithValue(reporter),
  ],
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
  Uint8List? bytes;
  Object? failure;
  final pending = captureProfileShareCard(key).then<void>(
    (value) {
      bytes = value;
    },
    onError: (Object error) {
      failure = error;
    },
  );
  // Image encoding runs on the engine while asset and frame completion use
  // the widget test's fake clock. Advance both until the capture resolves.
  for (
    var attempt = 0;
    attempt < 100 && bytes == null && failure == null;
    attempt++
  ) {
    await tester.pump();
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 10)),
    );
  }
  expect(failure, isNull);
  expect(bytes, isNotNull);
  await pending;
  return bytes!;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUpAll(CraftFoilSurface.prepareAssets);

  testWidgets(
    'Share exports PNG, prevents duplicate taps '
    'and permits retry after cancellation or failure',
    (tester) async {
      final transport = SerializedTransport();
      await tester.runAsync(
        () => Sentry.init((options) {
          options
            ..dsn = 'https://public@example.invalid/1'
            ..transport = transport;
          configureDiagnosticOptions(options, enableLogs: false);
        }),
      );
      addTearDown(Sentry.close);
      final directory = Directory.systemTemp.createTempSync(
        'card-dialog-test-',
      );
      final original = PathProviderPlatform.instance;
      PathProviderPlatform.instance = _Paths(directory.path);
      addTearDown(() {
        PathProviderPlatform.instance = original;
        directory.deleteSync(recursive: true);
      });
      const channel = MethodChannel('dev.fluttercommunity.plus/share');
      var calls = 0;
      var response = Completer<String>();
      File? image;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            calls++;
            final args = call.arguments as Map;
            image = File((args['paths'] as List).single as String);
            final decoded = raster.decodePng(image!.readAsBytesSync())!;
            expect(decoded.width, 1200);
            expect(decoded.height, 1500);
            expect(args['mimeTypes'], ['image/png']);
            expect(args['originWidth'], greaterThan(0));
            expect(args['originHeight'], greaterThan(0));
            return await response.future;
          });
      addTearDown(
        () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(channel, null),
      );
      await tester.pumpWidget(
        _app(
          ProfileShareCardDialog(profile: _profile()),
          reporter: const SentryErrorReporter(),
        ),
      );
      await tester.runAsync(() async {
        final context = tester.element(find.byType(ProfileShareCard));
        await precacheImage(const AssetImage(shareCardLogo), context);
        await precacheImage(
          const AssetImage(shareCardTransparentLogo),
          context,
        );
      });
      await tester.pumpAndSettle();
      final button = find.byKey(const Key('share-card-share'));
      await tester.ensureVisible(button);
      final idleSize = tester.getSize(button);
      final idleContent = find.descendant(
        of: button,
        matching: find.byType(Row),
      );
      expect(
        tester.getCenter(idleContent).dx,
        closeTo(tester.getCenter(button).dx, 1),
      );
      await tester.tap(button);
      await _advanceUntil(tester, () => calls == 1);
      expect(tester.getSize(button), idleSize);
      expect(find.text('Loading...'), findsOneWidget);
      expect(find.text('Share'), findsNothing);
      expect(tester.widget<ChunkyButton>(button).onPressed, isNull);
      expect(
        tester.widget<ChunkyButton>(button).disabledBackgroundColor,
        BrandColors.paper,
      );
      expect(
        find.descendant(
          of: button,
          matching: find.byType(StitchProgressIndicator),
        ),
        findsNothing,
      );
      expect(image!.existsSync(), isTrue);
      response.complete('');
      await _advanceUntil(
        tester,
        () => tester.widget<ChunkyButton>(button).onPressed != null,
      );
      expect(image!.existsSync(), isFalse);
      expect(
        find.text('Couldn’t share your card. Please try again.'),
        findsNothing,
      );
      response = Completer<String>();
      await tester.tap(button);
      await _advanceUntil(tester, () => calls == 2);
      response.completeError(
        PlatformException(code: 'unavailable', message: 'PRIVATE_CARD_PAYLOAD'),
      );
      await _advanceUntil(
        tester,
        () => tester.widget<ChunkyButton>(button).onPressed != null,
      );
      expect(
        find.text('Couldn’t share your card. Please try again.'),
        findsOneWidget,
      );
      expect(image!.existsSync(), isFalse);
      expect(tester.takeException(), isNull);
      final serialized = transport.payloads.join();
      expect(serialized, contains('profile.share_card.share_failed'));
      expect(serialized, contains('PlatformException'));
      expect(serialized, isNot(contains('PRIVATE_CARD_PAYLOAD')));
    },
  );

  testWidgets('bundled edge emboss shader compiles for the Flutter engine', (
    tester,
  ) async {
    final program = await tester.runAsync(
      () => ui.FragmentProgram.fromAsset('shaders/share_card_emboss.frag'),
    );
    expect(program, isNotNull);
    program!.fragmentShader().dispose();
  });

  testWidgets('craft foil catches tilt light while the QR stays unchanged', (
    tester,
  ) async {
    final frames = ValueNotifier(const ShareCardFrame());
    addTearDown(frames.dispose);
    final key = GlobalKey();
    await tester.pumpWidget(
      _app(
        SizedBox(
          width: 360,
          child: ProfileShareCard(
            profile: _profile(),
            frames: frames,
            artworkKey: key,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final still = raster.decodePng(await _capture(tester, key))!;
    frames.value = const ShareCardFrame(tilt: Offset(0.04, -0.035));
    await tester.pumpAndSettle();
    final tilted = raster.decodePng(await _capture(tester, key))!;
    var visibleGlyphPixels = 0;
    var litGlyphPixels = 0;
    // A strip inside the left border, outside all identity/QR content.
    for (var y = 100; y < 1350; y++) {
      for (var x = 30; x < 80; x++) {
        final a = still.getPixel(x, y);
        final b = tilted.getPixel(x, y);
        if (a.r != 21 || a.g != 53 || a.b != 214) visibleGlyphPixels++;
        if (a.r != b.r || a.g != b.g || a.b != b.b) litGlyphPixels++;
      }
    }
    expect(visibleGlyphPixels, greaterThan(100));
    expect(litGlyphPixels, greaterThan(100));
    for (var y = 840; y < 1260; y++) {
      for (var x = 390; x < 810; x++) {
        final a = still.getPixel(x, y);
        final b = tilted.getPixel(x, y);
        expect([a.r, a.g, a.b, a.a], [b.r, b.g, b.b, b.a]);
      }
    }
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
      expect(find.text('Share'), findsOneWidget);
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
