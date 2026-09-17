import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'app_harness.dart';

Widget craftskyTestWidget({
  required Widget child,
  List<dynamic> overrides = const [],
  Locale locale = const Locale('en'),
  ThemeData? theme,
  AppMessenger? messenger,
  EdgeInsetsGeometry padding = const EdgeInsets.all(24),
}) {
  Widget app = MaterialApp(
    theme: theme ?? AppTheme.lightThemeData,
    locale: locale,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(
      body: Padding(padding: padding, child: child),
    ),
  );
  if (messenger != null) {
    app = MessengerScope(messenger: messenger, child: app);
  }
  return appHarness(overrides: overrides, child: app);
}

Future<void> pumpCraftskyWidget(
  WidgetTester tester,
  Widget child, {
  List<dynamic> overrides = const [],
  Locale locale = const Locale('en'),
  ThemeData? theme,
  Size? surfaceSize,
  AppMessenger? messenger,
  EdgeInsetsGeometry padding = const EdgeInsets.all(24),
}) async {
  if (surfaceSize != null) {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = surfaceSize;
    addTearDown(() {
      tester.view.resetDevicePixelRatio();
      tester.view.resetPhysicalSize();
    });
  }
  await tester.pumpWidget(
    craftskyTestWidget(
      overrides: overrides,
      locale: locale,
      theme: theme,
      messenger: messenger,
      padding: padding,
      child: child,
    ),
  );
}
