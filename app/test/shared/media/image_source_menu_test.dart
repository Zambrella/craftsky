import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/media/image_source_menu.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('camera source support includes mobile and web but not desktop', () {
    expect(
      supportsCameraImageSource(
        isWeb: false,
        platform: TargetPlatform.android,
      ),
      isTrue,
    );
    expect(
      supportsCameraImageSource(isWeb: false, platform: TargetPlatform.iOS),
      isTrue,
    );
    expect(
      supportsCameraImageSource(isWeb: true, platform: TargetPlatform.linux),
      isTrue,
    );
    expect(
      supportsCameraImageSource(isWeb: false, platform: TargetPlatform.macOS),
      isFalse,
    );
    expect(
      supportsCameraImageSource(
        isWeb: false,
        platform: TargetPlatform.windows,
      ),
      isFalse,
    );
    expect(
      supportsCameraImageSource(isWeb: false, platform: TargetPlatform.linux),
      isFalse,
    );
  });

  testWidgets('offers camera and gallery and invokes the camera action', (
    tester,
  ) async {
    var cameraCalls = 0;
    var galleryCalls = 0;
    await tester.pumpWidget(
      _app(
        cameraSupported: true,
        onCamera: () async => cameraCalls++,
        onGallery: () async => galleryCalls++,
      ),
    );

    await tester.tap(find.byKey(const Key('open-source-menu')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('test-take-photo')), findsOneWidget);
    expect(find.byKey(const Key('test-choose-photos')), findsOneWidget);
    expect(find.text('Take a photo'), findsOneWidget);

    await tester.tap(find.byKey(const Key('test-take-photo')));
    await tester.pumpAndSettle();

    expect(cameraCalls, 1);
    expect(galleryCalls, 0);
  });

  testWidgets('opens gallery directly when camera is not supported', (
    tester,
  ) async {
    var galleryCalls = 0;
    await tester.pumpWidget(
      _app(
        cameraSupported: false,
        onCamera: () async {},
        onGallery: () async => galleryCalls++,
      ),
    );

    await tester.tap(find.byKey(const Key('open-source-menu')));
    await tester.pumpAndSettle();

    expect(galleryCalls, 1);
    expect(find.byKey(const Key('test-take-photo')), findsNothing);
    expect(find.byKey(const Key('test-choose-photos')), findsNothing);
  });
}

Widget _app({
  required bool cameraSupported,
  required Future<void> Function() onCamera,
  required Future<void> Function() onGallery,
}) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(
    body: Builder(
      builder: (context) => TextButton(
        key: const Key('open-source-menu'),
        onPressed: () => showImageSourceMenu(
          context,
          keyPrefix: 'test',
          cameraSupported: cameraSupported,
          onChoosePhotos: onGallery,
          onTakePhoto: onCamera,
        ),
        child: const Text('Open'),
      ),
    ),
  ),
);
