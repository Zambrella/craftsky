import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/router/home_page.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:shared_preferences/shared_preferences.dart';

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

  testWidgets('App boots and renders HomePage', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appDependenciesProvider.overrideWith((ref) async => stubDeps()),
        ],
        child: const App(),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.byType(HomePage), findsOneWidget);
    // These two assertions between them prove the AppLocalizations delegate
    // resolved and that both static and parameterized keys render: the
    // subtitle is a static string only reachable through `l10n.homeSubtitle`,
    // and the version label is built via `l10n.homeVersionLabel(version)`.
    expect(find.text('Scaffold ready'), findsOneWidget);
    expect(find.text('v1.0.0'), findsOneWidget);
  });
}
