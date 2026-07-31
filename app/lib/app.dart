import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_logging.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/widgets/active_account_initialization_gate.dart';
import 'package:craftsky_app/initialization_error_screen.dart';
import 'package:craftsky_app/initialization_loading_screen.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/providers/app_language_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_effect_host.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/text_scale_factor_clamper.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

final _log = Logger('App');

class App extends ConsumerStatefulWidget {
  const App({super.key});

  @override
  ConsumerState<App> createState() => _AppState();
}

class _AppState extends ConsumerState<App> {
  ProviderSubscription<AsyncValue<ActiveAccountInitialization?>>?
  _coldStartAccountInitialization;
  bool _coldStartComplete = false;
  bool _hasBuilt = false;

  @override
  void initState() {
    super.initState();
    final subscription = ref.listenManual(
      activeAccountInitializationProvider,
      _onAccountInitializationChanged,
    );
    _coldStartAccountInitialization = subscription;
    _onAccountInitializationChanged(null, subscription.read());
  }

  void _onAccountInitializationChanged(
    AsyncValue<ActiveAccountInitialization?>? previous,
    AsyncValue<ActiveAccountInitialization?> next,
  ) {
    if (_coldStartComplete || next is AsyncLoading) return;

    if (next case AsyncError()) {
      logActiveAccountInitializationFailure();
    }
    _coldStartComplete = true;
    _coldStartAccountInitialization?.close();
    _coldStartAccountInitialization = null;
    if (_hasBuilt && mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    _hasBuilt = true;
    // Log init failures once per transition into error, not on every rebuild.
    ref.listen(appDependenciesProvider, (prev, next) {
      if (next case AsyncError(:final error, :final stackTrace)) {
        _log.severe('App dependencies failed to initialize', error, stackTrace);
      }
    });

    final depsAsync = ref.watch(appDependenciesProvider);

    return switch (depsAsync) {
      AsyncData() when _coldStartComplete => const _ReadyApp(),
      AsyncData() => const _LoadingApp(),
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
    final appLanguage = ref.watch(appLanguageProvider);

    return MessengerScope(
      messenger: defaultAppMessenger,
      child: MaterialApp.router(
        scaffoldMessengerKey: appScaffoldMessengerKey,
        onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
        theme: AppTheme.lightThemeData,
        darkTheme: AppTheme.darkThemeData,
        themeMode: themeMode,
        debugShowCheckedModeBanner: false,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: appLanguage,
        routerConfig: router,
        builder: (context, child) {
          return TextScaleFactorClamper(
            child: FormFactorWidget(
              child: NotificationEffectHost(
                child: ActiveAccountInitializationGate(
                  child: child ?? const SizedBox.shrink(),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

class _LoadingApp extends StatelessWidget {
  const _LoadingApp();

  @override
  Widget build(BuildContext context) {
    return MessengerScope(
      messenger: defaultAppMessenger,
      child: MaterialApp(
        scaffoldMessengerKey: appScaffoldMessengerKey,
        debugShowCheckedModeBanner: false,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const InitializationLoadingScreen(),
      ),
    );
  }
}

class _ErrorApp extends ConsumerWidget {
  const _ErrorApp({required this.error});

  final Object error;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MessengerScope(
      messenger: defaultAppMessenger,
      child: MaterialApp(
        scaffoldMessengerKey: appScaffoldMessengerKey,
        debugShowCheckedModeBanner: false,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: InitializationErrorScreen(
          error: error,
          onRetry: () => ref.invalidate(appDependenciesProvider),
        ),
      ),
    );
  }
}
