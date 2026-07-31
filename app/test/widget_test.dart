import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'fakes/auth_session_fakes.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  SessionRegistry value = SessionRegistry.empty();

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  late SharedPreferences prefs;

  setUp(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    SharedPreferences.setMockInitialValues({});
    prefs = await SharedPreferences.getInstance();
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

  testWidgets('App boots unauthenticated and lands on WelcomePage', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appDependenciesProvider.overrideWith((ref) async => stubDeps()),
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(),
          ),
        ],
        child: const App(),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.byType(WelcomePage), findsOneWidget);
  });
}
