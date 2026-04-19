import 'dart:async';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/router/home_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  group('App initialisation', () {
    late SharedPreferences prefs;
    late List<LogRecord> records;
    late StreamSubscription<LogRecord> logSub;

    setUp(() async {
      TestWidgetsFlutterBinding.ensureInitialized();
      SharedPreferences.setMockInitialValues({});
      prefs = await SharedPreferences.getInstance();
      records = <LogRecord>[];
      logSub = Logger.root.onRecord.listen(records.add);
    });

    tearDown(() async {
      await logSub.cancel();
    });

    AppDependencies stubDeps() => AppDependencies(
      packageInfo: PackageInfo(
        appName: 'craftsky_app',
        packageName: 'social.craftsky.app',
        version: '1.0.0',
        buildNumber: '1',
      ),
      deviceInfo: CraftskyDeviceInfo(
        platform: 'Test',
        deviceId: 'test',
        model: 'test',
        brand: 'test',
        osVersion: '0',
      ),
      sharedPreferences: prefs,
      appVersion: Version.parse('1.0.0'),
    );

    // Tests go here.
  });
}
