import 'dart:async';

import 'package:dart_mappable/dart_mappable.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/date_symbol_data_local.dart';
import 'package:logging/logging.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';

part 'app_dependencies.mapper.dart';
part 'app_dependencies.g.dart';

final _log = Logger('AppDependencies');

@MappableClass()
class CraftskyDeviceInfo with CraftskyDeviceInfoMappable {
  CraftskyDeviceInfo({
    required this.platform,
    required this.deviceId,
    required this.model,
    required this.brand,
    required this.osVersion,
  });

  /// 'Android' | 'iOS' | 'Web'
  final String platform;
  final String deviceId;
  final String model;
  final String brand;
  final String osVersion;
}

@MappableClass()
class AppDependencies with AppDependenciesMappable {
  AppDependencies({
    required this.packageInfo,
    required this.deviceInfo,
    required this.sharedPreferences,
    required this.appVersion,
  });

  final PackageInfo packageInfo;
  final CraftskyDeviceInfo deviceInfo;
  final SharedPreferences sharedPreferences;
  final Version appVersion;
}

@Riverpod(keepAlive: true)
Future<AppDependencies> appDependencies(Ref ref) async {
  _log.info('initializing app dependencies');

  final packageInfo = await PackageInfo.fromPlatform();
  final deviceInfo = await _resolveDeviceInfo();
  final prefs = await SharedPreferences.getInstance();
  final appVersion = Version.parse(packageInfo.version);

  final deviceLocale = PlatformDispatcher.instance.locale.toString();
  await initializeDateFormatting(deviceLocale);
  _log
    ..fine('initialized date formatting for $deviceLocale')
    ..info(
      'app dependencies ready '
      '(version=$appVersion, platform=${deviceInfo.platform})',
    );

  return AppDependencies(
    packageInfo: packageInfo,
    deviceInfo: deviceInfo,
    sharedPreferences: prefs,
    appVersion: appVersion,
  );
}

Future<CraftskyDeviceInfo> _resolveDeviceInfo() async {
  final plugin = DeviceInfoPlugin();

  if (kIsWeb) {
    final web = await plugin.webBrowserInfo;
    return CraftskyDeviceInfo(
      platform: 'Web',
      deviceId: '',
      brand: web.browserName.name,
      model: web.userAgent ?? '',
      osVersion: web.platform ?? '',
    );
  }

  switch (defaultTargetPlatform) {
    case TargetPlatform.android:
      final android = await plugin.androidInfo;
      return CraftskyDeviceInfo(
        platform: 'Android',
        deviceId: android.id,
        brand: android.brand,
        model: android.model,
        osVersion: android.version.release,
      );
    case TargetPlatform.iOS:
      final ios = await plugin.iosInfo;
      return CraftskyDeviceInfo(
        platform: 'iOS',
        deviceId: ios.identifierForVendor ?? '',
        brand: ios.utsname.machine,
        model: ios.utsname.machine,
        osVersion: ios.systemVersion,
      );
    // Desktop and fuchsia are intentionally out of scope for v1 — spec
    // Section 2 restricts the scaffold to Android, iOS, and Web.
    case TargetPlatform.fuchsia:
    case TargetPlatform.linux:
    case TargetPlatform.macOS:
    case TargetPlatform.windows:
      throw UnsupportedError(
        'Platform not supported by this scaffold: $defaultTargetPlatform',
      );
  }
}

// Accessor providers return the resolved dependency synchronously. They use
// `requireValue`, which throws if called before `appDependenciesProvider`
// reaches `AsyncData`. The `App` widget only mounts the subtree that
// consumes these accessors (via `_ReadyApp`) once that provider resolves,
// so the exception path is unreachable in the normal flow.

@Riverpod(keepAlive: true)
SharedPreferences sharedPreferences(Ref ref) => ref.watch(
  appDependenciesProvider.select((a) => a.requireValue.sharedPreferences),
);

@Riverpod(keepAlive: true)
PackageInfo packageInfo(Ref ref) => ref.watch(
  appDependenciesProvider.select((a) => a.requireValue.packageInfo),
);

@Riverpod(keepAlive: true)
CraftskyDeviceInfo deviceInfo(Ref ref) =>
    ref.watch(appDependenciesProvider.select((a) => a.requireValue.deviceInfo));

@Riverpod(keepAlive: true)
Version appVersion(Ref ref) =>
    ref.watch(appDependenciesProvider.select((a) => a.requireValue.appVersion));
