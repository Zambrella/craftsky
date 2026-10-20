import 'dart:io';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:media_kit/media_kit.dart';
import 'package:patrol/patrol.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../modules/api_clients.dart';
import '../modules/modules.dart';
import '../modules/system.dart';

void testApp(
  String description,
  Future<void> Function(Modules modules, System system, ApiClients api) body,
) {
  patrolTest(
    description,
    ($) async {
      if (!const bool.fromEnvironment('E2E_LIVE')) {
        throw StateError(
          'Run the live journey with just app-test-e2e <device>.',
        );
      }
      initializeMappers();
      MediaKit.ensureInitialized();
      SharedPreferences.setPrefix('craftsky.e2e.');
      const testStorage = FlutterSecureStorage(
        iOptions: IOSOptions(accountName: 'craftsky.patrol.e2e'),
      );
      if (Platform.isIOS) {
        // Keychain survives uninstall. Reset only the E2E session namespace.
        await testStorage.delete(key: SecureSessionRegistryStorage.storageKey);
      }
      final container = ProviderContainer(
        retry: appProviderRetry,
        overrides: [
          if (Platform.isIOS)
            secureSessionRegistryStorageProvider.overrideWith(
              (ref) => SecureSessionRegistryStorage(testStorage),
            ),
        ],
      );
      try {
        await container.read(deviceIdProvider.future);
        // Use real auth storage, router, clients and plugins. Push delivery,
        // billing and remote crash reporting are outside this smoke journey.
        await $.pumpWidget(
          UncontrolledProviderScope(container: container, child: const App()),
        );
        await body(Modules($), System($), ApiClients(container));
      } finally {
        container.dispose();
      }
    },
    config: const PatrolTesterConfig(
      printLogs: true,
      visibleTimeout: Duration(seconds: 60),
    ),
    platformAutomatorConfig: PlatformAutomatorConfig.fromOptions(
      findTimeout: const Duration(seconds: 60),
      connectionTimeout: const Duration(seconds: 90),
      logger: (_) {},
    ),
  );
}
