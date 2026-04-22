import 'dart:async';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
// flutter_web_plugins ships with Flutter but is not declared in pubspec;
// it's the only place usePathUrlStrategy() lives.
// ignore: depend_on_referenced_packages
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:intl/intl.dart';
import 'package:logging/logging.dart';
import 'package:shared_preferences/shared_preferences.dart';

final _log = Logger('bootstrap');

/// Runs platform / Flutter init before `runApp`.
///
/// IMPORTANT: must never throw in production. Anything that *can* fail
/// belongs in `appDependenciesProvider`, which has loading/error UI.
Future<void> bootstrap(WidgetsBinding widgetsBinding) async {
  _log.fine('bootstrap starting');

  // Web: path URL strategy (no `#` in URLs).
  usePathUrlStrategy();

  if (kIsWeb) {
    _log.fine('web detected, skipping native init');
    runApp(
      const ProviderScope(child: App()),
    );
    return;
  }

  // Default locale for intl date/number formatting.
  final localeName = PlatformDispatcher.instance.locale.toString();
  Intl.defaultLocale = localeName;
  _log.fine('Intl.defaultLocale=$localeName');

  // dart_mappable mapper init — empty for now, grows as models are added.
  initializeMappers();

  // Namespace all shared_preferences keys.
  SharedPreferences.setPrefix('craftsky.');

  if (defaultTargetPlatform == TargetPlatform.android) {
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    SystemChrome.setSystemUIOverlayStyle(
      const SystemUiOverlayStyle(
        systemNavigationBarColor: Colors.transparent,
        systemNavigationBarDividerColor: Colors.transparent,
        systemNavigationBarContrastEnforced: true,
      ),
    );
  }

  // Fail fast if --dart-define=CRAFTSKY_API_BASE_URL is missing in a
  // release build. Building the provider throws StateError before any
  // networking is attempted. We dispose the throwaway container so the
  // check stays cheap; the real app creates its own via ProviderScope.
  final probe = ProviderContainer();
  try {
    probe.read(dioProvider);
  } finally {
    probe.dispose();
  }

  _log.fine('bootstrap complete');

  runApp(
    const ProviderScope(child: App()),
  );
}

/// Initialize all `dart_mappable` mappers here as models are added.
void initializeMappers() {
  AppDependenciesMapper.ensureInitialized();
  CraftskyDeviceInfoMapper.ensureInitialized();
  LoginResponseMapper.ensureInitialized();
  WhoAmIMapper.ensureInitialized();
  StoredSessionMapper.ensureInitialized();
  PendingAuthMapper.ensureInitialized();
}
