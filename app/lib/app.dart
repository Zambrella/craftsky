import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/text_scale_factor_clamper.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

final _log = Logger('App');

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Log init failures once per transition into error, not on every rebuild.
    ref.listen(appDependenciesProvider, (prev, next) {
      if (next case AsyncError(:final error, :final stackTrace)) {
        _log.severe('App dependencies failed to initialize', error, stackTrace);
      }
    });

    final depsAsync = ref.watch(appDependenciesProvider);

    return switch (depsAsync) {
      AsyncData() => const _ReadyApp(),
      AsyncError(:final error) => _ErrorApp(error: error),
      _ => const _LoadingApp(),
    };
  }
}

class _ReadyApp extends ConsumerWidget {
  const _ReadyApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // NOTE: Riverpod codegen strips the `Notifier` suffix from class-based
    // notifiers, so the generated provider is `themeModeProvider`, not
    // `themeModeNotifierProvider`.
    final themeMode = ref.watch(themeModeProvider);
    final router = ref.watch(goRouterProvider);

    return MaterialApp.router(
      title: 'Craftsky',
      theme: AppTheme.lightThemeData,
      darkTheme: AppTheme.darkThemeData,
      themeMode: themeMode,
      debugShowCheckedModeBanner: false,
      routerConfig: router,
      builder: (context, child) {
        return TextScaleFactorClamper(
          child: FormFactorWidget(
            child: child ?? const SizedBox.shrink(),
          ),
        );
      },
    );
  }
}

class _LoadingApp extends StatelessWidget {
  const _LoadingApp();

  @override
  Widget build(BuildContext context) {
    return const MaterialApp(
      debugShowCheckedModeBanner: false,
      home: InitializationLoadingScreen(),
    );
  }
}

class _ErrorApp extends ConsumerWidget {
  const _ErrorApp({required this.error});

  final Object error;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      home: InitializationErrorScreen(
        error: error,
        onRetry: () => ref.invalidate(appDependenciesProvider),
      ),
    );
  }
}

class InitializationLoadingScreen extends StatelessWidget {
  const InitializationLoadingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: CircularProgressIndicator()),
    );
  }
}

class InitializationErrorScreen extends StatelessWidget {
  const InitializationErrorScreen({
    required this.error,
    required this.onRetry,
    super.key,
  });

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.error_outline, color: theme.colorScheme.error, size: 64),
              const SizedBox(height: 16),
              Text(
                'Initialization Failed',
                style: theme.textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              Text(
                error.toString(),
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium,
              ),
              const SizedBox(height: 24),
              ElevatedButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh),
                label: const Text('Retry'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
