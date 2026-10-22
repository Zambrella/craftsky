import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/service_status_router_config.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/service_status/widgets/service_status_lifecycle_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:package_info_plus/package_info_plus.dart';

// Dedicated executable fixture: production runtime version/controller/actions,
// Simulated signed-in, onboarded audience, no sessions/API identity.
// The external runner survives reload.
const _compiledBuild = String.fromEnvironment('STATUS_HARNESS_COMPILED_BUILD');
final _buildMarker = FutureProvider<String>((ref) async {
  final info = await PackageInfo.fromPlatform();
  return '${info.version}+${info.buildNumber}';
});
void main() {
  WidgetsFlutterBinding.ensureInitialized();
  usePathUrlStrategy();
  SemanticsBinding.instance.ensureSemantics();
  runApp(
    ProviderScope(
      overrides: [statusMessageAudienceReadyProvider.overrideWithValue(true)],
      child: const ServiceStatusLifecycleHost(child: _Harness()),
    ),
  );
}

class _Harness extends ConsumerStatefulWidget {
  const new();
  @override
  ConsumerState<_Harness> createState() => _HarnessState();
}

class _HarnessState extends ConsumerState<_Harness> {
  final _router = GoRouter(
    routes: [
      GoRoute(path: '/', builder: (_, _) => const _Content()),
      GoRoute(path: '/journey', builder: (_, _) => const _Content()),
    ],
  );
  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp.router(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    routerConfig: serviceStatusRouterConfig(
      _router,
      () => ref.read(serviceStatusControllerProvider.notifier).blockingActive,
    ),
    builder: (_, child) => ServiceStatusHost(child: child!),
  );
}

class _Content extends ConsumerWidget {
  const new();
  @override
  Widget build(BuildContext context, WidgetRef ref) => Scaffold(
    body: Column(
      children: [
        const Text(
          'Compiled build $_compiledBuild',
        ),
        Text('Running version ${ref.watch(_buildMarker).value ?? 'pending'}'),
        TextButton(
          onPressed: () =>
              ref.read(serviceStatusControllerProvider.notifier).refresh(),
          child: const Text('Check status'),
        ),
        const TextField(
          decoration: InputDecoration(labelText: 'Disposable editor'),
        ),
      ],
    ),
  );
}
