import 'dart:async';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
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

final class ProviderLogger extends ProviderObserver {
  const ProviderLogger();

  static final _log = Logger('ProviderLogger');

  @override
  void didUpdateProvider(
    ProviderObserverContext context,
    Object? previousValue,
    Object? newValue,
  ) {
    _log.fine(
      'provider updated: '
      'provider=${context.provider}, '
      'previousValue=$previousValue, '
      'newValue=$newValue, '
      'mutation=${context.mutation}',
    );
  }

  @override
  void providerDidFail(
    ProviderObserverContext context,
    Object error,
    StackTrace stackTrace,
  ) {
    _log.warning(
      'provider failed: '
      'provider=${context.provider}, '
      'mutation=${context.mutation}',
      error,
      stackTrace,
    );
  }
}

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
      const ProviderScope(observers: [ProviderLogger()], child: App()),
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
  //
  // Also eagerly resolve deviceIdProvider so the first /v1/* request
  // has the header. The server enforces X-Craftsky-Device-Id on all
  // authenticated routes; if the provider hasn't resolved by the time
  // the session Dio fires its first request, the server 400s. The
  // eager await here guarantees the future is hot before runApp.
  final probe = ProviderContainer(observers: const [ProviderLogger()]);
  try {
    probe.read(dioProvider);
    await probe.read(deviceIdProvider.future);
  } finally {
    probe.dispose();
  }

  _log.fine('bootstrap complete');

  runApp(
    const ProviderScope(observers: [ProviderLogger()], child: App()),
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
  PostMapper.ensureInitialized();
  PostCommentSectionMapper.ensureInitialized();
  PostPageMapper.ensureInitialized();
  UserPostsStateMapper.ensureInitialized();
  InteractionWriteResponseMapper.ensureInitialized();
}
